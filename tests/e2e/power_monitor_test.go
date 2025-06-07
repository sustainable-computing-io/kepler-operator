// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"time"

	// "strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/internal/controller"

	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/test"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

func TestPowerMonitor_Deletion(t *testing.T) {
	f := test.NewFramework(t)

	// Pre-condition: Verify PowerMonitor doesn't exist
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{})

	// Create PowerMonitor
	if runningOnVM {
		configMapName := "my-custom-config"
		f.CreatePowerMonitor("power-monitor", f.WithAdditionalConfigMaps([]string{configMapName}))
		cfm := f.NewAdditionalConfigMap(configMapName, controller.PowerMonitorDeploymentNS, `dev:
  fake-cpu-meter:
    enabled: true`)
		err := f.Patch(cfm)
		assert.NoError(t, err)
	} else {
		f.CreatePowerMonitor("power-monitor")
	}

	// Wait until PowerMonitor is available
	pm := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)

	// Verify DaemonSet exists
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(
		pm.Name,
		controller.PowerMonitorDeploymentNS,
		&ds,
		test.Timeout(10*time.Second),
	)
}

func TestPowerMonitor_Reconciliation(t *testing.T) {
	f := test.NewFramework(t)

	// Pre-condition: Verify PowerMonitor doesn't exist
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{})

	// Create PowerMonitor
	if runningOnVM {
		configMapName := "my-custom-config"
		f.CreatePowerMonitor("power-monitor", f.WithAdditionalConfigMaps([]string{configMapName}))
		cfm := f.NewAdditionalConfigMap(configMapName, controller.PowerMonitorDeploymentNS, `dev:
  fake-cpu-meter:
    enabled: true`)
		err := f.Patch(cfm)
		assert.NoError(t, err)
	} else {
		f.CreatePowerMonitor("power-monitor")
	}

	// Verify reconciliation
	pm := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Reconciled, v1alpha1.ConditionTrue)
	f.AssertResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &ds)

	// Verify default toleration
	assert.Equal(t, []corev1.Toleration{{Operator: "Exists"}}, pm.Spec.Kepler.Deployment.Tolerations)
	reconciled, err := k8s.FindCondition(pm.Status.Conditions, v1alpha1.Reconciled)
	assert.NoError(t, err, "unable to get reconciled condition")
	assert.Equal(t, reconciled.ObservedGeneration, pm.Generation)
	assert.Equal(t, reconciled.Status, v1alpha1.ConditionTrue)

	// Verify available condition
	pm = f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
	available, err := k8s.FindCondition(pm.Status.Conditions, v1alpha1.Available)
	assert.NoError(t, err, "unable to get available condition")
	assert.Equal(t, available.ObservedGeneration, pm.Generation)
	assert.Equal(t, available.Status, v1alpha1.ConditionTrue)
}

func TestBadPowerMonitor_Reconciliation(t *testing.T) {
	f := test.NewFramework(t)

	// Pre-condition: Verify PowerMonitor doesn't exist
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{}, test.Timeout(10*time.Second))
	f.AssertNoResourceExists("invalid-name", "", &v1alpha1.PowerMonitor{})

	// Attempt to create PowerMonitor with invalid name
	powermonitor := f.NewPowerMonitor("invalid-name")
	err := f.Patch(&powermonitor)
	assert.ErrorContains(t, err, "denied the request")
}

