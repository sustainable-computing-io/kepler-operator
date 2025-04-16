package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components/exporter"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components/modelserver"
	"github.com/sustainable.computing.io/kepler-operator/pkg/reconciler"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	secv1 "github.com/openshift/api/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"
)

// KeplerInternalReconciler reconciles a Kepler object
type KeplerInternalReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	logger logr.Logger
}

// common to all components deployed by operator
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services;configmaps;serviceaccounts;persistentvolumeclaims,verbs=list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=*,verbs=*

// RBAC for running Kepler exporter
//+kubebuilder:rbac:groups=apps,resources=daemonsets;deployments,verbs=list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=list;watch
//+kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=list;watch;create;update;patch;delete;use
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors;prometheusrules,verbs=list;watch;create;update;patch;delete

// RBAC required by Kepler exporter
//+kubebuilder:rbac:groups=core,resources=nodes/metrics;nodes/proxy;nodes/stats,verbs=get;list;watch

// SetupWithManager sets up the controller with the Manager.
func (r *KeplerInternalReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// We only want to trigger a reconciliation when the generation
	// of a child changes. Until we need to update our the status for our own objects,
	// we can save CPU cycles by avoiding reconciliations triggered by
	// child status changes.
	//
	// TODO: consider using ResourceVersionChanged predicate for resources that support it

	genChanged := builder.WithPredicates(predicate.GenerationChangedPredicate{})

	c := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.KeplerInternal{}).
		Owns(&corev1.ConfigMap{}, genChanged).
		Owns(&corev1.ServiceAccount{}, genChanged).
		Owns(&corev1.Service{}, genChanged).
		Owns(&appsv1.DaemonSet{}, builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Owns(&rbacv1.ClusterRoleBinding{}, genChanged).
		Owns(&rbacv1.ClusterRole{}, genChanged)

	c = c.Watches(&corev1.Secret{},
		handler.EnqueueRequestsFromMapFunc(r.mapSecretToRequests),
		builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
	)

	if Config.Cluster == k8s.OpenShift {
		c = c.Owns(&secv1.SecurityContextConstraints{}, genChanged)
	}
	return c.Complete(r)
}

// mapSecretToRequests returns the reconcile requests for kepler-internal objects for which an associated redfish secret has been changed.
func (r *KeplerInternalReconciler) mapSecretToRequests(ctx context.Context, object client.Object) []reconcile.Request {
	secret, ok := object.(*corev1.Secret)
	if !ok {
		return nil
	}

	ks := v1alpha1.KeplerInternalList{}
	if err := r.List(ctx, &ks); err != nil {
		return nil
	}

	requests := []reconcile.Request{}
	for _, ki := range ks.Items {
		ex := ki.Spec.Exporter

		if ex.Redfish == nil {
			continue
		}

		if ex.Redfish.SecretRef == secret.GetName() &&
			ex.Deployment.Namespace == secret.GetNamespace() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: ki.ObjectMeta.Name, Namespace: ki.ObjectMeta.Namespace},
			})
		}
	}
	return requests
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Kepler object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *KeplerInternalReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	r.logger = logger

	logger.Info("Start of reconcile")
	defer logger.Info("End of reconcile")

	ki, err := r.getInternal(ctx, req)
	if err != nil {
		// retry since some error has occurred
		logger.V(6).Info("Get Error ", "error", err)
		return ctrl.Result{}, err
	}

	if ki == nil {
		// no kepler-internal found , so stop here
		logger.V(6).Info("Kepler Nil")
		return ctrl.Result{}, nil
	}

	logger.V(6).Info("Running sub reconcilers", "kepler-internal", ki.Spec)

	result, recErr := r.runReconcilers(ctx, ki)
	updateErr := r.updateStatus(ctx, req, recErr)

	if recErr != nil {
		return result, recErr
	}
	return result, updateErr
}

func (r *KeplerInternalReconciler) runReconcilers(ctx context.Context, ki *v1alpha1.KeplerInternal) (ctrl.Result, error) {
	reconcilers := r.reconcilersForInternal(ki)
	r.logger.V(6).Info("reconcilers ...", "count", len(reconcilers))

	return reconciler.Runner{
		Reconcilers: reconcilers,
		Client:      r.Client,
		Scheme:      r.Scheme,
		Logger:      r.logger,
	}.Run(ctx)
}

