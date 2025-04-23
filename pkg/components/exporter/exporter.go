// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	_ "embed"
	"fmt"
	"regexp"
	"strconv"

	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"

	secv1 "github.com/openshift/api/security/v1"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	ServicePortName = "http"

	overviewDashboardName = "power-monitoring-overview"
	nsInfoDashboardName   = "power-monitoring-by-ns"
	DashboardNs           = "openshift-config-managed"

	RedfishArgs             = "-redfish-cred-file-path=/etc/redfish/redfish.csv"
	RedfishCSV              = "redfish.csv"
	RedfishSecretAnnotation = "kepler.system.sustainable.computing.io/redfish-secret-ref"
	RedfishConfigHash       = "kepler.system.sustainable.computing.io/redfish-config-hash"
)

const (
	KeplerContainerIndex k8s.ContainerIndex = 0
)

var (
	linuxNodeSelector = k8s.StringMap{
		"kubernetes.io/os": "linux",
	}

	//go:embed assets/dashboards/power-monitoring-overview.json
	overviewDashboardJson string

	//go:embed assets/dashboards/power-monitoring-by-ns.json
	nsInfoDashboardJson string
)

// TODO:
func NewDaemonSet(detail components.Detail, k *v1alpha1.KeplerInternal) *appsv1.DaemonSet {
	if detail == components.Metadata {
		return &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{
				APIVersion: appsv1.SchemeGroupVersion.String(),
				Kind:       "DaemonSet",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      k.DaemonsetName(),
				Namespace: k.Namespace(),
				Labels:    labels(k),
			},
		}
	}

	deployment := k.Spec.Exporter.Deployment.ExporterDeploymentSpec
	nodeSelector := deployment.NodeSelector
	tolerations := deployment.Tolerations
	// NOTE: since 2 or more KeplerInternals can be deployed to the same namespace,
	// we need to make sure that the pod selector of each of the DaemonSet
	// create of each kepler is unique. Thus the daemonset name is added as
	// label to the pod

	exporterContainer := newExporterContainer(k.Name, k.DaemonsetName(), k.Spec.Exporter.Deployment)
	containers := []corev1.Container{exporterContainer}

	volumes := []corev1.Volume{
		k8s.VolumeFromHost("lib-modules", "/lib/modules"),
		k8s.VolumeFromHost("tracing", "/sys"),
		k8s.VolumeFromHost("proc", "/proc"),
		k8s.VolumeFromConfigMap("cfm", k.Name),
	} // exporter default Volumes

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k.Name,
			Namespace: k.Namespace(),
			Labels:    labels(k),
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: podSelector(k)},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      k.DaemonsetName(),
					Namespace: k.Namespace(),
					Labels:    podSelector(k),
				},
				Spec: corev1.PodSpec{
					HostPID:            true,
					NodeSelector:       linuxNodeSelector.Merge(nodeSelector),
					ServiceAccountName: k.Name,
					DNSPolicy:          corev1.DNSPolicy(corev1.DNSClusterFirstWithHostNet),
					Tolerations:        tolerations,
					Containers:         containers,
					Volumes:            volumes,
				}, // PodSpec
			}, // PodTemplateSpec
		}, // Spec
	}
}

func MountRedfishSecretToDaemonSet(ds *appsv1.DaemonSet, secret *corev1.Secret, hash uint64) {
	spec := &ds.Spec.Template.Spec
	keplerContainer := &spec.Containers[KeplerContainerIndex]
	keplerContainer.Command = append(keplerContainer.Command, RedfishArgs)
	keplerContainer.VolumeMounts = append(keplerContainer.VolumeMounts,
		corev1.VolumeMount{Name: "redfish-cred", MountPath: "/etc/redfish", ReadOnly: true},
	)
	spec.Volumes = append(spec.Volumes,
		k8s.VolumeFromSecret("redfish-cred", secret.ObjectMeta.Name))

	// NOTE: annotating the Pods with the secret's resource version
	// forces pods to be redeployed if the secret change
	ds.Spec.Template.Annotations = map[string]string{
		RedfishSecretAnnotation: secret.ResourceVersion,
		RedfishConfigHash:       strconv.FormatUint(hash, 10),
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

func NewOverviewDashboard(d components.Detail) *corev1.ConfigMap {
	objMeta := openshiftDashboardObjectMeta(overviewDashboardName)
	objMeta.Labels["console.openshift.io/odc-dashboard"] = "true"

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
			"power-monitoring-overview.json": overviewDashboardJson,
		},
	}
}

