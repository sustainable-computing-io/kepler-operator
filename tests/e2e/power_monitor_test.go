// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"strings"
	"time"

	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/internal/controller"

	powermonitor "github.com/sustainable.computing.io/kepler-operator/pkg/components/power-monitor"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"github.com/sustainable.computing.io/kepler-operator/tests/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

func TestPowerMonitor_Deletion(t *testing.T) {
	f := utils.NewFramework(t)

	// Pre-condition: Verify PowerMonitor doesn't exist
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{})

	// Create PowerMonitor
	if runningOnVM {
		configMapName := "my-custom-config"
		f.CreatePowerMonitor(
			"power-monitor",
			f.WithAdditionalConfigMaps([]string{configMapName}),
			f.WithPowerMonitorSecuritySet(
				v1alpha1.SecurityModeNone,
				[]string{},
			),
		)
		cfm := f.NewAdditionalConfigMap(configMapName, controller.PowerMonitorDeploymentNS, `dev:
  fake-cpu-meter:
    enabled: true`)
		err := f.Patch(cfm)
		assert.NoError(t, err)
	} else {
		f.CreatePowerMonitor(
			"power-monitor",
			f.WithPowerMonitorSecuritySet(
				v1alpha1.SecurityModeNone,
				[]string{},
			),
		)
	}

	// Wait until PowerMonitor is available
	pm := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)

	// Verify DaemonSet exists
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(
		pm.Name,
		controller.PowerMonitorDeploymentNS,
		&ds,
		utils.Timeout(10*time.Second),
	)
}

func TestPowerMonitor_Reconciliation(t *testing.T) {
	f := utils.NewFramework(t)

	// Pre-condition: Verify PowerMonitor doesn't exist
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{})

	// Create PowerMonitor
	if runningOnVM {
		configMapName := "my-custom-config"
		f.CreatePowerMonitor(
			"power-monitor",
			f.WithAdditionalConfigMaps([]string{configMapName}),
			f.WithPowerMonitorSecuritySet(
				v1alpha1.SecurityModeNone,
				[]string{},
			),
		)
		cfm := f.NewAdditionalConfigMap(configMapName, controller.PowerMonitorDeploymentNS, `dev:
  fake-cpu-meter:
    enabled: true`)
		err := f.Patch(cfm)
		assert.NoError(t, err)
	} else {
		f.CreatePowerMonitor(
			"power-monitor",
			f.WithPowerMonitorSecuritySet(
				v1alpha1.SecurityModeNone,
				[]string{},
			),
		)
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
	f := utils.NewFramework(t)

	// Pre-condition: Verify PowerMonitor doesn't exist
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{}, utils.Timeout(10*time.Second))
	f.AssertNoResourceExists("invalid-name", "", &v1alpha1.PowerMonitor{})

	// Attempt to create PowerMonitor with invalid name
	powermonitor := f.NewPowerMonitor("invalid-name")
	err := f.Patch(&powermonitor)
	assert.ErrorContains(t, err, "denied the request")
}

func TestPowerMonitorNodeSelector(t *testing.T) {
	f := utils.NewFramework(t)

	// Pre-condition: Verify PowerMonitor doesn't exist
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{}, utils.Timeout(10*time.Second))

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
		f.CreatePowerMonitor(
			"power-monitor",
			f.WithAdditionalConfigMaps([]string{configMapName}),
			f.WithPowerMonitorNodeSelector(labels),
			f.WithPowerMonitorSecuritySet(
				v1alpha1.SecurityModeNone,
				[]string{},
			),
		)
		cfm := f.NewAdditionalConfigMap(configMapName, controller.PowerMonitorDeploymentNS, `dev:
  fake-cpu-meter:
    enabled: true`)
		err := f.Patch(cfm)
		assert.NoError(t, err)
	} else {
		f.CreatePowerMonitor(
			"power-monitor",
			f.WithPowerMonitorNodeSelector(labels),
			f.WithPowerMonitorSecuritySet(
				v1alpha1.SecurityModeNone,
				[]string{},
			),
		)
	}

	// Verify PowerMonitor is available
	pm := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
	f.AssertResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &ds)
	assert.EqualValues(t, 1, pm.Status.Kepler.NumberAvailable)
}

