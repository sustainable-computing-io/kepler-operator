/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	keplerv1alpha1 "github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/klog/v2"
)

var (
	// CollectotContainerImage is the container image name of the collector
	CollectotContainerImage string
	// EstimatorContainerImage is the container image name of the estimator
	EstimatorContainerImage string
	// ModelServerContainerImage is the container image name of the model-server
	ModelServerContainerImage string
	// SCCName is the name of the scribe security context constraint
	SCCName string
)

// KeplerReconciler reconciles a Kepler object
type KeplerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=kepler.system.sustainable.computing.io,resources=keplers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kepler.system.sustainable.computing.io,resources=keplers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kepler.system.sustainable.computing.io,resources=keplers/finalizers,verbs=update

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
	_ = log.FromContext(ctx)

	logger := log.Log.WithValues("kepler", req.NamespacedName)
	r.Log = logger

	inst := &keplerv1alpha1.Kepler{}
	if err := r.Client.Get(ctx, req.NamespacedName, inst); err != nil {
		if kerrors.IsNotFound(err) {
			klog.Error(err, "Failed to get namespace")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	var result ctrl.Result
	var err error

	if inst.Spec.Collector != nil {
		result, err = CollectorReconciler(ctx, inst, r, logger)
	} else if inst.Spec.Estimator != nil {
		// result, err = EstimatorReconciler(ctx, inst, r, logger)
	} else if inst.Spec.ModelServer != nil {
		// result, err = ModelServerReconciler(ctx, inst, r, logger)
	} else {

		return result, nil
	}

	// Set reconcile status condition
	if err == nil {
		inst.Status.Conditions = metav1.Condition{
			Type:               keplerv1alpha1.ConditionReconciled,
			Status:             metav1.ConditionTrue,
			Reason:             keplerv1alpha1.ReconciledReasonComplete,
			Message:            "Reconcile complete",
			LastTransitionTime: inst.CreationTimestamp,
		}
	} else {
		inst.Status.Conditions = metav1.Condition{
			Type:               keplerv1alpha1.ConditionReconciled,
			Status:             metav1.ConditionTrue,
			Reason:             keplerv1alpha1.ReconciledReasonError,
			Message:            err.Error(),
			LastTransitionTime: inst.CreationTimestamp,
		}
	}
	// Update instance status
	statusErr := r.Client.Status().Update(ctx, inst)
	if err == nil { // Don't mask previous error
		err = statusErr
	}

	return result, err
}

//nolint:dupl
func CollectorReconciler(ctx context.Context, instance *keplerv1alpha1.Kepler, kr *KeplerReconciler, logger klog.Logger) (ctrl.Result, error) {
	r := collectorReconciler{
		Ctx:              ctx,
		Instance:         instance,
		KeplerReconciler: *kr,
	}

	l := logger.WithValues("method", "Collector")
	_, err := reconcileBatch(l,
		r.ensureServiceAccount,
		// apply all resoucres here like service account, scc etc here eg r.applyServiceAccount

	)

	return ctrl.Result{}, err

}

// SetupWithManager sets up the controller with the Manager.
func (r *KeplerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&keplerv1alpha1.Kepler{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}

type collectorReconciler struct {
	KeplerReconciler
	serviceAccount *corev1.ServiceAccount
	Instance       *keplerv1alpha1.Kepler
	Ctx            context.Context
}

func (r *collectorReconciler) ensureServiceAccount(l klog.Logger) (bool, error) {
	r.serviceAccount = &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Instance.Name,
			Namespace: r.Instance.Namespace,
		},
	}
	kepDesc := keplerSADescription{
		Context: r.Ctx,
		Client:  r.Client,
		Scheme:  r.Scheme,
		SA:      r.serviceAccount,
		Owner:   r.Instance,
	}
	return kepDesc.Reconcile(l)
}
