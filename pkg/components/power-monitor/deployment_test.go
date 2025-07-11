// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package powermonitor

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/cespare/xxhash/v2"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/internal/config"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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

func TestPowerMonitorDaemonSet(t *testing.T) {
	tt := []struct {
		spec            v1alpha1.PowerMonitorInternalKeplerSpec
		hostPID         bool
		exporterCommand []string
		volumeMounts    []corev1.VolumeMount
		volumes         []corev1.Volume
		containers      []string
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
				"--kube.enable",
				"--kube.node-name=$(NODE_NAME)",
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
			containers: []string{"power-monitor-internal"},
			scenario:   "default case",
		},
		{
			spec:    v1alpha1.PowerMonitorInternalKeplerSpec{},
			hostPID: true,
			exporterCommand: []string{
				"/usr/bin/kepler",
				fmt.Sprintf("--config.file=%s", filepath.Join(KeplerConfigMapPath, KeplerConfigFile)),
				"--kube.enable",
				"--kube.node-name=$(NODE_NAME)",
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
			containers:   []string{"power-monitor-internal"},
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
			scenario: "configmap case",
		},
		{
			spec: v1alpha1.PowerMonitorInternalKeplerSpec{
				Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
					PowerMonitorKeplerDeploymentSpec: v1alpha1.PowerMonitorKeplerDeploymentSpec{
						Security: v1alpha1.PowerMonitorKeplerDeploymentSecuritySpec{
							Mode: v1alpha1.SecurityModeRBAC,
						},
					},
				},
			},
			hostPID: true,
			exporterCommand: []string{
				"/usr/bin/kepler",
				fmt.Sprintf("--config.file=%s", filepath.Join(KeplerConfigMapPath, KeplerConfigFile)),
				"--kube.enable",
				"--kube.node-name=$(NODE_NAME)",
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
				k8s.VolumeFromSecret(SecretTLSCertName, SecretTLSCertName),
				k8s.VolumeFromSecret(SecretKubeRBACProxyConfigName, SecretKubeRBACProxyConfigName),
			},
			containers: []string{"power-monitor-internal", KubeRBACProxyContainerName},
			scenario:   "rbac case",
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
			if tc.addConfigMap {
				AnnotateDaemonSetWithConfigMapHash(ds, tc.configMap)
			}

			actualHostPID := k8s.HostPIDFromDS(ds)
			assert.Equal(t, tc.hostPID, actualHostPID)

			actualExporterCommand := k8s.CommandFromDS(ds, 0)
			assert.Equal(t, tc.exporterCommand, actualExporterCommand)

			actualVolumeMounts := k8s.VolumeMountsFromDS(ds, 0)
			assert.Equal(t, tc.volumeMounts, actualVolumeMounts)

			actualVolumes := k8s.VolumesFromDS(ds)
			assert.Equal(t, tc.volumes, actualVolumes)

			actualContainers := k8s.ContainerNamesFromDS(ds)
			assert.Equal(t, tc.containers, actualContainers)

			if tc.addConfigMap {
				actualAnnotation := k8s.AnnotationFromDS(ds)
				assert.Contains(t, actualAnnotation, ConfigMapHashAnnotation+"-power-monitor-internal")
			}
		})
	}
}

func TestPowerMonitorClusterRole(t *testing.T) {
	tt := []struct {
		spec     v1alpha1.PowerMonitorInternalKeplerSpec
		rules    []rbacv1.PolicyRule
		scenario string
	}{
		{
			spec: v1alpha1.PowerMonitorInternalKeplerSpec{},
			rules: []rbacv1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"nodes/metrics", "nodes/proxy", "nodes/stats", "pods"},
				Verbs:     []string{"get", "watch", "list"},
			}},
			scenario: "default case",
		},
		{
			spec: v1alpha1.PowerMonitorInternalKeplerSpec{
				Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
					PowerMonitorKeplerDeploymentSpec: v1alpha1.PowerMonitorKeplerDeploymentSpec{
						Security: v1alpha1.PowerMonitorKeplerDeploymentSecuritySpec{
							Mode: v1alpha1.SecurityModeRBAC,
						},
					},
				},
			},
			rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"nodes/metrics", "nodes/proxy", "nodes/stats", "pods"},
					Verbs:     []string{"get", "watch", "list"},
				},
				{
					APIGroups: []string{"authentication.k8s.io"},
					Resources: []string{"tokenreviews"},
					Verbs:     []string{"create"},
				},
				{
					APIGroups: []string{"authorization.k8s.io"},
					Resources: []string{"subjectaccessreviews"},
					Verbs:     []string{"create"},
				},
			},
			scenario: "rbac case",
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
			cr := NewPowerMonitorClusterRole(components.Full, &pmi)
			assert.Equal(t, tc.rules, cr.Rules)
		})
	}
}

