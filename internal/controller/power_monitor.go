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

var PowerMonitorDeploymentNS = "power-monitor"

// PowerMonitorReconciler reconciles a Kepler object
type PowerMonitorReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	logger logr.Logger
}

// Owned resource
//+kubebuilder:rbac:groups=kepler.system.sustainable.computing.io,resources=*,verbs=*

// SetupWithManager sets up the controller with the Manager.
func (r *PowerMonitorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.PowerMonitor{}).
		Owns(&v1alpha1.PowerMonitorInternal{},
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
func (r *PowerMonitorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// TODO: remove these keys from the log
	// "controller": "kepler", "controllerGroup": "kepler.system.sustainable.computing.io",
	// "controllerKind": "Kepler", "Kepler": {"name":"kepler"},

	logger := log.FromContext(ctx)
	r.logger = logger

	logger.Info("Start of  reconcile")
	defer logger.Info("End of reconcile")

	pm, err := r.getPowerMonitor(ctx, req)
	if err != nil {
		// retry since some error has occurred
		logger.V(6).Info("Get Error ", "error", err)
		return ctrl.Result{}, err
	}

	if pm == nil {
		// no kepler found , so stop here
		logger.V(6).Info("power-monitor Nil")
		return ctrl.Result{}, nil
	}

	// NOTE: validating webhook should ensure that this isn't possible, however,
	// if the webhook is removed, we should mark the instance as invalid.
	if pm.Name != v1alpha1.PowerMonitorInstanceName {
		return r.setInvalidStatus(ctx, req)
	}

	logger.V(6).Info("Running sub reconcilers", "power-monitor", pm.Spec)

	result, recErr := r.runPowerMonitorReconcilers(ctx, pm)
	updateErr := r.updatePowerMonitorStatus(ctx, req, recErr)

	if recErr != nil {
		return result, recErr
	}
	return result, updateErr
}

func (r PowerMonitorReconciler) runPowerMonitorReconcilers(ctx context.Context, pm *v1alpha1.PowerMonitor) (ctrl.Result, error) {
	reconcilers := r.reconcilersForPowerMonitor(pm)
	r.logger.V(6).Info("renconcilers ...", "count", len(reconcilers))

	return reconciler.Runner{
		Reconcilers: reconcilers,
		Client:      r.Client,
		Scheme:      r.Scheme,
		Logger:      r.logger,
	}.Run(ctx)
}

func (r PowerMonitorReconciler) updatePowerMonitorStatus(ctx context.Context, req ctrl.Request, recErr error) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		pm, _ := r.getPowerMonitor(ctx, req)
		// may be deleted
		if pm == nil || !pm.GetDeletionTimestamp().IsZero() {
			// retry since some error has occurred
			r.logger.V(6).Info("powermonitor has been deleted; skipping status update")
			return nil
		}

		internal, _ := r.getInternalForPowerMonitor(ctx, pm)
		// may be deleted
		if internal == nil || !internal.GetDeletionTimestamp().IsZero() {
			// retry since some error has occurred
			r.logger.V(6).Info("powermonitor has deleted; skipping status update")
			return nil
		}
		if !hasPowerMonitorInternalStatusChanged(internal) {
			r.logger.V(6).Info("powermonitor has not changed; skipping status update")
			return nil
		}

		// NOTE: although, this copies the internal status, the observed generation
		// should be set to kepler's current generation to indicate that the
		// current generation has been "observed"
		pm.Status = v1alpha1.PowerMonitorStatus{
			Kepler: v1alpha1.PowerMonitorKeplerStatus(internal.Status.Kepler), // this may fail
		}
		for i := range pm.Status.Kepler.Conditions {
			pm.Status.Kepler.Conditions[i].ObservedGeneration = pm.Generation
		}
		return r.Client.Status().Update(ctx, pm)
	})
}

// returns true (i.e. status has changed ) if any of the Conditions'
// ObservedGeneration is equal to the current generation
func hasPowerMonitorInternalStatusChanged(internal *v1alpha1.PowerMonitorInternal) bool {
	for i := range internal.Status.Kepler.Conditions {
		if internal.Status.Kepler.Conditions[i].ObservedGeneration == internal.Generation {
			return true
		}
	}
	return false
}