func (r *KeplerInternalReconciler) getInternal(ctx context.Context, req ctrl.Request) (*v1alpha1.KeplerInternal, error) {
	logger := r.logger.WithValues("keplerinternal", req.Name)
	ki := v1alpha1.KeplerInternal{}

	if err := r.Client.Get(ctx, req.NamespacedName, &ki); err != nil {
		if errors.IsNotFound(err) {
			logger.V(3).Info("keplerinternal could not be found; may be marked for deletion")
			return nil, nil
		}
		logger.Error(err, "failed to get keplerinternal")
		return nil, err
	}

	return &ki, nil
}

func (r *KeplerInternalReconciler) updateStatus(ctx context.Context, req ctrl.Request, recErr error) error {
	logger := r.logger.WithValues("keplerinternal", req.Name, "action", "update-status")
	logger.V(3).Info("Start of status update")
	defer logger.V(3).Info("End of status update")

	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		ki, _ := r.getInternal(ctx, req)
		// may be deleted
		if ki == nil || !ki.GetDeletionTimestamp().IsZero() {
			// retry since some error has occurred
			r.logger.V(6).Info("Reconcile has deleted kepler-internal; skipping status update")
			return nil
		}

		// sanitize the conditions so that all types are present and the order is predictable
		ki.Status.Exporter.Conditions = sanitizeConditions(ki.Status.Exporter.Conditions)

		{
			now := metav1.Now()
			reconciledChanged := r.updateReconciledStatus(ctx, ki, recErr, now)
			availableChanged := r.updateAvailableStatus(ctx, ki, recErr, now)
			logger.V(6).Info("conditions updated", "reconciled", reconciledChanged, "available", availableChanged)

			if !reconciledChanged && !availableChanged {
				logger.V(6).Info("no changes to existing status; skipping update")
				return nil
			}
		}

		return r.Client.Status().Update(ctx, ki)
	})
}

func sanitizeConditions(conditions []v1alpha1.Condition) []v1alpha1.Condition {
	required := map[v1alpha1.ConditionType]bool{
		v1alpha1.Reconciled: false,
		v1alpha1.Available:  false,
	}

	if len(conditions) == len(required) {
		return conditions
	}

	if len(conditions) == 0 {
		return []v1alpha1.Condition{{
			Type:   v1alpha1.Reconciled,
			Status: v1alpha1.ConditionFalse,
		}, {
			Type:   v1alpha1.Available,
			Status: v1alpha1.ConditionFalse,
		}}
	}

	for _, c := range conditions {
		required[c.Type] = true
	}

	for t, present := range required {
		if !present {
			conditions = append(conditions, v1alpha1.Condition{
				Type:   t,
				Status: v1alpha1.ConditionFalse,
			})
		}
	}
	return conditions
}

func (r *KeplerInternalReconciler) updateReconciledStatus(ctx context.Context, ki *v1alpha1.KeplerInternal, recErr error, time metav1.Time) bool {
	reconciled := v1alpha1.Condition{
		Type:               v1alpha1.Reconciled,
		Status:             v1alpha1.ConditionTrue,
		ObservedGeneration: ki.Generation,
		Reason:             v1alpha1.ReconcileComplete,
		Message:            "Reconcile succeeded",
	}

	if recErr != nil {
		reconciled.Status = v1alpha1.ConditionFalse
		reconciled.Reason = v1alpha1.ReconcileError
		reconciled.Message = recErr.Error()
	}

	return updateCondition(ki.Status.Exporter.Conditions, reconciled, time)
}

func findCondition(conditions []v1alpha1.Condition, t v1alpha1.ConditionType) *v1alpha1.Condition {
	for i, c := range conditions {
		if c.Type == t {
			return &conditions[i]
		}
	}
	return nil
}

// returns true if the condition has been updated
func updateCondition(conditions []v1alpha1.Condition, latest v1alpha1.Condition, time metav1.Time) bool {
	old := findCondition(conditions, latest.Type)
	if old == nil {
		panic("old condition not found; this should never happen after sanitizeConditions")
	}

	if old.ObservedGeneration == latest.ObservedGeneration &&
		old.Status == latest.Status &&
		old.Reason == latest.Reason &&
		old.Message == latest.Message {
		return false
	}

	old.ObservedGeneration = latest.ObservedGeneration
	old.Status = latest.Status
	old.Reason = latest.Reason
	old.Message = latest.Message
	// NOTE: last transition time changes only if the status changes
	old.LastTransitionTime = time
	return true
}

