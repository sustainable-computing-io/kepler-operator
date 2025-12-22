// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"slices"

	"github.com/go-logr/logr"
	secv1 "github.com/openshift/api/security/v1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	powermonitor "github.com/sustainable.computing.io/kepler-operator/pkg/components/power-monitor"
	"github.com/sustainable.computing.io/kepler-operator/pkg/reconciler"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	ctrl "sigs.k8s.io/controller-runtime"
)

// KeplerInternalReconciler reconciles a Kepler object
type PowerMonitorInternalReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

const (
	configMapField         = ".spec.kepler.config.additionalConfigMaps.name"
	deploymentSecretsField = ".spec.kepler.deployment.secrets.name"
)

// common to all components deployed by operator
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services;configmaps;serviceaccounts;persistentvolumeclaims,verbs=list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=serviceaccounts/token,verbs=create
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=*,verbs=*

// RBAC for running Kepler exporter
//+kubebuilder:rbac:groups=apps,resources=daemonsets;deployments,verbs=list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=list;watch;create;update;patch;delete;use
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors;prometheusrules,verbs=list;watch;create;update;patch;delete

// RBAC required by Kepler exporter
//+kubebuilder:rbac:groups=core,resources=nodes/metrics;nodes/proxy;nodes/stats,verbs=get;list;watch

// indexAdditonalConfigmaps sets up indexer for PowerMonitorInternal based on referenced ConfigMaps
func indexAdditonalConfigmaps(mgr ctrl.Manager, logger logr.Logger) error {
	return mgr.GetFieldIndexer().IndexField(context.Background(),
		&v1alpha1.PowerMonitorInternal{},
		configMapField,

		func(obj client.Object) []string {
			pmi, ok := obj.(*v1alpha1.PowerMonitorInternal)
			if !ok {
				logger.Info("failed to cast object to PowerMonitorInternal", "object", obj.GetName())
				return nil
			}
			var keys []string
			for _, cm := range pmi.Spec.Kepler.Config.AdditionalConfigMaps {
				keys = append(keys, cm.Name)
			}
			return keys
		})
}

// indexDeploymentSecrets sets up indexer for PowerMonitorInternal based on referenced Secrets
func indexDeploymentSecrets(mgr ctrl.Manager, logger logr.Logger) error {
	return mgr.GetFieldIndexer().IndexField(context.Background(),
		&v1alpha1.PowerMonitorInternal{},
		deploymentSecretsField,

		func(obj client.Object) []string {
			pmi, ok := obj.(*v1alpha1.PowerMonitorInternal)
			if !ok {
				logger.Info("failed to cast object to PowerMonitorInternal", "object", obj.GetName())
				return nil
			}
			var keys []string
			for _, sec := range pmi.Spec.Kepler.Deployment.Secrets {
				keys = append(keys, sec.Name)
			}
			return keys
		})
}

