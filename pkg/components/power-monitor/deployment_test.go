// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package powermonitor

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
						Effect: corev1.TaintEffectNoSchedule, Key: "key1",
					}},
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

/*
	func TestPowerMonitorDaemonSet(t *testing.T) {
		tt := []struct {
			spec            v1alpha1.PowerMonitorInternalKeplerSpec
			hostPID         bool
			exporterCommand []string
			volumeMounts    []corev1.VolumeMount
			volumes         []corev1.Volume
			cluster         k8s.Cluster
			scenario        string
			addConfigMap    bool
			configMap       *corev1.ConfigMap
			annotation      map[string]string
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
				cluster:  k8s.Kubernetes,
				scenario: "default case",
			},
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
				addConfigMap: true,
				configMap: &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: "power-monitor-internal",
					},
					Data: map[string]string{
						KeplerConfigFile: "test-config-content",
					},
				},
				annotation: map[string]string{
					ConfigMapHashAnnotation + "-power-monitor-internal": "123",
				},
				cluster:  k8s.Kubernetes,
				scenario: "configmap case",
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
				ds := NewPowerMonitorDaemonSet(components.Full, &pmi, tc.cluster)
				if tc.addConfigMap {
					MountConfigMapToDaemonSet(ds, tc.configMap)
				}

				actualHostPID := k8s.HostPIDFromDS(ds)
				assert.Equal(t, tc.hostPID, actualHostPID)

				actualExporterCommand := k8s.CommandFromDS(ds, 0)
				assert.Equal(t, tc.exporterCommand, actualExporterCommand)

				actualVolumeMounts := k8s.VolumeMountsFromDS(ds, 0)
				assert.Equal(t, tc.volumeMounts, actualVolumeMounts)

				actualVolumes := k8s.VolumesFromDS(ds)
				assert.Equal(t, tc.volumes, actualVolumes)

				if tc.addConfigMap {
					actualAnnotation := k8s.AnnotationFromDS(ds)
					assert.Contains(t, actualAnnotation, ConfigMapHashAnnotation+"-power-monitor-internal")
				}
			})
		}
	}
*/
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

func TestPowerMonitorDashboards(t *testing.T) {
	tt := []struct {
		createDashboard    func(d components.Detail) *corev1.ConfigMap
		labels             k8s.StringMap
		dashboardName      string
		dashboardNamespace string
		cmKey              string
		scenario           string
	}{
		{
			createDashboard: NewPowerMonitorInfoDashboard,
			labels: k8s.StringMap{
				"console.openshift.io/dashboard": "true",
				"app.kubernetes.io/managed-by":   "kepler-operator",
			},
			dashboardName:      InfoDashboardName,
			dashboardNamespace: DashboardNs,
			cmKey:              fmt.Sprintf("%s.json", InfoDashboardName),
			scenario:           "info dashboard case",
		},
	}
	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			nodeDashboard := tc.createDashboard(components.Full)

			actualName := nodeDashboard.Name
			assert.Equal(t, tc.dashboardName, actualName)

			actualNamespace := nodeDashboard.Namespace
			assert.Equal(t, tc.dashboardNamespace, actualNamespace)

			actualLabels := k8s.LabelsFromConfigMap(nodeDashboard)
			assert.Equal(t, tc.labels.ToMap(), actualLabels)

			actualData := k8s.DataFromConfigMap(nodeDashboard)
			assert.Contains(t, actualData, tc.cmKey)
			assert.Equal(t, actualData[tc.cmKey], readDashboardJSON(t, tc.cmKey))
		})
	}
}

func readDashboardJSON(t *testing.T, jsonFilename string) string {
	_, f, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get filepath")
	}
	path := filepath.Join(filepath.Dir(f), "assets", "dashboards", jsonFilename)
	dashboardData, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read dashboard file %s: %v", jsonFilename, err)
	}
	return string(dashboardData)
}