func NewNamespaceInfoDashboard(d components.Detail) *corev1.ConfigMap {
	objMeta := openshiftDashboardObjectMeta(nsInfoDashboardName)

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
			"power-monitoring-by-ns.json": nsInfoDashboardJson,
		},
	}
}

func NewConfigMap(d components.Detail, k *v1alpha1.KeplerInternal) *corev1.ConfigMap {
	if d == components.Metadata {
		return &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      k.Name,
				Namespace: k.Namespace(),
				Labels:    labels(k).ToMap(),
			},
		}
	}

	deployment := k.Spec.Exporter.Deployment.ExporterDeploymentSpec
	bindAddress := "0.0.0.0:" + strconv.Itoa(int(deployment.Port))

	modelConfig := ""

	exporterConfigMap := k8s.StringMap{
		"KEPLER_NAMESPACE":           k.Namespace(),
		"KEPLER_LOG_LEVEL":           "1",
		"METRIC_PATH":                "/metrics",
		"BIND_ADDRESS":               bindAddress,
		"ENABLE_GPU":                 "false",
		"ENABLE_QAT":                 "false",
		"ENABLE_EBPF_CGROUPID":       "true",
		"EXPOSE_HW_COUNTER_METRICS":  "true",
		"EXPOSE_IRQ_COUNTER_METRICS": "true",
		"EXPOSE_KUBELET_METRICS":     "true",
		"EXPOSE_CGROUP_METRICS":      "false",
		"ENABLE_PROCESS_METRICS":     "false",
		"CPU_ARCH_OVERRIDE":          "",
		"CGROUP_METRICS":             "*",
		"MODEL_CONFIG":               modelConfig,
	}

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k.Name,
			Namespace: k.Namespace(),
			Labels:    labels(k).ToMap(),
		},
		Data: exporterConfigMap,
	}
}

func NewClusterRole(c components.Detail, k *v1alpha1.KeplerInternal) *rbacv1.ClusterRole {
	if c == components.Metadata {
		return &rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacv1.SchemeGroupVersion.String(),
				Kind:       "ClusterRole",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   k.Name,
				Labels: labels(k),
			},
		}
	}

	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   k.Name,
			Labels: labels(k),
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"nodes/metrics", "nodes/proxy", "nodes/stats", "pods"},
			Verbs:     []string{"get", "watch", "list"},
		}},
	}
}

func NewClusterRoleBinding(c components.Detail, k *v1alpha1.KeplerInternal) *rbacv1.ClusterRoleBinding {
	if c == components.Metadata {
		return &rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacv1.SchemeGroupVersion.String(),
				Kind:       "ClusterRoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   k.Name,
				Labels: labels(k),
			},
		}
	}

	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   k.Name,
			Labels: labels(k),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     k.Name,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      k.Name,
			Namespace: k.Namespace(),
		}},
	}
}

