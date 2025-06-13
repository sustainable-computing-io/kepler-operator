// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package powermonitor

import (
	_ "embed"
	"fmt"
	"path/filepath"

	"github.com/cespare/xxhash/v2"
	secv1 "github.com/openshift/api/security/v1"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/internal/config"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

// TODO: Generate Unit Tests (require more thorough test cases)
// TODO: Convert every hard coded name to a variable

const (
	PowerMonitorServicePortName = "http"
	DashboardNs                 = "openshift-config-managed"
	PowerMonitorDSPort          = 28282

	// Dashboard
	InfoDashboardName = "power-monitor-node-info"

	SysFSMountPath      = "/host/sys"
	ProcFSMountPath     = "/host/proc"
	KeplerConfigMapPath = "/etc/kepler"
	KeplerConfigFile    = "config.yaml"
	EnableVMTestKey     = "powermonitor.sustainable.computing.io/test-env-vm"

	// ConfigMap annotations
	ConfigMapHashAnnotation = "powermonitor.sustainable.computing.io/config-map-hash"
)

var (
	linuxNodeSelector = k8s.StringMap{
		"kubernetes.io/os": "linux",
	}
	//go:embed assets/dashboards/power-monitor-node-info.json
	infoDashboardJson string
)

func NewPowerMonitorDaemonSet(detail components.Detail, pmi *v1alpha1.PowerMonitorInternal) *appsv1.DaemonSet {
	if detail == components.Metadata {
		return &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{
				APIVersion: appsv1.SchemeGroupVersion.String(),
				Kind:       "DaemonSet",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      pmi.DaemonsetName(),
				Namespace: pmi.Namespace(),
				Labels:    labels(pmi),
			},
		}
	}
	deployment := pmi.Spec.Kepler.Deployment
	nodeSelector := deployment.NodeSelector
	tolerations := deployment.Tolerations

	pmContainer := newPowerMonitorContainer(pmi)
	pmContainers := []corev1.Container{pmContainer}

	volumes := []corev1.Volume{
		k8s.VolumeFromHost("sysfs", "/sys"),
		k8s.VolumeFromHost("procfs", "/proc"),
		k8s.VolumeFromConfigMap("cfm", pmi.Name),
	}

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pmi.Name,
			Namespace: pmi.Namespace(),
			Labels:    labels(pmi),
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: podSelector(pmi)},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pmi.DaemonsetName(),
					Namespace: pmi.Namespace(),
					Labels:    podSelector(pmi),
				},
				Spec: corev1.PodSpec{
					HostPID:            true,
					NodeSelector:       linuxNodeSelector.Merge(nodeSelector),
					ServiceAccountName: pmi.Name,
					DNSPolicy:          corev1.DNSPolicy(corev1.DNSClusterFirstWithHostNet),
					Tolerations:        tolerations,
					Containers:         pmContainers,
					Volumes:            volumes,
				}, // PodSpec
			}, // PodTemplateSpec
		}, // Spec
	}
}

func NewPowerMonitorService(pmi *v1alpha1.PowerMonitorInternal) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pmi.Name,
			Namespace: pmi.Namespace(),
			Labels:    labels(pmi).ToMap(),
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  podSelector(pmi),
			Ports: []corev1.ServicePort{{
				Name: PowerMonitorServicePortName,
				Port: int32(PowerMonitorDSPort),
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: int32(PowerMonitorDSPort),
				},
			}},
		},
	}
}

func NewPowerMonitorInfoDashboard(d components.Detail) *corev1.ConfigMap {
	return openshiftDashboardConfigMap(d, InfoDashboardName, fmt.Sprintf("%s.json", InfoDashboardName), infoDashboardJson)
}

func NewPowerMonitorConfigMap(d components.Detail, pmi *v1alpha1.PowerMonitorInternal, additionalConfigs ...string) *corev1.ConfigMap {
	if d == components.Metadata {
		return &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      pmi.Name,
				Namespace: pmi.Namespace(),
				Labels:    labels(pmi).ToMap(),
			},
		}
	}

	config, _ := KeplerConfig(pmi, additionalConfigs...)

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pmi.Name,
			Namespace: pmi.Namespace(),
			Labels:    labels(pmi).ToMap(),
		},
		Data: k8s.StringMap{
			KeplerConfigFile: config,
		},
	}
}