// SetupWithManager sets up the controller with the Manager.
func (r *PowerMonitorInternalReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := indexAdditonalConfigmaps(mgr, r.logger); err != nil {
		r.logger.Error(err, "failed to set up index for PowerMonitorInternal additionalConfigMaps")
		return err
	}

	if err := indexDeploymentSecrets(mgr, r.logger); err != nil {
		r.logger.Error(err, "failed to set up index for PowerMonitorInternal additionalConfigMaps")
		return err
	}

	// We only want to trigger a reconciliation when the generation
	// of a child changes. Until we need to update our the status for our own objects,
	// we can save CPU cycles by avoiding reconciliations triggered by
	// child status changes.

	genChanged := builder.WithPredicates(predicate.GenerationChangedPredicate{})
	resVerChanged := builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})

	// watch for ConfigMap change events
	configMapHandler := handler.EnqueueRequestsFromMapFunc(r.mapConfigMapToRequests)
	secretHandler := handler.EnqueueRequestsFromMapFunc(r.mapDeploymentSecretsToRequests)

	c := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.PowerMonitorInternal{}).
		Owns(&corev1.ConfigMap{}, genChanged).
		Owns(&corev1.ServiceAccount{}, genChanged).
		Owns(&corev1.Service{}, genChanged).
		Owns(&appsv1.DaemonSet{}, resVerChanged).
		Owns(&monv1.ServiceMonitor{}, genChanged).
		Owns(&rbacv1.ClusterRoleBinding{}, genChanged).
		Owns(&rbacv1.ClusterRole{}, genChanged).
		// NOTE: requires resVerChanged for ConfigMap & Secret since
		// they don't have metadata.generation
		Watches(&corev1.ConfigMap{}, configMapHandler, resVerChanged).
		Watches(&corev1.Secret{}, secretHandler, resVerChanged)

	if Config.Cluster == k8s.OpenShift {
		c = c.Owns(&secv1.SecurityContextConstraints{}, genChanged)
		c = c.Owns(&corev1.Secret{}, genChanged)
		// GenerationChangedPredicate triggers when Spec has changed for the following resources.
		// AnnotationChangedPredicate triggers when Annotations have changed for the following resources.
		// These predicates are used to avoid unnecessary reconciliations from ResourceVersionChangedPredicate.
		c = c.Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.mapSecretToPowerMonitorRequests),
			builder.WithPredicates(
				predicate.GenerationChangedPredicate{},
				predicate.AnnotationChangedPredicate{},
			),
		)
		c = c.Watches(&corev1.ConfigMap{},
			handler.EnqueueRequestsFromMapFunc(r.mapCABundleConfigMapToPowerMonitorRequests),
			builder.WithPredicates(
				predicate.GenerationChangedPredicate{},
				predicate.AnnotationChangedPredicate{},
			),
		)
		c = c.Watches(&corev1.ServiceAccount{},
			handler.EnqueueRequestsFromMapFunc(r.mapServiceAccountToPowerMonitorRequests),
			builder.WithPredicates(
				predicate.GenerationChangedPredicate{},
				predicate.AnnotationChangedPredicate{},
			),
		)
	}
	return c.Complete(r)
}

// mapConfigMapToRequests returns the reconcile requests for power-monitor-internal objects for which an associated ConfigMap has changed
func (r *PowerMonitorInternalReconciler) mapConfigMapToRequests(ctx context.Context, object client.Object) []reconcile.Request {
	configMap, ok := object.(*corev1.ConfigMap)
	if !ok {
		r.logger.Info("failed to cast object to ConfigMap", "object", object.GetName())
		return nil
	}

	pmis := &v1alpha1.PowerMonitorInternalList{}
	err := r.Client.List(ctx, pmis, client.MatchingFields{configMapField: configMap.Name})
	if err != nil {
		r.logger.Error(err, "failed to list objects using index", "indexKey", configMap.Name)
		return nil
	}

	requests := []reconcile.Request{}
	for _, pmi := range pmis.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{Name: pmi.Name},
		})
	}
	return requests
}

// mapDeploymentSecretsToRequests returns the reconcile requests for power-monitor-internal objects for which an associated ConfigMap has changed
func (r *PowerMonitorInternalReconciler) mapDeploymentSecretsToRequests(ctx context.Context, object client.Object) []reconcile.Request {
	secret, ok := object.(*corev1.Secret)
	if !ok {
		r.logger.Info("failed to cast object to Secret", "object", object.GetName())
		return nil
	}

	pmis := &v1alpha1.PowerMonitorInternalList{}
	err := r.Client.List(ctx, pmis, client.MatchingFields{deploymentSecretsField: secret.Name})
	if err != nil {
		r.logger.Error(err, "failed to list objects using index", "indexKey", secret.Name)
		return nil
	}

	requests := []reconcile.Request{}
	r.logger.V(6).Info("pmis found for secret ", "secret", secret.Name, "pmis", len(pmis.Items))
	for _, pmi := range pmis.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{Name: pmi.Name},
		})
	}
	return requests
}

