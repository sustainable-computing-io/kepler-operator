package exporter

import (
	"testing"

	secv1 "github.com/openshift/api/security/v1"
	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/pkg/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type SCCAllows struct {
	AllowPrivilegedContainer bool
	AllowHostDirVolumePlugin bool
	AllowHostIPC             bool
	AllowHostNetwork         bool
	AllowHostPID             bool
	AllowHostPorts           bool
}

func NodeSelectorFromDS(ds *appsv1.DaemonSet) map[string]string {
	return ds.Spec.Template.Spec.NodeSelector
}

func TolerationsFromDS(ds *appsv1.DaemonSet) []corev1.Toleration {
	return ds.Spec.Template.Spec.Tolerations
}

func HostPIDFromDS(ds *appsv1.DaemonSet) bool {
	return ds.Spec.Template.Spec.HostPID
}

func VolumeMountsFromDS(ds *appsv1.DaemonSet) []corev1.VolumeMount {
	return ds.Spec.Template.Spec.Containers[0].VolumeMounts
}

func AllowsFromSCC(SCC *secv1.SecurityContextConstraints) SCCAllows {
	return SCCAllows{
		AllowPrivilegedContainer: SCC.AllowPrivilegedContainer,
		AllowHostDirVolumePlugin: SCC.AllowHostDirVolumePlugin,
		AllowHostIPC:             SCC.AllowHostIPC,
		AllowHostNetwork:         SCC.AllowHostNetwork,
		AllowHostPID:             SCC.AllowHostPID,
		AllowHostPorts:           SCC.AllowHostPorts,
	}
}

func TestNodeSelection(t *testing.T) {

	tt := []struct {
		spec     v1alpha1.ExporterSpec
		selector map[string]string
		scenario string
	}{
		{
			spec:     v1alpha1.ExporterSpec{},
			selector: map[string]string{"kubernetes.io/os": "linux"},
			scenario: "default case",
		},
		{
			spec:     v1alpha1.ExporterSpec{NodeSelector: map[string]string{"k1": "v1"}},
			selector: map[string]string{"k1": "v1", "kubernetes.io/os": "linux"},
			scenario: "user defined node selector",
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			k := v1alpha1.Kepler{
				Spec: v1alpha1.KeplerSpec{
					Exporter: tc.spec,
				},
			}
			actual := NodeSelectorFromDS(NewDaemonSet(&k))
			assert.Equal(t, actual, tc.selector)
		})
	}
}

func TestTolerations(t *testing.T) {

	tt := []struct {
		spec        v1alpha1.ExporterSpec
		tolerations []corev1.Toleration
		scenario    string
	}{
		{
			spec: v1alpha1.ExporterSpec{},
			tolerations: []corev1.Toleration{{
				Operator: corev1.TolerationOpExists}},
			scenario: "default case",
		},
		{
			spec: v1alpha1.ExporterSpec{Tolerations: []corev1.Toleration{{
				Effect: corev1.TaintEffectNoSchedule, Key: "key1"}}},
			tolerations: []corev1.Toleration{{
				Effect: corev1.TaintEffectNoSchedule, Key: "key1",
			}},
			scenario: "user defined toleration",
		},
	}
	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			k := v1alpha1.Kepler{
				Spec: v1alpha1.KeplerSpec{
					Exporter: tc.spec,
				},
			}
			actual := TolerationsFromDS(NewDaemonSet(&k))
			assert.Equal(t, actual, tc.tolerations)
		})
	}
}

func TestHostPID(t *testing.T) {
	tt := []struct {
		spec     v1alpha1.ExporterSpec
		hostPID  bool
		scenario string
	}{
		{
			spec:     v1alpha1.ExporterSpec{},
			hostPID:  true,
			scenario: "default case",
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			k := v1alpha1.Kepler{
				Spec: v1alpha1.KeplerSpec{
					Exporter: tc.spec,
				},
			}
			actual := HostPIDFromDS(NewDaemonSet(&k))
			assert.Equal(t, actual, tc.hostPID)
		})
	}
}
func TestVolumeMounts(t *testing.T) {
	tt := []struct {
		spec         v1alpha1.ExporterSpec
		volumeMounts []corev1.VolumeMount
		scenario     string
	}{
		{
			spec: v1alpha1.ExporterSpec{},
			volumeMounts: []corev1.VolumeMount{
				{Name: "lib-modules", MountPath: "/lib/modules", ReadOnly: true},
				{Name: "tracing", MountPath: "/sys", ReadOnly: true},
				{Name: "kernel-src", MountPath: "/usr/src/kernels", ReadOnly: true},
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
			k := v1alpha1.Kepler{
				Spec: v1alpha1.KeplerSpec{
					Exporter: tc.spec,
				},
			}
			actual := VolumeMountsFromDS(NewDaemonSet(&k))
			assert.Equal(t, actual, tc.volumeMounts)
		})
	}
}

func TestSCCAllowHostIPCIsFalse(t *testing.T) {
	tt := []struct {
		spec      v1alpha1.ExporterSpec
		sccAllows SCCAllows
		scenario  string
	}{
		{
			spec: v1alpha1.ExporterSpec{},
			sccAllows: SCCAllows{
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
			k := v1alpha1.Kepler{
				Spec: v1alpha1.KeplerSpec{
					Exporter: tc.spec,
				},
			}
			actual := AllowsFromSCC(NewSCC(components.Full, &k))
			assert.Equal(t, actual, tc.sccAllows)
		})
	}
}