func NewSCC(d components.Detail, ki *v1alpha1.KeplerInternal) *secv1.SecurityContextConstraints {
	if d == components.Metadata {
		return &secv1.SecurityContextConstraints{
			TypeMeta: metav1.TypeMeta{
				APIVersion: secv1.SchemeGroupVersion.String(),
				Kind:       "SecurityContextConstraints",
			},

			ObjectMeta: metav1.ObjectMeta{
				Name:   ki.Name,
				Labels: labels(ki),
			},
		}
	}

	return &secv1.SecurityContextConstraints{
		TypeMeta: metav1.TypeMeta{
			APIVersion: secv1.SchemeGroupVersion.String(),
			Kind:       "SecurityContextConstraints",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:   ki.Name,
			Labels: labels(ki),
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
		Users: []string{ki.FQServiceAccountName()},
		Volumes: []secv1.FSType{
			secv1.FSType("configMap"),
			secv1.FSType("secret"),
			secv1.FSType("projected"),
			secv1.FSType("emptyDir"),
			secv1.FSType("hostPath"),
		},
	}
}

func NewServiceAccount(ki *v1alpha1.KeplerInternal) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ki.Name,
			Namespace: ki.Namespace(),
			Labels:    labels(ki).ToMap(),
		},
	}
}

func NewService(k *v1alpha1.KeplerInternal) *corev1.Service {
	deployment := k.Spec.Exporter.Deployment.ExporterDeploymentSpec

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k.Name,
			Namespace: k.Namespace(),
			Labels:    labels(k).ToMap(),
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  podSelector(k),
			Ports: []corev1.ServicePort{
				{
					Name: ServicePortName,
					Port: int32(deployment.Port),
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: int32(deployment.Port),
					},
				},
			},
		},
	}
}

func NewServiceMonitor(k *v1alpha1.KeplerInternal) *monv1.ServiceMonitor {
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
			Name:      k.Name,
			Namespace: k.Namespace(),
			Labels:    labels(k).ToMap(),
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
				MatchLabels: labels(k),
			},
		},
	}
}

var promRuleInvalidChars = regexp.MustCompile(`[^a-zA-Z0-9]`)

func keplerRulePrefix(name string) string {
	ruleName := promRuleInvalidChars.ReplaceAllString(name, "_")
	return fmt.Sprintf("kepler:%s", ruleName)
}

func NewPrometheusRule(k *v1alpha1.KeplerInternal) *monv1.PrometheusRule {
	interval := monv1.Duration("15s")
	ns := k.Namespace()
	//
	// NOTE: recording rules have a kepler-internal name prefixed as
	// kepler:<kepler_internal> so that there is a unique rule created per
	// object and dashboards can rely on kepler:kepler:<rules> for the
	// `kepler` object.

	prefix := keplerRulePrefix(k.Name)

	return &monv1.PrometheusRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monv1.SchemeGroupVersion.String(),
			Kind:       "PrometheusRule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k.Name,
			Namespace: ns,
			Labels:    labels(k).ToMap(),
		},
		Spec: monv1.PrometheusRuleSpec{
			Groups: []monv1.RuleGroup{{
				Name:     "kepler.rules",
				Interval: &interval,
				Rules: []monv1.Rule{
					record(prefix, "container_joules_total:consumed:24h:all",
						fmt.Sprintf(`sum(
							increase(kepler_container_joules_total{namespace=%q}[24h:1m])
						)`, ns),
					),
					record(prefix, "container_joules_total:consumed:24h:by_ns",
						fmt.Sprintf(`sum by (container_namespace) (
								increase(kepler_container_joules_total{namespace=%q}[24h:1m])
						)`, ns),
					),

					record(prefix, "container_gpu_joules_total:consumed:1h:by_ns",
						fmt.Sprintf(`sum by (container_namespace) (
								increase(kepler_container_gpu_joules_total{namespace=%q}[1h:15s])
						)`, ns),
					),

					record(prefix, "container_dram_joules_total:consumed:1h:by_ns",
						fmt.Sprintf(`sum by (container_namespace) (
								increase(kepler_container_dram_joules_total{namespace=%q}[1h:15s])
						)`, ns),
					),

					record(prefix, "container_package_joules_total:consumed:1h:by_ns",
						fmt.Sprintf(`sum by (container_namespace) (
								increase(kepler_container_package_joules_total{namespace=%q}[1h:15s])
						)`, ns),
					),

					record(prefix, "container_other_joules_total:consumed:1h:by_ns",
						fmt.Sprintf(`sum by (container_namespace) (
								increase(kepler_container_other_joules_total{namespace=%q}[1h:15s])
						)`, ns),
					),

					// irate of joules = joules per second -> watts
					record(prefix, "container_gpu_watts:1m:by_ns_pod",
						fmt.Sprintf(`sum by (container_namespace, pod_name) (
								irate(kepler_container_gpu_joules_total{namespace=%q}[1m])
						)`, ns),
					),

					record(prefix, "container_package_watts:1m:by_ns_pod",
						fmt.Sprintf(`sum by (container_namespace, pod_name) (
								irate(kepler_container_package_joules_total{namespace=%q}[1m])
						)`, ns),
					),

					record(prefix, "container_other_watts:1m:by_ns_pod",
						fmt.Sprintf(`sum by (container_namespace, pod_name) (
								irate(kepler_container_other_joules_total{namespace=%q}[1m])
						)`, ns),
					),

					record(prefix, "container_dram_watts:1m:by_ns_pod",
						fmt.Sprintf(`sum by (container_namespace, pod_name) (
								irate(kepler_container_dram_joules_total{namespace=%q}[1m])
						)`, ns),
					),
				},
			}},
		},
	}
}