func (r *KeplerInternalReconciler) updateAvailableStatus(ctx context.Context, ki *v1alpha1.KeplerInternal, recErr error, time metav1.Time) bool {
	// get daemonset owned by kepler
	dset := appsv1.DaemonSet{}
	key := types.NamespacedName{Name: ki.DaemonsetName(), Namespace: ki.Namespace()}
	if err := r.Client.Get(ctx, key, &dset); err != nil {
		return updateCondition(ki.Status.Exporter.Conditions, availableConditionForGetError(err), time)
	}

	ds := dset.Status
	ki.Status.Exporter.NumberMisscheduled = ds.NumberMisscheduled
	ki.Status.Exporter.CurrentNumberScheduled = ds.CurrentNumberScheduled
	ki.Status.Exporter.DesiredNumberScheduled = ds.DesiredNumberScheduled
	ki.Status.Exporter.NumberReady = ds.NumberReady
	ki.Status.Exporter.UpdatedNumberScheduled = ds.UpdatedNumberScheduled
	ki.Status.Exporter.NumberAvailable = ds.NumberAvailable
	ki.Status.Exporter.NumberUnavailable = ds.NumberUnavailable

	available := availableCondition(&dset)

	if recErr == nil {
		available.ObservedGeneration = ki.Generation
	} else {
		// failure to reconcile is a Degraded condition
		available.Status = v1alpha1.ConditionDegraded
		available.Reason = v1alpha1.ReconcileError
	}

	updated := updateCondition(ki.Status.Exporter.Conditions, available, time)

	estimatorStatus := v1alpha1.EstimatorStatus{
		Status: v1alpha1.DeploymentNotInstalled,
	}
	if ki.Spec.Estimator != nil && len(dset.Spec.Template.Spec.Containers) > 1 {
		// estimator enabled and has a sidecar container
		estimatorStatus.Status = v1alpha1.DeploymentNotReady
		if ds.NumberReady == ds.DesiredNumberScheduled {
			estimatorStatus.Status = v1alpha1.DeploymentRunning
		}
	}

	ki.Status.Estimator = estimatorStatus

	// update model server status
	modelServerStatus := v1alpha1.ModelServerStatus{
		Status: v1alpha1.DeploymentNotInstalled,
	}
	if ki.Spec.ModelServer != nil {
		key = types.NamespacedName{Name: ki.ModelServerDeploymentName(), Namespace: ki.Namespace()}
		deploy := appsv1.Deployment{}
		if err := r.Client.Get(ctx, key, &deploy); err == nil {
			modelServerStatus.Status = v1alpha1.DeploymentNotReady
			if deploy.Status.ReadyReplicas > 0 {
				modelServerStatus.Status = v1alpha1.DeploymentRunning
			}
		}
	}
	ki.Status.ModelServer = modelServerStatus
	return updated
}

func availableConditionForGetError(err error) v1alpha1.Condition {
	if errors.IsNotFound(err) {
		return v1alpha1.Condition{
			Type:    v1alpha1.Available,
			Status:  v1alpha1.ConditionFalse,
			Reason:  v1alpha1.DaemonSetNotFound,
			Message: err.Error(),
		}
	}

	return v1alpha1.Condition{
		Type:    v1alpha1.Available,
		Status:  v1alpha1.ConditionUnknown,
		Reason:  v1alpha1.DaemonSetError,
		Message: err.Error(),
	}
}

