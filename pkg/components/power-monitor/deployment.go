// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package powermonitor

import (
	_ "embed"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/cespare/xxhash/v2"
	secv1 "github.com/openshift/api/security/v1"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/internal/config"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
	OverviewDashboardName      = "power-monitor-overview"
	NamespaceInfoDashboardName = "power-monitor-namespace-info"

	SysFSMountPath      = "/host/sys"
	ProcFSMountPath     = "/host/proc"
	KeplerConfigMapPath = "/etc/kepler"
	KeplerConfigFile    = "config.yaml"

	// ConfigMap annotations
	ConfigMapHashAnnotation = "powermonitor.sustainable.computing.io/config-map-hash"

	// Secure Endpoint
	KubeRBACProxyContainerName    = "kube-rbac-proxy"
	SecurePort                    = 8443
	SecurePortName                = "https"
	KubeRBACProxyConfigMountPath  = "/etc/kube-rbac-proxy"
	PowerMonitorTLSMountPath      = "/etc/tls/private"
	SecretTokenHashAnnotation     = "powermonitor.sustainable.computing.io/secret-token-hash"
	SecretTLSHashAnnotation       = "powermonitor.sustainable.computing.io/secret-tls-hash"
	ConfigMapCAHashAnnotation     = "powermonitor.sustainable.computing.io/configmap-ca-hash"
	SecretTLSCertName             = "power-monitor-tls"
	SecretKubeRBACProxyConfigName = "power-monitor-kube-rbac-proxy-config"
	SecretUWMTokenName            = "prometheus-user-workload-token"
	PowerMonitorCertsCABundleName = "power-monitor-serving-certs-ca-bundle"
	ServiceAccountTokenKey        = "token"
	UWMServiceAccountName         = "prometheus-user-workload"
	UWMNamespace                  = "openshift-user-workload-monitoring"
)

var (
	linuxNodeSelector = k8s.StringMap{
		"kubernetes.io/os": "linux",
	}
	//go:embed assets/dashboards/power-monitor-overview.json
	infoDashboardJson string

	//go:embed assets/dashboards/power-monitor-namespace-info.json
	namespaceInfoDashboardJson string
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

	if pmi.Spec.Kepler.Deployment.Security.Mode == v1alpha1.SecurityModeRBAC {
		rbacContainer := newKubeRBACProxyContainer(pmi)
		pmContainers = append(pmContainers, rbacContainer)
		volumes = append(volumes,
			k8s.VolumeFromSecret(SecretTLSCertName, SecretTLSCertName),
			k8s.VolumeFromSecret(SecretKubeRBACProxyConfigName, SecretKubeRBACProxyConfigName),
		)
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
	service := &corev1.Service{
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
	if pmi.Spec.Kepler.Deployment.Security.Mode == v1alpha1.SecurityModeRBAC {
		service.Annotations = map[string]string{
			"service.beta.openshift.io/serving-cert-secret-name": SecretTLSCertName,
		}
		service.Spec.Ports = []corev1.ServicePort{{
			Name: SecurePortName,
			Port: int32(SecurePort),
			TargetPort: intstr.IntOrString{
				Type:   intstr.String,
				StrVal: SecurePortName,
			},
		}}
	}
	return service
}

func NewPowerMonitorNamespaceInfoDashboard(d components.Detail) *corev1.ConfigMap {
	return openshiftDashboardConfigMap(d, NamespaceInfoDashboardName, fmt.Sprintf("%s.json", NamespaceInfoDashboardName), namespaceInfoDashboardJson)
}

func NewPowerMonitorInfoDashboard(d components.Detail) *corev1.ConfigMap {
	return openshiftDashboardConfigMap(d, OverviewDashboardName, fmt.Sprintf("%s.json", OverviewDashboardName), infoDashboardJson)
}

func NewPowerMonitorConfigMap(d components.Detail, pmi *v1alpha1.PowerMonitorInternal, additionalConfigs ...string) (*corev1.ConfigMap, error) {
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
		}, nil
	}

	config, err := KeplerConfig(pmi, additionalConfigs...)

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
	}, err
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

	cr := &rbacv1.ClusterRole{
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
	if pmi.Spec.Kepler.Deployment.Security.Mode == v1alpha1.SecurityModeRBAC {
		tokenReviewRule := rbacv1.PolicyRule{
			APIGroups: []string{"authentication.k8s.io"},
			Resources: []string{"tokenreviews"},
			Verbs:     []string{"create"},
		}
		subjectAccessReviewsRule := rbacv1.PolicyRule{
			APIGroups: []string{"authorization.k8s.io"},
			Resources: []string{"subjectaccessreviews"},
			Verbs:     []string{"create"},
		}
		cr.Rules = append(cr.Rules, tokenReviewRule, subjectAccessReviewsRule)
	}
	return cr
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
		AllowHostPID:             true,
		ReadOnlyRootFilesystem:   true,

		FSGroup: secv1.FSGroupStrategyOptions{
			Type: secv1.FSGroupStrategyRunAsAny,
		},
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

func NewPowerMonitorServiceMonitor(d components.Detail, pmi *v1alpha1.PowerMonitorInternal) *monv1.ServiceMonitor {
	if d == components.Metadata {
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
		}
	}
	relabelings := []*monv1.RelabelConfig{{
		Action:      "replace",
		Regex:       "(.*)",
		Replacement: "$1",
		SourceLabels: []monv1.LabelName{
			"__meta_kubernetes_pod_node_name",
		},
		TargetLabel: "instance",
	}}

	sm := &monv1.ServiceMonitor{
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
	if pmi.Spec.Kepler.Deployment.Security.Mode == v1alpha1.SecurityModeRBAC {
		sm.Spec.Endpoints = []monv1.Endpoint{{
			Port:           SecurePortName,
			Scheme:         "https",
			RelabelConfigs: relabelings,
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
					ServerName: fmt.Sprintf("%s.%s.svc", pmi.Name, pmi.Namespace()),
				},
			},
		}}
	}
	return sm
}