func TestPowerMonitorClusterRoleBinding(t *testing.T) {
	tt := []struct {
		name      string
		namespace string
		roleRef   rbacv1.RoleRef
		subjects  []rbacv1.Subject
		scenario  string
	}{
		{
			name:      "power-monitor-internal",
			namespace: "power-monitor-internal",
			roleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "power-monitor-internal",
			},
			subjects: []rbacv1.Subject{{
				Kind:      "ServiceAccount",
				Name:      "power-monitor-internal",
				Namespace: "power-monitor-internal",
			}},
			scenario: "default case",
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			pmi := v1alpha1.PowerMonitorInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name: tc.name,
				},
				Spec: v1alpha1.PowerMonitorInternalSpec{
					Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
						Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
							Namespace: tc.namespace,
						},
					},
				},
			}
			crb := NewPowerMonitorClusterRoleBinding(components.Full, &pmi)
			assert.Equal(t, tc.name, crb.Name)
			assert.Equal(t, tc.roleRef, crb.RoleRef)
			assert.Equal(t, tc.subjects, crb.Subjects)
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
		spec        v1alpha1.PowerMonitorInternalKeplerSpec
		podSelector k8s.StringMap
		port        int
		portName    string
		targetPort  intstr.IntOrString
		annotations map[string]string
		scenario    string
	}{
		{
			spec: v1alpha1.PowerMonitorInternalKeplerSpec{},
			podSelector: k8s.StringMap{
				"app.kubernetes.io/name":                     "power-monitor-exporter",
				"app.kubernetes.io/component":                "exporter",
				"operator.sustainable-computing.io/internal": "power-monitor-internal",
				"app.kubernetes.io/part-of":                  "power-monitor-internal",
				"app.kubernetes.io/managed-by":               "kepler-operator",
			},
			port:       PowerMonitorDSPort,
			portName:   PowerMonitorServicePortName,
			targetPort: intstr.FromInt(PowerMonitorDSPort),
			scenario:   "default case",
		},
		{
			spec: v1alpha1.PowerMonitorInternalKeplerSpec{
				Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
					PowerMonitorKeplerDeploymentSpec: v1alpha1.PowerMonitorKeplerDeploymentSpec{
						Security: v1alpha1.PowerMonitorKeplerDeploymentSecuritySpec{
							Mode: v1alpha1.SecurityModeRBAC,
						},
					},
				},
			},
			podSelector: k8s.StringMap{
				"app.kubernetes.io/name":                     "power-monitor-exporter",
				"app.kubernetes.io/component":                "exporter",
				"operator.sustainable-computing.io/internal": "power-monitor-internal",
				"app.kubernetes.io/part-of":                  "power-monitor-internal",
				"app.kubernetes.io/managed-by":               "kepler-operator",
			},
			port:       SecurePort,
			portName:   SecurePortName,
			targetPort: intstr.FromString(SecurePortName),
			annotations: map[string]string{
				"service.beta.openshift.io/serving-cert-secret-name": SecretTLSCertName,
			},
			scenario: "rbac case",
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
			s := NewPowerMonitorService(&pmi)
			actualPodSelector := s.Spec.Selector
			assert.Equal(t, tc.podSelector.ToMap(), actualPodSelector)

			actualPorts := k8s.PortsFromService(s)
			assert.Equal(t, len(actualPorts), 1)
			assert.Equal(t, int(actualPorts[0].Port), tc.port)
			assert.Equal(t, actualPorts[0].Name, tc.portName)
			assert.Equal(t, tc.targetPort, actualPorts[0].TargetPort)
			if tc.scenario == "rbac case" {
				assert.Contains(t, s.Annotations, "service.beta.openshift.io/serving-cert-secret-name")
				assert.Equal(t, s.Annotations["service.beta.openshift.io/serving-cert-secret-name"], SecretTLSCertName)
			} else {
				assert.NotContains(t, s.Annotations, "service.beta.openshift.io/serving-cert-secret-name")
			}
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
			spec: v1alpha1.PowerMonitorInternalKeplerSpec{
				Config: v1alpha1.PowerMonitorInternalKeplerConfigSpec{
					LogLevel: "info",
				},
			},
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
			cm, err := NewPowerMonitorConfigMap(components.Full, &pmi)
			assert.NoError(t, err)

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
			dashboardName:      OverviewDashboardName,
			dashboardNamespace: DashboardNs,
			cmKey:              fmt.Sprintf("%s.json", OverviewDashboardName),
			scenario:           "info dashboard case",
		},
		{
			createDashboard: NewPowerMonitorNamespaceInfoDashboard,
			labels: k8s.StringMap{
				"console.openshift.io/dashboard": "true",
				"app.kubernetes.io/managed-by":   "kepler-operator",
			},
			dashboardName:      NamespaceInfoDashboardName,
			dashboardNamespace: DashboardNs,
			cmKey:              fmt.Sprintf("%s.json", NamespaceInfoDashboardName),
			scenario:           "namespace info dashboard case",
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

func TestPowerMonitorServiceMonitor(t *testing.T) {
	tt := []struct {
		spec      v1alpha1.PowerMonitorInternalKeplerSpec
		namespace string
		endpoints []monv1.Endpoint
		selector  metav1.LabelSelector
		scenario  string
	}{
		{
			spec: v1alpha1.PowerMonitorInternalKeplerSpec{},
			endpoints: []monv1.Endpoint{{
				Port:   PowerMonitorServicePortName,
				Scheme: "http",
				RelabelConfigs: []*monv1.RelabelConfig{{
					Action:      "replace",
					Regex:       "(.*)",
					Replacement: "$1",
					SourceLabels: []monv1.LabelName{
						"__meta_kubernetes_pod_node_name",
					},
					TargetLabel: "instance",
				}},
			}},
			selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/component":                "exporter",
					"operator.sustainable-computing.io/internal": "power-monitor-internal",
					"app.kubernetes.io/part-of":                  "power-monitor-internal",
					"app.kubernetes.io/managed-by":               "kepler-operator",
				},
			},
			scenario: "default case",
		},
		{
			spec: v1alpha1.PowerMonitorInternalKeplerSpec{
				Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
					PowerMonitorKeplerDeploymentSpec: v1alpha1.PowerMonitorKeplerDeploymentSpec{
						Security: v1alpha1.PowerMonitorKeplerDeploymentSecuritySpec{
							Mode: v1alpha1.SecurityModeRBAC,
						},
					},
					Namespace: "power-monitor-internal",
				},
			},
			endpoints: []monv1.Endpoint{{
				Port:   SecurePortName,
				Scheme: "https",
				RelabelConfigs: []*monv1.RelabelConfig{{
					Action:      "replace",
					Regex:       "(.*)",
					Replacement: "$1",
					SourceLabels: []monv1.LabelName{
						"__meta_kubernetes_pod_node_name",
					},
					TargetLabel: "instance",
				}},
				Authorization: &monv1.SafeAuthorization{
					Type: "Bearer",
					Credentials: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: SecretUWMTokenName,
						},
						Key: ServiceAccountTokenKey,
					},
				},
				TLSConfig: &monv1.TLSConfig{
					SafeTLSConfig: monv1.SafeTLSConfig{
						CA: monv1.SecretOrConfigMap{
							ConfigMap: &corev1.ConfigMapKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: PowerMonitorCertsCABundleName,
								},
								Key: "service-ca.crt",
							},
						},
						ServerName: "power-monitor-internal.power-monitor-internal.svc",
					},
				},
			}},
			selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/component":                "exporter",
					"operator.sustainable-computing.io/internal": "power-monitor-internal",
					"app.kubernetes.io/part-of":                  "power-monitor-internal",
					"app.kubernetes.io/managed-by":               "kepler-operator",
				},
			},
			scenario: "rbac case",
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
			sm := NewPowerMonitorServiceMonitor(components.Full, &pmi)
			assert.Equal(t, tc.endpoints, sm.Spec.Endpoints)
			assert.Equal(t, tc.selector, sm.Spec.Selector)
		})
	}
}