func TestPowerMonitorNodeSelectorUnavailableLabel(t *testing.T) {
	f := utils.NewFramework(t)

	// Pre-condition: Verify PowerMonitor doesn't exist
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{}, utils.Timeout(10*time.Second))

	// Verify nodes exist
	nodes := f.GetSchedulableNodes()
	assert.NotZero(t, len(nodes), "got zero nodes")

	// Create PowerMonitor with unavailable node selector
	unavailableLabels := k8s.StringMap{"e2e-test": "true"}
	if runningOnVM {
		configMapName := "my-custom-config"
		f.CreatePowerMonitor(
			"power-monitor",
			f.WithAdditionalConfigMaps([]string{configMapName}), f.WithPowerMonitorNodeSelector(unavailableLabels),
			f.WithPowerMonitorSecuritySet(
				v1alpha1.SecurityModeNone,
				[]string{},
			),
		)
		cfm := f.NewAdditionalConfigMap(configMapName, controller.PowerMonitorDeploymentNS, `dev:
  fake-cpu-meter:
    enabled: true`)
		err := f.Patch(cfm)
		assert.NoError(t, err)
	} else {
		f.CreatePowerMonitor(
			"power-monitor",
			f.WithPowerMonitorNodeSelector(unavailableLabels),
			f.WithPowerMonitorSecuritySet(
				v1alpha1.SecurityModeNone,
				[]string{},
			),
		)
	}

	// Verify PowerMonitor is unavailable
	pm := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionFalse)
	f.AssertResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &ds)
	assert.EqualValues(t, 0, pm.Status.Kepler.NumberAvailable)
}

func TestPowerMonitorTaint_WithToleration(t *testing.T) {
	f := utils.NewFramework(t)

	// Pre-condition: Verify PowerMonitor doesn't exist
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{}, utils.Timeout(10*time.Second))

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
		f.CreatePowerMonitor(
			"power-monitor",
			f.WithAdditionalConfigMaps([]string{configMapName}),
			f.WithPowerMonitorTolerations(append(node.Spec.Taints, e2eTestTaint)),
			f.WithPowerMonitorSecuritySet(
				v1alpha1.SecurityModeNone,
				[]string{},
			),
		)
		cfm := f.NewAdditionalConfigMap(configMapName, controller.PowerMonitorDeploymentNS, `dev:
  fake-cpu-meter:
    enabled: true`)
		err := f.Patch(cfm)
		assert.NoError(t, err)
	} else {
		f.CreatePowerMonitor(
			"power-monitor",
			f.WithPowerMonitorTolerations(append(node.Spec.Taints, e2eTestTaint)),
			f.WithPowerMonitorSecuritySet(
				v1alpha1.SecurityModeNone,
				[]string{},
			),
		)
	}

	// Verify PowerMonitor is available
	pm := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
	f.AssertResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &ds)
	assert.EqualValues(t, len(nodes), pm.Status.Kepler.NumberAvailable)
}

func TestBadPowerMonitorTaint_WithToleration(t *testing.T) {
	f := utils.NewFramework(t)

	// Pre-condition: Verify PowerMonitor doesn't exist
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{}, utils.Timeout(10*time.Second))

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
		f.CreatePowerMonitor(
			"power-monitor",
			f.WithAdditionalConfigMaps([]string{configMapName}),
			f.WithPowerMonitorTolerations(append(node.Spec.Taints, badTestTaint)),
			f.WithPowerMonitorSecuritySet(
				v1alpha1.SecurityModeNone,
				[]string{},
			),
		)
		cfm := f.NewAdditionalConfigMap(configMapName, controller.PowerMonitorDeploymentNS, `dev:
  fake-cpu-meter:
    enabled: true`)
		err := f.Patch(cfm)
		assert.NoError(t, err)
	} else {
		f.CreatePowerMonitor(
			"power-monitor",
			f.WithPowerMonitorTolerations(append(node.Spec.Taints, badTestTaint)),
			f.WithPowerMonitorSecuritySet(
				v1alpha1.SecurityModeNone,
				[]string{},
			),
		)
	}
	// Verify PowerMonitor is available but with reduced nodes
	pm := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
	f.AssertResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &ds)
	assert.EqualValues(t, len(nodes)-1, pm.Status.Kepler.NumberAvailable)
}

