// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/internal/controller"
	"github.com/sustainable.computing.io/kepler-operator/pkg/components"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/test"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPowerMonitor_Deletion(t *testing.T) {
	f := test.NewFramework(t)

	// pre-condition: ensure powermonitor exists
	f.CreatePowerMonitor("power-monitor", f.WithPowerMonitorAnnotation(vmAnnotationKey, strconv.FormatBool(enableVMTest)))
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
	pm := f.CreatePowerMonitor("power-monitor", f.WithPowerMonitorAnnotation(vmAnnotationKey, strconv.FormatBool(enableVMTest)))

	// then
	f.AssertResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &ds)

	powermonitor := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Reconciled, v1alpha1.ConditionTrue)
	// default toleration
	assert.Equal(t, []corev1.Toleration{{Operator: "Exists"}}, powermonitor.Spec.Kepler.Deployment.Tolerations)
	reconciled, err := k8s.FindCondition(powermonitor.Status.Conditions, v1alpha1.Reconciled)
	assert.NoError(t, err, "unable to get reconciled condition")
	assert.Equal(t, reconciled.ObservedGeneration, powermonitor.Generation)
	assert.Equal(t, reconciled.Status, v1alpha1.ConditionTrue)

	powermonitor = f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
	available, err := k8s.FindCondition(powermonitor.Status.Conditions, v1alpha1.Available)
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
		f.WithPowerMonitorAnnotation(vmAnnotationKey, strconv.FormatBool(enableVMTest)))

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
		f.WithPowerMonitorAnnotation(vmAnnotationKey, strconv.FormatBool(enableVMTest)))

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
		f.WithPowerMonitorAnnotation(vmAnnotationKey, strconv.FormatBool(enableVMTest)))
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
		f.WithPowerMonitorAnnotation(vmAnnotationKey, strconv.FormatBool(enableVMTest)))

	f.AssertResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &ds)

	powermonitor := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
	assert.EqualValues(t, len(nodes)-1, powermonitor.Status.Kepler.NumberAvailable)

	f.DeletePowerMonitor("power-monitor")

	f.AssertNoResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	f.AssertNoResourceExists(ds.Name, ds.Namespace, &ds)
}

func TestPowerMonitor_ReconciliationWithAdditionalConfigMap(t *testing.T) {
	f := test.NewFramework(t)
	configMapName := "my-custom-config"

	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{})

	pm := f.CreatePowerMonitor("power-monitor",
		f.WithPowerMonitorAdditionalConfigMaps([]string{configMapName}),
		f.WithPowerMonitorAnnotation(vmAnnotationKey, strconv.FormatBool(enableVMTest)))

	f.AssertResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})

	ds := appsv1.DaemonSet{}
	f.AssertNoResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &ds)

	// NOTE: condition should be false since the configmap is not created yet
	pm = f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Reconciled, v1alpha1.ConditionFalse)
	reconciled, _ := k8s.FindCondition(pm.Status.Conditions, v1alpha1.Reconciled)
	assert.Contains(t, reconciled.Message, fmt.Sprintf("configMap %s not found in %s namespace", configMapName, controller.PowerMonitorDeploymentNS))

	// create custom configmap with additional config
	customConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: controller.PowerMonitorDeploymentNS,
		},
		Data: map[string]string{
			"config.yaml": `log:
  format: json
  level: debug`,
		},
	}
	err := f.Client().Create(context.TODO(), &customConfigMap)
	assert.NoError(t, err)

	// wait for DaemonSet to be created
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &ds)

	// expect reconcile to be true after configmap is created
	pm = f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Reconciled, v1alpha1.ConditionTrue)

	// verify that the main configmap contains merged configuration
	mainConfigMap := corev1.ConfigMap{}
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &mainConfigMap)

	// check that the merged config contains both default and custom settings
	configData := mainConfigMap.Data["config.yaml"]
	assert.Contains(t, configData, "format: json", "custom log format should be merged")
	assert.Contains(t, configData, "sysfs: /host/sys", "default sysfs path should be present")
	assert.Contains(t, configData, "procfs: /host/proc", "default procfs path should be present")

	// verify that DaemonSet has the config map hash annotation for rollout trigger
	assert.Contains(t, ds.Spec.Template.Annotations, "powermonitor.sustainable.computing.io/config-map-hash-"+pm.Name)

	og := ds.Status.ObservedGeneration
	assert.Equal(t, og, int64(1))

	// update custom configmap to trigger rollout
	customConfigMap.Data["config.yaml"] = `log:
  format: text
  level: warn`
	err = f.Client().Update(context.TODO(), &customConfigMap)
	assert.NoError(t, err)

	// wait for DaemonSet to restart
	ds = appsv1.DaemonSet{}
	f.WaitUntil("Daemonset to restart", func(ctx context.Context) (bool, error) {
		err := f.Client().Get(ctx,
			client.ObjectKey{Namespace: controller.PowerMonitorDeploymentNS, Name: pm.Name}, &ds)
		if errors.IsNotFound(err) {
			return false, nil
		} else if err != nil {
			return false, err
		}
		return ds.Status.ObservedGeneration == og+1, nil
	})

	// verify updated config is merged
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &mainConfigMap)
	updatedConfigData := mainConfigMap.Data["config.yaml"]
	assert.Contains(t, updatedConfigData, "format: text", "updated log format should be merged")
	assert.Contains(t, updatedConfigData, "level: info", "config set inside spec should have precedence over config set in configmap")

	// test expected status
	pm = f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
	available, err := k8s.FindCondition(pm.Status.Conditions, v1alpha1.Available)
	assert.NoError(t, err, "unable to get available condition")
	assert.Equal(t, available.ObservedGeneration, pm.Generation)
	assert.Equal(t, available.Status, v1alpha1.ConditionTrue)
}