func (r *PowerMonitorInternalReconciler) mapSecretToPowerMonitorRequests(ctx context.Context, object client.Object) []reconcile.Request {
	secret, ok := object.(*corev1.Secret)
	if !ok {
		r.logger.Info("failed to cast object to Secret", "object", object.GetName())
		return nil
	}

	if secret.GetName() != powermonitor.SecretTLSCertName {
		r.logger.V(6).Info("ignoring secret", "name", secret.GetName())
		return nil
	}

	pmis := &v1alpha1.PowerMonitorInternalList{}
	err := r.List(ctx, pmis)
	if err != nil {
		r.logger.Error(err, "failed to list objects using index", "indexKey", secret.Name)
		return nil
	}

	requests := []reconcile.Request{}
	for _, pmi := range pmis.Items {
		enabledSecurity := (pmi.Spec.Kepler.Deployment.Security.Mode == v1alpha1.SecurityModeRBAC)
		if !enabledSecurity {
			continue
		}
		ns := pmi.Spec.Kepler.Deployment.Namespace
		if ns == secret.GetNamespace() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      pmi.Name,
					Namespace: pmi.ObjectMeta.Namespace,
				},
			})
		}
	}
	return requests
}

func (r *PowerMonitorInternalReconciler) mapCABundleConfigMapToPowerMonitorRequests(ctx context.Context, object client.Object) []reconcile.Request {
	configMap, ok := object.(*corev1.ConfigMap)
	if !ok {
		r.logger.Info("failed to cast object to ConfigMap", "object", object.GetName())
		return nil
	}

	if configMap.GetName() != powermonitor.PowerMonitorCertsCABundleName {
		r.logger.V(6).Info("ignoring configmap", "name", configMap.GetName())
		return nil
	}

	pmis := &v1alpha1.PowerMonitorInternalList{}
	err := r.List(ctx, pmis)
	if err != nil {
		r.logger.Error(err, "failed to list objects using index", "indexKey", configMap.Name)
		return nil
	}

	requests := []reconcile.Request{}
	for _, pmi := range pmis.Items {
		enabledSecurity := (pmi.Spec.Kepler.Deployment.Security.Mode == v1alpha1.SecurityModeRBAC)
		if !enabledSecurity {
			continue
		}
		ns := pmi.Spec.Kepler.Deployment.Namespace
		if ns == configMap.GetNamespace() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      pmi.Name,
					Namespace: pmi.ObjectMeta.Namespace,
				},
			})
		}
	}
	return requests
}

func (r *PowerMonitorInternalReconciler) mapServiceAccountToPowerMonitorRequests(ctx context.Context, object client.Object) []reconcile.Request {
	sa, ok := object.(*corev1.ServiceAccount)
	if !ok {
		r.logger.Info("failed to cast object to ServiceAccount", "object", object.GetName())
		return nil
	}

	if sa.GetName() != powermonitor.UWMServiceAccountName || sa.GetNamespace() != powermonitor.UWMNamespace {
		r.logger.V(6).Info("ignoring service account", "name", sa.GetName())
		return nil
	}

	pmis := &v1alpha1.PowerMonitorInternalList{}
	err := r.List(ctx, pmis)
	if err != nil {
		r.logger.Error(err, "failed to list objects using index", "indexKey", sa.Name)
		return nil
	}

	requests := []reconcile.Request{}
	for _, pmi := range pmis.Items {
		enabledSecurity := (pmi.Spec.Kepler.Deployment.Security.Mode == v1alpha1.SecurityModeRBAC)
		if !enabledSecurity {
			continue
		}

		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      pmi.Name,
				Namespace: pmi.ObjectMeta.Namespace,
			},
		})
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
func (r *PowerMonitorInternalReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	r.logger = logger

	logger.Info("Start of reconcile")
	defer logger.Info("End of reconcile")

	pmi, err := r.getPowerMonitorInternal(ctx, req)
	if err != nil {
		// retry since some error has occurred
		logger.V(6).Info("Get Error ", "error", err)
		return ctrl.Result{}, err
	}

	if pmi == nil {
		// no kepler-x found , so stop here
		logger.V(6).Info("power-monitor-internal Nil")
		return ctrl.Result{}, nil
	}

	logger.V(6).Info("Running sub reconcilers", "power-monitor-internal", pmi.Spec)

	result, recErr := r.runPowerMonitorReconcilers(ctx, pmi)
	updateErr := r.updatePowerMonitorStatus(ctx, req, recErr)
	if recErr != nil {
		return result, recErr
	}
	return result, updateErr
}