func record(prefix, name, expr string) monv1.Rule {
	return monv1.Rule{
		Expr:   intstr.IntOrString{Type: intstr.String, StrVal: expr},
		Record: prefix + ":" + name,
	}
}

func podSelector(ki *v1alpha1.KeplerInternal) k8s.StringMap {
	return labels(ki).Merge(k8s.StringMap{
		"app.kubernetes.io/name": "kepler-exporter",
	})
}

func labels(ki *v1alpha1.KeplerInternal) k8s.StringMap {
	return components.CommonLabels.Merge(k8s.StringMap{
		"app.kubernetes.io/component":                "exporter",
		"operator.sustainable-computing.io/internal": ki.Name,
		"app.kubernetes.io/part-of":                  ki.Name,
	})
}

func newExporterContainer(kiName, dsName string, deployment v1alpha1.InternalExporterDeploymentSpec) corev1.Container {
	bindAddress := "0.0.0.0:" + strconv.Itoa(int(deployment.Port))
	return corev1.Container{
		Name:            dsName,
		SecurityContext: &corev1.SecurityContext{Privileged: ptr.To(true)},
		Image:           deployment.Image,
		Command: []string{
			"/usr/bin/kepler",
			"-address", bindAddress,
			"-enable-cgroup-id=$(ENABLE_EBPF_CGROUPID)",
			"-enable-gpu=$(ENABLE_GPU)",
			"-v=$(KEPLER_LOG_LEVEL)",
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
			TimeoutSeconds:      10,
		},
		Env: []corev1.EnvVar{
			{Name: "NODE_IP", ValueFrom: k8s.EnvFromField("status.hostIP")},
			{Name: "NODE_NAME", ValueFrom: k8s.EnvFromField("spec.nodeName")},
			{Name: "KEPLER_LOG_LEVEL", ValueFrom: k8s.EnvFromConfigMap("KEPLER_LOG_LEVEL", kiName)},
			{Name: "ENABLE_GPU", ValueFrom: k8s.EnvFromConfigMap("ENABLE_GPU", kiName)},
			{Name: "ENABLE_EBPF_CGROUPID", ValueFrom: k8s.EnvFromConfigMap("ENABLE_EBPF_CGROUPID", kiName)},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "lib-modules", MountPath: "/lib/modules", ReadOnly: true},
			{Name: "tracing", MountPath: "/sys", ReadOnly: true},
			{Name: "proc", MountPath: "/proc"},
			{Name: "cfm", MountPath: "/etc/kepler/kepler.config"},
		},
	}
}