func TestPowerMonitorCABundleConfigMap(t *testing.T) {
	tt := []struct {
		name        string
		namespace   string
		annotations map[string]string
		scenario    string
	}{
		{
			name:      PowerMonitorCertsCABundleName,
			namespace: "power-monitor-internal",
			annotations: map[string]string{
				"service.beta.openshift.io/inject-cabundle": "true",
			},
			scenario: "tls case",
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
						Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
							Namespace: tc.namespace,
						},
					},
				},
			}
			cm := NewPowerMonitorCABundleConfigMap(components.Full, &pmi)
			assert.Equal(t, tc.name, cm.Name)
			assert.Equal(t, tc.namespace, cm.Namespace)
			assert.Equal(t, tc.annotations, cm.Annotations)
		})
	}
}

func TestPowerMonitorKubeRBACProxyConfig(t *testing.T) {
	tt := []struct {
		spec     v1alpha1.PowerMonitorInternalKeplerSpec
		name     string
		labels   k8s.StringMap
		scenario string
	}{
		{
			spec: v1alpha1.PowerMonitorInternalKeplerSpec{
				Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
					PowerMonitorKeplerDeploymentSpec: v1alpha1.PowerMonitorKeplerDeploymentSpec{
						Security: v1alpha1.PowerMonitorKeplerDeploymentSecuritySpec{
							Mode:           v1alpha1.SecurityModeRBAC,
							AllowedSANames: []string{"test-sa"},
						},
					},
					Namespace: "power-monitor-internal",
				},
			},
			name: SecretKubeRBACProxyConfigName,
			labels: k8s.StringMap{
				"app.kubernetes.io/component":                "exporter",
				"operator.sustainable-computing.io/internal": "power-monitor-internal",
				"app.kubernetes.io/part-of":                  "power-monitor-internal",
				"app.kubernetes.io/managed-by":               "kepler-operator",
			},
			scenario: "rbac case",
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
			rbacConfig, err := createKubeRBACConfig([]string{"test-sa"})
			assert.NoError(t, err)
			configData := map[string]string{
				"config.yaml": rbacConfig,
			}
			secret, err := NewPowerMonitorKubeRBACProxyConfig(components.Full, &pmi)
			assert.NoError(t, err)
			assert.Equal(t, tc.name, secret.Name)
			assert.Equal(t, pmi.Spec.Kepler.Deployment.Namespace, secret.Namespace)
			assert.Equal(t, tc.labels.ToMap(), secret.Labels)
			assert.Equal(t, configData, secret.StringData)
			assert.Equal(t, corev1.SecretTypeOpaque, secret.Type)
		})
	}
}