func NewPowerMonitorCABundleConfigMap(d components.Detail, pmi *v1alpha1.PowerMonitorInternal) *corev1.ConfigMap {
	if d == components.Metadata {
		return &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      PowerMonitorCertsCABundleName,
				Namespace: pmi.Namespace(),
			},
		}
	}
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      PowerMonitorCertsCABundleName,
			Namespace: pmi.Namespace(),
			Annotations: map[string]string{
				"service.beta.openshift.io/inject-cabundle": "true",
			},
		},
		Data: map[string]string{},
	}
}

func NewPowerMonitorKubeRBACProxyConfig(d components.Detail, pmi *v1alpha1.PowerMonitorInternal) (*corev1.Secret, error) {
	if d == components.Metadata {
		return &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      SecretKubeRBACProxyConfigName,
				Namespace: pmi.Namespace(),
				Labels:    labels(pmi),
			},
		}, nil
	}
	configYAML, err := createKubeRBACConfig(pmi.Spec.Kepler.Deployment.Security.AllowedSANames)
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      SecretKubeRBACProxyConfigName,
			Namespace: pmi.Namespace(),
			Labels:    labels(pmi),
		},
		StringData: map[string]string{
			"config.yaml": configYAML,
		},
		Type: corev1.SecretTypeOpaque,
	}, err
}

func NewPowerMonitorUWMTokenSecret(d components.Detail, pmi *v1alpha1.PowerMonitorInternal, saToken string) *corev1.Secret {
	if d == components.Metadata {
		return &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      SecretUWMTokenName,
				Namespace: pmi.Namespace(),
				Labels:    labels(pmi),
			},
		}
	}
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      SecretUWMTokenName,
			Namespace: pmi.Namespace(),
			Labels:    labels(pmi),
		},
		StringData: map[string]string{
			ServiceAccountTokenKey: saToken,
		},
		Type: corev1.SecretTypeOpaque,
	}
}

// AnnotateDaemonSetWithConfigMapHash sets annotations on the DaemonSet's pod template to trigger a rollout when the ConfigMap changes
func AnnotateDaemonSetWithConfigMapHash(ds *appsv1.DaemonSet, cfm *corev1.ConfigMap) {
	if ds.Spec.Template.Annotations == nil {
		ds.Spec.Template.Annotations = make(map[string]string)
	}

	hash := xxhash.Sum64([]byte(cfm.Data[KeplerConfigFile]))

	ds.Spec.Template.Annotations[ConfigMapHashAnnotation+"-"+cfm.Name] = fmt.Sprintf("%x", hash)
}

// AnnotateDaemonSetWithSecretHash sets annotations on the DaemonSet's pod template to trigger a rollout when the Secret changes
func AnnotateDaemonSetWithSecretHash(ds *appsv1.DaemonSet, s *corev1.Secret) {
	if ds.Spec.Template.Annotations == nil {
		ds.Spec.Template.Annotations = make(map[string]string)
	}
	var data []byte
	keys := make([]string, 0, len(s.Data))
	for k := range s.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		data = append(data, []byte(k)...)
		data = append(data, s.Data[k]...)
	}
	hash := xxhash.Sum64([]byte(data))

	ds.Spec.Template.Annotations[SecretTLSHashAnnotation+"-"+s.Name] = fmt.Sprintf("%x", hash)
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
	webListenAddress := fmt.Sprintf("0.0.0.0:%d", PowerMonitorDSPort)

	if pmi.Spec.Kepler.Deployment.Security.Mode == v1alpha1.SecurityModeRBAC {
		webListenAddress = fmt.Sprintf("127.0.0.1:%d", PowerMonitorDSPort)
	}

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
			fmt.Sprintf("--web.listen-address=%s", webListenAddress),
		},
		Ports: []corev1.ContainerPort{{
			ContainerPort: int32(PowerMonitorDSPort),
			Name:          PowerMonitorServicePortName,
		}},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "sysfs", MountPath: SysFSMountPath, ReadOnly: true},
			{Name: "procfs", MountPath: ProcFSMountPath, ReadOnly: true},
			{Name: "cfm", MountPath: KeplerConfigMapPath},
		},
	}
}

