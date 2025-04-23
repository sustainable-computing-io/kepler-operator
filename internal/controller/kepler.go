// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	"github.com/sustainable.computing.io/kepler-operator/pkg/reconciler"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	Finalizer = "kepler.system.sustainable.computing.io/finalizer"
)

var KeplerDeploymentNS = "kepler-operator"

// KeplerReconciler reconciles a Kepler object
type KeplerReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	logger logr.Logger
}

// Owned resource
//+kubebuilder:rbac:groups=kepler.system.sustainable.computing.io,resources=*,verbs=*

// SetupWithManager sets up the controller with the Manager.
func (r *KeplerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Kepler{}).
		Owns(&v1alpha1.KeplerInternal{},
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Complete(r)
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
func (r *KeplerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// TODO: remove these keys from the log
	// "controller": "kepler", "controllerGroup": "kepler.system.sustainable.computing.io",
	// "controllerKind": "Kepler", "Kepler": {"name":"kepler"},

	logger := log.FromContext(ctx)
	r.logger = logger

	logger.Info("Start of  reconcile")
	defer logger.Info("End of reconcile")

	kepler, err := r.getKepler(ctx, req)
	if err != nil {
		// retry since some error has occurred
		logger.V(6).Info("Get Error ", "error", err)
		return ctrl.Result{}, err
	}

	if kepler == nil {
		// no kepler found , so stop here
		logger.V(6).Info("Kepler Nil")
		return ctrl.Result{}, nil
	}

	// NOTE: validating webhook should ensure that this isn't possible, however,
	// if the webhook is removed, we should mark the instance as invalid.
	if kepler.Name != v1alpha1.KeplerInstanceName {
		return r.setInvalidStatus(ctx, req)
	}

	logger.V(6).Info("Running sub reconcilers", "kepler", kepler.Spec)

	result, recErr := r.runKeplerReconcilers(ctx, kepler)
	updateErr := r.updateStatus(ctx, req, recErr)

	if recErr != nil {
		return result, recErr
	}
	return result, updateErr
}

func (r KeplerReconciler) runKeplerReconcilers(ctx context.Context, kepler *v1alpha1.Kepler) (ctrl.Result, error) {
	reconcilers := r.reconcilersForKepler(kepler)
	r.logger.V(6).Info("renconcilers ...", "count", len(reconcilers))

	return reconciler.Runner{
		Reconcilers: reconcilers,
		Client:      r.Client,
		Scheme:      r.Scheme,
		Logger:      r.logger,
	}.Run(ctx)
}

func (r KeplerReconciler) updateStatus(ctx context.Context, req ctrl.Request, recErr error) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		k, _ := r.getKepler(ctx, req)
		// may be deleted
		if k == nil || !k.GetDeletionTimestamp().IsZero() {
			// retry since some error has occurred
			r.logger.V(6).Info("kepler has been deleted; skipping status update")
			return nil
		}

		internal, _ := r.getInternalForKepler(ctx, k)
		// may be deleted
		if internal == nil || !internal.GetDeletionTimestamp().IsZero() {
			// retry since some error has occurred
			r.logger.V(6).Info("keplerinternal has deleted; skipping status update")
			return nil
		}
		if !hasInternalStatusChanged(internal) {
			r.logger.V(6).Info("keplerinternal has not changed; skipping status update")
			return nil
		}

		// NOTE: although, this copies the internal status, the observed generation
		// should be set to kepler's current generation to indicate that the
		// current generation has been "observed"
		k.Status = v1alpha1.KeplerStatus{
			Exporter: internal.Status.Exporter,
		}
		for i := range k.Status.Exporter.Conditions {
			k.Status.Exporter.Conditions[i].ObservedGeneration = k.Generation
		}
		return r.Client.Status().Update(ctx, k)
	})
}

// returns true (i.e. status has changed ) if any of the Conditions'
// ObservedGeneration is equal to the current generation
func hasInternalStatusChanged(internal *v1alpha1.KeplerInternal) bool {
	for i := range internal.Status.Exporter.Conditions {
		if internal.Status.Exporter.Conditions[i].ObservedGeneration == internal.Generation {
			return true
		}
	}
	return false
}