func TestPowerMonitorNodeSelector(t *testing.T) {
	f := test.NewFramework(t)

	// Pre-condition: Verify PowerMonitor doesn't exist
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{}, test.Timeout(10*time.Second))

	// Label a node
	nodes := f.GetSchedulableNodes()
	assert.NotZero(t, len(nodes), "got zero nodes")
	node := nodes[0]
	var labels k8s.StringMap = map[string]string{"e2e-test": "true"}
	err := f.AddResourceLabels("node", node.Name, labels)
	assert.NoError(t, err, "could not label node")

	// Create PowerMonitor with node selector
	if runningOnVM {
		configMapName := "my-custom-config"
		f.CreatePowerMonitor("power-monitor", f.WithAdditionalConfigMaps([]string{configMapName}),
			f.WithPowerMonitorNodeSelector(labels))
		cfm := f.NewAdditionalConfigMap(configMapName, controller.PowerMonitorDeploymentNS, `dev:
  fake-cpu-meter:
    enabled: true`)
		err := f.Patch(cfm)
		assert.NoError(t, err)
	} else {
		f.CreatePowerMonitor("power-monitor", f.WithPowerMonitorNodeSelector(labels))
	}

	// Verify PowerMonitor is available
	pm := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
	f.AssertResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &ds)
	assert.EqualValues(t, 1, pm.Status.Kepler.NumberAvailable)
}

func TestPowerMonitorNodeSelectorUnavailableLabel(t *testing.T) {
	f := test.NewFramework(t)

	// Pre-condition: Verify PowerMonitor doesn't exist
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{}, test.Timeout(10*time.Second))

	// Verify nodes exist
	nodes := f.GetSchedulableNodes()
	assert.NotZero(t, len(nodes), "got zero nodes")

	// Create PowerMonitor with unavailable node selector
	unavailableLabels := k8s.StringMap{"e2e-test": "true"}
	if runningOnVM {
		configMapName := "my-custom-config"
		f.CreatePowerMonitor("power-monitor", f.WithAdditionalConfigMaps([]string{configMapName}), f.WithPowerMonitorNodeSelector(unavailableLabels))
		cfm := f.NewAdditionalConfigMap(configMapName, controller.PowerMonitorDeploymentNS, `dev:
  fake-cpu-meter:
    enabled: true`)
		err := f.Patch(cfm)
		assert.NoError(t, err)
	} else {
		f.CreatePowerMonitor("power-monitor", f.WithPowerMonitorNodeSelector(unavailableLabels))
	}

	// Verify PowerMonitor is unavailable
	pm := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionFalse)
	f.AssertResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &ds)
	assert.EqualValues(t, 0, pm.Status.Kepler.NumberAvailable)
}

func TestPowerMonitorTaint_WithToleration(t *testing.T) {
	f := test.NewFramework(t)

	// Pre-condition: Verify PowerMonitor doesn't exist
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{}, test.Timeout(10*time.Second))

	// Taint a node
	nodes := f.GetSchedulableNodes()
	node := nodes[0]
	e2eTestTaint := corev1.Taint{
		Key:    "key1",
		Value:  "value1",
		Effect: corev1.TaintEffectNoSchedule,
	}
	err := f.TaintNode(node.Name, e2eTestTaint.ToString())
	assert.NoError(t, err, "failed to taint node %s", node)

	// Create PowerMonitor with toleration
	if runningOnVM {
		configMapName := "my-custom-config"
		f.CreatePowerMonitor("power-monitor", f.WithAdditionalConfigMaps([]string{configMapName}), f.WithPowerMonitorTolerations(append(node.Spec.Taints, e2eTestTaint)))
		cfm := f.NewAdditionalConfigMap(configMapName, controller.PowerMonitorDeploymentNS, `dev:
  fake-cpu-meter:
    enabled: true`)
		err := f.Patch(cfm)
		assert.NoError(t, err)
	} else {
		f.CreatePowerMonitor("power-monitor", f.WithPowerMonitorTolerations(append(node.Spec.Taints, e2eTestTaint)))
	}

	// Verify PowerMonitor is available
	pm := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
	f.AssertResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &ds)
	assert.EqualValues(t, len(nodes), pm.Status.Kepler.NumberAvailable)
}

