// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/internal/controller"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/test"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestPowerMonitor_Deletion(t *testing.T) {
	f := test.NewFramework(t)

	// pre-condition: ensure powermonitor exists
	f.CreatePowerMonitor("power-monitor", f.WithPowerMonitorAnnotation(vmAnnotationKey, enableVMEnv))
	pm := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)

	//
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(
		pm.Name,
		controller.PowerMonitorDeploymentNS,
		&ds,
		test.Timeout(10*time.Second),
	)

	f.DeletePowerMonitor("power-monitor")

	ns := components.NewNamespace(controller.PowerMonitorDeploymentNS)
	f.AssertNoResourceExists(ns.Name, "", ns)
	f.AssertNoResourceExists(ds.Name, ds.Namespace, &ds)
}

func TestPowerMonitor_Reconciliation(t *testing.T) {
	f := test.NewFramework(t)

	// pre-condition
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{})

	// when
	pm := f.CreatePowerMonitor("power-monitor", f.WithPowerMonitorAnnotation(vmAnnotationKey, enableVMEnv))

	// then
	f.AssertResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &ds)

	powermonitor := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Reconciled, v1alpha1.ConditionTrue)
	// default toleration
	assert.Equal(t, []corev1.Toleration{{Operator: "Exists"}}, powermonitor.Spec.Kepler.Deployment.Tolerations)
	reconciled, err := k8s.FindCondition(powermonitor.Status.Kepler.Conditions, v1alpha1.Reconciled)
	assert.NoError(t, err, "unable to get reconciled condition")
	assert.Equal(t, reconciled.ObservedGeneration, powermonitor.Generation)
	assert.Equal(t, reconciled.Status, v1alpha1.ConditionTrue)

	powermonitor = f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
	available, err := k8s.FindCondition(powermonitor.Status.Kepler.Conditions, v1alpha1.Available)
	assert.NoError(t, err, "unable to get available condition")
	assert.Equal(t, available.ObservedGeneration, powermonitor.Generation)
	assert.Equal(t, available.Status, v1alpha1.ConditionTrue)
}

func TestBadPowerMonitor_Reconciliation(t *testing.T) {
	f := test.NewFramework(t)
	// Ensure PowerMonitor is not deployed (by any chance)
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{}, test.Timeout(10*time.Second))
	f.AssertNoResourceExists("invalid-name", "", &v1alpha1.PowerMonitor{})
	powermonitor := f.NewPowerMonitor("invalid-name")
	err := f.Patch(&powermonitor)
	assert.ErrorContains(t, err, "denied the request")
}

func TestPowerMonitorNodeSelector(t *testing.T) {
	f := test.NewFramework(t)
	// Ensure PowerMonitor is not deployed (by any chance)
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{}, test.Timeout(10*time.Second))

	nodes := f.GetSchedulableNodes()
	assert.NotZero(t, len(nodes), "got zero nodes")

	node := nodes[0]
	var labels k8s.StringMap = map[string]string{"e2e-test": "true"}
	err := f.AddResourceLabels("node", node.Name, labels)
	assert.NoError(t, err, "could not label node")

	pm := f.CreatePowerMonitor("power-monitor",
		f.WithPowerMonitorNodeSelector(labels),
		f.WithPowerMonitorAnnotation(vmAnnotationKey, enableVMEnv))

	f.AssertResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &ds)
	powermonitor := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
	assert.EqualValues(t, 1, powermonitor.Status.Kepler.NumberAvailable)
	f.DeletePowerMonitor("power-monitor")
	f.AssertNoResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	f.AssertNoResourceExists(ds.Name, ds.Namespace, &ds)
}

func TestPowerMonitorNodeSelectorUnavailableLabel(t *testing.T) {
	f := test.NewFramework(t)
	// Ensure PowerMonitor is not deployed (by any chance)
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{}, test.Timeout(10*time.Second))

	nodes := f.GetSchedulableNodes()
	assert.NotZero(t, len(nodes), "got zero nodes")

	var unavailableLabels k8s.StringMap = map[string]string{"e2e-test": "true"}

	pm := f.CreatePowerMonitor("power-monitor",
		f.WithPowerMonitorNodeSelector(unavailableLabels),
		f.WithPowerMonitorAnnotation(vmAnnotationKey, enableVMEnv))

	f.AssertResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &ds)

	powermonitor := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionFalse)
	assert.EqualValues(t, 0, powermonitor.Status.Kepler.NumberAvailable)

	f.DeletePowerMonitor("power-monitor")

	f.AssertNoResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	f.AssertNoResourceExists(ds.Name, ds.Namespace, &ds)
}

func TestPowerMonitorTaint_WithToleration(t *testing.T) {
	f := test.NewFramework(t)
	// Ensure PowerMonitor is not deployed (by any chance)
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{}, test.Timeout(10*time.Second))

	var err error
	// choose one node
	nodes := f.GetSchedulableNodes()
	node := nodes[0]

	e2eTestTaint := corev1.Taint{
		Key:    "key1",
		Value:  "value1",
		Effect: corev1.TaintEffectNoSchedule,
	}

	err = f.TaintNode(node.Name, e2eTestTaint.ToString())
	assert.NoError(t, err, "failed to taint node %s", node)

	pm := f.CreatePowerMonitor("power-monitor",
		f.WithPowerMonitorTolerations(append(node.Spec.Taints, e2eTestTaint)),
		f.WithPowerMonitorAnnotation(vmAnnotationKey, enableVMEnv))
	f.AssertResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &ds)

	powermonitor := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
	assert.EqualValues(t, len(nodes), powermonitor.Status.Kepler.NumberAvailable)

	f.DeletePowerMonitor("power-monitor")

	f.AssertNoResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	f.AssertNoResourceExists(ds.Name, ds.Namespace, &ds)
}

func TestBadPowerMonitorTaint_WithToleration(t *testing.T) {
	f := test.NewFramework(t)
	// Ensure PowerMonitor is not deployed (by any chance)
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{}, test.Timeout(10*time.Second))

	// choose one node
	nodes := f.GetSchedulableNodes()
	node := nodes[0]
	e2eTestTaint := corev1.Taint{
		Key:    "key1",
		Value:  "value1",
		Effect: corev1.TaintEffectNoSchedule,
	}
	badTestTaint := corev1.Taint{
		Key:    "key2",
		Value:  "value2",
		Effect: corev1.TaintEffectNoSchedule,
	}

	err := f.TaintNode(node.Name, e2eTestTaint.ToString())
	assert.NoError(t, err, "failed to taint node %s", node)

	pm := f.CreatePowerMonitor("power-monitor",
		f.WithPowerMonitorTolerations(append(node.Spec.Taints, badTestTaint)),
		f.WithPowerMonitorAnnotation(vmAnnotationKey, enableVMEnv))

	f.AssertResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &ds)

	powermonitor := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
	assert.EqualValues(t, len(nodes)-1, powermonitor.Status.Kepler.NumberAvailable)

	f.DeletePowerMonitor("power-monitor")

	f.AssertNoResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	f.AssertNoResourceExists(ds.Name, ds.Namespace, &ds)
}
