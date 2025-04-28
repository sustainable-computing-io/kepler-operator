// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package powermonitor

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPowerMonitorNodeSelection(t *testing.T) {
	tt := []struct {
		spec     v1alpha1.PowerMonitorInternalKeplerSpec
		selector map[string]string
		scenario string
	}{
		{
			spec:     v1alpha1.PowerMonitorInternalKeplerSpec{},
			selector: map[string]string{"kubernetes.io/os": "linux"},
			scenario: "default case",
		},
		{
			spec: v1alpha1.PowerMonitorInternalKeplerSpec{
				Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
					PowerMonitorKeplerDeploymentSpec: v1alpha1.PowerMonitorKeplerDeploymentSpec{
						NodeSelector: map[string]string{"k1": "v1"},
					},
				},
			},
			selector: map[string]string{"k1": "v1", "kubernetes.io/os": "linux"},
			scenario: "user defined node selector",
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			pmi := v1alpha1.PowerMonitorInternal{
				Spec: v1alpha1.PowerMonitorInternalSpec{
					Kepler: tc.spec,
				},
			}
			actual := k8s.NodeSelectorFromDS(NewPowerMonitorDaemonSet(components.Full, &pmi))
			assert.Equal(t, actual, tc.selector)
		})
	}
}

func TestPowerMonitorTolerations(t *testing.T) {
	tt := []struct {
		spec        v1alpha1.PowerMonitorInternalKeplerSpec
		tolerations []corev1.Toleration
		scenario    string
	}{{
		spec: v1alpha1.PowerMonitorInternalKeplerSpec{},
		// NOTE: default toleration { "operator": "Exists" } is set by k8s API server (CRD default)
		tolerations: nil,
		scenario:    "default case",
	}, {
		spec: v1alpha1.PowerMonitorInternalKeplerSpec{
			Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
				PowerMonitorKeplerDeploymentSpec: v1alpha1.PowerMonitorKeplerDeploymentSpec{
					Tolerations: []corev1.Toleration{{
						Effect: corev1.TaintEffectNoSchedule, Key: "key1"}},
				},
			},
		},
		tolerations: []corev1.Toleration{{
			Effect: corev1.TaintEffectNoSchedule, Key: "key1",
		}},
		scenario: "user defined toleration",
	}}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			pmi := v1alpha1.PowerMonitorInternal{
				Spec: v1alpha1.PowerMonitorInternalSpec{
					Kepler: tc.spec,
				},
			}
			actual := k8s.TolerationsFromDS(NewPowerMonitorDaemonSet(components.Full, &pmi))
			assert.Equal(t, actual, tc.tolerations)
		})
	}
}

func TestPowerMonitorDaemonSet(t *testing.T) {
	tt := []struct {
		spec            v1alpha1.PowerMonitorInternalKeplerSpec
		hostPID         bool
		exporterCommand []string
		volumeMounts    []corev1.VolumeMount
		volumes         []corev1.Volume
		scenario        string
	}{
		{
			spec:    v1alpha1.PowerMonitorInternalKeplerSpec{},
			hostPID: true,
			exporterCommand: []string{
				"/usr/bin/kepler",
				fmt.Sprintf("--config.file=%s", filepath.Join(KeplerConfigMapPath, KeplerConfigFile)),
			},
			volumeMounts: []corev1.VolumeMount{
				{Name: "sysfs", MountPath: SysFSMountPath, ReadOnly: true},
				{Name: "procfs", MountPath: ProcFSMountPath, ReadOnly: true},
				{Name: "cfm", MountPath: KeplerConfigMapPath},
			},
			volumes: []corev1.Volume{
				k8s.VolumeFromHost("sysfs", "/sys"),
				k8s.VolumeFromHost("procfs", "/proc"),
				k8s.VolumeFromConfigMap("cfm", "power-monitor-internal"),
			},
			scenario: "default case",
		},
	}
	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			pmi := v1alpha1.PowerMonitorInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name: "power-monitor-internal",
				},
				Spec: v1alpha1.PowerMonitorInternalSpec{
					Kepler: tc.spec,
				},
			}
			ds := NewPowerMonitorDaemonSet(components.Full, &pmi)

			actualHostPID := k8s.HostPIDFromDS(ds)
			assert.Equal(t, tc.hostPID, actualHostPID)

			actualExporterCommand := k8s.CommandFromDS(ds, 0)
			assert.Equal(t, tc.exporterCommand, actualExporterCommand)

			actualVolumeMounts := k8s.VolumeMountsFromDS(ds, 0)
			assert.Equal(t, tc.volumeMounts, actualVolumeMounts)

			actualVolumes := k8s.VolumesFromDS(ds)
			assert.Equal(t, tc.volumes, actualVolumes)
		})
	}
}

