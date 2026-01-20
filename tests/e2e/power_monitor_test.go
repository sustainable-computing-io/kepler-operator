// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

var _ = Describe("PowerMonitor", func() {
	var f *utils.Framework

	BeforeEach(func() {
		f = utils.NewFramework()
	})

	Describe("Deletion", func() {
		It("should delete PowerMonitor successfully", func() {
			// Pre-condition: Verify PowerMonitor doesn't exist
			f.ExpectNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{})

			// Create PowerMonitor
			f.CreateTestPowerMonitor("power-monitor", runningOnVM)

			// Wait until PowerMonitor is available
			pm := f.ExpectPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)

			// Verify DaemonSet exists
			ds := &appsv1.DaemonSet{}
			f.ExpectResourceExists(
				pm.Name,
				controller.PowerMonitorDeploymentNS,
				ds,
				utils.Timeout(10*time.Second),
			)
		})
	})

	Describe("Reconciliation", func() {
		It("should reconcile PowerMonitor and create DaemonSet", func() {
			// Pre-condition: Verify PowerMonitor doesn't exist
			f.ExpectNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{})

			// Create PowerMonitor
			f.CreateTestPowerMonitor("power-monitor", runningOnVM)

			// Verify reconciliation
			pm := f.ExpectPowerMonitorCondition("power-monitor", v1alpha1.Reconciled, v1alpha1.ConditionTrue)
			f.ExpectResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
			ds := &appsv1.DaemonSet{}
			f.ExpectResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, ds)

			// Verify default toleration
			Expect(pm.Spec.Kepler.Deployment.Tolerations).To(Equal([]corev1.Toleration{{Operator: "Exists"}}))
			reconciled, err := k8s.FindCondition(pm.Status.Conditions, v1alpha1.Reconciled)
			Expect(err).NotTo(HaveOccurred(), "unable to get reconciled condition")
			Expect(reconciled.ObservedGeneration).To(Equal(pm.Generation))
			Expect(reconciled.Status).To(Equal(v1alpha1.ConditionTrue))

			// Verify available condition
			pm = f.ExpectPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
			available, err := k8s.FindCondition(pm.Status.Conditions, v1alpha1.Available)
			Expect(err).NotTo(HaveOccurred(), "unable to get available condition")
			Expect(available.ObservedGeneration).To(Equal(pm.Generation))
			Expect(available.Status).To(Equal(v1alpha1.ConditionTrue))
		})

		It("should reject PowerMonitor with invalid name", func() {
			// Pre-condition: Verify PowerMonitor doesn't exist
			f.ExpectNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{}, utils.Timeout(10*time.Second))
			f.ExpectNoResourceExists("invalid-name", "", &v1alpha1.PowerMonitor{})

			// Attempt to create PowerMonitor with invalid name
			powermonitor := f.NewPowerMonitor("invalid-name")
			err := f.Patch(&powermonitor)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("denied the request"))
		})
	})

	Describe("NodeSelector", func() {
		It("should deploy only on labeled nodes", func() {
			// Pre-condition: Verify PowerMonitor doesn't exist
			f.ExpectNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{}, utils.Timeout(10*time.Second))

			// Label a node
			nodes := f.GetSchedulableNodes()
			Expect(len(nodes)).NotTo(BeZero(), "got zero nodes")
			node := nodes[0]
			var labels k8s.StringMap = map[string]string{"e2e-test": "true"}
			err := f.AddResourceLabels("node", node.Name, labels)
			Expect(err).NotTo(HaveOccurred(), "could not label node")

			// Create PowerMonitor with node selector
			f.CreateTestPowerMonitor("power-monitor", runningOnVM, f.WithPowerMonitorNodeSelector(labels))

			// Verify PowerMonitor is available
			pm := f.ExpectPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
			f.ExpectResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
			ds := &appsv1.DaemonSet{}
			f.ExpectResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, ds)
			Expect(pm.Status.Kepler.NumberAvailable).To(BeEquivalentTo(1))
		})

		It("should remain unavailable with unavailable node selector", func() {
			// Pre-condition: Verify PowerMonitor doesn't exist
			f.ExpectNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{}, utils.Timeout(10*time.Second))

			// Verify nodes exist
			nodes := f.GetSchedulableNodes()
			Expect(len(nodes)).NotTo(BeZero(), "got zero nodes")

			// Create PowerMonitor with unavailable node selector
			unavailableLabels := k8s.StringMap{"e2e-test": "true"}
			f.CreateTestPowerMonitor("power-monitor", runningOnVM, f.WithPowerMonitorNodeSelector(unavailableLabels))

			// Verify PowerMonitor is unavailable
			pm := f.ExpectPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionFalse)
			f.ExpectResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
			ds := &appsv1.DaemonSet{}
			f.ExpectResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, ds)
			Expect(pm.Status.Kepler.NumberAvailable).To(BeEquivalentTo(0))
		})
	})

	Describe("Taints and Tolerations", func() {
		It("should deploy with correct tolerations", func() {
			// Pre-condition: Verify PowerMonitor doesn't exist
			f.ExpectNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{}, utils.Timeout(10*time.Second))

			// Taint a node
			nodes := f.GetSchedulableNodes()
			node := nodes[0]
			e2eTestTaint := corev1.Taint{
				Key:    "key1",
				Value:  "value1",
				Effect: corev1.TaintEffectNoSchedule,
			}
			err := f.TaintNode(node.Name, e2eTestTaint.ToString())
			Expect(err).NotTo(HaveOccurred(), "failed to taint node %s", node)

			// Create PowerMonitor with toleration
			f.CreateTestPowerMonitor("power-monitor", runningOnVM, f.WithPowerMonitorTolerations(append(node.Spec.Taints, e2eTestTaint)))

			// Verify PowerMonitor is available
			pm := f.ExpectPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
			f.ExpectResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
			ds := &appsv1.DaemonSet{}
			f.ExpectResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, ds)
			Expect(pm.Status.Kepler.NumberAvailable).To(BeEquivalentTo(len(nodes)))
		})

		It("should not deploy on nodes with missing tolerations", func() {
			// Pre-condition: Verify PowerMonitor doesn't exist
			f.ExpectNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{}, utils.Timeout(10*time.Second))

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
			Expect(err).NotTo(HaveOccurred(), "failed to taint node %s", node)

			// Create PowerMonitor with incorrect toleration
			f.CreateTestPowerMonitor("power-monitor", runningOnVM, f.WithPowerMonitorTolerations(append(node.Spec.Taints, badTestTaint)))
			// Verify PowerMonitor is available but with reduced nodes
			pm := f.ExpectPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
			f.ExpectResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
			ds := &appsv1.DaemonSet{}
			f.ExpectResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, ds)
			Expect(pm.Status.Kepler.NumberAvailable).To(BeEquivalentTo(len(nodes) - 1))
		})
	})

	Describe("AdditionalConfigMap", func() {
		It("should reconcile with additional config map", func() {
			configMapName := "my-custom-config"

			// Pre-condition: Verify PowerMonitor doesn't exist
			f.ExpectNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{})

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
			ds := &appsv1.DaemonSet{}
			f.ExpectNoResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, ds)

			// Verify reconcillation fails without config map
			pm = f.ExpectPowerMonitorCondition("power-monitor", v1alpha1.Reconciled, v1alpha1.ConditionFalse)
			reconciled, err := k8s.FindCondition(pm.Status.Conditions, v1alpha1.Reconciled)
			Expect(err).NotTo(HaveOccurred(), "unable to get reconciled condition")
			Expect(reconciled.Message).To(ContainSubstring(fmt.Sprintf("configMap %s not found in %s namespace", configMapName, controller.PowerMonitorDeploymentNS)))

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
			Expect(err).NotTo(HaveOccurred())

			// Verify reconcillation succeeds
			pm = f.ExpectPowerMonitorCondition("power-monitor", v1alpha1.Reconciled, v1alpha1.ConditionTrue)

			// Verify Daemonset exists
			f.ExpectResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, ds)

			// Verify merged config map
			mainConfigMap := &corev1.ConfigMap{}
			f.ExpectResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, mainConfigMap)
			configData := mainConfigMap.Data["config.yaml"]
			Expect(configData).To(ContainSubstring("format: json"), "custom log format should be merged")
			Expect(configData).To(ContainSubstring("sysfs: /host/sys"), "default sysfs path should be present")
			Expect(configData).To(ContainSubstring("procfs: /host/proc"), "default procfs path should be present")

			// Verify Daemonset annotation
			Expect(ds.Spec.Template.Annotations).To(HaveKey("powermonitor.sustainable.computing.io/config-map-hash-" + pm.Name))
			og := ds.Status.ObservedGeneration
			Expect(og).To(Equal(int64(1)))

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
			Expect(err).NotTo(HaveOccurred())

			// Wait for Daemonset restart
			newDs := &appsv1.DaemonSet{}
			Eventually(func(g Gomega) {
				err := f.Client().Get(context.Background(),
					client.ObjectKey{Namespace: controller.PowerMonitorDeploymentNS, Name: pm.Name}, newDs)
				if errors.IsNotFound(err) {
					g.Expect(err).NotTo(HaveOccurred())
					return
				}
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(newDs.Status.ObservedGeneration).To(Equal(og + 1))
			}).Should(Succeed())

			// Verify updated config
			f.ExpectResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, mainConfigMap)
			updatedConfigData := mainConfigMap.Data["config.yaml"]
			Expect(updatedConfigData).To(ContainSubstring("format: text"), "updated log format should be merged")
			Expect(updatedConfigData).To(ContainSubstring("level: info"), "config set inside spec should have precedence over config set in configmap")

			// Verify availability
			pm = f.ExpectPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
			available, err := k8s.FindCondition(pm.Status.Conditions, v1alpha1.Available)
			Expect(err).NotTo(HaveOccurred(), "unable to get available condition")
			Expect(available.ObservedGeneration).To(Equal(pm.Generation))
			Expect(available.Status).To(Equal(v1alpha1.ConditionTrue))
		})
	})

	Describe("RBAC Reconciliation", func() {
		It("should reconcile with RBAC security mode", func() {
			// pre-condition
			f.ExpectNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{})

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
			f.ExpectResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
			ds := &appsv1.DaemonSet{}
			f.ExpectResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, ds)

			retrievedPm := f.ExpectPowerMonitorCondition("power-monitor", v1alpha1.Reconciled, v1alpha1.ConditionTrue)
			// default toleration
			Expect(retrievedPm.Spec.Kepler.Deployment.Tolerations).To(Equal([]corev1.Toleration{{Operator: "Exists"}}))
			reconciled, err := k8s.FindCondition(retrievedPm.Status.Conditions, v1alpha1.Reconciled)
			Expect(err).NotTo(HaveOccurred(), "unable to get reconciled condition")
			Expect(reconciled.ObservedGeneration).To(Equal(retrievedPm.Generation))
			Expect(reconciled.Status).To(Equal(v1alpha1.ConditionTrue))

			retrievedPm = f.ExpectPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)
			available, err := k8s.FindCondition(retrievedPm.Status.Conditions, v1alpha1.Available)
			Expect(err).NotTo(HaveOccurred(), "unable to get available condition")
			Expect(available.ObservedGeneration).To(Equal(retrievedPm.Generation))
			Expect(available.Status).To(Equal(v1alpha1.ConditionTrue))

			audience := fmt.Sprintf("%s.%s.svc", pm.Name, controller.PowerMonitorDeploymentNS)
			serviceURL := fmt.Sprintf(
				"https://%s.%s.svc:%d/metrics",
				pm.Name,
				controller.PowerMonitorDeploymentNS,
				powermonitor.SecurePort,
			)

			// wait for relevant secrets to be created
			tlsSecret := &corev1.Secret{}
			f.ExpectResourceExists(
				tlsCertSecretName,
				controller.PowerMonitorDeploymentNS,
				tlsSecret,
				utils.Timeout(5*time.Minute),
			)
			Expect(tlsSecret.Data["tls.crt"]).NotTo(BeEmpty(), "TLS cert should be present")
			Expect(tlsSecret.Data["tls.key"]).NotTo(BeEmpty(), "TLS key should be present")

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
			Expect(jobLogs).To(ContainSubstring("HTTP/2 200"),
				"expected %s to successfully access (200) the secure endpoint", successfulJobName)

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
			Expect(jobLogs).To(ContainSubstring("HTTP/2 403"),
				"expected %s to receive forbidden error (403) when accessing secure endpoint", failedJobname)
			f.DeletePowerMonitor("power-monitor")
			f.ExpectNoResourceExists(controller.PowerMonitorDeploymentNS, "", &corev1.Namespace{})
			f.ExpectNoResourceExists(ds.Name, ds.Namespace, ds)
		})
	})

	Describe("NewConfigFields", func() {
		It("should apply custom config fields", func() {
			// Pre-condition: Verify PowerMonitor doesn't exist
			f.ExpectNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{})

			// Create PowerMonitor with all new config fields
			f.CreateTestPowerMonitor("power-monitor", runningOnVM,
				f.WithMaxTerminated(250), // Custom MaxTerminated
				f.WithStaleness("1s"),    // Custom Staleness (1 second)
				f.WithSampleRate("10s"))  // Custom SampleRate (10 seconds)

			// Wait until PowerMonitor is available
			pm := f.ExpectPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)

			// Verify DaemonSet exists
			ds := &appsv1.DaemonSet{}
			f.ExpectResourceExists(
				pm.Name,
				controller.PowerMonitorDeploymentNS,
				ds,
				utils.Timeout(10*time.Second),
			)

			// Verify the spec values are properly set
			Expect(pm.Spec.Kepler.Config.MaxTerminated).NotTo(BeNil(), "MaxTerminated should be set")
			Expect(*pm.Spec.Kepler.Config.MaxTerminated).To(Equal(int32(250)), "MaxTerminated should be 250")

			Expect(pm.Spec.Kepler.Config.Staleness).NotTo(BeNil(), "Staleness should be set")
			Expect(pm.Spec.Kepler.Config.Staleness.Duration).To(Equal(1*time.Second), "Staleness should be 1 second")

			Expect(pm.Spec.Kepler.Config.SampleRate).NotTo(BeNil(), "SampleRate should be set")
			Expect(pm.Spec.Kepler.Config.SampleRate.Duration).To(Equal(10*time.Second), "SampleRate should be 10 seconds")

			// Verify the configuration is applied in the ConfigMap
			mainConfigMap := &corev1.ConfigMap{}
			f.ExpectResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, mainConfigMap)
			configData := mainConfigMap.Data["config.yaml"]

			// Check that our custom values are present in the generated config
			Expect(configData).To(ContainSubstring("maxTerminated: 250"), "Config should contain custom MaxTerminated value")
			Expect(configData).To(ContainSubstring("staleness: 1s"), "Config should contain custom Staleness value")
			Expect(configData).To(ContainSubstring("interval: 10s"), "Config should contain custom SampleRate value")

			// Verify standard default values are still present
			Expect(configData).To(ContainSubstring("sysfs: /host/sys"), "Default sysfs path should be present")
			Expect(configData).To(ContainSubstring("procfs: /host/proc"), "Default procfs path should be present")
		})

		It("should handle zero value config fields", func() {
			// Pre-condition: Verify PowerMonitor doesn't exist
			f.ExpectNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{})

			// Create PowerMonitor with zero values for new config fields
			f.CreateTestPowerMonitor("power-monitor", runningOnVM,
				f.WithMaxTerminated(0), // Zero MaxTerminated (disabled)
				f.WithStaleness("0s"),  // Zero Staleness
				f.WithSampleRate("0s")) // Zero SampleRate

			// Wait until PowerMonitor is available
			pm := f.ExpectPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)

			// Verify DaemonSet exists
			ds := &appsv1.DaemonSet{}
			f.ExpectResourceExists(
				pm.Name,
				controller.PowerMonitorDeploymentNS,
				ds,
				utils.Timeout(10*time.Second),
			)

			// Verify the spec values are properly set to zero
			Expect(pm.Spec.Kepler.Config.MaxTerminated).NotTo(BeNil(), "MaxTerminated should be set")
			Expect(*pm.Spec.Kepler.Config.MaxTerminated).To(Equal(int32(0)), "MaxTerminated should be 0")

			Expect(pm.Spec.Kepler.Config.Staleness).NotTo(BeNil(), "Staleness should be set")
			Expect(pm.Spec.Kepler.Config.Staleness.Duration).To(Equal(0*time.Second), "Staleness should be 0")

			Expect(pm.Spec.Kepler.Config.SampleRate).NotTo(BeNil(), "SampleRate should be set")
			Expect(pm.Spec.Kepler.Config.SampleRate.Duration).To(Equal(0*time.Second), "SampleRate should be 0")

			// Verify the configuration is applied in the ConfigMap
			mainConfigMap := &corev1.ConfigMap{}
			f.ExpectResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, mainConfigMap)
			configData := mainConfigMap.Data["config.yaml"]

			// Check that our zero values are present in the generated config
			Expect(configData).To(ContainSubstring("maxTerminated: 0"), "Config should contain zero MaxTerminated value")
			Expect(configData).To(ContainSubstring("staleness: 0s"), "Config should contain zero Staleness value")
			Expect(configData).To(ContainSubstring("interval: 0s"), "Config should contain zero SampleRate value")
		})
	})

	Describe("Security", func() {
		It("should have correct pod and container security context", func() {
			f.ExpectNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{})

			f.CreateTestPowerMonitor("power-monitor", runningOnVM)
			pm := f.ExpectPowerMonitorCondition("power-monitor", v1alpha1.Available, v1alpha1.ConditionTrue)

			ds := &appsv1.DaemonSet{}
			f.ExpectResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, ds)

			// Verify pod-level security: hostPID should be enabled for process monitoring
			Expect(ds.Spec.Template.Spec.HostPID).To(BeTrue(), "Pod should have hostPID enabled")

			// Verify Kepler container security context
			containers := ds.Spec.Template.Spec.Containers
			keplerCntr, err := f.ContainerWithName(containers, pm.Name)
			Expect(err).NotTo(HaveOccurred(), "Should find the Kepler container")

			secCtx := keplerCntr.SecurityContext
			Expect(secCtx).NotTo(BeNil(), "Container should have security context")
			Expect(secCtx.Capabilities).NotTo(BeNil(), "Container should have capabilities set")

			// Verify capabilities: drop ALL, add only SYS_PTRACE
			Expect(secCtx.Capabilities.Drop).To(ContainElement(corev1.Capability("ALL")),
				"Container should drop ALL capabilities")
			Expect(secCtx.Capabilities.Add).To(Equal([]corev1.Capability{"SYS_PTRACE"}),
				"Container should only add SYS_PTRACE capability")
		})
	})

	Describe("Secrets Lifecycle CRUD", func() {
		It("should handle secret lifecycle correctly", func() {
			name := "power-monitor"
			testNs := controller.PowerMonitorDeploymentNS
			secretName := "lifecycle-test-secret"

			// Pre-condition: Verify PowerMonitor doesn't exist
			f.ExpectNoResourceExists(name, "", &v1alpha1.PowerMonitor{})

			// Step 1: Create PowerMonitor without any secrets initially
			pm := f.CreateTestPowerMonitor(name, runningOnVM)

			// Wait for PowerMonitor to be reconciled successfully (no secrets = no issues)
			pm = f.ExpectPowerMonitorCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)
			pm = f.ExpectPowerMonitorCondition(name, v1alpha1.Available, v1alpha1.ConditionTrue)

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
				Expect(err).NotTo(HaveOccurred(), "Should be able to update PM with secret reference")
			}

			// Step 3: Wait for PowerMonitor to reach degraded state due to missing secret
			pm = f.ExpectPowerMonitorCondition(name, v1alpha1.Available, v1alpha1.ConditionDegraded, utils.Timeout(2*time.Minute))

			// Assert that the degraded condition is specifically due to missing secret
			availableCondition, err := k8s.FindCondition(pm.Status.Conditions, v1alpha1.Available)
			Expect(err).NotTo(HaveOccurred(), "Should find Available condition")
			Expect(availableCondition.Reason).To(Equal(v1alpha1.SecretNotFound), "PowerMonitor should be degraded due to missing secret")
			Expect(availableCondition.Message).To(ContainSubstring(secretName), "Error message should mention the missing secret")
			Expect(availableCondition.Message).To(ContainSubstring(testNs), "Error message should mention the namespace")
			GinkgoWriter.Printf("PowerMonitor correctly degraded due to missing secret: %s\n", availableCondition.Message)

			// Step 4: Create the missing secret
			f.CreateTestSecret(secretName, testNs, map[string]string{
				"redfish.yaml": "database: lifecycle-test\\nmode: testing",
				"app.conf":     "lifecycle=enabled\\ntesting=true",
			})

			// Wait for PowerMonitor to recover and become available
			pm = f.ExpectPowerMonitorCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)
			pm = f.ExpectPowerMonitorCondition(name, v1alpha1.Available, v1alpha1.ConditionTrue)

			// Verify the secret is now properly mounted in the underlying DaemonSet
			ds := &appsv1.DaemonSet{}
			f.ExpectResourceExists(pm.Name, testNs, ds)

			// Verify secret volume exists
			secretVolumes := 0
			for _, vol := range ds.Spec.Template.Spec.Volumes {
				if vol.Secret != nil && vol.Secret.SecretName == secretName {
					secretVolumes++
					Expect(vol.Name).To(Equal(secretName), "Volume name should match secret name")
				}
			}
			Expect(secretVolumes).To(Equal(1), "Should have exactly 1 secret volume")

			// Verify secret volume mount in container
			containers := ds.Spec.Template.Spec.Containers
			keplerCntr, err := f.ContainerWithName(containers, pm.Name)
			Expect(err).NotTo(HaveOccurred(), "Should find the kepler container")

			secretMounts := 0
			for _, mount := range keplerCntr.VolumeMounts {
				if mount.Name == secretName {
					secretMounts++
					Expect(mount.MountPath).To(Equal("/etc/kepler/lifecycle-config"), "Mount path should match specification")
					Expect(mount.ReadOnly).To(BeTrue(), "Secret should be mounted read-only")
				}
			}
			Expect(secretMounts).To(Equal(1), "Should have exactly 1 secret mount")

			// Assert that the Available condition is no longer showing SecretNotFound
			recoveredCondition, err := k8s.FindCondition(pm.Status.Conditions, v1alpha1.Available)
			Expect(err).NotTo(HaveOccurred(), "Should find Available condition")
			Expect(recoveredCondition.Reason).NotTo(Equal(v1alpha1.SecretNotFound),
				"Available condition should no longer have SecretNotFound reason after secret creation")
			Expect(recoveredCondition.Status).To(Equal(v1alpha1.ConditionTrue), "PowerMonitor should be available")
			GinkgoWriter.Printf("PowerMonitor successfully recovered: status=%s, reason=%s\n", recoveredCondition.Status, recoveredCondition.Reason)

			// Step 5: Delete the secret to trigger degradation again
			f.DeleteTestSecret(secretName, testNs)

			// Wait for PowerMonitor to become degraded again due to missing secret
			pm = f.ExpectPowerMonitorCondition(name, v1alpha1.Available, v1alpha1.ConditionDegraded, utils.Timeout(2*time.Minute))

			// Assert that the degraded condition is again due to missing secret
			degradedAgainCondition, err := k8s.FindCondition(pm.Status.Conditions, v1alpha1.Available)
			Expect(err).NotTo(HaveOccurred(), "Should find Available condition")
			Expect(degradedAgainCondition.Reason).To(Equal(v1alpha1.SecretNotFound), "PowerMonitor should be degraded again due to missing secret")
			Expect(degradedAgainCondition.Message).To(ContainSubstring(secretName), "Error message should mention the missing secret")
			GinkgoWriter.Printf("PowerMonitor correctly degraded again after secret deletion: %s\n", degradedAgainCondition.Message)

			// Step 6: Remove the secret reference from PM spec using DeepCopy
			pm = f.GetPowerMonitor(name)
			{
				GinkgoWriter.Println("Remove the secret reference from PM")
				removePatchPm := pm.DeepCopy()
				removePatchPm.Spec.Kepler.Deployment.Secrets = []v1alpha1.SecretRef{} // Empty slice to remove secrets
				err = f.Patch(removePatchPm)
				Expect(err).NotTo(HaveOccurred(), "Should be able to update PM to remove secret reference")
			}

			// Wait for PowerMonitor to become available again (no secret requirements)
			pm = f.ExpectPowerMonitorCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)
			pm = f.ExpectPowerMonitorCondition(name, v1alpha1.Available, v1alpha1.ConditionTrue)

			// Assert that the Available condition is healthy again
			finalCondition, err := k8s.FindCondition(pm.Status.Conditions, v1alpha1.Available)
			Expect(err).NotTo(HaveOccurred(), "Should find Available condition")
			Expect(finalCondition.Reason).NotTo(Equal(v1alpha1.SecretNotFound),
				"Available condition should no longer have SecretNotFound reason after removing secret reference")
			Expect(finalCondition.Status).To(Equal(v1alpha1.ConditionTrue), "PowerMonitor should be available")
			GinkgoWriter.Printf("PowerMonitor final recovery after removing secret reference: status=%s, reason=%s\n", finalCondition.Status, finalCondition.Reason)

			// Verify that the secret volume is no longer present in the DaemonSet
			finalDs := &appsv1.DaemonSet{}
			f.ExpectResourceExists(pm.Name, testNs, finalDs)

			secretVolumesAfterRemoval := 0
			for _, vol := range finalDs.Spec.Template.Spec.Volumes {
				if vol.Secret != nil && vol.Secret.SecretName == secretName {
					secretVolumesAfterRemoval++
				}
			}
			Expect(secretVolumesAfterRemoval).To(Equal(0), "Should have no secret volumes after removing secret reference")

			// Verify that the secret volume mount is no longer present in the container
			finalContainers := finalDs.Spec.Template.Spec.Containers
			finalKeplerCntr, err := f.ContainerWithName(finalContainers, pm.Name)
			Expect(err).NotTo(HaveOccurred(), "Should find the kepler container")

			secretMountsAfterRemoval := 0
			for _, mount := range finalKeplerCntr.VolumeMounts {
				if mount.Name == secretName {
					secretMountsAfterRemoval++
				}
			}
			Expect(secretMountsAfterRemoval).To(Equal(0), "Should have no secret mounts after removing secret reference")

			GinkgoWriter.Println("Lifecycle test completed successfully: PM went through all expected state transitions")
		})
	})
})