func TestPowerMonitorUWMTokenSecret(t *testing.T) {
	tt := []struct {
		name      string
		namespace string
		labels    k8s.StringMap
		tokenData map[string]string
		scenario  string
	}{
		{
			name:      SecretUWMTokenName,
			namespace: "power-monitor-internal",
			labels: k8s.StringMap{
				"app.kubernetes.io/component":                "exporter",
				"operator.sustainable-computing.io/internal": "power-monitor-internal",
				"app.kubernetes.io/part-of":                  "power-monitor-internal",
				"app.kubernetes.io/managed-by":               "kepler-operator",
			},
			tokenData: map[string]string{
				ServiceAccountTokenKey: "test-token",
			},
			scenario: "uwm and rbac case",
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
						Deployment: v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{
							Namespace: tc.namespace,
						},
					},
				},
			}
			secret := NewPowerMonitorUWMTokenSecret(components.Full, &pmi, "test-token")
			assert.Equal(t, tc.name, secret.Name)
			assert.Equal(t, tc.namespace, secret.Namespace)
			assert.Equal(t, tc.labels.ToMap(), secret.Labels)
			assert.Equal(t, tc.tokenData, secret.StringData)
			assert.Equal(t, corev1.SecretTypeOpaque, secret.Type)
		})
	}
}

func TestAnnotateDaemonSetWithSecretHash(t *testing.T) {
	tt := []struct {
		secret     *corev1.Secret
		annotation map[string]string
		scenario   string
	}{
		{
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: SecretTLSCertName,
				},
				Data: map[string][]byte{
					"tls.crt": []byte("cert-data"),
					"tls.key": []byte("key-data"),
				},
			},
			annotation: map[string]string{
				SecretTLSHashAnnotation + "-" + SecretTLSCertName: fmt.Sprintf("%x", xxhash.Sum64([]byte("tls.crtcert-datatls.keykey-data"))),
			},
			scenario: "tls secret case",
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
			ds := NewPowerMonitorDaemonSet(components.Full, &pmi)
			AnnotateDaemonSetWithSecretHash(ds, tc.secret)
			actualAnnotation := k8s.AnnotationFromDS(ds)
			assert.Equal(t, tc.annotation, actualAnnotation)
		})
	}
}