func NewPowerMonitorClusterRole(c components.Detail, pmi *v1alpha1.PowerMonitorInternal) *rbacv1.ClusterRole {
	if c == components.Metadata {
		return &rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacv1.SchemeGroupVersion.String(),
				Kind:       "ClusterRole",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   pmi.Name,
				Labels: labels(pmi),
			},
		}
	}

	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   pmi.Name,
			Labels: labels(pmi),
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"nodes/metrics", "nodes/proxy", "nodes/stats", "pods"},
			Verbs:     []string{"get", "watch", "list"},
		}},
	}
}

func NewPowerMonitorClusterRoleBinding(c components.Detail, pmi *v1alpha1.PowerMonitorInternal) *rbacv1.ClusterRoleBinding {
	if c == components.Metadata {
		return &rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacv1.SchemeGroupVersion.String(),
				Kind:       "ClusterRoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   pmi.Name,
				Labels: labels(pmi),
			},
		}
	}

	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   pmi.Name,
			Labels: labels(pmi),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     pmi.Name,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      pmi.Name,
			Namespace: pmi.Namespace(),
		}},
	}
}

func NewPowerMonitorSCC(d components.Detail, pmi *v1alpha1.PowerMonitorInternal) *secv1.SecurityContextConstraints {
	if d == components.Metadata {
		return &secv1.SecurityContextConstraints{
			TypeMeta: metav1.TypeMeta{
				APIVersion: secv1.SchemeGroupVersion.String(),
				Kind:       "SecurityContextConstraints",
			},

			ObjectMeta: metav1.ObjectMeta{
				Name:   pmi.Name,
				Labels: labels(pmi),
			},
		}
	}

	return &secv1.SecurityContextConstraints{
		TypeMeta: metav1.TypeMeta{
			APIVersion: secv1.SchemeGroupVersion.String(),
			Kind:       "SecurityContextConstraints",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:   pmi.Name,
			Labels: labels(pmi),
		},

		AllowPrivilegedContainer: true,
		AllowHostDirVolumePlugin: true,
		AllowHostIPC:             false,
		AllowHostNetwork:         false,
		AllowHostPID:             true,
		AllowHostPorts:           false,
		DefaultAddCapabilities:   []corev1.Capability{corev1.Capability("SYS_ADMIN")},

		FSGroup: secv1.FSGroupStrategyOptions{
			Type: secv1.FSGroupStrategyRunAsAny,
		},
		ReadOnlyRootFilesystem: true,
		RunAsUser: secv1.RunAsUserStrategyOptions{
			Type: secv1.RunAsUserStrategyRunAsAny,
		},
		SELinuxContext: secv1.SELinuxContextStrategyOptions{
			Type: secv1.SELinuxStrategyRunAsAny,
		},
		Users: []string{pmi.FQServiceAccountName()},
		Volumes: []secv1.FSType{
			secv1.FSType("configMap"),
			secv1.FSType("secret"),
			secv1.FSType("projected"),
			secv1.FSType("emptyDir"),
			secv1.FSType("hostPath"),
		},
	}
}

func NewPowerMonitorServiceAccount(pmi *v1alpha1.PowerMonitorInternal) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pmi.Name,
			Namespace: pmi.Namespace(),
			Labels:    labels(pmi).ToMap(),
		},
	}
}

func NewPowerMonitorServiceMonitor(pmi *v1alpha1.PowerMonitorInternal) *monv1.ServiceMonitor {
	relabelings := []*monv1.RelabelConfig{{
		Action:      "replace",
		Regex:       "(.*)",
		Replacement: "$1",
		SourceLabels: []monv1.LabelName{
			"__meta_kubernetes_pod_node_name",
		},
		TargetLabel: "instance",
	}}

	return &monv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monv1.SchemeGroupVersion.String(),
			Kind:       "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pmi.Name,
			Namespace: pmi.Namespace(),
			Labels:    labels(pmi).ToMap(),
		},
		Spec: monv1.ServiceMonitorSpec{
			Endpoints: []monv1.Endpoint{{
				Port:           PowerMonitorServicePortName,
				Scheme:         "http",
				RelabelConfigs: relabelings,
			}},
			JobLabel: "app.kubernetes.io/name",
			Selector: metav1.LabelSelector{
				MatchLabels: labels(pmi),
			},
		},
	}
}

func openshiftDashboardConfigMap(d components.Detail, dashboardName, dashboardJSONName, dashboardJSONPath string) *corev1.ConfigMap {
	objMeta := openshiftDashboardObjectMeta(dashboardName)

	if d == components.Metadata {
		return &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "ConfigMap",
			},
			ObjectMeta: objMeta,
		}
	}

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: objMeta,
		Data: map[string]string{
			dashboardJSONName: dashboardJSONPath,
		},
	}
}

