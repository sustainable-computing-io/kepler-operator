package exporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/pkg/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func NodeSelectorFromDS(ds *appsv1.DaemonSet) map[string]string {
	return ds.Spec.Template.Spec.NodeSelector
}

func TolerationsFromDS(ds *appsv1.DaemonSet) []corev1.Toleration {
	return ds.Spec.Template.Spec.Tolerations
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
