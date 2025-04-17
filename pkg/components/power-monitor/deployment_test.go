package powermonitor

import (
	"fmt"
	"path/filepath"
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
		spec            v1alpha1.PowerMonitorInternalKeplerDeploymentSpec
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
			spec:    v1alpha1.PowerMonitorInternalKeplerDeploymentSpec{},
			hostPID: true,
			exporterCommand: []string{
				"/usr/bin/kepler",
				fmt.Sprintf("--config.file=%s", filepath.Join(KeplerConfigMapPath, KeplerConfigFile)),
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
				Spec: v1alpha1.PowerMonitorInternalSpec{
					Kepler: v1alpha1.PowerMonitorInternalKeplerSpec{
						Deployment: tc.spec,
					},
				},
			}
			ds := NewPowerMonitorDaemonSet(components.Full, &pmi)

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