func availableCondition(dset *appsv1.DaemonSet) v1alpha1.Condition {
	ds := dset.Status
	dsName := dset.Namespace + "/" + dset.Name

	if gen, ogen := dset.Generation, ds.ObservedGeneration; gen > ogen {
		return v1alpha1.Condition{
			Type:   v1alpha1.Available,
			Status: v1alpha1.ConditionUnknown,
			Reason: v1alpha1.DaemonSetOutOfSync,
			Message: fmt.Sprintf(
				"Generation %d of kepler daemonset %q is out of sync with the observed generation: %d",
				gen, dsName, ogen),
		}
	}

	c := v1alpha1.Condition{Type: v1alpha1.Available}

	// NumberReady: The number of nodes that should be running the daemon pod and
	// have one or more of the daemon pod running with a Ready Condition.
	//
	// DesiredNumberScheduled: The total number of nodes that should be running
	// the daemon pod (including nodes correctly running the daemon pod).
	if ds.NumberReady == 0 || ds.DesiredNumberScheduled == 0 {
		c.Status = v1alpha1.ConditionFalse
		c.Reason = v1alpha1.DaemonSetPodsNotRunning
		c.Message = fmt.Sprintf("Kepler daemonset %q is not rolled out to any node; check nodeSelector and tolerations", dsName)
		return c
	}

	// UpdatedNumberScheduled: The total number of nodes that are running updated daemon pod
	//
	// DesiredNumberScheduled: The total number of nodes that should be running
	// the daemon pod (including nodes correctly running the daemon pod).

	if ds.UpdatedNumberScheduled < ds.DesiredNumberScheduled {
		c.Status = v1alpha1.ConditionUnknown
		c.Reason = v1alpha1.DaemonSetRolloutInProgress
		c.Message = fmt.Sprintf(
			"Waiting for kepler daemonset %q rollout to finish: %d out of %d new pods have been updated",
			dsName, ds.UpdatedNumberScheduled, ds.DesiredNumberScheduled)
		return c
	}

	// NumberAvailable: The number of nodes that should be running the daemon pod
	// and have one or more of the daemon pod running and available (ready for at
	// least spec.minReadySeconds)

	if ds.NumberAvailable < ds.DesiredNumberScheduled {
		c.Status = v1alpha1.ConditionUnknown
		c.Reason = v1alpha1.DaemonSetPartiallyAvailable
		c.Message = fmt.Sprintf("Rollout of kepler daemonset %q is in progress: %d of %d updated pods are available",
			dsName, ds.NumberAvailable, ds.DesiredNumberScheduled)
		return c
	}

	// NumberUnavailable:  The number of nodes that should be running the daemon
	// pod and have none of the daemon pod running and available (ready for at
	// least spec.minReadySeconds)
	if ds.NumberUnavailable > 0 {
		c.Status = v1alpha1.ConditionFalse
		c.Reason = v1alpha1.DaemonSetPartiallyAvailable
		c.Message = fmt.Sprintf("Waiting for kepler daemonset %q to rollout on %d nodes", dsName, ds.NumberUnavailable)
		return c
	}

	c.Status = v1alpha1.ConditionTrue
	c.Reason = v1alpha1.DaemonSetReady
	c.Message = fmt.Sprintf("Kepler daemonset %q is deployed to all nodes and available; ready %d/%d",
		dsName, ds.NumberReady, ds.DesiredNumberScheduled)
	return c
}

func (r *KeplerInternalReconciler) reconcilersForInternal(ki *v1alpha1.KeplerInternal) []reconciler.Reconciler {
	rs := []reconciler.Reconciler{}

	cleanup := !ki.DeletionTimestamp.IsZero()
	if !cleanup {
		// NOTE: create namespace first and for deletion, reverse the order
		rs = append(rs, reconciler.Updater{
			Owner:    ki,
			Resource: components.NewNamespace(ki.Namespace()),
			OnError:  reconciler.Requeue,
			Logger:   r.logger,
		})
	}

	if ki.Spec.Estimator != nil {
		if ki.Spec.Estimator.Image == "" {
			ki.Spec.Estimator.Image = InternalConfig.EstimatorImage
		}
	}

	rs = append(rs, exporterReconcilers(ki, Config.Cluster)...)

	if ki.Spec.ModelServer != nil && ki.Spec.ModelServer.Enabled {
		if ki.Spec.ModelServer.Image == "" {
			ki.Spec.ModelServer.Image = InternalConfig.ModelServerImage
		}
		reconcilers, err := modelServerInternalReconcilers(ki)
		if err != nil {
			r.logger.Info(fmt.Sprintf("cannot init model server reconciler from config: %v", err))
		} else {
			rs = append(rs, reconcilers...)
		}
	}

	if cleanup {
		rs = append(rs, reconciler.Deleter{
			OnError:     reconciler.Requeue,
			Resource:    components.NewNamespace(ki.Namespace()),
			WaitTimeout: 2 * time.Minute,
		})
	}

	// WARN: only run finalizer if theren't any errors
	// this bug 🐛 must be FIXED
	rs = append(rs, reconciler.Finalizer{
		Resource:  ki,
		Finalizer: Finalizer,
		Logger:    r.logger,
	})
	return rs
}