func openshiftDashboardObjectMeta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: DashboardNs,
		Labels: components.CommonLabels.Merge(k8s.StringMap{
			"console.openshift.io/dashboard": "true",
		}),
		Annotations: k8s.StringMap{
			"include.release.openshift.io/self-managed-high-availability": "true",
			"include.release.openshift.io/single-node-developer":          "true",
		},
	}
}

func labels(pmi *v1alpha1.PowerMonitorInternal) k8s.StringMap {
	return components.CommonLabels.Merge(k8s.StringMap{
		"app.kubernetes.io/component":                "exporter",
		"operator.sustainable-computing.io/internal": pmi.Name,
		"app.kubernetes.io/part-of":                  pmi.Name,
	})
}

func podSelector(pmi *v1alpha1.PowerMonitorInternal) k8s.StringMap {
	return labels(pmi).Merge(k8s.StringMap{
		"app.kubernetes.io/name":      "power-monitor-exporter",
		"app.kubernetes.io/component": "exporter",
	})
}

func newPowerMonitorContainer(pmi *v1alpha1.PowerMonitorInternal) corev1.Container {
	deployment := pmi.Spec.Kepler.Deployment
	configMapPath := filepath.Join(KeplerConfigMapPath, KeplerConfigFile)
	return corev1.Container{
		Name:            pmi.DaemonsetName(),
		SecurityContext: &corev1.SecurityContext{Privileged: ptr.To(true)},
		Image:           deployment.Image,
		ImagePullPolicy: corev1.PullAlways,
		Env: []corev1.EnvVar{{
			Name:      "NODE_NAME",
			ValueFrom: k8s.EnvFromField("spec.nodeName"),
		}},
		Command: []string{
			"/usr/bin/kepler",
			fmt.Sprintf("--config.file=%s", configMapPath),
			"--kube.enable",
			"--kube.node-name=$(NODE_NAME)",
		},
		Ports: []corev1.ContainerPort{{
			ContainerPort: int32(PowerMonitorDSPort),
			Name:          PowerMonitorServicePortName,
		}},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/metrics",
					Port:   intstr.IntOrString{Type: intstr.Int, IntVal: int32(PowerMonitorDSPort)},
					Scheme: "HTTP",
				},
			},
			FailureThreshold:    5,
			InitialDelaySeconds: 10,
			PeriodSeconds:       60,
			SuccessThreshold:    1,
			TimeoutSeconds:      10,
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "sysfs", MountPath: SysFSMountPath, ReadOnly: true},
			{Name: "procfs", MountPath: ProcFSMountPath, ReadOnly: true},
			{Name: "cfm", MountPath: KeplerConfigMapPath},
		},
	}
}

// KeplerConfig returns the config for the power-monitor
func KeplerConfig(pmi *v1alpha1.PowerMonitorInternal, additionalConfigs ...string) (string, error) {
	// Start with default config
	b := &config.Builder{}

	for _, additionalConfig := range additionalConfigs {
		b.Merge(additionalConfig)
	}

	cfg, err := b.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build config: %w", err)
	}

	val, ok := pmi.Annotations[EnableVMTestKey]
	if ok {
		cfg.Dev.FakeCpuMeter.Enabled = ptr.To(val == "true")
	}

	cfg.Log.Level = pmi.Spec.Kepler.Config.LogLevel
	cfg.Host.SysFS = SysFSMountPath
	cfg.Host.ProcFS = ProcFSMountPath

	if err := cfg.Validate(config.SkipHostValidation); err != nil {
		return config.DefaultConfig().String(), err
	}
	return cfg.String(), nil
}

// MountConfigMapToDaemonSet sets annotations on the DaemonSet's pod template to trigger a rollout when the ConfigMap changes
func MountConfigMapToDaemonSet(ds *appsv1.DaemonSet, cfm *corev1.ConfigMap) {
	if ds.Spec.Template.Annotations == nil {
		ds.Spec.Template.Annotations = make(map[string]string)
	}

	hash := xxhash.Sum64([]byte(cfm.Data[KeplerConfigFile]))

	ds.Spec.Template.Annotations[ConfigMapHashAnnotation+"-"+cfm.Name] = fmt.Sprintf("%x", hash)
}