/*
func TestKeplerConfig(t *testing.T) {
	t.Run("With default config", func(t *testing.T) {
		pmi := &v1alpha1.PowerMonitorInternal{
			ObjectMeta: metav1.ObjectMeta{
				Name: "power-monitor-internal",
			},
			Spec: v1alpha1.PowerMonitorInternalSpec{
				Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
					Config: v1alpha1.PowerMonitorInternalKeplerConfigSpec{
						LogLevel: "info",
					},
				},
			},
		}

		configStr, err := KeplerConfig(pmi)

		defaultConfig := config.DefaultConfig()
		defaultConfig.Host.ProcFS = ProcFSMountPath
		defaultConfig.Host.SysFS = SysFSMountPath

		assert.NoError(t, err)
		assert.Equal(t, defaultConfig.String(), configStr)
	})

	t.Run("With debug log level", func(t *testing.T) {
		pmi := &v1alpha1.PowerMonitorInternal{
			ObjectMeta: metav1.ObjectMeta{
				Name: "power-monitor-internal",
			},
			Spec: v1alpha1.PowerMonitorInternalSpec{
				Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
					Config: v1alpha1.PowerMonitorInternalKeplerConfigSpec{
						LogLevel: "debug",
					},
				},
			},
		}

		configStr, err := KeplerConfig(pmi)

		defaultConfig := config.DefaultConfig()
		defaultConfig.Log.Level = "debug"
		defaultConfig.Host.ProcFS = ProcFSMountPath
		defaultConfig.Host.SysFS = SysFSMountPath

		assert.NoError(t, err)
		assert.Equal(t, defaultConfig.String(), configStr)
	})

	t.Run("With additional config", func(t *testing.T) {
		pmi := &v1alpha1.PowerMonitorInternal{
			ObjectMeta: metav1.ObjectMeta{
				Name: "power-monitor-internal",
			},
			Spec: v1alpha1.PowerMonitorInternalSpec{
				Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
					Config: v1alpha1.PowerMonitorInternalKeplerConfigSpec{
						LogLevel: "info",
					},
				},
			},
		}

		additionalConfig := `log:
  level: debug`

		configStr, err := KeplerConfig(pmi, additionalConfig)

		defaultConfig := config.DefaultConfig()
		defaultConfig.Host.ProcFS = ProcFSMountPath
		defaultConfig.Host.SysFS = SysFSMountPath

		assert.NoError(t, err)
		assert.Equal(t, defaultConfig.String(), configStr) // PMI spec config takes precedence over additional config
	})

	t.Run("With additional config affecting log format", func(t *testing.T) {
		pmi := &v1alpha1.PowerMonitorInternal{
			ObjectMeta: metav1.ObjectMeta{
				Name: "power-monitor-internal",
			},
			Spec: v1alpha1.PowerMonitorInternalSpec{
				Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
					Config: v1alpha1.PowerMonitorInternalKeplerConfigSpec{
						LogLevel: "info",
					},
				},
			},
		}

		additionalConfig := `log:
  format: json`

		configStr, err := KeplerConfig(pmi, additionalConfig)

		defaultConfig := config.DefaultConfig()
		defaultConfig.Host.ProcFS = ProcFSMountPath
		defaultConfig.Host.SysFS = SysFSMountPath
		defaultConfig.Log.Format = "json"

		assert.NoError(t, err)
		assert.Equal(t, defaultConfig.String(), configStr)
	})

	t.Run("With fake CPU meter enabled", func(t *testing.T) { // TODO: remove this test
		pmi := &v1alpha1.PowerMonitorInternal{
			ObjectMeta: metav1.ObjectMeta{
				Name: "power-monitor-internal",
				Annotations: map[string]string{
					EnableVMTestKey: "true",
				},
			},
			Spec: v1alpha1.PowerMonitorInternalSpec{
				Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
					Config: v1alpha1.PowerMonitorInternalKeplerConfigSpec{
						LogLevel: "info",
					},
				},
			},
		}

		configStr, err := KeplerConfig(pmi)

		defaultConfig := config.DefaultConfig()
		defaultConfig.Host.ProcFS = ProcFSMountPath
		defaultConfig.Host.SysFS = SysFSMountPath
		defaultConfig.Dev.FakeCpuMeter.Enabled = ptr.To(true)

		assert.NoError(t, err)
		assert.Equal(t, defaultConfig.String(), configStr)
	})
	t.Run("With invalid additional config", func(t *testing.T) {
		pmi := &v1alpha1.PowerMonitorInternal{
			ObjectMeta: metav1.ObjectMeta{
				Name: "power-monitor-internal",
			},
			Spec: v1alpha1.PowerMonitorInternalSpec{
				Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
					Config: v1alpha1.PowerMonitorInternalKeplerConfigSpec{
						LogLevel: "info",
					},
				},
			},
		}

		invalidConfig := `{[invalid]yaml}`

		configStr, err := KeplerConfig(pmi, invalidConfig)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to build config")
		assert.Empty(t, configStr)
	})
}
*/
