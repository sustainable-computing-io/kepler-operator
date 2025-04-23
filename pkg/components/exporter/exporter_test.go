// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
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

func TestDaemonSet(t *testing.T) {
	tt := []struct {
		spec            v1alpha1.InternalExporterSpec
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
			spec: v1alpha1.InternalExporterSpec{
				Deployment: v1alpha1.InternalExporterDeploymentSpec{
					ExporterDeploymentSpec: v1alpha1.ExporterDeploymentSpec{
						Port: 9103,
					},
				},
			},
			hostPID: true,
			exporterCommand: []string{
				"/usr/bin/kepler",
				"-address",
				"0.0.0.0:9103",
				"-enable-cgroup-id=$(ENABLE_EBPF_CGROUPID)",
				"-enable-gpu=$(ENABLE_GPU)",
				"-v=$(KEPLER_LOG_LEVEL)",
			},
			volumeMounts: []corev1.VolumeMount{
				{Name: "lib-modules", MountPath: "/lib/modules", ReadOnly: true},
				{Name: "tracing", MountPath: "/sys", ReadOnly: true},
				{Name: "proc", MountPath: "/proc"},
				{Name: "cfm", MountPath: "/etc/kepler/kepler.config"},
			},
			volumes: []corev1.Volume{
				k8s.VolumeFromHost("lib-modules", "/lib/modules"),
				k8s.VolumeFromHost("tracing", "/sys"),
				k8s.VolumeFromHost("proc", "/proc"),
				k8s.VolumeFromConfigMap("cfm", "kepler-internal"),
			},
			scenario: "default case",
		},
		{
			spec: v1alpha1.InternalExporterSpec{
				Deployment: v1alpha1.InternalExporterDeploymentSpec{
					ExporterDeploymentSpec: v1alpha1.ExporterDeploymentSpec{
						Port: 9103,
					},
				},
			},
			hostPID: true,
			exporterCommand: []string{
				"/usr/bin/kepler",
				"-address",
				"0.0.0.0:9103",
				"-enable-cgroup-id=$(ENABLE_EBPF_CGROUPID)",
				"-enable-gpu=$(ENABLE_GPU)",
				"-v=$(KEPLER_LOG_LEVEL)",
				"-redfish-cred-file-path=/etc/redfish/redfish.csv",
			},
			volumeMounts: []corev1.VolumeMount{
				{Name: "lib-modules", MountPath: "/lib/modules", ReadOnly: true},
				{Name: "tracing", MountPath: "/sys", ReadOnly: true},
				{Name: "proc", MountPath: "/proc"},
				{Name: "cfm", MountPath: "/etc/kepler/kepler.config"},
				{Name: "redfish-cred", MountPath: "/etc/redfish", ReadOnly: true},
			},
			volumes: []corev1.Volume{
				k8s.VolumeFromHost("lib-modules", "/lib/modules"),
				k8s.VolumeFromHost("tracing", "/sys"),
				k8s.VolumeFromHost("proc", "/proc"),
				k8s.VolumeFromConfigMap("cfm", "kepler-internal"),
				k8s.VolumeFromSecret("redfish-cred", "my-redfish-secret"),
			},
			addRedfish: true,
			redfishSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-redfish-secret",
					ResourceVersion: "123",
				},
			},
			annotation: map[string]string{
				"kepler.system.sustainable.computing.io/redfish-secret-ref":  "123",
				"kepler.system.sustainable.computing.io/redfish-config-hash": "1337",
			},
			scenario: "redfish case",
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
			ds := NewDaemonSet(components.Full, &k)
			if tc.addRedfish {
				MountRedfishSecretToDaemonSet(ds, tc.redfishSecret, 1337)
			}

			actualHostPID := k8s.HostPIDFromDS(ds)
			assert.Equal(t, tc.hostPID, actualHostPID)

			actualExporterCommand := k8s.CommandFromDS(ds, KeplerContainerIndex)
			assert.Equal(t, tc.exporterCommand, actualExporterCommand)

			actualVolumeMounts := k8s.VolumeMountsFromDS(ds, KeplerContainerIndex)
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