func exporterReconcilers(ki *v1alpha1.KeplerInternal, cluster k8s.Cluster) []reconciler.Reconciler {
	if cleanup := !ki.DeletionTimestamp.IsZero(); cleanup {
		rs := resourceReconcilers(
			deleteResource,
			// cluster-scoped
			exporter.NewClusterRoleBinding(components.Metadata, ki),
			exporter.NewClusterRole(components.Metadata, ki),
		)
		rs = append(rs, resourceReconcilers(deleteResource, openshiftNamespacedResources(ki, cluster)...)...)
		return rs
	}

	updateResource := newUpdaterWithOwner(ki)
	// cluster-scoped resources first
	rs := resourceReconcilers(updateResource,
		exporter.NewClusterRole(components.Full, ki),
		exporter.NewClusterRoleBinding(components.Full, ki),
	)
	rs = append(rs, resourceReconcilers(updateResource, openshiftClusterResources(ki, cluster)...)...)

	// namespace scoped
	rs = append(rs, resourceReconcilers(updateResource,
		exporter.NewServiceAccount(ki),
		exporter.NewService(ki),
		exporter.NewServiceMonitor(ki),
		exporter.NewPrometheusRule(ki),
	)...)

	if ki.Spec.Exporter.Redfish == nil {
		rs = append(rs, resourceReconcilers(updateResource,
			exporter.NewDaemonSet(components.Full, ki),
			exporter.NewConfigMap(components.Full, ki),
		)...)
	} else {
		rs = append(rs,
			reconciler.KeplerReconciler{
				Ki: ki,
				Ds: exporter.NewDaemonSet(components.Full, ki),
			},
			reconciler.KeplerConfigMapReconciler{
				Ki:  ki,
				Cfm: exporter.NewConfigMap(components.Full, ki),
			},
		)
	}
	rs = append(rs, resourceReconcilers(updateResource, openshiftNamespacedResources(ki, cluster)...)...)
	return rs
}

func openshiftClusterResources(ki *v1alpha1.KeplerInternal, cluster k8s.Cluster) []client.Object {
	oshift := ki.Spec.OpenShift
	if cluster != k8s.OpenShift || !oshift.Enabled {
		return nil
	}
	// NOTE: SCC is required for kepler deployment even if openshift is not enabled
	return []client.Object{
		exporter.NewSCC(components.Full, ki),
	}
}

func openshiftNamespacedResources(ki *v1alpha1.KeplerInternal, cluster k8s.Cluster) []client.Object {
	oshift := ki.Spec.OpenShift

	if cluster != k8s.OpenShift || !oshift.Enabled {
		return nil
	}

	res := []client.Object{}
	if oshift.Dashboard.Enabled {
		res = append(res,
			exporter.NewOverviewDashboard(components.Full),
			exporter.NewNamespaceInfoDashboard(components.Full),
		)
	}
	return res
}

func modelServerInternalReconcilers(ki *v1alpha1.KeplerInternal) ([]reconciler.Reconciler, error) {
	ms := ki.Spec.ModelServer
	msName := ki.ModelServerDeploymentName()
	namespace := ki.Spec.Exporter.Deployment.Namespace
	cm := modelserver.NewConfigMap(msName, components.Full, ms, namespace)
	deploy := modelserver.NewDeployment(msName, ms, namespace)
	svc := modelserver.NewService(msName, ms, namespace)

	resources := []client.Object{cm, deploy, svc}

	if ms.Storage.PersistentVolumeClaim != nil {
		pvc := modelserver.NewPVC(msName, namespace, ms.Storage.PersistentVolumeClaim)
		resources = append(resources, pvc)
	}

	rs := updatersForInternalResources(ki, resources...)
	return rs, nil
}

func updatersForInternalResources(ki *v1alpha1.KeplerInternal, resources ...client.Object) []reconciler.Reconciler {
	rs := []reconciler.Reconciler{}
	resourceUpdater := newUpdaterWithOwner(ki)
	for _, res := range resources {
		rs = append(rs, resourceUpdater(res))
	}
	return rs
}
