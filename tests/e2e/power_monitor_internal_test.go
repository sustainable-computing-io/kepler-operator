// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"strconv"
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
	// test namespace must be the deployment namespace for controller
	// to watch the deployments / daemonsets etc
	testNs := controller.PowerMonitorDeploymentNS

	// pre-condition
	f.AssertNoResourceExists(name, "", &v1alpha1.PowerMonitorInternal{})

	// when
	b := test.PowerMonitorInternalBuilder{}
	pmi := f.CreatePowerMonitorInternal(name,
		b.WithNamespace(testNs),
		b.WithKeplerImage(testKeplerRebootImage),
		b.WithCluster(Cluster),
		b.WithAnnotation(vmAnnotationKey, strconv.FormatBool(enableVMTest)),
	)

	// then the following resources will be created
	f.AssertResourceExists(testNs, "", &corev1.Namespace{})

	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pmi.Name, testNs, &ds)
	containers := ds.Spec.Template.Spec.Containers
	assert.Equal(t, 1, len(containers))
	assert.Equal(t, 1, len(containers[0].Ports))
	assert.EqualValues(t, 28282, containers[0].Ports[0].ContainerPort)
	// test expected status (PowerMonitor)
	f.AssertPowerMonitorInternalStatus(pmi.Name, test.Timeout(5*time.Minute))
}
