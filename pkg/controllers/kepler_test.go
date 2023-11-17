package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/pkg/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBpfAttachMethod(t *testing.T) {

	tt := []struct {
		annotations map[string]string
		scenario    string
		IsLibbpf    bool
	}{
		{
			annotations: map[string]string{},
			IsLibbpf:    false,
			scenario:    "no annotation",
		},
		{
			annotations: map[string]string{
				KeplerBpfAttachMethodAnnotation: "junk",
			},
			IsLibbpf: false,
			scenario: "annotation present but not libbpf",
		},
		{
			annotations: map[string]string{
				KeplerBpfAttachMethodAnnotation: "bcc",
			},
			IsLibbpf: false,
			scenario: "annotation present with bcc",
		},
		{
			annotations: map[string]string{
				KeplerBpfAttachMethodAnnotation: "libbpf",
			},
			IsLibbpf: true,
			scenario: "annotation present with libbpf",
		},
	}
	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			k := v1alpha1.Kepler{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: tc.annotations,
				},
				Spec: v1alpha1.KeplerSpec{
					Exporter: v1alpha1.ExporterSpec{},
				},
			}
			actual := IsLibbpfAttachType(&k)
			assert.Equal(t, actual, tc.IsLibbpf)
		})
	}
}