func TestPowerMonitor_ReconciliationWithAdditionalConfigMap(t *testing.T) {
	f := utils.NewFramework(t)
	configMapName := "my-custom-config"

	// Pre-condition: Verify PowerMonitor doesn't exist
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{})

	// Create PowerMonitor with additional config map
	pm := f.CreatePowerMonitor(
		"power-monitor",
		f.WithAdditionalConfigMaps([]string{configMapName}),
		f.WithPowerMonitorSecuritySet(
			v1alpha1.SecurityModeNone,
			[]string{},
		),
	)

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

func TestPowerMonitor_RBAC_Reconciliation(t *testing.T) {
	f := utils.NewFramework(t)

	// pre-condition
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{})

	// Create PowerMonitor with additional config map
	var pm *v1alpha1.PowerMonitor
	if runningOnVM {
		configMapName := "my-custom-config"
		pm = f.CreatePowerMonitor(
			"power-monitor",
			f.WithAdditionalConfigMaps([]string{configMapName}),
			f.WithPowerMonitorSecuritySet(
				v1alpha1.SecurityModeRBAC,
				[]string{
					"successful-test-namespace:successful-test-curl-sa",
				},
			),
		)
		cfm := f.NewAdditionalConfigMap(configMapName, controller.PowerMonitorDeploymentNS, `dev:
  fake-cpu-meter:
    enabled: true`)
		err := f.Patch(cfm)
		assert.NoError(t, err)
	} else {
		pm = f.CreatePowerMonitor(
			"power-monitor",
			f.WithPowerMonitorSecuritySet(
				v1alpha1.SecurityModeRBAC,
				[]string{
					"successful-test-namespace:successful-test-curl-sa",
				},
			),
		)
	}

	tlsCertSecretName := powermonitor.SecretTLSCertName
	var caCertSource string

	if Cluster == k8s.Kubernetes {
		// For Kubernetes clusters, deploy cert-manager and dependencies
		clusterIssuerName := "selfsigned-cluster-issuer"
		caCertName := "power-monitor-ca"
		caCertSecretName := "power-monitor-ca-secret"
		pmIssuerName := "power-monitor-ca-issuer"
		f.DeployOpenshiftCerts(
			pm.Name,
			controller.PowerMonitorDeploymentNS,
			clusterIssuerName,
			caCertName,
			caCertSecretName,
			pmIssuerName,
			tlsCertSecretName,
			tlsCertSecretName,
		)
		caCertSource = caCertSecretName
	} else {
		f.WaitUntilPowerMonitorCondition(pm.Name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)
		f.WaitForOpenshiftCerts(pm.Name, controller.PowerMonitorDeploymentNS, tlsCertSecretName)
		caCertSource = tlsCertSecretName
	}

	// then
	f.AssertResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &ds)

	retrievedPm := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Reconciled, v1alpha1.ConditionTrue)
	// default toleration
	assert.Equal(t, []corev1.Toleration{{Operator: "Exists"}}, retrievedPm.Spec.Kepler.Deployment.Tolerations)
	reconciled, err := k8s.FindCondition(retrievedPm.Status.Conditions, v1alpha1.Reconciled)
	assert.NoError(t, err, "unable to get reconciled condition")
	assert.Equal(t, reconciled.ObservedGeneration, retrievedPm.Generation)
	assert.Equal(t, reconciled.Status, v1alpha1.ConditionTrue)

	retrievedPm = f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
	available, err := k8s.FindCondition(retrievedPm.Status.Conditions, v1alpha1.Available)
	assert.NoError(t, err, "unable to get available condition")
	assert.Equal(t, available.ObservedGeneration, retrievedPm.Generation)
	assert.Equal(t, available.Status, v1alpha1.ConditionTrue)

	audience := fmt.Sprintf("%s.%s.svc", pm.Name, controller.PowerMonitorDeploymentNS)
	serviceURL := fmt.Sprintf(
		"https://%s.%s.svc:%d/metrics",
		pm.Name,
		controller.PowerMonitorDeploymentNS,
		powermonitor.SecurePort,
	)

	// wait for relevant secrets to be created
	tlsSecret := corev1.Secret{}
	f.AssertResourceExists(
		tlsCertSecretName,
		controller.PowerMonitorDeploymentNS,
		&tlsSecret,
		utils.Timeout(5*time.Minute),
	)
	assert.NotEmpty(t, tlsSecret.Data["tls.crt"], "TLS cert should be present")
	assert.NotEmpty(t, tlsSecret.Data["tls.key"], "TLS key should be present")

	// deploy successful curl job
	successfulJobName := "successful-test-curl"
	successfulTestSAName := "successful-test-curl-sa"
	successfulTestCurlNs := "successful-test-namespace"
	var jobLogs string

	if Cluster == k8s.Kubernetes {
		jobLogs = f.CreateCurlPowerMonitorTestSuite(
			successfulJobName,
			successfulTestSAName,
			successfulTestCurlNs,
			audience,
			serviceURL,
			caCertSource,
			controller.PowerMonitorDeploymentNS,
		)
	} else {
		jobLogs = f.CreateCurlPowerMonitorTestSuiteForOpenShift(
			successfulJobName,
			successfulTestSAName,
			successfulTestCurlNs,
			audience,
			serviceURL,
			caCertSource,
			controller.PowerMonitorDeploymentNS,
		)
	}
	assert.True(t, strings.Contains(jobLogs, "HTTP/2 200"), fmt.Sprintf("expected %s to successfully access (200) the secure endpoint but it did not", successfulJobName))

	// deploy blocked curl job
	failedJobname := "failed-test-curl"
	failedTestSAName := "failed-test-curl-sa"
	failedTestCurlNs := "failed-test-namespace"

	if Cluster == k8s.Kubernetes {
		jobLogs = f.CreateCurlPowerMonitorTestSuite(
			failedJobname,
			failedTestSAName,
			failedTestCurlNs,
			audience,
			serviceURL,
			caCertSource,
			controller.PowerMonitorDeploymentNS,
		)
	} else {
		jobLogs = f.CreateCurlPowerMonitorTestSuiteForOpenShift(
			failedJobname,
			failedTestSAName,
			failedTestCurlNs,
			audience,
			serviceURL,
			caCertSource,
			controller.PowerMonitorDeploymentNS,
		)
	}
	assert.True(t, strings.Contains(jobLogs, "HTTP/2 403"), fmt.Sprintf("expected %s to receive a forbidden error (403) when attempting to access secure endpoint but did not", failedJobname))
	f.DeletePowerMonitor("power-monitor")
	f.AssertNoResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
	f.AssertNoResourceExists(ds.Name, ds.Namespace, &ds)
}