func (r PowerMonitorInternalReconciler) getPowerMonitorInternal(ctx context.Context, req ctrl.Request) (*v1alpha1.PowerMonitorInternal, error) {
	logger := r.logger.WithValues("power-monitor-internal", req.Name)
	pmi := v1alpha1.PowerMonitorInternal{}

	if err := r.Client.Get(ctx, req.NamespacedName, &pmi); err != nil {
		if errors.IsNotFound(err) {
			logger.V(3).Info("power-monitor-internal could not be found; may be marked for deletion")
			return nil, nil
		}
		logger.Error(err, "failed to get power-monitor-internal")
		return nil, err
	}

	return &pmi, nil
}

func (r PowerMonitorInternalReconciler) runPowerMonitorReconcilers(ctx context.Context, pmi *v1alpha1.PowerMonitorInternal) (ctrl.Result, error) {
	reconcilers, err := r.reconcilersForPowerMonitor(pmi)
	if err != nil {
		return ctrl.Result{}, err
	}
	r.logger.V(6).Info("reconcilers ...", "count", len(reconcilers))

	return reconciler.Runner{
		Reconcilers: reconcilers,
		Client:      r.Client,
		Scheme:      r.Scheme,
		Logger:      r.logger,
	}.Run(ctx)
}

func openshiftPowerMonitorClusterResources(pmi *v1alpha1.PowerMonitorInternal, cluster k8s.Cluster) []client.Object {
	oshift := pmi.Spec.OpenShift
	if cluster != k8s.OpenShift || !oshift.Enabled {
		return nil
	}
	// NOTE: SCC is required for kepler deployment even if openshift is not enabled
	return []client.Object{
		powermonitor.NewPowerMonitorSCC(components.Full, pmi),
	}
}

func openshiftPowerMonitorNamespacedResources(pmi *v1alpha1.PowerMonitorInternal, cluster k8s.Cluster) []client.Object {
	oshift := pmi.Spec.OpenShift

	if cluster != k8s.OpenShift || !oshift.Enabled {
		return nil
	}

	res := []client.Object{}
	if oshift.Dashboard.Enabled {
		res = append(res,
			powermonitor.NewPowerMonitorInfoDashboard(components.Full),
			powermonitor.NewPowerMonitorNamespaceInfoDashboard(components.Full),
		)
	}
	return res
}

func securityPowerMonitorReconcilers(pmi *v1alpha1.PowerMonitorInternal, cluster k8s.Cluster, enableRBAC, enableUWM bool) []reconciler.Reconciler {
	rs := []reconciler.Reconciler{}
	rs = append(rs,
		reconciler.KubeRBACProxyConfigReconciler{
			Pmi:        pmi,
			EnableRBAC: enableRBAC,
			EnableUWM:  enableUWM,
		},
		reconciler.CABundleConfigReconciler{
			Pmi:        pmi,
			EnableRBAC: enableRBAC,
			EnableUWM:  enableUWM,
		},
		reconciler.UWMSecretTokenReconciler{
			Pmi:        pmi,
			Cluster:    cluster,
			EnableRBAC: enableRBAC,
			EnableUWM:  enableUWM,
		},
	)
	return rs
}