func TestSCCAllows(t *testing.T) {
	tt := []struct {
		sccAllows k8s.SCCAllows
		scenario  string
	}{
		{
			sccAllows: k8s.SCCAllows{
				AllowPrivilegedContainer: true,
				AllowHostDirVolumePlugin: true,
				AllowHostIPC:             false,
				AllowHostNetwork:         false,
				AllowHostPID:             true,
				AllowHostPorts:           false,
			},
			scenario: "default case",
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			pmi := v1alpha1.PowerMonitorInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name: "power-monitor-internal",
				},
			}
			actual := k8s.AllowsFromSCC(NewPowerMonitorSCC(components.Full, &pmi))
			assert.Equal(t, actual, tc.sccAllows)
		})
	}
}

func TestPowerMonitorService(t *testing.T) {
	tt := []struct {
		podSelector k8s.StringMap
		port        int
		portName    string
		targetPort  int
		scenario    string
	}{
		{
			podSelector: k8s.StringMap{
				"app.kubernetes.io/name":                     "power-monitor-exporter",
				"app.kubernetes.io/component":                "exporter",
				"operator.sustainable-computing.io/internal": "power-monitor-internal",
				"app.kubernetes.io/part-of":                  "power-monitor-internal",
				"app.kubernetes.io/managed-by":               "kepler-operator",
			},
			port:       PowerMonitorDSPort,
			portName:   PowerMonitorServicePortName,
			targetPort: PowerMonitorDSPort,
			scenario:   "default case",
		},
	}
	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			pmi := v1alpha1.PowerMonitorInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name: "power-monitor-internal",
				},
				Spec: v1alpha1.PowerMonitorInternalSpec{
					Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
						Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{},
					},
				},
			}
			s := NewPowerMonitorService(&pmi)
			actualPodSelector := s.Spec.Selector
			assert.Equal(t, tc.podSelector.ToMap(), actualPodSelector)

			actualPorts := k8s.PortsFromService(s)
			assert.Equal(t, len(actualPorts), 1)
			assert.Equal(t, int(actualPorts[0].Port), tc.port)
			assert.Equal(t, actualPorts[0].Name, tc.portName)
			assert.Equal(t, actualPorts[0].TargetPort.IntValue(), tc.targetPort)
		})
	}
}

func TestPowerMonitorConfigMap(t *testing.T) {
	tt := []struct {
		spec           v1alpha1.PowerMonitorInternalKeplerSpec
		labels         k8s.StringMap
		logLevel       string
		configFileName string
		scenario       string
	}{
		{
			spec: v1alpha1.PowerMonitorInternalKeplerSpec{},
			labels: k8s.StringMap{
				"app.kubernetes.io/component":                "exporter",
				"operator.sustainable-computing.io/internal": "power-monitor-internal",
				"app.kubernetes.io/part-of":                  "power-monitor-internal",
				"app.kubernetes.io/managed-by":               "kepler-operator",
			},
			logLevel:       "info",
			configFileName: KeplerConfigFile,
			scenario:       "default case",
		},
		{
			spec: v1alpha1.PowerMonitorInternalKeplerSpec{
				Config: v1alpha1.PowerMonitorInternalKeplerConfigSpec{
					LogLevel: "debug",
				},
			},
			labels: k8s.StringMap{
				"app.kubernetes.io/component":                "exporter",
				"operator.sustainable-computing.io/internal": "power-monitor-internal",
				"app.kubernetes.io/part-of":                  "power-monitor-internal",
				"app.kubernetes.io/managed-by":               "kepler-operator",
			},
			logLevel:       "debug",
			configFileName: KeplerConfigFile,
			scenario:       "debug case",
		},
	}
	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			pmi := v1alpha1.PowerMonitorInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name: "power-monitor-internal",
				},
				Spec: v1alpha1.PowerMonitorInternalSpec{
					Kepler: tc.spec,
				},
			}
			cm := NewPowerMonitorConfigMap(components.Full, &pmi)

			actualLabels := k8s.LabelsFromConfigMap(cm)
			assert.Equal(t, tc.labels.ToMap(), actualLabels)

			actualData := k8s.DataFromConfigMap(cm)
			assert.Contains(t, actualData, tc.configFileName)
			assert.Contains(t, actualData[tc.configFileName], tc.logLevel)
		})
	}
}