func TestBadPowerMonitorTaint_WithToleration(t *testing.T) {
	f := test.NewFramework(t)

	// Pre-condition: Verify PowerMonitor doesn't exist
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{}, test.Timeout(10*time.Second))

	// Taint a node
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

	// Create PowerMonitor with incorrect toleration
	if runningOnVM {
		configMapName := "my-custom-config"
		f.CreatePowerMonitor("power-monitor", f.WithAdditionalConfigMaps([]string{configMapName}), f.WithPowerMonitorTolerations(append(node.Spec.Taints, badTestTaint)))
		cfm := f.NewAdditionalConfigMap(configMapName, controller.PowerMonitorDeploymentNS, `dev:
  fake-cpu-meter:
    enabled: true`)
		err := f.Patch(cfm)
		assert.NoError(t, err)
	} else {
		f.CreatePowerMonitor("power-monitor", f.WithPowerMonitorTolerations(append(node.Spec.Taints, badTestTaint)))
	}
	// Verify PowerMonitor is available but with reduced nodes
	pm := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
	f.AssertResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &ds)
	assert.EqualValues(t, len(nodes)-1, pm.Status.Kepler.NumberAvailable)
}

func TestPowerMonitor_ReconciliationWithAdditionalConfigMap(t *testing.T) {
	f := test.NewFramework(t)
	configMapName := "my-custom-config"

	// Pre-condition: Verify PowerMonitor doesn't exist
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{})

	// Create PowerMonitor with additional config map
	pm := f.CreatePowerMonitor("power-monitor", f.WithAdditionalConfigMaps([]string{configMapName}))

	// Verify Daemonset doesn't exist
	ds := appsv1.DaemonSet{}
	f.AssertNoResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &ds)

	// Verify reconcillation fails without config map
	pm = f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Reconciled, v1alpha1.ConditionFalse)
	reconciled, _ := k8s.FindCondition(pm.Status.Conditions, v1alpha1.Reconciled)
	assert.Contains(t, reconciled.Message, fmt.Sprintf("configMap %s not found in %s namespace", configMapName, controller.PowerMonitorDeploymentNS))

	// Create config map
	conf := `log:
  format: json
  level: debug`

	if runningOnVM {
		conf += `
dev:
  fake-cpu-meter:
    enabled: true`
	}
	cfm := f.NewAdditionalConfigMap(configMapName, controller.PowerMonitorDeploymentNS, conf)
	err := f.Patch(cfm)
	assert.NoError(t, err)

	// Verify reconcillation succeeds
	pm = f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Reconciled, v1alpha1.ConditionTrue)

	// Verify Daemonset exists
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &ds)

	// Verify merged config map
	mainConfigMap := corev1.ConfigMap{}
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &mainConfigMap)
	configData := mainConfigMap.Data["config.yaml"]
	assert.Contains(t, configData, "format: json", "custom log format should be merged")
	assert.Contains(t, configData, "sysfs: /host/sys", "default sysfs path should be present")
	assert.Contains(t, configData, "procfs: /host/proc", "default procfs path should be present")

	// Verify Daemonset annotation
	assert.Contains(t, ds.Spec.Template.Annotations, "powermonitor.sustainable.computing.io/config-map-hash-"+pm.Name)
	og := ds.Status.ObservedGeneration
	assert.Equal(t, og, int64(1))

	// Update config map
	updatedConf := `log:
  format: text
  level: warn`

	if runningOnVM {
		updatedConf += `
dev:
  fake-cpu-meter:
    enabled: true`
	}
	cfm = f.NewAdditionalConfigMap(configMapName, controller.PowerMonitorDeploymentNS, updatedConf)
	err = f.Patch(cfm)
	assert.NoError(t, err)

	// Wait for Daemonset restart
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

	// Verify updated config
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &mainConfigMap)
	updatedConfigData := mainConfigMap.Data["config.yaml"]
	assert.Contains(t, updatedConfigData, "format: text", "updated log format should be merged")
	assert.Contains(t, updatedConfigData, "level: info", "config set inside spec should have precedence over config set in configmap")

	// Verify availability
	pm = f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
	available, err := k8s.FindCondition(pm.Status.Conditions, v1alpha1.Available)
	assert.NoError(t, err, "unable to get available condition")
	assert.Equal(t, available.ObservedGeneration, pm.Generation)
	assert.Equal(t, available.Status, v1alpha1.ConditionTrue)
}
