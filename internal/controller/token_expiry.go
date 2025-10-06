// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	powermonitor "github.com/sustainable.computing.io/kepler-operator/pkg/components/power-monitor"
	"github.com/sustainable.computing.io/kepler-operator/pkg/reconciler"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"

	corev1 "k8s.io/api/core/v1"

	ctrl "sigs.k8s.io/controller-runtime"
)

type TokenExpiryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// RBAC for TokenExpirationReconciler
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;delete

// SetupWithManager sets up the controller with the Manager.
func (r *TokenExpiryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	secretPredicate := builder.WithPredicates(predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return r.inPowerMonitorNamespace(e.Object) && r.isPrometheusUserWorkloadToken(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return r.inPowerMonitorNamespace(e.ObjectNew) && r.isPrometheusUserWorkloadToken(e.ObjectNew)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return r.inPowerMonitorNamespace(e.Object) && r.isPrometheusUserWorkloadToken(e.Object)
		},
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}, secretPredicate).
		Complete(r)
}

// inPowerMonitorNamespace checks if object is in the PowerMonitorDeploymentNS namespace
func (r *TokenExpiryReconciler) inPowerMonitorNamespace(obj client.Object) bool {
	return obj.GetNamespace() == PowerMonitorDeploymentNS
}

// isPrometheusUserWorkloadToken checks if the secret is the prometheus-user-workload-token
func (r *TokenExpiryReconciler) isPrometheusUserWorkloadToken(obj client.Object) bool {
	return obj.GetName() == powermonitor.SecretUWMTokenName
}

// hasExpirationAnnotation checks if the secret has an expiration annotation
func (r *TokenExpiryReconciler) hasExpirationAnnotation(obj client.Object) bool {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return false
	}

	annotations := secret.GetAnnotations()
	if annotations == nil {
		return false
	}

	_, exists := annotations[powermonitor.SecretTokenExpirationAnnotation]
	return exists
}

// deleteResources is a helper function that creates and runs deleter reconcilers for the given resources
func (r *TokenExpiryReconciler) deleteResources(ctx context.Context, resources ...client.Object) (ctrl.Result, error) {
	reconcilers := resourceReconcilers(deleteResource, resources...)

	return reconciler.Runner{
		Reconcilers: reconcilers,
		Client:      r.Client,
		Scheme:      r.Scheme,
		Logger:      r.logger,
	}.Run(ctx)
}

func (r *TokenExpiryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	r.logger = logger

	logger.Info("Start of reconcile")
	defer logger.Info("End of reconcile")

	secret := &corev1.Secret{}
	err := r.Get(ctx, req.NamespacedName, secret)
	if err != nil {
		if errors.IsNotFound(err) {
			r.logger.Info("secret not found, continue without error")
			return ctrl.Result{}, nil
		}
		r.logger.Error(err, "failed to retrieve secret")
		return ctrl.Result{}, err
	}

	if !r.hasExpirationAnnotation(secret) {
		r.logger.Info("prometheus-user-workload-token does not have expiration annotation, deleting it")
		return r.deleteResources(ctx, secret)
	}

	expired, expirationTime, err := r.isSecretExpired(secret)
	if err != nil {
		r.logger.Error(err, "failed to extract expiration time")
		return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
	}

	if expired {
		r.logger.Info("secret has expired, reconciling", "expiration-time", expirationTime)
		return r.deleteResources(ctx, secret)
	}

	timeUntilExpiration := time.Until(expirationTime)
	r.logger.Info("secret not expired yet, requeuing", "expiration-time", expirationTime, "time-until-expiration", timeUntilExpiration)

	return ctrl.Result{RequeueAfter: Config.TokenRefreshInterval}, nil
}

// isSecretExpired checks if the secret has expired according to the expiration annotation
func (r *TokenExpiryReconciler) isSecretExpired(secret *corev1.Secret) (bool, time.Time, error) {
	expirationTime, err := powermonitor.GetExpirationFromAnnotation(&secret.ObjectMeta, powermonitor.SecretTokenExpirationAnnotation)
	if err != nil {
		return false, time.Time{}, err
	}
	if expirationTime == nil {
		return false, time.Time{}, nil
	}
	return time.Now().After(expirationTime.Add(-(Config.TokenRefreshInterval * 2))), *expirationTime, nil
}
