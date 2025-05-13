// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"
	"fmt"

	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	powermonitor "github.com/sustainable.computing.io/kepler-operator/pkg/components/power-monitor"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PowerMonitorDeployer deploys the PowerMonitor DaemonSet and ConfigMap for the given PowerMonitorInternal
type PowerMonitorDeployer struct {
	Pmi *v1alpha1.PowerMonitorInternal
}

// Reconcile implements the PowerMonitorDeployer interface
func (r PowerMonitorDeployer) Reconcile(ctx context.Context, c client.Client, s *runtime.Scheme) Result {
	additionalConfigs, err := r.readAdditionalConfigs(ctx, c)
	if err != nil {
		return Result{Action: Stop, Error: fmt.Errorf("error creating config: %w", err)}
	}

	cfm := powermonitor.NewPowerMonitorConfigMap(components.Full, r.Pmi, additionalConfigs...)

	ds := powermonitor.NewPowerMonitorDaemonSet(components.Full, r.Pmi)

	powermonitor.MountConfigMapToDaemonSet(ds, cfm)

	// Update the ConfigMap in the cluster
	result := Updater{Owner: r.Pmi, Resource: cfm}.Reconcile(ctx, c, s)
	if result.Error != nil {
		return result
	}

	// Update the DaemonSet
	return Updater{Owner: r.Pmi, Resource: ds}.Reconcile(ctx, c, s)
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
