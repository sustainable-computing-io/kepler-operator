// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

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
	"k8s.io/utils/ptr"
)

func TestPowerMonitor_Deletion(t *testing.T) {
	f := utils.NewFramework(t)

	// Pre-condition: Verify PowerMonitor doesn't exist
	f.AssertNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{})

	// Create PowerMonitor
	f.CreateTestPowerMonitor("power-monitor", runningOnVM)

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
	f.CreateTestPowerMonitor("power-monitor", runningOnVM)

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
	f.CreateTestPowerMonitor("power-monitor", runningOnVM, f.WithPowerMonitorNodeSelector(labels))

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
	f.CreateTestPowerMonitor("power-monitor", runningOnVM, f.WithPowerMonitorNodeSelector(unavailableLabels))

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
	f.CreateTestPowerMonitor("power-monitor", runningOnVM, f.WithPowerMonitorTolerations(append(node.Spec.Taints, e2eTestTaint)))

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
	f.CreateTestPowerMonitor("power-monitor", runningOnVM, f.WithPowerMonitorTolerations(append(node.Spec.Taints, badTestTaint)))
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
	reconciled, err := k8s.FindCondition(pm.Status.Conditions, v1alpha1.Reconciled)
	assert.NoError(t, err, "unable to get reconciled condition")
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
	err = f.Patch(cfm)
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

	// Create PowerMonitor with RBAC security mode
	pm := f.CreateTestPowerMonitor("power-monitor", runningOnVM,
		f.WithPowerMonitorSecuritySet(
			v1alpha1.SecurityModeRBAC,
			[]string{
				"successful-test-namespace:successful-test-curl-sa",
			},
		))

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
	f.CreateTestPowerMonitor("power-monitor", runningOnVM,
		f.WithMaxTerminated(250), // Custom MaxTerminated
		f.WithStaleness("1s"),    // Custom Staleness (1 second)
		f.WithSampleRate("10s"))  // Custom SampleRate (10 seconds)

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
	f.CreateTestPowerMonitor("power-monitor", runningOnVM,
		f.WithMaxTerminated(0), // Zero MaxTerminated (disabled)
		f.WithStaleness("0s"),  // Zero Staleness
		f.WithSampleRate("0s")) // Zero SampleRate

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