func powerMonitorExporters(pmi *v1alpha1.PowerMonitorInternal, ds *appsv1.DaemonSet, cluster k8s.Cluster) ([]reconciler.Reconciler, error) {
	if cleanup := !pmi.DeletionTimestamp.IsZero(); cleanup {
		rs := resourceReconcilers(
			deleteResource,
			// cluster-scoped
			// remove cluster role binding first, then remove cluster role
			powermonitor.NewPowerMonitorClusterRoleBinding(components.Metadata, pmi),
			powermonitor.NewPowerMonitorClusterRole(components.Metadata, pmi),
		)
		rs = append(rs, resourceReconcilers(deleteResource, openshiftPowerMonitorNamespacedResources(pmi, cluster)...)...)
		return rs, nil
	}

	updateResource := newUpdaterWithOwner(pmi)

	// flags to check if rbac and uwm are set
	enableRBAC := pmi.Spec.Kepler.Deployment.Security.Mode == v1alpha1.SecurityModeRBAC
	enableUWM := slices.Contains(
		pmi.Spec.Kepler.Deployment.Security.AllowedSANames,
		fmt.Sprintf("%s:%s", powermonitor.UWMNamespace, powermonitor.UWMServiceAccountName),
	)

	sm := powermonitor.NewPowerMonitorServiceMonitor(components.Full, pmi)

	// cluster-scoped resources first
	// update cluster role before cluster role binding
	rs := resourceReconcilers(updateResource,
		powermonitor.NewPowerMonitorClusterRole(components.Full, pmi),
		powermonitor.NewPowerMonitorClusterRoleBinding(components.Full, pmi),
	)
	rs = append(rs, resourceReconcilers(updateResource, openshiftPowerMonitorClusterResources(pmi, cluster)...)...)

	// kube rbac proxy resources
	rs = append(rs, securityPowerMonitorReconcilers(pmi, cluster, enableRBAC, enableUWM)...)

	// namespace scoped
	rs = append(rs, resourceReconcilers(updateResource,
		powermonitor.NewPowerMonitorServiceAccount(pmi),
		powermonitor.NewPowerMonitorService(pmi),
		// powermonitor.NewPowerMonitorPrometheusRule(kx), prometheus rule is not necessary at the moment
	)...)

	// check that all required objects have been created for kube rbac proxy
	rs = append(rs,
		reconciler.PowerMonitorDeployer{
			Pmi: pmi,
			Ds:  ds,
		},
		reconciler.KubeRBACProxyObjectsChecker{
			Pmi:        pmi,
			Cluster:    cluster,
			Ds:         ds,
			Sm:         sm,
			EnableRBAC: enableRBAC,
			EnableUWM:  enableUWM,
		},
	)

	// deploy daemonset
	rs = append(rs, resourceReconcilers(updateResource, ds)...)

	// deploy service monitor
	rs = append(rs,
		reconciler.PowerMonitorServiceMonitorReconciler{
			Pmi:        pmi,
			Sm:         sm,
			EnableRBAC: enableRBAC,
			EnableUWM:  enableUWM,
		},
	)

	rs = append(rs, resourceReconcilers(updateResource, openshiftPowerMonitorNamespacedResources(pmi, cluster)...)...)
	return rs, nil
}

func (r PowerMonitorInternalReconciler) reconcilersForPowerMonitor(pmi *v1alpha1.PowerMonitorInternal) ([]reconciler.Reconciler, error) {
	rs := []reconciler.Reconciler{}

	cleanup := !pmi.DeletionTimestamp.IsZero()
	// not set for deletion
	if !cleanup {
		rs = append(rs, reconciler.Updater{
			Owner:    pmi,
			Resource: components.NewNamespace(pmi.Namespace()),
			OnError:  reconciler.Requeue,
			Logger:   r.logger,
		})
	}

	// Create DaemonSet early so it can be used by SecretMounter
	var ds *appsv1.DaemonSet
	if !cleanup {
		ds = powermonitor.NewPowerMonitorDaemonSet(components.Full, pmi)
	}

	// Mount secrets (validate and annotate DaemonSet) before deploying
	if !cleanup {
		rs = append(rs, reconciler.SecretMounter{
			Pmi:    pmi,
			Ds:     ds,
			Logger: r.logger,
		})
	}

	// update with image to be used (initial setup for testing then fix to be top level)
	exporterReconcilers, err := powerMonitorExporters(pmi, ds, Config.Cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to create power monitor exporters: %w", err)
	}
	rs = append(rs, exporterReconcilers...)

	if cleanup {
		// The Deleter returns Requeue until the resource is gone, ensuring
		// the Finalizer won't run until cleanup is complete
		rs = append(rs, reconciler.Deleter{
			OnError:  reconciler.Requeue,
			Resource: components.NewNamespace(pmi.Namespace()),
		})
	}

	rs = append(rs, reconciler.Finalizer{
		Resource:  pmi,
		Finalizer: Finalizer,
		Logger:    r.logger,
	})
	return rs, nil
}