func (r KeplerReconciler) getKepler(ctx context.Context, req ctrl.Request) (*v1alpha1.Kepler, error) {
	logger := r.logger

	kepler := v1alpha1.Kepler{}

	if err := r.Client.Get(ctx, req.NamespacedName, &kepler); err != nil {
		if errors.IsNotFound(err) {
			logger.V(3).Info("kepler could not be found; may be marked for deletion")
			return nil, nil
		}
		logger.Error(err, "failed to get kepler")
		return nil, err
	}

	return &kepler, nil
}

func (r KeplerReconciler) getInternalForKepler(ctx context.Context, k *v1alpha1.Kepler) (*v1alpha1.KeplerInternal, error) {
	logger := r.logger.WithValues("kepler-internal", k.Name)

	internal := v1alpha1.KeplerInternal{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: k.Name}, &internal); err != nil {
		if errors.IsNotFound(err) {
			logger.V(3).Info("kepler-internal could not be found; may be marked for deletion")
			return nil, nil
		}
		logger.Error(err, "failed to get kepler-internal")
		return nil, err
	}
	return &internal, nil
}

func (r KeplerReconciler) reconcilersForKepler(k *v1alpha1.Kepler) []reconciler.Reconciler {
	op := deleteResource
	detail := components.Metadata

	if update := k.DeletionTimestamp.IsZero(); update {
		op = newUpdaterWithOwner(k)
		detail = components.Full
	}

	rs := []reconciler.Reconciler{
		op(newKeplerInternal(detail, k)),
		reconciler.Finalizer{
			Resource: k, Finalizer: Finalizer, Logger: r.logger,
		},
	}
	return rs
}

func (r KeplerReconciler) setInvalidStatus(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		invalidKepler, _ := r.getKepler(ctx, req)
		// may be deleted
		if invalidKepler == nil || !invalidKepler.GetDeletionTimestamp().IsZero() {
			return nil
		}

		now := metav1.Now()
		invalidKepler.Status.Exporter.Conditions = []v1alpha1.Condition{{
			Type:               v1alpha1.Reconciled,
			Status:             v1alpha1.ConditionFalse,
			ObservedGeneration: invalidKepler.Generation,
			LastTransitionTime: now,
			Reason:             v1alpha1.InvalidKeplerResource,
			Message:            "Only a single instance of Kepler named kepler is reconciled",
		}, {
			Type:               v1alpha1.Available,
			Status:             v1alpha1.ConditionUnknown,
			ObservedGeneration: invalidKepler.Generation,
			LastTransitionTime: now,
			Reason:             v1alpha1.InvalidKeplerResource,
			Message:            "This instance of Kepler is invalid",
		}}
		return r.Client.Status().Update(ctx, invalidKepler)
	})

	// retry only on error
	return ctrl.Result{}, err
}

func newKeplerInternal(d components.Detail, k *v1alpha1.Kepler) *v1alpha1.KeplerInternal {
	if d == components.Metadata {
		return &v1alpha1.KeplerInternal{
			TypeMeta: metav1.TypeMeta{
				Kind:       "KeplerInternal",
				APIVersion: v1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        k.Name,
				Annotations: k.Annotations,
			},
		}
	}

	isOpenShift := Config.Cluster == k8s.OpenShift

	return &v1alpha1.KeplerInternal{
		TypeMeta:   metav1.TypeMeta{Kind: "KeplerInternal", APIVersion: v1alpha1.GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{Name: k.Name, Annotations: k.Annotations},
		Spec: v1alpha1.KeplerInternalSpec{
			Exporter: v1alpha1.InternalExporterSpec{
				Deployment: v1alpha1.InternalExporterDeploymentSpec{
					ExporterDeploymentSpec: k.Spec.Exporter.Deployment,
					Image:                  Config.Image,
					Namespace:              KeplerDeploymentNS,
				},
				Redfish: k.Spec.Exporter.Redfish,
			},
			OpenShift: v1alpha1.OpenShiftSpec{
				Enabled: isOpenShift,
				Dashboard: v1alpha1.DashboardSpec{
					Enabled: isOpenShift,
				},
			},
		},
	}
}
