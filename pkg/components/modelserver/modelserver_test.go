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

package modelserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	corev1 "k8s.io/api/core/v1"
)

func TestConfigMap(t *testing.T) {

	tt := []struct {
		spec     *v1alpha1.InternalModelServerSpec
		data     map[string]string
		scenario string
	}{
		{
			spec: &v1alpha1.InternalModelServerSpec{},
			data: map[string]string{
				"MODEL_PATH":   "/mnt/models",
				"RESOURCE_DIR": "/data/resource",
			},
			scenario: "default case",
		},
		{
			spec: &v1alpha1.InternalModelServerSpec{
				URL:         "fake-url",
				Path:        "fake-model-path",
				RequestPath: "fake-request-path",
				ListPath:    "fake-model-list-path",
				PipelineURL: "fake-pipeline",
				ErrorKey:    "fake-error-key",
			},
			data: map[string]string{
				"MODEL_PATH":                   "fake-model-path",
				"MODEL_SERVER_REQ_PATH":        "fake-request-path",
				"MODEL_SERVER_MODEL_LIST_PATH": "fake-model-list-path",
				"INITIAL_PIPELINE_URL":         "fake-pipeline",
				"ERROR_KEY":                    "fake-error-key",
				"RESOURCE_DIR":                 "/data/resource",
			},
			scenario: "user defined server-api config",
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			k := v1alpha1.KeplerInternal{
				Spec: v1alpha1.KeplerInternalSpec{
					ModelServer: tc.spec,
				},
			}
			actual := NewConfigMap(k.ModelServerDeploymentName(), components.Full, k.Spec.ModelServer, k.Spec.Exporter.Deployment.Namespace)
			assert.Equal(t, len(tc.data), len(actual.Data))
			for k, v := range tc.data {
				assert.Equal(t, v, actual.Data[k])
			}
		})
	}

}

func TestService(t *testing.T) {

	tt := []struct {
		spec        v1alpha1.InternalModelServerSpec
		servicePort int32
		scenario    string
	}{
		{
			spec: v1alpha1.InternalModelServerSpec{
				Port: 8080,
			},
			servicePort: 8080,
			scenario:    "user defined port",
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			k := v1alpha1.KeplerInternal{
				Spec: v1alpha1.KeplerInternalSpec{
					ModelServer: &tc.spec,
				},
			}
			actual := NewService(k.ModelServerDeploymentName(), k.Spec.ModelServer, k.Spec.Exporter.Deployment.Namespace)
			assert.Equal(t, actual.Spec.Ports[0].Port, tc.servicePort)
		})
	}

}

func TestServerAPIContainer(t *testing.T) {

	tt := []struct {
		spec        v1alpha1.InternalModelServerSpec
		servicePort int32
		scenario    string
	}{
		{
			spec: v1alpha1.InternalModelServerSpec{
				Port: 8080,
			},
			servicePort: 8080,
			scenario:    "user defined port",
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			k := v1alpha1.KeplerInternal{
				Spec: v1alpha1.KeplerInternalSpec{
					ModelServer: &tc.spec,
				},
			}
			actual := NewDeployment(k.ModelServerDeploymentName(), k.Spec.ModelServer, k.Spec.Exporter.Deployment.Namespace)
			containers := actual.Spec.Template.Spec.Containers
			assert.Equal(t, len(containers), 1)
			assert.Equal(t, containers[0].Ports[0].ContainerPort, tc.servicePort)
		})
	}

}

func TestServerContainer(t *testing.T) {
	exporterContainer := &corev1.Container{
		Command: []string{"kepler", "-v=1"},
	}
	ms := v1alpha1.InternalModelServerSpec{
		Port: 8080,
	}
	deployName := "kepler-internal"
	deployNamespace := "default"
	expectedCommand := []string{"/usr/bin/bash", "-c"}
	expectedArgs := []string{"until [[ \"$(curl -s -o /dev/null -w %{http_code} http://kepler-internal-svc.default.svc.cluster.local:8080/best-models)\" -eq 200 ]]; do sleep 1; done && kepler -v=1"}

	t.Run("server container", func(t *testing.T) {
		t.Parallel()
		_ = v1alpha1.KeplerInternal{
			Spec: v1alpha1.KeplerInternalSpec{
				ModelServer: &ms,
			},
		}
		exporterContainer := AddModelServerDependency(exporterContainer, deployName, deployNamespace, &ms)
		actualCommand := exporterContainer.Command
		actualArgs := exporterContainer.Args
		assert.Equal(t, expectedCommand, actualCommand)
		assert.Equal(t, expectedArgs, actualArgs)
	})
}
