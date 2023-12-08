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
package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/pkg/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/pkg/controllers"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/test"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestKeplerInternal_Reconciliation(t *testing.T) {
	f := test.NewFramework(t)
	name := "e2e-kepler-internal"
	// test namespace must be the deployment namespace for controller
	// to watch the deployments / daemonsets etc
	testNs := controllers.KeplerDeploymentNS

	// pre-condition
	f.AssertNoResourceExists(name, "", &v1alpha1.KeplerInternal{}, test.NoWait())

	// when
	b := test.InternalBuilder{}
	ki := f.CreateInternal(name,
		b.WithNamespace(testNs),
		b.WithExporterLibBpfImage(),
		b.WithExporterPort(9108),
	)

	// then the following resources will be created
	f.AssertResourceExists(testNs, "", &corev1.Namespace{})

	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(ki.Name, testNs, &ds)
	containers := ds.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(containers))
	assert.Equal(t, 1, len(containers[0].Ports))
	assert.EqualValues(t, 9108, containers[0].Ports[0].ContainerPort)

	// the status will be updated
	ki = f.WaitUntilInternalCondition(ki.Name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)
	assert.Equal(t, []corev1.Toleration{{Operator: "Exists"}}, ki.Spec.Exporter.Deployment.Tolerations)

	reconciled, err := k8s.FindCondition(ki.Status.Exporter.Conditions, v1alpha1.Reconciled)
	assert.NoError(t, err, "unable to get reconciled condition")
	assert.Equal(t, reconciled.ObservedGeneration, ki.Generation)
	assert.Equal(t, reconciled.Status, v1alpha1.ConditionTrue)
	//
	ki = f.WaitUntilInternalCondition(ki.Name, v1alpha1.Available, v1alpha1.ConditionTrue)
	available, err := k8s.FindCondition(ki.Status.Exporter.Conditions, v1alpha1.Available)
	assert.NoError(t, err, "unable to get available condition")
	assert.Equal(t, available.ObservedGeneration, ki.Generation)
	assert.Equal(t, available.Status, v1alpha1.ConditionTrue)
}
