package exporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/pkg/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNodeSelection(t *testing.T) {

	tt := []struct {
		spec     v1alpha1.InternalExporterSpec
		selector map[string]string
		scenario string
	}{
		{
			spec:     v1alpha1.InternalExporterSpec{},
			selector: map[string]string{"kubernetes.io/os": "linux"},
			scenario: "default case",
		},
		{
			spec: v1alpha1.InternalExporterSpec{
				Deployment: v1alpha1.InternalExporterDeploymentSpec{
					ExporterDeploymentSpec: v1alpha1.ExporterDeploymentSpec{
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
			k := v1alpha1.KeplerInternal{
				Spec: v1alpha1.KeplerInternalSpec{
					Exporter: tc.spec,
				},
			}
			actual := k8s.NodeSelectorFromDS(NewDaemonSet(components.Full, &k))
			assert.Equal(t, actual, tc.selector)
		})
	}
}

func TestTolerations(t *testing.T) {

	tt := []struct {
		spec        v1alpha1.InternalExporterSpec
		tolerations []corev1.Toleration
		scenario    string
	}{{
		spec: v1alpha1.InternalExporterSpec{},
		// NOTE: default toleration { "operator": "Exists" } is set by k8s API server (CRD default)
		// see: Kepler_Reconciliation e2e test
		tolerations: nil,
		scenario:    "default case",
	}, {
		spec: v1alpha1.InternalExporterSpec{
			Deployment: v1alpha1.InternalExporterDeploymentSpec{
				ExporterDeploymentSpec: v1alpha1.ExporterDeploymentSpec{
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
			k := v1alpha1.KeplerInternal{
				Spec: v1alpha1.KeplerInternalSpec{
					Exporter: tc.spec,
				},
			}
			actual := k8s.TolerationsFromDS(NewDaemonSet(components.Full, &k))
			assert.Equal(t, actual, tc.tolerations)
		})
	}
}

func TestHostPID(t *testing.T) {
	tt := []struct {
		spec     v1alpha1.InternalExporterSpec
		hostPID  bool
		scenario string
	}{
		{
			spec:     v1alpha1.InternalExporterSpec{},
			hostPID:  true,
			scenario: "default case",
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			k := v1alpha1.KeplerInternal{
				Spec: v1alpha1.KeplerInternalSpec{
					Exporter: tc.spec,
				},
			}
			actual := k8s.HostPIDFromDS(NewDaemonSet(components.Full, &k))
			assert.Equal(t, actual, tc.hostPID)
		})
	}
}
func TestVolumeMounts(t *testing.T) {
	tt := []struct {
		spec         v1alpha1.InternalExporterSpec
		volumeMounts []corev1.VolumeMount
		scenario     string
	}{
		{
			spec: v1alpha1.InternalExporterSpec{},
			volumeMounts: []corev1.VolumeMount{
				{Name: "lib-modules", MountPath: "/lib/modules", ReadOnly: true},
				{Name: "tracing", MountPath: "/sys", ReadOnly: true},
				{Name: "kernel-src", MountPath: "/usr/src/kernels", ReadOnly: true},
				{Name: "kernel-debug", MountPath: "/sys/kernel/debug"},
				{Name: "proc", MountPath: "/proc"},
				{Name: "cfm", MountPath: "/etc/kepler/kepler.config"},
			},
			scenario: "default case",
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			k := v1alpha1.KeplerInternal{
				Spec: v1alpha1.KeplerInternalSpec{
					Exporter: tc.spec,
				},
			}
			actual := k8s.VolumeMountsFromDS(NewDaemonSet(components.Full, &k))
			assert.Equal(t, actual, tc.volumeMounts)
		})
	}
}
func TestVolumes(t *testing.T) {
	tt := []struct {
		spec     v1alpha1.InternalExporterSpec
		volumes  []corev1.Volume
		scenario string
	}{
		{
			spec: v1alpha1.InternalExporterSpec{},
			volumes: []corev1.Volume{
				k8s.VolumeFromHost("lib-modules", "/lib/modules"),
				k8s.VolumeFromHost("tracing", "/sys"),
				k8s.VolumeFromHost("proc", "/proc"),
				k8s.VolumeFromHost("kernel-src", "/usr/src/kernels"),
				k8s.VolumeFromHost("kernel-debug", "/sys/kernel/debug"),
				k8s.VolumeFromConfigMap("cfm", "kepler-internal"),
			},
			scenario: "default case",
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			k := v1alpha1.KeplerInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kepler-internal",
				},
				Spec: v1alpha1.KeplerInternalSpec{
					Exporter: tc.spec,
				},
			}
			actual := k8s.VolumesFromDS(NewDaemonSet(components.Full, &k))
			assert.Equal(t, actual, tc.volumes)
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
			k := v1alpha1.KeplerInternal{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kepler-internal",
				},
			}
			actual := k8s.AllowsFromSCC(NewSCC(components.Full, &k))
			assert.Equal(t, actual, tc.sccAllows)
		})
	}
}

func TestRecordingRuleName(t *testing.T) {
	tt := []struct {
		keplerName string
		recRule    string
	}{
		{"kepler", "kepler:kepler"},
		{"kepler-internal", "kepler:kepler_internal"},
		{"kep-ler-inte.rnal", "kepler:kep_ler_inte_rnal"},
	}
	for _, tc := range tt {
		tc := tc
		t.Run(tc.keplerName, func(t *testing.T) {
			actual := keplerRulePrefix(tc.keplerName)
			assert.Equal(t, tc.recRule, actual)
		})
	}
}
