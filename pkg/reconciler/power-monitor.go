// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	powermonitor "github.com/sustainable.computing.io/kepler-operator/pkg/components/power-monitor"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PowerMonitorDeployer deploys the PowerMonitor ConfigMap for the given PowerMonitorInterna
// land annotates the DaemonSet so that it is reloaded if the ConfigMap changes
type PowerMonitorDeployer struct {
	Pmi *v1alpha1.PowerMonitorInternal
	Ds  *appsv1.DaemonSet
}

// Reconcile implements the PowerMonitorDeployer interface
func (r PowerMonitorDeployer) Reconcile(ctx context.Context, c client.Client, s *runtime.Scheme) Result {
	additionalConfigs, err := r.readAdditionalConfigs(ctx, c)
	if err != nil {
		return Result{Action: Stop, Error: fmt.Errorf("error creating config: %w", err)}
	}

	cfm, err := powermonitor.NewPowerMonitorConfigMap(components.Full, r.Pmi, additionalConfigs...)
	if err != nil {
		return Result{Action: Stop, Error: fmt.Errorf("error creating configmap: %w", err)}
	}
	powermonitor.AnnotateDaemonSetWithConfigMapHash(r.Ds, cfm)

	// Update the ConfigMap
	return Updater{Owner: r.Pmi, Resource: cfm}.Reconcile(ctx, c, s)
}

// readAdditionalConfigs fetches the ConfigMaps referenced in the spec, merges them, and returns the final config data
func (r PowerMonitorDeployer) readAdditionalConfigs(ctx context.Context, c client.Client) ([]string, error) {
	cfmRefs := r.Pmi.Spec.Kepler.Config.AdditionalConfigMaps

	if len(cfmRefs) == 0 {
		return nil, nil
	}

	configMaps := make([]*corev1.ConfigMap, 0, len(cfmRefs))
	ns := r.Pmi.Namespace()

	// Fetch all referenced ConfigMaps
	for _, ref := range cfmRefs {
		cfm := &corev1.ConfigMap{}
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: ref.Name}, cfm); err != nil {
			if errors.IsNotFound(err) {
				return nil, fmt.Errorf("configMap %s not found in %s namespace", ref.Name, ns)
			}
			return nil, fmt.Errorf("failed to get ConfigMap %s: %w", ref.Name, err)
		}
		configMaps = append(configMaps, cfm)
	}

	// Extract YAML configurations from additional ConfigMaps
	var additionalConfigs []string
	for _, cfm := range configMaps {
		if cfm != nil && cfm.Data != nil {
			for filename, content := range cfm.Data {
				if content != "" && filename == powermonitor.KeplerConfigFile {
					additionalConfigs = append(additionalConfigs, content)
				}
			}
		}
	}
	return additionalConfigs, nil
}

// SecretNotFoundError represents an error when one or more referenced secrets are missing
type SecretNotFoundError struct {
	MissingSecrets []string
	Namespace      string
}

func (e *SecretNotFoundError) Error() string {
	if len(e.MissingSecrets) == 1 {
		return fmt.Sprintf("secret %s not found in %s namespace", e.MissingSecrets[0], e.Namespace)
	}
	return fmt.Sprintf("secrets %v not found in %s namespace", e.MissingSecrets, e.Namespace)
}

// SecretMounter validates that all referenced secrets exist and annotates the DaemonSet
// with secret hashes to trigger pod restarts when secrets change
type SecretMounter struct {
	Pmi    *v1alpha1.PowerMonitorInternal
	Ds     *appsv1.DaemonSet
	Logger logr.Logger
}

// Reconcile implements the SecretMounter interface
func (r SecretMounter) Reconcile(ctx context.Context, c client.Client, s *runtime.Scheme) Result {
	secretRefs := r.Pmi.Spec.Kepler.Deployment.Secrets
	if len(secretRefs) == 0 {
		return Result{Action: Continue}
	}

	ns := r.Pmi.Namespace()
	var missingSecrets []string

	// Validate all referenced secrets exist and annotate DaemonSet for found secrets
	for _, secretRef := range secretRefs {
		secret := &corev1.Secret{}
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: secretRef.Name}, secret); err != nil {
			if errors.IsNotFound(err) {
				missingSecrets = append(missingSecrets, secretRef.Name)
				r.Logger.Info("skipping hash annotation for missing secret",
					"secret", secretRef.Name, "namespace", ns)
				// Continue processing remaining secrets to collect all missing ones for status reporting
				continue
			}
			// For other errors (not NotFound), we should still stop reconciliation
			return Result{Action: Stop, Error: fmt.Errorf("failed to get secret %s: %w", secretRef.Name, err)}
		}

		// Secret exists - annotate DaemonSet with its hash for auto-reload
		powermonitor.AnnotateDaemonSetWithSecretHash(r.Ds, secret)
		r.Logger.Info("annotated DaemonSet with secret hash",
			"secret", secretRef.Name, "namespace", ns)
	}

	// If some secrets are missing, continue reconciliation but return an error
	// that can be detected by the controller to set degraded status
	if len(missingSecrets) > 0 {
		return Result{
			Action: Continue,
			Error: &SecretNotFoundError{
				MissingSecrets: missingSecrets,
				Namespace:      ns,
			},
		}
	}

	return Result{Action: Continue}
}