func (r PowerMonitorReconciler) getPowerMonitor(ctx context.Context, req ctrl.Request) (*v1alpha1.PowerMonitor, error) {
	logger := r.logger

	pm := v1alpha1.PowerMonitor{}

	if err := r.Client.Get(ctx, req.NamespacedName, &pm); err != nil {
		if errors.IsNotFound(err) {
			logger.V(3).Info("power-monitor could not be found; may be marked for deletion")
			return nil, nil
		}
		logger.Error(err, "failed to get power-monitor")
		return nil, err
	}

	return &pm, nil
}

func (r PowerMonitorReconciler) getInternalForPowerMonitor(ctx context.Context, pm *v1alpha1.PowerMonitor) (*v1alpha1.PowerMonitorInternal, error) {
	logger := r.logger.WithValues("power-monitor-internal", pm.Name)

	internal := v1alpha1.PowerMonitorInternal{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: pm.Name}, &internal); err != nil {
		if errors.IsNotFound(err) {
			logger.V(3).Info("power-monitor-internal could not be found; may be marked for deletion")
			return nil, nil
		}
		logger.Error(err, "failed to get power-monitor-internal")
		return nil, err
	}
	return &internal, nil
}

func (r PowerMonitorReconciler) reconcilersForPowerMonitor(pm *v1alpha1.PowerMonitor) []reconciler.Reconciler {
	op := deleteResource
	detail := components.Metadata

	if update := pm.DeletionTimestamp.IsZero(); update {
		op = newUpdaterWithOwner(pm)
		detail = components.Full
	}

	rs := []reconciler.Reconciler{
		op(newPowerMonitorInternal(detail, pm)),
		reconciler.Finalizer{
			Resource: pm, Finalizer: Finalizer, Logger: r.logger,
		},
	}
	return rs
}

func (r PowerMonitorReconciler) setInvalidStatus(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		invalidpm, _ := r.getPowerMonitor(ctx, req)
		// may be deleted
		if invalidpm == nil || !invalidpm.GetDeletionTimestamp().IsZero() {
			return nil
		}

		now := metav1.Now()
		invalidpm.Status.Kepler.Conditions = []v1alpha1.Condition{{
			Type:               v1alpha1.Reconciled,
			Status:             v1alpha1.ConditionFalse,
			ObservedGeneration: invalidpm.Generation,
			LastTransitionTime: now,
			Reason:             v1alpha1.InvalidPowerMonitorResource,
			Message:            "Only a single instance of PowerMonitor named powermonitor is reconciled",
		}, {
			Type:               v1alpha1.Available,
			Status:             v1alpha1.ConditionUnknown,
			ObservedGeneration: invalidpm.Generation,
			LastTransitionTime: now,
			Reason:             v1alpha1.InvalidPowerMonitorResource,
			Message:            "This instance of PowerMonitor is invalid",
		}}
		return r.Client.Status().Update(ctx, invalidpm)
	})

	// retry only on error
	return ctrl.Result{}, err
}

func newPowerMonitorInternal(d components.Detail, pm *v1alpha1.PowerMonitor) *v1alpha1.PowerMonitorInternal {
	if d == components.Metadata {
		return &v1alpha1.PowerMonitorInternal{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PowerMonitorInternal",
				APIVersion: v1alpha1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        pm.Name,
				Annotations: pm.Annotations,
			},
		}
	}

	isOpenShift := Config.Cluster == k8s.OpenShift

	return &v1alpha1.PowerMonitorInternal{
		TypeMeta:   metav1.TypeMeta{Kind: "PowerMonitorInternal", APIVersion: v1alpha1.GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{Name: pm.Name, Annotations: pm.Annotations},
		Spec: v1alpha1.PowerMonitorInternalSpec{
			Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
				Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
					PowerMonitorKeplerDeploymentSpec: pm.Spec.Kepler.Deployment,

					Image:     Config.RebootImage,
					Namespace: PowerMonitorDeploymentNS,
				},
				Config: v1alpha1.PowerMonitorInternalKeplerConfigSpec{
					LogLevel: pm.Spec.Kepler.Config.LogLevel,
				},
			},
			OpenShift: v1alpha1.PowerMonitorInternalOpenShiftSpec{
				Enabled: isOpenShift,
				Dashboard: v1alpha1.PowerMonitorInternalDashboardSpec{
					Enabled: isOpenShift,
				},
			},
		},
	}
}