func TestPowerMonitor_Secrets_Lifecycle_CRUD(t *testing.T) {
	f := utils.NewFramework(t)
	name := "power-monitor"

	testNs := controller.PowerMonitorDeploymentNS
	secretName := "lifecycle-test-secret"

	// Pre-condition: Verify PowerMonitor doesn't exist
	f.AssertNoResourceExists(name, "", &v1alpha1.PowerMonitor{})

	// Step 1: Create PowerMonitor without any secrets initially
	pm := f.CreateTestPowerMonitor(name, runningOnVM)

	// Wait for PowerMonitor to be reconciled successfully (no secrets = no issues)
	pm = f.WaitUntilPowerMonitorCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)
	pm = f.WaitUntilPowerMonitorCondition(name, v1alpha1.Available, v1alpha1.ConditionTrue)

	// Step 2: Update PowerMonitor to add a secret reference (secret doesn't exist yet)
	secretRef := v1alpha1.SecretRef{
		Name:      secretName,
		MountPath: "/etc/kepler/lifecycle-config",
		ReadOnly:  ptr.To(true),
	}

	// Update the PM to include the secret reference using DeepCopy to avoid managedFields issues
	pm = f.GetPowerMonitor(name)
	{
		patchPm := pm.DeepCopy()
		patchPm.Spec.Kepler.Deployment.Secrets = []v1alpha1.SecretRef{secretRef}
		err := f.Patch(patchPm)
		assert.NoError(t, err, "Should be able to update PM with secret reference")
	}

	// Step 3: Wait for PowerMonitor to reach degraded state due to missing secret
	pm = f.WaitUntilPowerMonitorCondition(name, v1alpha1.Available, v1alpha1.ConditionDegraded, utils.Timeout(2*time.Minute))

	// Assert that the degraded condition is specifically due to missing secret
	availableCondition, err := k8s.FindCondition(pm.Status.Conditions, v1alpha1.Available)
	assert.NoError(t, err, "Should find Available condition")
	assert.Equal(t, v1alpha1.SecretNotFound, availableCondition.Reason, "PowerMonitor should be degraded due to missing secret")
	assert.Contains(t, availableCondition.Message, secretName, "Error message should mention the missing secret")
	assert.Contains(t, availableCondition.Message, testNs, "Error message should mention the namespace")
	t.Logf("PowerMonitor correctly degraded due to missing secret: %s", availableCondition.Message)

	// Step 4: Create the missing secret
	f.CreateTestSecret(secretName, testNs, map[string]string{
		"redfish.yaml": "database: lifecycle-test\\nmode: testing",
		"app.conf":     "lifecycle=enabled\\ntesting=true",
	})

	// Wait for PowerMonitor to recover and become available
	pm = f.WaitUntilPowerMonitorCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)
	pm = f.WaitUntilPowerMonitorCondition(name, v1alpha1.Available, v1alpha1.ConditionTrue)

	// Verify the secret is now properly mounted in the underlying DaemonSet
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pm.Name, testNs, &ds)

	// Verify secret volume exists
	secretVolumes := 0
	for _, vol := range ds.Spec.Template.Spec.Volumes {
		if vol.Secret != nil && vol.Secret.SecretName == secretName {
			secretVolumes++
			assert.Equal(t, secretName, vol.Name, "Volume name should match secret name")
		}
	}
	assert.Equal(t, 1, secretVolumes, "Should have exactly 1 secret volume")

	// Verify secret volume mount in container
	containers := ds.Spec.Template.Spec.Containers
	keplerCntr, err := f.ContainerWithName(containers, pm.Name)
	assert.NoError(t, err, "Should find the kepler container")

	secretMounts := 0
	for _, mount := range keplerCntr.VolumeMounts {
		if mount.Name == secretName {
			secretMounts++
			assert.Equal(t, "/etc/kepler/lifecycle-config", mount.MountPath, "Mount path should match specification")
			assert.True(t, mount.ReadOnly, "Secret should be mounted read-only")
		}
	}
	assert.Equal(t, 1, secretMounts, "Should have exactly 1 secret mount")

	// Assert that the Available condition is no longer showing SecretNotFound
	recoveredCondition, err := k8s.FindCondition(pm.Status.Conditions, v1alpha1.Available)
	assert.NoError(t, err, "Should find Available condition")
	assert.NotEqual(t, v1alpha1.SecretNotFound, recoveredCondition.Reason,
		"Available condition should no longer have SecretNotFound reason after secret creation")
	assert.Equal(t, v1alpha1.ConditionTrue, recoveredCondition.Status, "PowerMonitor should be available")
	t.Logf("PowerMonitor successfully recovered: status=%s, reason=%s", recoveredCondition.Status, recoveredCondition.Reason)

	// Step 5: Delete the secret to trigger degradation again
	f.DeleteTestSecret(secretName, testNs)

	// Wait for PowerMonitor to become degraded again due to missing secret
	pm = f.WaitUntilPowerMonitorCondition(name, v1alpha1.Available, v1alpha1.ConditionDegraded, utils.Timeout(2*time.Minute))

	// Assert that the degraded condition is again due to missing secret
	degradedAgainCondition, err := k8s.FindCondition(pm.Status.Conditions, v1alpha1.Available)
	assert.NoError(t, err, "Should find Available condition")
	assert.Equal(t, v1alpha1.SecretNotFound, degradedAgainCondition.Reason, "PowerMonitor should be degraded again due to missing secret")
	assert.Contains(t, degradedAgainCondition.Message, secretName, "Error message should mention the missing secret")
	t.Logf("PowerMonitor correctly degraded again after secret deletion: %s", degradedAgainCondition.Message)

	// Step 6: Remove the secret reference from PM spec using DeepCopy
	pm = f.GetPowerMonitor(name)
	{
		t.Logf("Remove the secret reference from PM")
		removePatchPm := pm.DeepCopy()
		removePatchPm.Spec.Kepler.Deployment.Secrets = []v1alpha1.SecretRef{} // Empty slice to remove secrets
		err = f.Patch(removePatchPm)
		assert.NoError(t, err, "Should be able to update PM to remove secret reference")
	}

	// Wait for PowerMonitor to become available again (no secret requirements)
	pm = f.WaitUntilPowerMonitorCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)
	pm = f.WaitUntilPowerMonitorCondition(name, v1alpha1.Available, v1alpha1.ConditionTrue)

	// Assert that the Available condition is healthy again
	finalCondition, err := k8s.FindCondition(pm.Status.Conditions, v1alpha1.Available)
	assert.NoError(t, err, "Should find Available condition")
	assert.NotEqual(t, v1alpha1.SecretNotFound, finalCondition.Reason,
		"Available condition should no longer have SecretNotFound reason after removing secret reference")
	assert.Equal(t, v1alpha1.ConditionTrue, finalCondition.Status, "PowerMonitor should be available")
	t.Logf("PowerMonitor final recovery after removing secret reference: status=%s, reason=%s", finalCondition.Status, finalCondition.Reason)

	// Verify that the secret volume is no longer present in the DaemonSet
	finalDs := appsv1.DaemonSet{}
	f.AssertResourceExists(pm.Name, testNs, &finalDs)

	secretVolumesAfterRemoval := 0
	for _, vol := range finalDs.Spec.Template.Spec.Volumes {
		if vol.Secret != nil && vol.Secret.SecretName == secretName {
			secretVolumesAfterRemoval++
		}
	}
	assert.Equal(t, 0, secretVolumesAfterRemoval, "Should have no secret volumes after removing secret reference")

	// Verify that the secret volume mount is no longer present in the container
	finalContainers := finalDs.Spec.Template.Spec.Containers
	finalKeplerCntr, err := f.ContainerWithName(finalContainers, pm.Name)
	assert.NoError(t, err, "Should find the kepler container")

	secretMountsAfterRemoval := 0
	for _, mount := range finalKeplerCntr.VolumeMounts {
		if mount.Name == secretName {
			secretMountsAfterRemoval++
		}
	}
	assert.Equal(t, 0, secretMountsAfterRemoval, "Should have no secret mounts after removing secret reference")

	t.Logf("Lifecycle test completed successfully: PM went through all expected state transitions")
}