func (r PowerMonitorInternalReconciler) updatePowerMonitorStatus(ctx context.Context, req ctrl.Request, recErr error) error {
	logger := r.logger.WithValues("power-monitor-internal", req.Name, "action", "update-status")
	logger.V(3).Info("Start of status update")
	defer logger.V(3).Info("End of status update")

	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		pmi, _ := r.getPowerMonitorInternal(ctx, req)
		// may be deleted
		if pmi == nil || !pmi.GetDeletionTimestamp().IsZero() {
			// retry since some error has occurred
			r.logger.V(6).Info("Reconcile has deleted power-monitor-internal; skipping status update")
			return nil
		}
		// sanitize the conditions so that all types are present and the order is predictable
		pmi.Status.Conditions = sanitizePowerMonitorConditions(pmi.Status.Conditions)

		{
			now := metav1.Now()
			reconciledChanged := r.updatePowerMonitorReconciledStatus(ctx, pmi, recErr, now)
			availableChanged := r.updatePowerMonitorAvailableStatus(ctx, pmi, recErr, now)
			logger.V(6).Info("conditions updated", "reconciled", reconciledChanged, "available", availableChanged)

			if !reconciledChanged && !availableChanged {
				logger.V(6).Info("no changes to existing status; skipping update")
				return nil
			}
		}

		return r.Client.Status().Update(ctx, pmi)
	})
}

