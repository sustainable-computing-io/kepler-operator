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

package test

import (
	"github.com/sustainable.computing.io/kepler-operator/pkg/api/v1alpha1"
)

const (
	libBPFImage = "quay.io/sustainable_computing_io/kepler:release-0.6.1-libbpf"
	bccImage    = "quay.io/sustainable_computing_io/kepler:release-0.6.1"
)

type InternalBuilder struct {
}

func (b InternalBuilder) WithNamespace(ns string) func(k *v1alpha1.KeplerInternal) {
	return func(k *v1alpha1.KeplerInternal) {
		k.Spec.Exporter.Deployment.Namespace = ns
	}
}

func (b InternalBuilder) WithExporterLibBpfImage() func(k *v1alpha1.KeplerInternal) {
	return b.WithExporterImage(libBPFImage)
}

func (b InternalBuilder) WithExporterImage(img string) func(k *v1alpha1.KeplerInternal) {
	return func(k *v1alpha1.KeplerInternal) {
		k.Spec.Exporter.Deployment.Image = img
	}
}

func (b InternalBuilder) WithExporterPort(p int) func(k *v1alpha1.KeplerInternal) {
	return func(k *v1alpha1.KeplerInternal) {
		k.Spec.Exporter.Deployment.Port = int32(p)
	}
}

func (b InternalBuilder) WithEstimator() func(k *v1alpha1.KeplerInternal) {
	return func(k *v1alpha1.KeplerInternal) {
		k.Spec.Estimator = &v1alpha1.InternalEstimatorSpec{
			Node: v1alpha1.EstimatorGroup{
				Total: &v1alpha1.EstimatorConfig{
					SidecarEnabled: true,
				},
				Components: &v1alpha1.EstimatorConfig{
					SidecarEnabled: true,
				},
			},
		}
	}
}

func (b InternalBuilder) WithModelServer() func(k *v1alpha1.KeplerInternal) {
	return func(k *v1alpha1.KeplerInternal) {
		k.Spec.ModelServer = &v1alpha1.InternalModelServerSpec{
			Enabled: true,
		}
	}
}
