// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"
	"fmt"
	"time"

	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	powermonitor "github.com/sustainable.computing.io/kepler-operator/pkg/components/power-monitor"
	appsv1 "k8s.io/api/apps/v1"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	timeout    = 60 * time.Second
	retryDelay = 3 * time.Second
)

// KubeRBACProxyConfigReconciler reconciles configuration for allowed SAs
type KubeRBACProxyConfigReconciler struct {
	Pmi        *v1alpha1.PowerMonitorInternal
	EnableRBAC bool
	EnableUWM  bool
}

func (r KubeRBACProxyConfigReconciler) Reconcile(ctx context.Context, c client.Client, s *runtime.Scheme) Result {
	if !r.EnableRBAC {
		secretKubeRBACConfig := powermonitor.NewPowerMonitorKubeRBACProxyConfig(
			components.Metadata,
			r.Pmi,
		)
		return Deleter{Resource: secretKubeRBACConfig}.Reconcile(ctx, c, s)
	}
	secretKubeRBACConfig := powermonitor.NewPowerMonitorKubeRBACProxyConfig(
		components.Full,
		r.Pmi,
	)
	return Updater{Owner: r.Pmi, Resource: secretKubeRBACConfig}.Reconcile(ctx, c, s)
}

// CABundleConfigReconciler reconciles the CA Injected Bundle ConfigMap to be used by ServiceMonitor
type CABundleConfigReconciler struct {
	Pmi        *v1alpha1.PowerMonitorInternal
	EnableRBAC bool
	EnableUWM  bool
}

func (r CABundleConfigReconciler) Reconcile(ctx context.Context, c client.Client, s *runtime.Scheme) Result {
	if !r.EnableRBAC || !r.EnableUWM {
		caBundle := powermonitor.NewPowerMonitorCABundleConfigMap(
			components.Metadata,
			r.Pmi,
		)
		return Deleter{Resource: caBundle}.Reconcile(ctx, c, s)
	}
	caBundle := powermonitor.NewPowerMonitorCABundleConfigMap(
		components.Full,
		r.Pmi,
	)
	return Updater{Owner: r.Pmi, Resource: caBundle}.Reconcile(ctx, c, s)
}

// UWMSecretTokenReconciler reconciles the User Workload Monitoring Secret Token
type UWMSecretTokenReconciler struct {
	Pmi        *v1alpha1.PowerMonitorInternal
	EnableRBAC bool
	EnableUWM  bool
}

func (r UWMSecretTokenReconciler) Reconcile(ctx context.Context, c client.Client, s *runtime.Scheme) Result {
	if !r.EnableRBAC || !r.EnableUWM {
		tokenSecret := powermonitor.NewPowerMonitorUWMTokenSecret(
			components.Metadata,
			r.Pmi,
			"",
		)
		return Deleter{Resource: tokenSecret}.Reconcile(ctx, c, s)
	}
	var promAccount *corev1.ServiceAccount
	if err := retryWithTimeout(ctx, timeout, retryDelay, func() error {
		var err error
		promAccount, err = getServiceAccount(
			ctx,
			c,
			powermonitor.UWMServiceAccountName,
			powermonitor.UWMNamespace,
		)
		if err != nil {
			return fmt.Errorf(
				"error occurred while getting %q service account %w",
				powermonitor.UWMServiceAccountName,
				err,
			)
		}
		if promAccount == nil {
			return fmt.Errorf(
				"missing %q in %q namespace yet; please enable user workload monitoring",
				powermonitor.UWMServiceAccountName,
				powermonitor.UWMNamespace,
			)
		}
		return nil
	}); err != nil {
		return Result{
			Action: Stop,
			Error:  err,
		}
	}
	audiences := []string{
		fmt.Sprintf("%s.%s.svc", r.Pmi.Name, r.Pmi.Namespace()),
	}
	token, err := requestToken(
		ctx,
		c,
		promAccount,
		audiences,
		365*24*time.Hour,
	)
	if err != nil {
		return Result{
			Action: Stop,
			Error: fmt.Errorf(
				"error occurred while generating %q token %w",
				powermonitor.UWMServiceAccountName,
				err,
			),
		}
	}
	tokenSecret := powermonitor.NewPowerMonitorUWMTokenSecret(components.Full, r.Pmi, token)
	return Updater{Owner: r.Pmi, Resource: tokenSecret}.Reconcile(ctx, c, s)
}

// KubeRBACProxyObjectsChecker checks if all required objects for kube-rbac-proxy are present
type KubeRBACProxyObjectsChecker struct {
	Pmi        *v1alpha1.PowerMonitorInternal
	Ds         *appsv1.DaemonSet
	EnableRBAC bool
	EnableUWM  bool
}

