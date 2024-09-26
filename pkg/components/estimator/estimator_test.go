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

package estimator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	corev1 "k8s.io/api/core/v1"
)

func sidecarEnabledSpec() *v1alpha1.EstimatorConfig {
	return &v1alpha1.EstimatorConfig{
		InitUrl:        "fake-url.zip",
		SidecarEnabled: true,
	}
}

func sidecarDisabledSpec() *v1alpha1.EstimatorConfig {
	return &v1alpha1.EstimatorConfig{
		InitUrl:        "fake-url.json",
		SidecarEnabled: false,
	}
}

func TestModelConfig(t *testing.T) {

	nodeTotalEnabled := "NODE_TOTAL_ESTIMATOR=true\nNODE_TOTAL_INIT_URL=fake-url.zip\n"
	nodeComponentsEnabled := "NODE_COMPONENTS_ESTIMATOR=true\nNODE_COMPONENTS_INIT_URL=fake-url.zip\n"

	containerTotalEnabled := "CONTAINER_TOTAL_ESTIMATOR=true\nCONTAINER_TOTAL_INIT_URL=fake-url.zip\n"
	containerComponentsEnabled := "CONTAINER_COMPONENTS_ESTIMATOR=true\nCONTAINER_COMPONENTS_INIT_URL=fake-url.zip\n"

	nodeTotalDisabled := "NODE_TOTAL_ESTIMATOR=false\nNODE_TOTAL_INIT_URL=fake-url.json\n"
	nodeComponentsDisabled := "NODE_COMPONENTS_ESTIMATOR=false\nNODE_COMPONENTS_INIT_URL=fake-url.json\n"

	containerTotalDisabled := "CONTAINER_TOTAL_ESTIMATOR=false\nCONTAINER_TOTAL_INIT_URL=fake-url.json\n"
	containerComponentsDisabled := "CONTAINER_COMPONENTS_ESTIMATOR=false\nCONTAINER_COMPONENTS_INIT_URL=fake-url.json\n"

	tt := []struct {
		spec      *v1alpha1.KeplerInternalSpec
		configStr string
		scenario  string
	}{
		{
			spec: &v1alpha1.KeplerInternalSpec{
				Estimator: &v1alpha1.InternalEstimatorSpec{
					Node: v1alpha1.EstimatorGroup{
						Total:      sidecarDisabledSpec(),
						Components: sidecarDisabledSpec(),
					},
					Container: v1alpha1.EstimatorGroup{
						Total:      sidecarDisabledSpec(),
						Components: sidecarDisabledSpec(),
					},
				},
			},
			configStr: nodeTotalDisabled + nodeComponentsDisabled + containerTotalDisabled + containerComponentsDisabled,
			scenario:  "all enable case",
		},
		{
			spec: &v1alpha1.KeplerInternalSpec{
				Estimator: &v1alpha1.InternalEstimatorSpec{
					Node: v1alpha1.EstimatorGroup{
						Total:      sidecarEnabledSpec(),
						Components: sidecarEnabledSpec(),
					},
					Container: v1alpha1.EstimatorGroup{
						Total:      sidecarEnabledSpec(),
						Components: sidecarEnabledSpec(),
					},
				},
			},
			configStr: nodeTotalEnabled + nodeComponentsEnabled + containerTotalEnabled + containerComponentsEnabled,
			scenario:  "all enable case",
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()
			k := v1alpha1.KeplerInternal{
				Spec: *tc.spec,
			}
			actual := ModelConfig(k.Spec.Estimator)
			assert.Equal(t, actual, tc.configStr)
		})
	}
}

func TestModifiedContainer(t *testing.T) {
	exporterContainer := &corev1.Container{
		Command: []string{"kepler", "-v=1"},
	}
	expectedCommand := []string{"/usr/bin/bash", "-c"}
	expectedArgs := []string{"until [ -e /tmp/estimator.sock ]; do sleep 1; done && kepler -v=1"}
	expectedVolumeMounts := []string{"cfm", "mnt", "tmp"}
	keplerVolumes := []corev1.Volume{k8s.VolumeFromEmptyDir("kepler-volume")}
	expectedVolumes := []string{"kepler-volume", "mnt", "tmp"}
	t.Run("modified container", func(t *testing.T) {
		t.Parallel()
		k := v1alpha1.KeplerInternal{
			Spec: v1alpha1.KeplerInternalSpec{
				Estimator: &v1alpha1.InternalEstimatorSpec{
					Image: StableImage,
					Node: v1alpha1.EstimatorGroup{
						Total:      sidecarEnabledSpec(),
						Components: sidecarDisabledSpec(),
					},
				},
			},
		}
		need := NeedsEstimatorSidecar(k.Spec.Estimator)
		assert.Equal(t, need, true)
		container := Container(k.Spec.Estimator.Image)
		assert.Equal(t, len(container.VolumeMounts), len(expectedVolumeMounts))
		for index, mnt := range container.VolumeMounts {
			assert.Equal(t, mnt.Name, expectedVolumeMounts[index])
		}
		volumes := append(keplerVolumes, Volumes()...)
		assert.Equal(t, len(volumes), len(expectedVolumes))
		for index, volume := range volumes {
			assert.Equal(t, volume.Name, expectedVolumes[index])
		}
		exporterContainer := AddEstimatorDependency(exporterContainer)
		actualCommand := exporterContainer.Command
		actualArgs := exporterContainer.Args
		assert.Equal(t, len(actualCommand), len(expectedCommand))
		assert.Equal(t, len(actualArgs), len(expectedArgs))
		for index, actual := range actualCommand {
			assert.Equal(t, actual, expectedCommand[index])
		}
		for index, actual := range actualArgs {
			assert.Equal(t, actual, expectedArgs[index])
		}
		exporterVolumeMounts := exporterContainer.VolumeMounts
		assert.Equal(t, len(exporterVolumeMounts), 1)
		assert.Equal(t, exporterVolumeMounts[0].Name, "tmp")
	})
}
