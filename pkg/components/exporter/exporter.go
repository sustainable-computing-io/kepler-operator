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

package exporter

import (
	"strconv"

	"github.com/sustainable.computing.io/kepler-operator/pkg/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"

	secv1 "github.com/openshift/api/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

const (
	prefix = "kepler-exporter-"

	SCCName = prefix + "scc"

	ServiceAccountName   = prefix + "sa"
	FQServiceAccountName = "system:serviceaccount:" + components.Namespace + ":" + ServiceAccountName

	ClusterRoleName        = prefix + "cr"
	ClusterRoleBindingName = prefix + "crb"

	ConfigmapName = prefix + "cm"
	DaemonSetName = prefix + "ds"
	ServiceName   = prefix + "svc"

	StableImage = "quay.io/sustainable_computing_io/kepler:release-0.5.4"
)

// Config that will be set from outside
var (
	Config = struct {
		Image string
	}{
		Image: StableImage,
	}
)

var (
	labels = components.CommonLabels.Merge(k8s.StringMap{
		"app.kubernetes.io/component":  "exporter",
		"sustainable-computing.io/app": "kepler",
	})

	podSelector = labels.Merge(k8s.StringMap{
		"app.kubernetes.io/name": "kepler-exporter",
	})

	linuxNodeSelector = k8s.StringMap{
		"kubernetes.io/os": "linux",
	}

	// Default Toleration is allow all nodes
	defaultTolerations = []corev1.Toleration{{
		Operator: corev1.TolerationOpExists,
	}}
)

func NewDaemonSet(detail components.Detail, k *v1alpha1.Kepler) *appsv1.DaemonSet {
	if detail == components.Metadata {
		return &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{APIVersion: appsv1.SchemeGroupVersion.String(), Kind: "DaemonSet"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      DaemonSetName,
				Namespace: components.Namespace,
				Labels:    labels,
			},
		}
	}

	exporter := k.Spec.Exporter
	nodeSelector := exporter.NodeSelector

	tolerations := exporter.Tolerations
	if len(tolerations) == 0 {
		tolerations = defaultTolerations
	}

	bindAddress := "0.0.0.0:" + strconv.Itoa(int(exporter.Port))

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      DaemonSetName,
			Namespace: components.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: podSelector},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      DaemonSetName,
					Namespace: components.Namespace,
					Labels:    podSelector,
				},
				Spec: corev1.PodSpec{
					HostPID:            true,
					NodeSelector:       linuxNodeSelector.Merge(nodeSelector),
					ServiceAccountName: ServiceAccountName,
					DNSPolicy:          corev1.DNSPolicy(corev1.DNSClusterFirstWithHostNet),
					Tolerations:        tolerations,
					Containers: []corev1.Container{{
						Name:            "kepler-exporter",
						SecurityContext: &corev1.SecurityContext{Privileged: pointer.Bool(true)},
						Image:           Config.Image,
						Command: []string{
							"/usr/bin/kepler",
							"-address", bindAddress,
							"-enable-gpu=true",
							"-enable-cgroup-id=true",
							"-v=1",
							"-kernel-source-dir=/usr/share/kepler/kernel_sources",
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: int32(exporter.Port),
							Name:          "http",
						}},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/healthz",
									Port:   intstr.IntOrString{Type: intstr.Int, IntVal: exporter.Port},
									Scheme: "HTTP",
								},
							},
							FailureThreshold:    5,
							InitialDelaySeconds: 10,
							PeriodSeconds:       60,
							SuccessThreshold:    1,
							TimeoutSeconds:      10},
						Env: []corev1.EnvVar{
							{Name: "NODE_IP", ValueFrom: k8s.EnvFromField("status.hostIP")},
							{Name: "NODE_NAME", ValueFrom: k8s.EnvFromField("spec.nodeName")}},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "lib-modules", MountPath: "/lib/modules", ReadOnly: true},
							{Name: "tracing", MountPath: "/sys", ReadOnly: true},
							{Name: "kernel-src", MountPath: "/usr/src/kernels", ReadOnly: true},
							{Name: "proc", MountPath: "/proc"},
							{Name: "cfm", MountPath: "/etc/kepler/kepler.config"},
						}, // VolumeMounts
					}}, // Container: kepler /  Containers
					Volumes: []corev1.Volume{
						k8s.VolumeFromHost("lib-modules", "/lib/modules"),
						k8s.VolumeFromHost("tracing", "/sys"),
						k8s.VolumeFromHost("proc", "/proc"),
						k8s.VolumeFromHost("kernel-src", "/usr/src/kernels"),
						k8s.VolumeFromConfigMap("cfm", ConfigmapName),
					}, // Volumes
				}, // PodSpec
			}, // PodTemplateSpec
		}, // Spec
	}

}

