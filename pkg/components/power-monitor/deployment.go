/*
Copyright 2023.

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

package powermonitor

import (
	_ "embed"

	secv1 "github.com/openshift/api/security/v1"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"github.com/sustainable.computing.io/kepler/pkg/config"
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
	PowerMonitorDSPort          = 28282
	DashboardNs                 = "openshift-config-managed"
)

var (
	linuxNodeSelector = k8s.StringMap{
		"kubernetes.io/os": "linux",
	}
)

func NewPowerMonitorDaemonSet(detail components.Detail, kx *v1alpha1.KeplerX) *appsv1.DaemonSet {
	if detail == components.Metadata {
		return &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{
				APIVersion: appsv1.SchemeGroupVersion.String(),
				Kind:       "DaemonSet",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      kx.DaemonsetName(),
				Namespace: kx.Namespace(),
				Labels:    labels(kx),
			},
		}
	}
	deployment := kx.Spec.Deployment
	nodeSelector := deployment.NodeSelector
	tolerations := deployment.Tolerations

	pmContainer := newPowerMonitorContainer(kx)
	pmContainers := []corev1.Container{pmContainer}

	var volumes = []corev1.Volume{
		k8s.VolumeFromHost("lib-modules", "/lib/modules"),
		k8s.VolumeFromHost("tracing", "/sys"),
		k8s.VolumeFromHost("proc", "/proc"),
		k8s.VolumeFromConfigMap("cfm", kx.Name),
	}

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kx.Name,
			Namespace: kx.Namespace(),
			Labels:    labels(kx),
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: podSelector(kx)},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kx.DaemonsetName(),
					Namespace: kx.Namespace(),
					Labels:    podSelector(kx),
				},
				Spec: corev1.PodSpec{
					HostPID:            true,
					NodeSelector:       linuxNodeSelector.Merge(nodeSelector),
					ServiceAccountName: kx.Name,
					DNSPolicy:          corev1.DNSPolicy(corev1.DNSClusterFirstWithHostNet),
					Tolerations:        tolerations,
					Containers:         pmContainers,
					Volumes:            volumes,
				}, // PodSpec
			}, // PodTemplateSpec
		}, // Spec
	}
}

func NewPowerMonitorService(kx *v1alpha1.KeplerX) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kx.Name,
			Namespace: kx.Namespace(),
			Labels:    labels(kx).ToMap(),
		},
		Spec: corev1.ServiceSpec{

			ClusterIP: "None",
			Selector:  podSelector(kx),
			Ports: []corev1.ServicePort{{
				Name: PowerMonitorServicePortName,
				Port: int32(PowerMonitorDSPort),
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: int32(PowerMonitorDSPort),
				}},
			},
		},
	}
}

func NewPowerMonitorConfigMap(d components.Detail, kx *v1alpha1.KeplerX) *corev1.ConfigMap {
	if d == components.Metadata {
		return &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      kx.Name,
				Namespace: kx.Namespace(),
				Labels:    labels(kx).ToMap(),
			},
		}
	}

	config, _ := keplerConfig(&kx.Spec.Configuration)

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kx.Name,
			Namespace: kx.Namespace(),
			Labels:    labels(kx).ToMap(),
		},
		Data: k8s.StringMap{
			"kepler-config.yaml": config,
		},
	}
}

func NewPowerMonitorClusterRole(c components.Detail, kx *v1alpha1.KeplerX) *rbacv1.ClusterRole {
	if c == components.Metadata {
		return &rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacv1.SchemeGroupVersion.String(),
				Kind:       "ClusterRole",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   kx.Name,
				Labels: labels(kx),
			},
		}
	}

	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   kx.Name,
			Labels: labels(kx),
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"nodes/metrics", "nodes/proxy", "nodes/stats", "pods"},
			Verbs:     []string{"get", "watch", "list"},
		}},
	}
}

func NewPowerMonitorClusterRoleBinding(c components.Detail, kx *v1alpha1.KeplerX) *rbacv1.ClusterRoleBinding {
	if c == components.Metadata {
		return &rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacv1.SchemeGroupVersion.String(),
				Kind:       "ClusterRoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   kx.Name,
				Labels: labels(kx),
			},
		}
	}

	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   kx.Name,
			Labels: labels(kx),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     kx.Name,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      kx.Name,
			Namespace: kx.Namespace(),
		}},
	}
}

func NewPowerMonitorSCC(d components.Detail, kx *v1alpha1.KeplerX) *secv1.SecurityContextConstraints {
	if d == components.Metadata {
		return &secv1.SecurityContextConstraints{
			TypeMeta: metav1.TypeMeta{
				APIVersion: secv1.SchemeGroupVersion.String(),
				Kind:       "SecurityContextConstraints",
			},

			ObjectMeta: metav1.ObjectMeta{
				Name:   kx.Name,
				Labels: labels(kx),
			},
		}
	}

	return &secv1.SecurityContextConstraints{
		TypeMeta: metav1.TypeMeta{
			APIVersion: secv1.SchemeGroupVersion.String(),
			Kind:       "SecurityContextConstraints",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:   kx.Name,
			Labels: labels(kx),
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
		Users: []string{kx.FQServiceAccountName()},
		Volumes: []secv1.FSType{
			secv1.FSType("configMap"),
			secv1.FSType("secret"),
			secv1.FSType("projected"),
			secv1.FSType("emptyDir"),
			secv1.FSType("hostPath")},
	}
}

func NewPowerMonitorServiceAccount(kx *v1alpha1.KeplerX) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kx.Name,
			Namespace: kx.Namespace(),
			Labels:    labels(kx).ToMap(),
		},
	}
}

func NewPowerMonitorServiceMonitor(kx *v1alpha1.KeplerX) *monv1.ServiceMonitor {
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
			Name:      kx.Name,
			Namespace: kx.Namespace(),
			Labels:    labels(kx).ToMap(),
		},
		Spec: monv1.ServiceMonitorSpec{
			Endpoints: []monv1.Endpoint{{
				Port:           PowerMonitorServicePortName,
				Interval:       "15s",
				Scheme:         "http",
				RelabelConfigs: relabelings,
			}},
			JobLabel: "app.kubernetes.io/name",
			Selector: metav1.LabelSelector{
				MatchLabels: labels(kx),
			},
		},
	}
}

func labels(kx *v1alpha1.KeplerX) k8s.StringMap {
	return components.CommonLabels.Merge(k8s.StringMap{
		"app.kubernetes.io/component":                "exporter",
		"operator.sustainable-computing.io/internal": kx.Name,
		"app.kubernetes.io/part-of":                  kx.Name,
	})
}

func podSelector(kx *v1alpha1.KeplerX) k8s.StringMap {
	return labels(kx).Merge(k8s.StringMap{
		"app.kubernetes.io/name":      "power-monitor-exporter",
		"app.kubernetes.io/component": "exporter",
	})
}

func keplerCommand() []string {
	return []string{
		"/usr/bin/kepler",
		"--config.file=/etc/kepler/kepler-config.yaml",
	}
}

func newPowerMonitorContainer(kx *v1alpha1.KeplerX) corev1.Container {
	deployment := kx.Spec.Deployment
	return corev1.Container{
		Name:            kx.DaemonsetName(),
		SecurityContext: &corev1.SecurityContext{Privileged: ptr.To(true)},
		Image:           deployment.Image,
		Command:         keplerCommand(),
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
			TimeoutSeconds:      10},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "lib-modules", MountPath: "/lib/modules", ReadOnly: true},
			{Name: "tracing", MountPath: "/sys", ReadOnly: true},
			{Name: "proc", MountPath: "/proc"},
			{Name: "cfm", MountPath: "/etc/kepler"}, // TODO: Modify configmap mountpath
		},
	}
}

func keplerConfig(kxConfig *v1alpha1.KeplerXConfigSpec) (string, error) {
	// Generate DefaultConfig
	cf := config.DefaultConfig()
	cf.Log.Level = "debug"
	// Modify DefaultFields according to KeplerXConfigSpec
	// 	"MONITOR_INTERVAL":                 strconv.FormatInt(int64(config.MonitorInterval), 10),
	// 	"EXPORTER_INTERVAL":                strconv.FormatInt(int64(config.ExporterInterval), 10),
	// 	"MAX_TRACKED_TERMINATED_PROCESSES": strconv.FormatInt(int64(config.MaxTrackedTerminatedProcesses), 10),
	// 	"PROCESS_RETENTION_PERIOD":         strconv.FormatInt(int64(config.ProcessRetentionPeriod), 10),

	// Validate Config
	if err := cf.Validate(); err != nil {
		cf = config.DefaultConfig()
	}

	// Return Config in form of Yaml String
	return cf.String(), nil
}