func sanitizePowerMonitorConditions(conditions []v1alpha1.Condition) []v1alpha1.Condition {
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

func (r PowerMonitorInternalReconciler) updatePowerMonitorReconciledStatus(ctx context.Context, pmi *v1alpha1.PowerMonitorInternal, recErr error, time metav1.Time) bool {
	reconciled := v1alpha1.Condition{
		Type:               v1alpha1.Reconciled,
		Status:             v1alpha1.ConditionTrue,
		ObservedGeneration: pmi.Generation,
		Reason:             v1alpha1.ReconcileComplete,
		Message:            "Reconcile succeeded",
	}

	if recErr != nil {
		reconciled.Status = v1alpha1.ConditionFalse
		reconciled.Reason = v1alpha1.ReconcileError
		reconciled.Message = recErr.Error()
	}

	return updatePowerMonitorCondition(pmi.Status.Conditions, reconciled, time)
}

func findPowerMonitorCondition(conditions []v1alpha1.Condition, t v1alpha1.ConditionType) *v1alpha1.Condition {
	for i, c := range conditions {
		if c.Type == t {
			return &conditions[i]
		}
	}
	return nil
}

// returns true if the condition has been updated
func updatePowerMonitorCondition(conditions []v1alpha1.Condition, latest v1alpha1.Condition, time metav1.Time) bool {
	old := findPowerMonitorCondition(conditions, latest.Type)
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

func (r PowerMonitorInternalReconciler) updatePowerMonitorAvailableStatus(ctx context.Context, pmi *v1alpha1.PowerMonitorInternal, recErr error, time metav1.Time) bool {
	// get daemonset owned by powermonitor
	dset := appsv1.DaemonSet{}
	key := types.NamespacedName{Name: pmi.DaemonsetName(), Namespace: pmi.Namespace()}
	if err := r.Client.Get(ctx, key, &dset); err != nil {
		return updatePowerMonitorCondition(pmi.Status.Conditions, availablePowerMonitorConditionForGetError(err), time)
	}

	ds := dset.Status
	pmi.Status.Kepler.NumberMisscheduled = ds.NumberMisscheduled
	pmi.Status.Kepler.CurrentNumberScheduled = ds.CurrentNumberScheduled
	pmi.Status.Kepler.DesiredNumberScheduled = ds.DesiredNumberScheduled
	pmi.Status.Kepler.NumberReady = ds.NumberReady
	pmi.Status.Kepler.UpdatedNumberScheduled = ds.UpdatedNumberScheduled
	pmi.Status.Kepler.NumberAvailable = ds.NumberAvailable
	pmi.Status.Kepler.NumberUnavailable = ds.NumberUnavailable

	available := availablePowerMonitorCondition(&dset)

	if recErr == nil {
		available.ObservedGeneration = pmi.Generation
	} else {
		// failure to reconcile is a Degraded condition
		available.Status = v1alpha1.ConditionDegraded

		// Check if the error is specifically a SecretNotFoundError
		if secretErr, ok := recErr.(*reconciler.SecretNotFoundError); ok {
			available.Reason = v1alpha1.SecretNotFound
			available.Message = secretErr.Error()
		} else {
			available.Reason = v1alpha1.ReconcileError
		}
	}

	updated := updatePowerMonitorCondition(pmi.Status.Conditions, available, time)
	return updated
}

func availablePowerMonitorConditionForGetError(err error) v1alpha1.Condition {
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

func availablePowerMonitorCondition(dset *appsv1.DaemonSet) v1alpha1.Condition {
	ds := dset.Status
	dsName := dset.Namespace + "/" + dset.Name

	if gen, ogen := dset.Generation, ds.ObservedGeneration; gen > ogen {
		return v1alpha1.Condition{
			Type:   v1alpha1.Available,
			Status: v1alpha1.ConditionUnknown,
			Reason: v1alpha1.DaemonSetOutOfSync,
			Message: fmt.Sprintf(
				"Generation %d of power-monitor daemonset %q is out of sync with the observed generation: %d",
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
		c.Message = fmt.Sprintf("power-monitor daemonset %q is not rolled out to any node; check nodeSelector and tolerations", dsName)
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
			"Waiting for power-monitor daemonset %q rollout to finish: %d out of %d new pods have been updated",
			dsName, ds.UpdatedNumberScheduled, ds.DesiredNumberScheduled)
		return c
	}

	// NumberAvailable: The number of nodes that should be running the daemon pod
	// and have one or more of the daemon pod running and available (ready for at
	// least spec.minReadySeconds)

	if ds.NumberAvailable < ds.DesiredNumberScheduled {
		c.Status = v1alpha1.ConditionUnknown
		c.Reason = v1alpha1.DaemonSetPartiallyAvailable
		c.Message = fmt.Sprintf("Rollout of power-monitor daemonset %q is in progress: %d of %d updated pods are available",
			dsName, ds.NumberAvailable, ds.DesiredNumberScheduled)
		return c
	}

	// NumberUnavailable:  The number of nodes that should be running the daemon
	// pod and have none of the daemon pod running and available (ready for at
	// least spec.minReadySeconds)
	if ds.NumberUnavailable > 0 {
		c.Status = v1alpha1.ConditionFalse
		c.Reason = v1alpha1.DaemonSetPartiallyAvailable
		c.Message = fmt.Sprintf("Waiting for power-monitor daemonset %q to rollout on %d nodes", dsName, ds.NumberUnavailable)
		return c
	}

	c.Status = v1alpha1.ConditionTrue
	c.Reason = v1alpha1.DaemonSetReady
	c.Message = fmt.Sprintf("power-monitor daemonset %q is deployed to all nodes and available; ready %d/%d",
		dsName, ds.NumberReady, ds.DesiredNumberScheduled)
	return c
}
