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
	"github.com/sustainable.computing.io/kepler-operator/pkg/components/estimator"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components/modelserver"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"

	secv1 "github.com/openshift/api/security/v1"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
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

	ServiceName        = prefix + "svc"
	ServicePortName    = "http"
	ServiceMonitorName = prefix + "smon"

	StableImage = "quay.io/sustainable_computing_io/kepler:release-0.5.5"
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

func newExporterContainer(deployment v1alpha1.ExporterDeploymentSpec) *corev1.Container {
	bindAddress := "0.0.0.0:" + strconv.Itoa(int(deployment.Port))
	return &corev1.Container{
		Name:            "kepler-exporter",
		SecurityContext: &corev1.SecurityContext{Privileged: pointer.Bool(true)},
		Image:           Config.Image,
		Command: []string{
			"/usr/bin/kepler",
			"-address", bindAddress,
			"-enable-gpu=$(ENABLE_GPU)",
			"-enable-cgroup-id=true",
			"-v=$(KEPLER_LOG_LEVEL)",
			"-kernel-source-dir=/usr/share/kepler/kernel_sources",
			"-redfish-cred-file-path=/etc/redfish/redfish.csv",
		},
		Ports: []corev1.ContainerPort{{
			ContainerPort: int32(deployment.Port),
			Name:          "http",
		}},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/healthz",
					Port:   intstr.IntOrString{Type: intstr.Int, IntVal: deployment.Port},
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
			{Name: "NODE_NAME", ValueFrom: k8s.EnvFromField("spec.nodeName")},
			{Name: "KEPLER_LOG_LEVEL", ValueFrom: k8s.EnvFromConfigMap("KEPLER_LOG_LEVEL", ConfigmapName)},
			{Name: "ENABLE_GPU", ValueFrom: k8s.EnvFromConfigMap("ENABLE_GPU", ConfigmapName)}},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "lib-modules", MountPath: "/lib/modules", ReadOnly: true},
			{Name: "tracing", MountPath: "/sys", ReadOnly: true},
			{Name: "kernel-src", MountPath: "/usr/src/kernels", ReadOnly: true},
			{Name: "kernel-debug", MountPath: "/sys/kernel/debug"},
			{Name: "proc", MountPath: "/proc"},
			{Name: "cfm", MountPath: "/etc/kepler/kepler.config"},
		},
	}
}

func addEstimatorSidecar(exporterContainer *corev1.Container, volumes []corev1.Volume) ([]corev1.Container, []corev1.Volume) {
	sidecarContainer := estimator.NewContainer()
	volumes = append(volumes, estimator.NewVolumes()...)
	exporterContainer = estimator.AddEstimatorDependency(exporterContainer)
	containers := []corev1.Container{*exporterContainer, sidecarContainer}
	return containers, volumes
}

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

	deployment := k.Spec.Exporter.Deployment
	nodeSelector := deployment.NodeSelector

	tolerations := deployment.Tolerations
	if len(tolerations) == 0 {
		tolerations = defaultTolerations
	}

	exporterContainer := newExporterContainer(deployment)

	var containers []corev1.Container

	// exporter volumes
	var volumes = []corev1.Volume{
		k8s.VolumeFromHost("lib-modules", "/lib/modules"),
		k8s.VolumeFromHost("tracing", "/sys"),
		k8s.VolumeFromHost("proc", "/proc"),
		k8s.VolumeFromHost("kernel-src", "/usr/src/kernels"),
		k8s.VolumeFromHost("kernel-debug", "/sys/kernel/debug"),
		k8s.VolumeFromConfigMap("cfm", ConfigmapName),
	}

	containers = []corev1.Container{*exporterContainer}

	if k.Spec.Estimator != nil && estimator.NeedsEstimatorSidecar(k.Spec.Estimator) {
		containers, volumes = addEstimatorSidecar(exporterContainer, volumes)
	}

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
					Containers:         containers,
					Volumes:            volumes,
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

	deployment := k.Spec.Exporter.Deployment
	bindAddress := "0.0.0.0:" + strconv.Itoa(int(deployment.Port))

	modelConfig := ""
	if k.Spec.Estimator != nil {
		modelConfig = estimator.ModelConfig(k.Spec.Estimator)
	}

	exporterConfigMap := k8s.StringMap{
		"KEPLER_NAMESPACE":                  components.Namespace,
		"KEPLER_LOG_LEVEL":                  "1",
		"METRIC_PATH":                       "/metrics",
		"BIND_ADDRESS":                      bindAddress,
		"ENABLE_GPU":                        "true",
		"ENABLE_QAT":                        "false",
		"ENABLE_EBPF_CGROUPID":              "true",
		"EXPOSE_HW_COUNTER_METRICS":         "true",
		"EXPOSE_IRQ_COUNTER_METRICS":        "true",
		"EXPOSE_KUBELET_METRICS":            "true",
		"EXPOSE_CGROUP_METRICS":             "true",
		"ENABLE_PROCESS_METRICS":            "false",
		"CPU_ARCH_OVERRIDE":                 "",
		"CGROUP_METRICS":                    "*",
		"REDFISH_PROBE_INTERVAL_IN_SECONDS": "60",
		"REDFISH_SKIP_SSL_VERIFY":           "true",
		"MODEL_CONFIG":                      modelConfig,
	}

	ms := k.Spec.ModelServer
	if ms != nil {
		if ms.Enabled {
			exporterConfigMap["MODEL_SERVER_ENABLE"] = "true"
		}
		modelServerConfig := modelserver.ModelServerConfigForClient(k.Spec.ModelServer)
		exporterConfigMap = exporterConfigMap.Merge(modelServerConfig)
	}

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
		Data: exporterConfigMap,
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
	deployment := k.Spec.Exporter.Deployment

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
				Name: ServicePortName,
				Port: int32(deployment.Port),
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: int32(deployment.Port),
				}},
			},
		},
	}
}

func NewServiceMonitor() *monv1.ServiceMonitor {
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
			Name:      ServiceMonitorName,
			Namespace: components.Namespace,
			Labels:    labels,
		},
		Spec: monv1.ServiceMonitorSpec{
			Endpoints: []monv1.Endpoint{{
				Port:           ServicePortName,
				Interval:       "3s",
				Scheme:         "http",
				RelabelConfigs: relabelings,
			}},
			JobLabel: "app.kubernetes.io/name",
			Selector: metav1.LabelSelector{
				MatchLabels: labels,
			},
		},
	}
}