func newKubeRBACProxyContainer(pmi *v1alpha1.PowerMonitorInternal) corev1.Container {
	deployment := pmi.Spec.Kepler.Deployment
	return corev1.Container{
		Name:  KubeRBACProxyContainerName,
		Image: deployment.KubeRbacProxyImage,
		Args: []string{
			fmt.Sprintf("--secure-listen-address=0.0.0.0:%d", SecurePort),
			fmt.Sprintf("--upstream=http://127.0.0.1:%d", PowerMonitorDSPort),
			fmt.Sprintf("--auth-token-audiences=%s.%s.svc", pmi.Name, pmi.Namespace()),
			fmt.Sprintf("--config-file=%s/config.yaml", KubeRBACProxyConfigMountPath),
			fmt.Sprintf("--tls-cert-file=%s/tls.crt", PowerMonitorTLSMountPath),
			fmt.Sprintf("--tls-private-key-file=%s/tls.key", PowerMonitorTLSMountPath),
			"--allow-paths=/metrics",
			"--logtostderr=true",
			"--v=3",
		},
		Ports: []corev1.ContainerPort{{
			Name:          SecurePortName,
			ContainerPort: int32(SecurePort),
		}},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1m"),
				corev1.ResourceMemory: resource.MustParse("15Mi"),
			},
		},
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.To(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		},
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		VolumeMounts: []corev1.VolumeMount{
			{Name: SecretKubeRBACProxyConfigName, MountPath: KubeRBACProxyConfigMountPath, ReadOnly: true},
			{Name: SecretTLSCertName, MountPath: PowerMonitorTLSMountPath, ReadOnly: true},
		},
	}
}

func createKubeRBACConfig(serviceAccountNames []string) (string, error) {
	var config k8s.KubeRBACProxyConfig

	for _, serviceAccountName := range serviceAccountNames {
		newRule := k8s.StaticRule{
			Path:            "/metrics",
			ResourceRequest: false,
			User: k8s.User{
				Name: fmt.Sprintf("system:serviceaccount:%s", serviceAccountName),
			},
			Verb: "get",
		}
		config.Authorization.StaticRules = append(config.Authorization.StaticRules, newRule)
	}
	yamlData, err := yaml.Marshal(&config)
	return string(yamlData), err
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

	cfg.Log.Level = pmi.Spec.Kepler.Config.LogLevel
	cfg.Host.SysFS = SysFSMountPath
	cfg.Host.ProcFS = ProcFSMountPath

	// Set metrics level if specified, otherwise use PowerMonitor default
	if len(pmi.Spec.Kepler.Config.MetricLevels) > 0 {
		level, err := config.ParseLevel(pmi.Spec.Kepler.Config.MetricLevels)
		if err != nil {
			cfg.Exporter.Prometheus.MetricsLevel = v1alpha1.MetricsLevelDefault
		} else {
			cfg.Exporter.Prometheus.MetricsLevel = level
		}
	} else {
		cfg.Exporter.Prometheus.MetricsLevel = v1alpha1.MetricsLevelDefault
	}

	// Set staleness if specified, otherwise use PowerMonitor default
	if pmi.Spec.Kepler.Config.Staleness != nil {
		cfg.Monitor.Staleness = pmi.Spec.Kepler.Config.Staleness.Duration
	} else {
		cfg.Monitor.Staleness = 500 * time.Millisecond
	}

	// Set sample rate if specified, otherwise use PowerMonitor default
	if pmi.Spec.Kepler.Config.SampleRate != nil {
		cfg.Monitor.Interval = pmi.Spec.Kepler.Config.SampleRate.Duration
	} else {
		cfg.Monitor.Interval = 5 * time.Second
	}

	// Set max terminated if specified, otherwise use PowerMonitor default
	if pmi.Spec.Kepler.Config.MaxTerminated != nil {
		cfg.Monitor.MaxTerminated = int(*pmi.Spec.Kepler.Config.MaxTerminated)
	} else {
		cfg.Monitor.MaxTerminated = 500
	}

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