func TestPowerMonitor_NewConfigFields(t *testing.T) {
	f := utils.NewFramework(t)

	// Pre-condition: Verify PowerMonitor doesn't exist
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{})

	// Create PowerMonitor with all new config fields
	if runningOnVM {
		configMapName := "my-custom-config"
		f.CreatePowerMonitor(
			"power-monitor",
			f.WithAdditionalConfigMaps([]string{configMapName}),
			f.WithMaxTerminated(250), // Custom MaxTerminated
			f.WithStaleness("1s"),    // Custom Staleness (1 second)
			f.WithSampleRate("10s"),  // Custom SampleRate (10 seconds)
			f.WithPowerMonitorSecuritySet(
				v1alpha1.SecurityModeNone,
				[]string{},
			),
		)
		cfm := f.NewAdditionalConfigMap(configMapName, controller.PowerMonitorDeploymentNS, `dev:
  fake-cpu-meter:
    enabled: true`)
		err := f.Patch(cfm)
		assert.NoError(t, err)
	} else {
		f.CreatePowerMonitor(
			"power-monitor",
			f.WithMaxTerminated(250), // Custom MaxTerminated
			f.WithStaleness("1s"),    // Custom Staleness (1 second)
			f.WithSampleRate("10s"),  // Custom SampleRate (10 seconds)
			f.WithPowerMonitorSecuritySet(
				v1alpha1.SecurityModeNone,
				[]string{},
			),
		)
	}

	// Wait until PowerMonitor is available
	pm := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)

	// Verify DaemonSet exists
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(
		pm.Name,
		controller.PowerMonitorDeploymentNS,
		&ds,
		utils.Timeout(10*time.Second),
	)

	// Verify the spec values are properly set
	assert.NotNil(t, pm.Spec.Kepler.Config.MaxTerminated, "MaxTerminated should be set")
	assert.Equal(t, int32(250), *pm.Spec.Kepler.Config.MaxTerminated, "MaxTerminated should be 250")

	assert.NotNil(t, pm.Spec.Kepler.Config.Staleness, "Staleness should be set")
	assert.Equal(t, 1*time.Second, pm.Spec.Kepler.Config.Staleness.Duration, "Staleness should be 1 second")

	assert.NotNil(t, pm.Spec.Kepler.Config.SampleRate, "SampleRate should be set")
	assert.Equal(t, 10*time.Second, pm.Spec.Kepler.Config.SampleRate.Duration, "SampleRate should be 10 seconds")

	// Verify the configuration is applied in the ConfigMap
	mainConfigMap := corev1.ConfigMap{}
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &mainConfigMap)
	configData := mainConfigMap.Data["config.yaml"]

	// Check that our custom values are present in the generated config
	assert.Contains(t, configData, "maxTerminated: 250", "Config should contain custom MaxTerminated value")
	assert.Contains(t, configData, "staleness: 1s", "Config should contain custom Staleness value")
	assert.Contains(t, configData, "interval: 10s", "Config should contain custom SampleRate value")

	// Verify standard default values are still present
	assert.Contains(t, configData, "sysfs: /host/sys", "Default sysfs path should be present")
	assert.Contains(t, configData, "procfs: /host/proc", "Default procfs path should be present")
}