func NewConfigMap(d components.Detail, k *v1alpha1.Kepler) *corev1.ConfigMap {
	if d == components.Metadata {
		return &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      ConfigmapName,
				Namespace: components.Namespace,
				Labels:    labels,
			},
		}
	}

	exporter := k.Spec.Exporter
	bindAddress := "0.0.0.0:" + strconv.Itoa(int(exporter.Port))

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigmapName,
			Namespace: components.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			// TODO: decide what this should be
			"KEPLER_NAMESPACE":     components.Namespace,
			"KEPLER_LOG_LEVEL":     "5",
			"METRIC_PATH":          "/metrics",
			"BIND_ADDRESS":         bindAddress,
			"ENABLE_GPU":           "true",
			"ENABLE_EBPF_CGROUPID": "true",
			"CPU_ARCH_OVERRIDE":    "",
			"CGROUP_METRICS":       "*",
			// TODO: clean this long time
			"MODEL_CONFIG": "| CONTAINER_COMPONENTS_ESTIMATOR=false CONTAINER_COMPONENTS_INIT_URL=https://raw.githubusercontent.com/sustainable-computing-io/kepler-model-server/main/tests/test_models/DynComponentModelWeight/CgroupOnly/ScikitMixed/ScikitMixed.json",

			"EXPOSE_HW_COUNTER_METRICS": "true",
			"EXPOSE_CGROUP_METRICS":     "true",
		},
	}
}

func NewClusterRole(c components.Detail) *rbacv1.ClusterRole {
	if c == components.Metadata {
		return &rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacv1.SchemeGroupVersion.String(),
				Kind:       "ClusterRole",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   ClusterRoleName,
				Labels: labels,
			},
		}
	}

	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   ClusterRoleName,
			Labels: labels,
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"nodes/metrics", "nodes/proxy", "nodes/stats", "pods"},
			Verbs:     []string{"get", "watch", "list"},
		}},
	}
}

func NewClusterRoleBinding(c components.Detail) *rbacv1.ClusterRoleBinding {
	if c == components.Metadata {
		return &rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacv1.SchemeGroupVersion.String(),
				Kind:       "ClusterRoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   ClusterRoleBindingName,
				Labels: labels,
			},
		}
	}

	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   ClusterRoleBindingName,
			Labels: labels,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     ClusterRoleName,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      ServiceAccountName,
			Namespace: components.Namespace,
		}},
	}
}

func NewSCC(d components.Detail, k *v1alpha1.Kepler) *secv1.SecurityContextConstraints {
	if d == components.Metadata {
		return &secv1.SecurityContextConstraints{
			TypeMeta: metav1.TypeMeta{
				APIVersion: secv1.SchemeGroupVersion.String(),
				Kind:       "SecurityContextConstraints",
			},

			ObjectMeta: metav1.ObjectMeta{
				Name:   SCCName,
				Labels: labels,
			},
		}
	}

	return &secv1.SecurityContextConstraints{
		TypeMeta: metav1.TypeMeta{
			APIVersion: secv1.SchemeGroupVersion.String(),
			Kind:       "SecurityContextConstraints",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:   SCCName,
			Labels: labels,
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
		//TODO: decide if "kepler" is really needed?
		Users: []string{"kepler", FQServiceAccountName},
		Volumes: []secv1.FSType{
			secv1.FSType("configMap"),
			secv1.FSType("projected"),
			secv1.FSType("emptyDir"),
			secv1.FSType("hostPath")},
	}
}

func NewServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceAccountName,
			Namespace: components.Namespace,
			Labels:    labels,
		},
	}
}

func NewService(k *v1alpha1.Kepler) *corev1.Service {
	exporter := k.Spec.Exporter

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceName,
			Namespace: components.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{

			ClusterIP: "None",
			Selector:  podSelector,
			Ports: []corev1.ServicePort{{
				Name: "http",
				Port: int32(exporter.Port),
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: int32(exporter.Port),
				}},
			},
		},
	}
}