func (r KubeRBACProxyObjectsChecker) Reconcile(ctx context.Context, c client.Client, s *runtime.Scheme) Result {
	if !r.EnableRBAC {
		return Result{}
	}
	// check kube rbac proxy config secret
	var proxyConfig *corev1.Secret
	if err := retryWithTimeout(ctx, timeout, retryDelay, func() error {
		var err error
		proxyConfig, err = getSecret(
			ctx,
			c,
			powermonitor.SecretKubeRBACProxyConfigName,
			r.Pmi.Spec.Kepler.Deployment.Namespace,
		)
		if err != nil {
			return fmt.Errorf(
				"error occurred while getting %q secret %w",
				powermonitor.SecretKubeRBACProxyConfigName,
				err,
			)
		}
		if proxyConfig == nil {
			return fmt.Errorf(
				"%q secret not created in %q namespace yet",
				powermonitor.SecretKubeRBACProxyConfigName,
				r.Pmi.Namespace(),
			)
		}
		return nil
	}); err != nil {
		return Result{
			Action: Stop,
			Error:  err,
		}
	}
	powermonitor.AnnotateDaemonSetWithSecretHash(r.Ds, proxyConfig)
	// check power monitor tls secret
	var pmTLS *corev1.Secret
	if err := retryWithTimeout(ctx, timeout, retryDelay, func() error {
		var err error
		pmTLS, err = getSecret(
			ctx,
			c,
			powermonitor.SecretTLSCertName,
			r.Pmi.Spec.Kepler.Deployment.Namespace,
		)
		if err != nil {
			return fmt.Errorf(
				"error occurred while getting %q secret %w",
				powermonitor.SecretTLSCertName,
				err,
			)
		}
		if pmTLS == nil {
			return fmt.Errorf(
				"%q secret not created in %q namespace yet",
				powermonitor.SecretTLSCertName,
				r.Pmi.Namespace(),
			)
		}
		return nil
	}); err != nil {
		return Result{
			Action: Stop,
			Error:  err,
		}
	}
	powermonitor.AnnotateDaemonSetWithSecretHash(r.Ds, pmTLS)

	if r.EnableUWM {
		// check ca bundle
		if err := retryWithTimeout(ctx, timeout, retryDelay, func() error {
			caBundle, err := getConfigMap(
				ctx,
				c,
				powermonitor.PowerMonitorCertsCABundleName,
				r.Pmi.Spec.Kepler.Deployment.Namespace,
			)
			if err != nil {
				return fmt.Errorf(
					"error occurred while getting %q configmap %w",
					powermonitor.PowerMonitorCertsCABundleName,
					err,
				)
			}
			if caBundle == nil {
				return fmt.Errorf(
					"missing %q in %q namespace yet; openshift is yet to create ca bundle validation",
					powermonitor.PowerMonitorCertsCABundleName, r.Pmi.Namespace(),
				)
			}
			return nil
		}); err != nil {
			return Result{
				Action: Stop,
				Error:  err,
			}
		}

		// check uwm token secret
		if err := retryWithTimeout(ctx, timeout, retryDelay, func() error {
			promUWMSecretToken, err := getSecret(
				ctx,
				c,
				powermonitor.SecretUWMTokenName,
				r.Pmi.Spec.Kepler.Deployment.Namespace,
			)
			if err != nil {
				return fmt.Errorf(
					"error occurred while getting %q secret %w",
					powermonitor.SecretUWMTokenName,
					err,
				)
			}
			if promUWMSecretToken == nil {
				return fmt.Errorf(
					"missing %q in %q namespace yet; operator is yet to create the token for %q sa",
					powermonitor.SecretUWMTokenName,
					r.Pmi.Namespace(),
					powermonitor.UWMServiceAccountName,
				)
			}
			return nil
		}); err != nil {
			return Result{
				Action: Stop,
				Error:  err,
			}
		}
	}
	return Result{}
}

// retryWithTimeout retries the operation until it succeeds or times out
func retryWithTimeout(ctx context.Context, timeout, retryDelay time.Duration, operation func() error) error {
	var lastErr error

	err := wait.PollUntilContextTimeout(ctx, retryDelay, timeout, true, func(ctx context.Context) (bool, error) {
		if err := operation(); err != nil {
			lastErr = err
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		if lastErr != nil {
			return fmt.Errorf("timeout after %v: %w", timeout, lastErr)
		}
		return fmt.Errorf("timeout after %v: %w", timeout, err)
	}

	return nil
}

func getSecret(ctx context.Context, c client.Client, secretName, ns string) (*corev1.Secret, error) {
	proxyConfig := corev1.Secret{}
	if err := c.Get(ctx, types.NamespacedName{Name: secretName, Namespace: ns}, &proxyConfig); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return &proxyConfig, nil
}

func getConfigMap(ctx context.Context, c client.Client, cmName, ns string) (*corev1.ConfigMap, error) {
	caBundle := corev1.ConfigMap{}
	if err := c.Get(ctx, types.NamespacedName{Name: cmName, Namespace: ns}, &caBundle); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return &caBundle, nil
}

func getServiceAccount(ctx context.Context, c client.Client, saName, ns string) (*corev1.ServiceAccount, error) {
	sa := corev1.ServiceAccount{}
	if err := c.Get(ctx, types.NamespacedName{Name: saName, Namespace: ns}, &sa); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return &sa, nil
}

func requestToken(ctx context.Context, c client.Client, serviceAccount *corev1.ServiceAccount, audiences []string, duration time.Duration) (string, error) {
	copiedAudiences := make([]string, len(audiences))
	copy(copiedAudiences, audiences)
	tokenRequest := &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			Audiences:         copiedAudiences,
			ExpirationSeconds: ptr.To(int64(duration.Seconds())),
		},
	}
	err := c.SubResource("token").Create(ctx, serviceAccount, tokenRequest)
	if err != nil {
		return "", err
	}
	return tokenRequest.Status.Token, nil
}
