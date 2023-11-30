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
				BpfAttachMethodAnnotation: "junk",
			},
			IsLibbpf: false,
			scenario: "annotation present but not libbpf",
		},
		{
			annotations: map[string]string{
				BpfAttachMethodAnnotation: "bcc",
			},
			IsLibbpf: false,
			scenario: "annotation present with bcc",
		},
		{
			annotations: map[string]string{
				BpfAttachMethodAnnotation: "libbpf",
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
			actual := hasLibBPFAnnotation(&k)
			assert.Equal(t, actual, tc.IsLibbpf)
		})
	}
}