func TestPowerMonitor_ZeroValueConfigFields(t *testing.T) {
	f := utils.NewFramework(t)

	// Pre-condition: Verify PowerMonitor doesn't exist
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{})

	// Create PowerMonitor with zero values for new config fields
	if runningOnVM {
		configMapName := "my-custom-config"
		f.CreatePowerMonitor(
			"power-monitor",
			f.WithAdditionalConfigMaps([]string{configMapName}),
			f.WithMaxTerminated(0), // Zero MaxTerminated (disabled)
			f.WithStaleness("0s"),  // Zero Staleness
			f.WithSampleRate("0s"), // Zero SampleRate
			f.WithPowerMonitorSecuritySet(
				v1alpha1.SecurityModeNone,
				[]string{},
			),
		)
		cfm := f.NewAdditionalConfigMap(configMapName, controller.PowerMonitorDeploymentNS, `dev:
  fake-cpu-meter:
    enabled: true`)
		err := f.Patch(cfm)
		assert.NoError(t, err)
	} else {
		f.CreatePowerMonitor(
			"power-monitor",
			f.WithMaxTerminated(0), // Zero MaxTerminated (disabled)
			f.WithStaleness("0s"),  // Zero Staleness
			f.WithSampleRate("0s"), // Zero SampleRate
			f.WithPowerMonitorSecuritySet(
				v1alpha1.SecurityModeNone,
				[]string{},
			),
		)
	}

	// Wait until PowerMonitor is available
	pm := f.WaitUntilPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)

	// Verify DaemonSet exists
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(
		pm.Name,
		controller.PowerMonitorDeploymentNS,
		&ds,
		utils.Timeout(10*time.Second),
	)

	// Verify the spec values are properly set to zero
	assert.NotNil(t, pm.Spec.Kepler.Config.MaxTerminated, "MaxTerminated should be set")
	assert.Equal(t, int32(0), *pm.Spec.Kepler.Config.MaxTerminated, "MaxTerminated should be 0")

	assert.NotNil(t, pm.Spec.Kepler.Config.Staleness, "Staleness should be set")
	assert.Equal(t, 0*time.Second, pm.Spec.Kepler.Config.Staleness.Duration, "Staleness should be 0")

	assert.NotNil(t, pm.Spec.Kepler.Config.SampleRate, "SampleRate should be set")
	assert.Equal(t, 0*time.Second, pm.Spec.Kepler.Config.SampleRate.Duration, "SampleRate should be 0")

	// Verify the configuration is applied in the ConfigMap
	mainConfigMap := corev1.ConfigMap{}
	f.AssertResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, &mainConfigMap)
	configData := mainConfigMap.Data["config.yaml"]

	// Check that our zero values are present in the generated config
	assert.Contains(t, configData, "maxTerminated: 0", "Config should contain zero MaxTerminated value")
	assert.Contains(t, configData, "staleness: 0s", "Config should contain zero Staleness value")
	assert.Contains(t, configData, "interval: 0s", "Config should contain zero SampleRate value")
}
