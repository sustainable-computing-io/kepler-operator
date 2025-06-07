// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/internal/controller"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/test"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestPowerMonitorInternal_Reconciliation(t *testing.T) {
	f := test.NewFramework(t)
	name := "e2e-pmi"
	testNs := controller.PowerMonitorDeploymentNS

	// Pre-condition: Verify PowerMonitorInternal doesn't exist
	f.AssertNoResourceExists(name, "", &v1alpha1.PowerMonitorInternal{})

	// Create PowerMonitorInternal
	b := test.PowerMonitorInternalBuilder{}
	if runningOnVM {
		configMapName := "my-custom-config"
		f.CreatePowerMonitorInternal(name,
			b.WithNamespace(testNs),
			b.WithKeplerImage(testKeplerRebootImage),
			b.WithCluster(Cluster),
			b.WithAdditionalConfigMaps([]string{configMapName}),
		)
		cfm := f.NewAdditionalConfigMap(configMapName, testNs, `dev:
  fake-cpu-meter:
    enabled: true`)
		err := f.Patch(cfm)
		assert.NoError(t, err)
	} else {
		f.CreatePowerMonitorInternal(name,
			b.WithNamespace(testNs),
			b.WithKeplerImage(testKeplerRebootImage),
			b.WithCluster(Cluster),
		)
	}

	// Verify namespace exists
	f.AssertResourceExists(testNs, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}

	// Wait for PowerMonitorInternal to be reconciled
	pmi := f.WaitUntilPowerMonitorInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)

	// Verify daemonset exists and has correct configuration
	f.AssertResourceExists(pmi.Name, testNs, &ds)
	containers := ds.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(containers))
	assert.Equal(t, 1, len(containers[0].Ports))
	assert.EqualValues(t, 28282, containers[0].Ports[0].ContainerPort)

	// Verify PowerMonitorInternal status
	f.AssertPowerMonitorInternalStatus(pmi.Name, test.Timeout(5*time.Minute))
}
