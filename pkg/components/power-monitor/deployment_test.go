package powermonitor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPowerMonitorDaemonSet(t *testing.T) {
	tt := []struct {
		spec            v1alpha1.KeplerXDeploymentSpec
		hostPID         bool
		exporterCommand []string
		volumeMounts    []corev1.VolumeMount
		volumes         []corev1.Volume
		scenario        string
		addRedfish      bool
		redfishSecret   *corev1.Secret
		annotation      map[string]string
	}{
		{
			spec: v1alpha1.KeplerXDeploymentSpec{
				Port: 9103,
			},
			hostPID: true,
			exporterCommand: []string{
				"/usr/bin/kepler",
				"--config.file=/etc/kepler/kepler-config.yaml",
			},
			volumeMounts: []corev1.VolumeMount{
				{Name: "lib-modules", MountPath: "/lib/modules", ReadOnly: true},
				{Name: "tracing", MountPath: "/sys", ReadOnly: true},
				{Name: "proc", MountPath: "/proc"},
				{Name: "cfm", MountPath: "/etc/kepler"},
			},
			volumes: []corev1.Volume{
				k8s.VolumeFromHost("lib-modules", "/lib/modules"),
				k8s.VolumeFromHost("tracing", "/sys"),
				k8s.VolumeFromHost("proc", "/proc"),
				k8s.VolumeFromConfigMap("cfm", "kepler-x"),
			},
			scenario: "default case",
		},
	}
	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			kx := v1alpha1.KeplerX{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kepler-x",
				},
				Spec: v1alpha1.KeplerXSpec{
					Deployment: tc.spec,
				},
			}
			ds := NewPowerMonitorDaemonSet(components.Full, &kx)

			actualHostPID := k8s.HostPIDFromDS(ds)
			assert.Equal(t, tc.hostPID, actualHostPID)

			actualExporterCommand := k8s.CommandFromDS(ds, 0)
			assert.Equal(t, tc.exporterCommand, actualExporterCommand)

			actualVolumeMounts := k8s.VolumeMountsFromDS(ds, 0)
			assert.Equal(t, tc.volumeMounts, actualVolumeMounts)

			actualVolumes := k8s.VolumesFromDS(ds)
			assert.Equal(t, tc.volumes, actualVolumes)

			actualAnnotation := k8s.AnnotationFromDS(ds)
			assert.Equal(t, tc.annotation, actualAnnotation)
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
			kx := v1alpha1.KeplerX{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kepler-x",
				},
			}
			actual := k8s.AllowsFromSCC(NewPowerMonitorSCC(components.Full, &kx))
			assert.Equal(t, actual, tc.sccAllows)
		})
	}
}
