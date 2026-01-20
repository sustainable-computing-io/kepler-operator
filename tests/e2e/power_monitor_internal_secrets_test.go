// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/internal/controller"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"github.com/sustainable.computing.io/kepler-operator/tests/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

var _ = Describe("PowerMonitorInternal Secrets", func() {
	var f *utils.Framework

	BeforeEach(func() {
		f = utils.NewFramework()
	})

	Describe("Single Secret Mount", func() {
		It("should handle missing secret then recover when created", func() {
			name := "e2e-pmi-single-secret"
			testNs := controller.PowerMonitorDeploymentNS
			secretName := "test-secret-1"

			// Pre-condition: Verify PowerMonitorInternal doesn't exist
			f.ExpectNoResourceExists(name, "", &v1alpha1.PowerMonitorInternal{}, utils.ShortWait())

			// Define secret reference for a secret that doesn't exist yet
			secretRef := v1alpha1.SecretRef{
				Name:      secretName,
				MountPath: "/etc/kepler/subdir",
				ReadOnly:  ptr.To(true),
			}

			// Create PowerMonitorInternal with reference to non-existent secret
			pmi := f.CreateTestPowerMonitorInternal(name, testNs, runningOnVM, testKeplerImage, testKubeRbacProxyImage, Cluster, v1alpha1.SecurityModeNone, []string{},
				utils.PowerMonitorInternalBuilder{}.WithSecrets([]v1alpha1.SecretRef{secretRef}),
			)

			// Wait for namespace to be created (reconciliation should proceed despite missing secret)
			f.WaitForNamespace(testNs, utils.Timeout(30*time.Second))
			ds := &appsv1.DaemonSet{}
			f.ExpectResourceExists(pmi.DaemonsetName(), testNs, ds, utils.Timeout(1*time.Minute))

			// Wait for PowerMonitorInternal to reach degraded state due to missing secret
			pmi = f.ExpectPowerMonitorInternalCondition(name, v1alpha1.Available, v1alpha1.ConditionDegraded, utils.Timeout(5*time.Second))

			// Assert that the degraded condition is specifically due to missing secret
			availableCondition, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Available)
			Expect(err).NotTo(HaveOccurred(), "Should find Available condition")
			Expect(availableCondition.Reason).To(Equal(v1alpha1.SecretNotFound), "PowerMonitorInternal should be degraded due to missing secret")
			Expect(availableCondition.Message).To(ContainSubstring(secretName), "Error message should mention the missing secret")
			Expect(availableCondition.Message).To(ContainSubstring(testNs), "Error message should mention the namespace")
			GinkgoWriter.Printf("PowerMonitorInternal is correctly marked as degraded due to missing secret: %s\n", availableCondition.Message)

			// Verify namespace was created despite missing secret
			ns := &corev1.Namespace{}
			f.ExpectResourceExists(testNs, "", ns)

			// Now create the missing secret
			f.CreateTestSecret(secretName, testNs, map[string]string{
				"config.yaml": "database: mysql\nuser: kepler",
				"token.txt":   "secret-token-123",
			})

			// Wait for PowerMonitorInternal to be reconciled successfully after secret creation
			pmi = f.ExpectPowerMonitorInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)

			// Verify secret volume is added
			secretVolumes := 0
			for _, vol := range ds.Spec.Template.Spec.Volumes {
				if vol.Secret != nil && vol.Secret.SecretName == secretName {
					secretVolumes++
					Expect(vol.Name).To(Equal(secretName), "Volume name should match secret name")
				}
			}
			Expect(secretVolumes).To(Equal(1), "Should have exactly 1 secret volume")

			// Verify secret volume mount in main container
			containers := ds.Spec.Template.Spec.Containers
			Expect(len(containers)).To(BeNumerically(">=", 1), "Should have at least 1 container")

			keplerCntr, err := f.ContainerWithName(containers, pmi.DaemonsetName())
			Expect(err).NotTo(HaveOccurred(), "Should find the kepler container")
			secretMounts := 0
			for _, mount := range keplerCntr.VolumeMounts {
				if mount.Name == secretName {
					secretMounts++
					Expect(mount.MountPath).To(Equal("/etc/kepler/subdir"), "Mount path should match specification")
					Expect(mount.ReadOnly).To(BeTrue(), "Secret should be mounted read-only")
				}
			}
			Expect(secretMounts).To(Equal(1), "Should have exactly 1 secret mount")

			// Verify PowerMonitorInternal final status is healthy (no longer degraded)
			f.ExpectPowerMonitorInternalStatus(pmi.Name, utils.Timeout(2*time.Minute))

			// Assert that the Available condition is no longer showing SecretNotFound
			pmi = &v1alpha1.PowerMonitorInternal{}
			f.ExpectResourceExists(name, "", pmi)
			finalCondition, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Available)
			Expect(err).NotTo(HaveOccurred(), "Should find Available condition")
			Expect(finalCondition.Reason).NotTo(Equal(v1alpha1.SecretNotFound),
				"Available condition should no longer have SecretNotFound reason after secret creation")
			GinkgoWriter.Printf("Final PowerMonitorInternal condition: status=%s, reason=%s\n",
				finalCondition.Status, finalCondition.Reason)
		})
	})

	Describe("Multiple Secrets Mount", func() {
		It("should mount multiple secrets correctly", func() {
			name := "e2e-pmi-multiple-secrets"
			testNs := controller.PowerMonitorDeploymentNS

			// Pre-condition: Verify PowerMonitorInternal doesn't exist
			f.ExpectNoResourceExists(name, "", &v1alpha1.PowerMonitorInternal{})

			// Define multiple test secrets to be created after PMI
			secret1Name := "test-config-secret"
			secret2Name := "test-credentials-secret"
			secret3Name := "test-token-secret"

			// Define secret references with different readOnly settings
			secretRefs := []v1alpha1.SecretRef{{
				Name:      secret1Name,
				MountPath: "/etc/kepler/db",
				ReadOnly:  ptr.To(true),
			}, {
				Name:      secret2Name,
				MountPath: "/etc/kepler/credentials",
				ReadOnly:  ptr.To(true),
			}, {
				Name:      secret3Name,
				MountPath: "/etc/kepler/tokens",
				ReadOnly:  ptr.To(false),
			}}

			// Create PowerMonitorInternal with multiple secrets
			pmi := f.CreateTestPowerMonitorInternal(name, testNs, runningOnVM, testKeplerImage, testKubeRbacProxyImage, Cluster, v1alpha1.SecurityModeNone, []string{},
				utils.PowerMonitorInternalBuilder{}.WithSecrets(secretRefs),
			)

			// Wait for namespace to be created by the controller
			f.WaitForNamespace(testNs, utils.Timeout(2*time.Minute))

			// Now create the test secrets after namespace exists
			f.CreateTestSecret(secret1Name, testNs, map[string]string{
				"config.yaml": "database: postgresql\nhost: localhost",
			})
			f.CreateTestSecret(secret2Name, testNs, map[string]string{
				"credentials.json": `{"username": "kepler", "password": "secret123"}`,
			})
			f.CreateTestSecret(secret3Name, testNs, map[string]string{
				"token.txt": "bearer-token-xyz789",
			})

			// Wait for PowerMonitorInternal to be reconciled
			pmi = f.ExpectPowerMonitorInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)

			// Verify daemonset exists
			ds := &appsv1.DaemonSet{}
			f.ExpectResourceExists(pmi.Name, testNs, ds)

			// Verify all secret volumes are added
			expectedSecrets := map[string]bool{
				secret1Name: false,
				secret2Name: false,
				secret3Name: false,
			}

			secretVolumes := 0
			for _, vol := range ds.Spec.Template.Spec.Volumes {
				if vol.Secret != nil {
					if _, exists := expectedSecrets[vol.Secret.SecretName]; exists {
						secretVolumes++
						expectedSecrets[vol.Secret.SecretName] = true
						Expect(vol.Name).To(Equal(vol.Secret.SecretName), "Volume name should match secret name")
					}
				}
			}
			Expect(secretVolumes).To(Equal(3), "Should have exactly 3 secret volumes")

			// Verify all secrets were found
			for secretName, found := range expectedSecrets {
				Expect(found).To(BeTrue(), "Secret volume %s should be present", secretName)
			}

			// Verify secret volume mounts in main container
			containers := ds.Spec.Template.Spec.Containers
			Expect(len(containers)).To(BeNumerically(">=", 1), "Should have at least 1 container")

			keplerContainer, err := f.ContainerWithName(containers, pmi.DaemonsetName())
			Expect(err).NotTo(HaveOccurred(), "Should find the main Kepler container")
			expectedMounts := map[string]struct {
				mountPath string
				readOnly  bool
			}{
				secret1Name: {"/etc/kepler/db", true},
				secret2Name: {"/etc/kepler/credentials", true},
				secret3Name: {"/etc/kepler/tokens", false},
			}

			secretMounts := 0
			for _, mount := range keplerContainer.VolumeMounts {
				if expectedMount, exists := expectedMounts[mount.Name]; exists {
					secretMounts++
					Expect(mount.MountPath).To(Equal(expectedMount.mountPath), "Mount path should match specification for %s", mount.Name)
					Expect(mount.ReadOnly).To(Equal(expectedMount.readOnly), "ReadOnly setting should match specification for %s", mount.Name)
				}
			}
			Expect(secretMounts).To(Equal(3), "Should have exactly 3 secret mounts")

			// Verify PowerMonitorInternal status
			f.ExpectPowerMonitorInternalStatus(pmi.Name, utils.Timeout(5*time.Minute))
		})
	})

	Describe("Validation Failure", func() {
		It("should mark as degraded when secret does not exist", func() {
			name := "e2e-pmi-secret-validation"
			testNs := controller.PowerMonitorDeploymentNS
			nonExistentSecretName := "non-existent-secret"

			// Pre-condition: Verify PowerMonitorInternal doesn't exist
			f.ExpectNoResourceExists(name, "", &v1alpha1.PowerMonitorInternal{})

			// Define reference to non-existent secret
			secretRef := v1alpha1.SecretRef{
				Name:      nonExistentSecretName,
				MountPath: "/etc/kepler/missing",
				ReadOnly:  ptr.To(true),
			}

			// Create PowerMonitorInternal with reference to non-existent secret
			pmi := f.CreateTestPowerMonitorInternal(name, testNs, runningOnVM, testKeplerImage, testKubeRbacProxyImage, Cluster, v1alpha1.SecurityModeNone, []string{},
				utils.PowerMonitorInternalBuilder{}.WithSecrets([]v1alpha1.SecretRef{secretRef}),
			)

			// Wait for namespace to be created (reconciliation should proceed despite missing secret)
			f.WaitForNamespace(testNs, utils.Timeout(2*time.Minute))

			// Wait for PowerMonitorInternal to reach degraded state due to missing secret
			pmi = f.ExpectPowerMonitorInternalCondition(name, v1alpha1.Available, v1alpha1.ConditionDegraded, utils.Timeout(3*time.Minute))

			// Assert that the degraded condition is specifically due to missing secret
			availableCondition, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Available)
			Expect(err).NotTo(HaveOccurred(), "Should find Available condition")
			Expect(availableCondition.Reason).To(Equal(v1alpha1.SecretNotFound), "PowerMonitorInternal should be degraded due to missing secret")
			Expect(availableCondition.Message).To(ContainSubstring(nonExistentSecretName), "Error message should mention the missing secret")
			Expect(availableCondition.Message).To(ContainSubstring(testNs), "Error message should mention the namespace")
			GinkgoWriter.Printf("PowerMonitorInternal is correctly marked as degraded due to missing secret: %s\n", availableCondition.Message)

			// Verify that despite the missing secret, some resources are still created (namespace, etc.)
			// The reconciliation should continue but mark the resource as degraded
		})
	})

	Describe("Secrets Lifecycle CRUD", func() {
		It("should handle complete secret lifecycle correctly", func() {
			name := "e2e-pmi-secrets-lifecycle"
			testNs := controller.PowerMonitorDeploymentNS
			secretName := "lifecycle-test-secret"

			// Pre-condition: Verify PowerMonitorInternal doesn't exist
			f.ExpectNoResourceExists(name, "", &v1alpha1.PowerMonitorInternal{})

			// Step 1: Create PowerMonitorInternal without any secrets initially
			pmi := f.CreateTestPowerMonitorInternal(name, testNs, runningOnVM, testKeplerImage, testKubeRbacProxyImage, Cluster, v1alpha1.SecurityModeNone, []string{})

			// Wait for namespace to be created
			f.WaitForNamespace(testNs, utils.Timeout(2*time.Minute))

			// Wait for PowerMonitorInternal to be reconciled successfully (no secrets = no issues)
			pmi = f.ExpectPowerMonitorInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)
			pmi = f.ExpectPowerMonitorInternalCondition(name, v1alpha1.Available, v1alpha1.ConditionTrue)

			// Step 2: Update PowerMonitorInternal to add a secret reference (secret doesn't exist yet)
			secretRef := v1alpha1.SecretRef{
				Name:      secretName,
				MountPath: "/etc/kepler/lifecycle-config",
				ReadOnly:  ptr.To(true),
			}

			// Update the PMI to include the secret reference using DeepCopy to avoid managedFields issues
			pmi = f.GetPowerMonitorInternal(name)
			{
				patchPmi := pmi.DeepCopy()
				patchPmi.Spec.Kepler.Deployment.Secrets = []v1alpha1.SecretRef{secretRef}
				err := f.Patch(patchPmi)
				Expect(err).NotTo(HaveOccurred(), "Should be able to update PMI with secret reference")
			}

			// Step 3: Wait for PowerMonitorInternal to reach degraded state due to missing secret
			pmi = f.ExpectPowerMonitorInternalCondition(name, v1alpha1.Available, v1alpha1.ConditionDegraded, utils.Timeout(2*time.Minute))

			// Assert that the degraded condition is specifically due to missing secret
			availableCondition, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Available)
			Expect(err).NotTo(HaveOccurred(), "Should find Available condition")
			Expect(availableCondition.Reason).To(Equal(v1alpha1.SecretNotFound), "PowerMonitorInternal should be degraded due to missing secret")
			Expect(availableCondition.Message).To(ContainSubstring(secretName), "Error message should mention the missing secret")
			Expect(availableCondition.Message).To(ContainSubstring(testNs), "Error message should mention the namespace")
			GinkgoWriter.Printf("PowerMonitorInternal correctly degraded due to missing secret: %s\n", availableCondition.Message)

			// Step 4: Create the missing secret
			f.CreateTestSecret(secretName, testNs, map[string]string{
				"redfish.yaml": "database: lifecycle-test\nmode: testing",
				"app.conf":     "lifecycle=enabled\ntesting=true",
			})

			// Wait for PowerMonitorInternal to recover and become available
			pmi = f.ExpectPowerMonitorInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)
			pmi = f.ExpectPowerMonitorInternalCondition(name, v1alpha1.Available, v1alpha1.ConditionTrue)

			// Verify the secret is now properly mounted
			ds := &appsv1.DaemonSet{}
			f.ExpectResourceExists(pmi.Name, testNs, ds)

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
			keplerCntr, err := f.ContainerWithName(containers, pmi.DaemonsetName())
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
			recoveredCondition, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Available)
			Expect(err).NotTo(HaveOccurred(), "Should find Available condition")
			Expect(recoveredCondition.Reason).NotTo(Equal(v1alpha1.SecretNotFound),
				"Available condition should no longer have SecretNotFound reason after secret creation")
			Expect(recoveredCondition.Status).To(Equal(v1alpha1.ConditionTrue), "PowerMonitorInternal should be available")
			GinkgoWriter.Printf("PowerMonitorInternal successfully recovered: status=%s, reason=%s\n", recoveredCondition.Status, recoveredCondition.Reason)

			// Step 5: Delete the secret to trigger degradation again
			f.DeleteTestSecret(secretName, testNs)

			// Wait for PowerMonitorInternal to become degraded again due to missing secret
			pmi = f.ExpectPowerMonitorInternalCondition(name, v1alpha1.Available, v1alpha1.ConditionDegraded, utils.Timeout(2*time.Minute))

			// Assert that the degraded condition is again due to missing secret
			degradedAgainCondition, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Available)
			Expect(err).NotTo(HaveOccurred(), "Should find Available condition")
			Expect(degradedAgainCondition.Reason).To(Equal(v1alpha1.SecretNotFound), "PowerMonitorInternal should be degraded again due to missing secret")
			Expect(degradedAgainCondition.Message).To(ContainSubstring(secretName), "Error message should mention the missing secret")
			GinkgoWriter.Printf("PowerMonitorInternal correctly degraded again after secret deletion: %s\n", degradedAgainCondition.Message)

			// Step 6: Remove the secret reference from PMI spec using DeepCopy
			pmi = f.GetPowerMonitorInternal(name)
			{
				GinkgoWriter.Println("Remove the secret reference from PMI")
				removePatchPmi := pmi.DeepCopy()
				removePatchPmi.Spec.Kepler.Deployment.Secrets = []v1alpha1.SecretRef{} // Empty slice to remove secrets
				err = f.Patch(removePatchPmi)
				Expect(err).NotTo(HaveOccurred(), "Should be able to update PMI to remove secret reference")
			}

			// Wait for PowerMonitorInternal to become available again (no secret requirements)
			pmi = f.ExpectPowerMonitorInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)
			pmi = f.ExpectPowerMonitorInternalCondition(name, v1alpha1.Available, v1alpha1.ConditionTrue)

			// Assert that the Available condition is healthy again
			finalCondition, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Available)
			Expect(err).NotTo(HaveOccurred(), "Should find Available condition")
			Expect(finalCondition.Reason).NotTo(Equal(v1alpha1.SecretNotFound),
				"Available condition should no longer have SecretNotFound reason after removing secret reference")
			Expect(finalCondition.Status).To(Equal(v1alpha1.ConditionTrue), "PowerMonitorInternal should be available")
			GinkgoWriter.Printf("PowerMonitorInternal final recovery after removing secret reference: status=%s, reason=%s\n", finalCondition.Status, finalCondition.Reason)

			// Verify that the secret volume is no longer present in the DaemonSet
			finalDs := &appsv1.DaemonSet{}
			f.ExpectResourceExists(pmi.Name, testNs, finalDs)

			secretVolumesAfterRemoval := 0
			for _, vol := range finalDs.Spec.Template.Spec.Volumes {
				if vol.Secret != nil && vol.Secret.SecretName == secretName {
					secretVolumesAfterRemoval++
				}
			}
			Expect(secretVolumesAfterRemoval).To(Equal(0), "Should have no secret volumes after removing secret reference")

			// Verify that the secret volume mount is no longer present in the container
			finalContainers := finalDs.Spec.Template.Spec.Containers
			finalKeplerCntr, err := f.ContainerWithName(finalContainers, pmi.DaemonsetName())
			Expect(err).NotTo(HaveOccurred(), "Should find the kepler container")

			secretMountsAfterRemoval := 0
			for _, mount := range finalKeplerCntr.VolumeMounts {
				if mount.Name == secretName {
					secretMountsAfterRemoval++
				}
			}
			Expect(secretMountsAfterRemoval).To(Equal(0), "Should have no secret mounts after removing secret reference")

			GinkgoWriter.Println("Lifecycle test completed successfully: PMI went through all expected state transitions")
		})
	})

	Describe("Duplicate Mount Path", func() {
		It("should fail with duplicate mount paths", func() {
			name := "e2e-pmi-duplicate-mount"
			testNs := controller.PowerMonitorDeploymentNS

			// Pre-condition: Verify PowerMonitorInternal doesn't exist
			f.ExpectNoResourceExists(name, "", &v1alpha1.PowerMonitorInternal{})

			// Define secret reference that would cause duplicate mount path with Kepler config
			conflictingSecretName := "conflicting-mount-secret"
			conflictingSecretRef := v1alpha1.SecretRef{
				Name:      conflictingSecretName,
				MountPath: "/etc/kepler", // This conflicts with the Kepler config mount!
				ReadOnly:  ptr.To(true),
			}

			// Create PowerMonitorInternal with the conflicting secret reference
			// NOTE: runningOnVM is set to false here so that the test doesn't try to create
			// configmap which will fail due to invalid daemsonset
			pmi := f.CreateTestPowerMonitorInternal(name, testNs, false,
				testKeplerImage, testKubeRbacProxyImage,
				Cluster, v1alpha1.SecurityModeNone, []string{},
				utils.PowerMonitorInternalBuilder{}.WithSecrets([]v1alpha1.SecretRef{conflictingSecretRef}),
			)

			// Wait for namespace to be created
			f.WaitForNamespace(testNs, utils.Timeout(1*time.Minute))
			f.CreateTestSecret(conflictingSecretName, testNs, map[string]string{
				"config.yaml": "broken-yaml { file",
			})

			// Wait for PowerMonitorInternal to reach degraded state due to DaemonSet creation failure
			pmi = f.ExpectPowerMonitorInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionFalse, utils.Timeout(30*time.Second))
			f.ExpectNoResourceExists(pmi.DaemonsetName(), testNs, &appsv1.DaemonSet{}, utils.ShortWait())

			// Assert that the degraded condition mentions the duplicate mount path issue
			reconciledCondition, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Reconciled)
			Expect(err).NotTo(HaveOccurred(), "Should find Reconciled condition")
			// The error should mention duplicate mount paths (Kubernetes validation error)
			GinkgoWriter.Printf("reconciledCondition message: %s\n", reconciledCondition.Message)
			Expect(reconciledCondition.Message).To(ContainSubstring("duplicate entries for key"),
				"Error message should mention duplicate mount path validation failure")

			availableCondition, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Available)
			Expect(err).NotTo(HaveOccurred(), "Should find Available condition")
			Expect(availableCondition.Status).To(Equal(v1alpha1.ConditionFalse), "Available condition should be false")
			GinkgoWriter.Printf("availableCondition message: %s\n", availableCondition.Message)
			Expect(availableCondition.Message).To(ContainSubstring(`DaemonSet.apps "e2e-pmi-duplicate-mount" not found`),
				"Available condition should mention DaemonSet not found")

			GinkgoWriter.Printf("PowerMonitorInternal correctly failed due to duplicate mount path: %s\n", reconciledCondition.Message)
		})
	})

	Describe("Auto Redeploy On Secret Change", func() {
		It("should redeploy when secret content changes", func() {
			name := "e2e-pmi-secret-redeploy"
			testNs := controller.PowerMonitorDeploymentNS
			secretName := "changing-secret"

			// Pre-condition: Verify PowerMonitorInternal doesn't exist
			f.ExpectNoResourceExists(name, "", &v1alpha1.PowerMonitorInternal{}, utils.ShortWait())

			// Define secret reference
			secretRef := v1alpha1.SecretRef{
				Name:      secretName,
				MountPath: "/etc/kepler/configs",
				ReadOnly:  ptr.To(true),
			}

			// Create PowerMonitorInternal with secret reference
			pmi := f.CreateTestPowerMonitorInternal(name, testNs, runningOnVM, testKeplerImage, testKubeRbacProxyImage, Cluster, v1alpha1.SecurityModeNone, []string{},
				utils.PowerMonitorInternalBuilder{}.WithSecrets([]v1alpha1.SecretRef{secretRef}),
			)

			// Wait for namespace to be created
			f.WaitForNamespace(testNs, utils.Timeout(1*time.Minute))

			// Create the initial secret
			initialSecretData := map[string]string{
				"app.yaml":    "version: 1.0\nconfig: initial",
				"config.json": `{"setting": "initial", "value": 100}`,
			}
			f.CreateTestSecret(secretName, testNs, initialSecretData)

			// Wait for PowerMonitorInternal to be reconciled and available successfully
			pmi = f.ExpectPowerMonitorInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)
			pmi = f.ExpectPowerMonitorInternalCondition(name, v1alpha1.Available, v1alpha1.ConditionTrue)

			// Get the initial DaemonSet and capture its initial hash annotation
			initialDs := &appsv1.DaemonSet{}
			f.ExpectResourceExists(pmi.Name, testNs, initialDs)

			// Find the initial secret hash annotation
			var initialHashAnnotation string
			for key, value := range initialDs.Spec.Template.Annotations {
				if key == fmt.Sprintf("powermonitor.sustainable.computing.io/secret-tls-hash-%s", secretName) {
					initialHashAnnotation = value
					break
				}
			}
			Expect(initialHashAnnotation).NotTo(BeEmpty(), "Should have initial secret hash annotation")
			GinkgoWriter.Printf("Initial secret hash annotation: %s\n", initialHashAnnotation)

			// Capture initial DaemonSet generation for comparison
			initialGeneration := initialDs.Generation
			GinkgoWriter.Printf("Initial DaemonSet generation: %d\n", initialGeneration)

			// Update the secret with different content
			updatedSecretData := map[string]string{
				"app.yaml":    "version: 2.0\nconfig: updated",          // Changed content
				"config.json": `{"setting": "updated", "value": 200}`,   // Changed content
				"new.txt":     "This is a new file added to the secret", // New file
			}

			// Delete and recreate the secret to simulate secret content change
			f.DeleteTestSecret(secretName, testNs)
			f.CreateTestSecret(secretName, testNs, updatedSecretData)

			// Wait for the controller to detect the secret change and update the DaemonSet
			// This should happen within a reasonable time due to the reconciliation loop
			Eventually(func(g Gomega) {
				updatedDs := &appsv1.DaemonSet{}
				err := f.Client().Get(context.Background(), types.NamespacedName{Name: pmi.Name, Namespace: testNs}, updatedDs)
				g.Expect(err).NotTo(HaveOccurred())

				// Check if the secret hash annotation has changed
				for key, value := range updatedDs.Spec.Template.Annotations {
					if key == fmt.Sprintf("powermonitor.sustainable.computing.io/secret-tls-hash-%s", secretName) {
						g.Expect(value).NotTo(Equal(initialHashAnnotation), "Secret hash should change")
						return
					}
				}
				g.Expect(false).To(BeTrue(), "Secret hash annotation not found")
			}).WithTimeout(2 * time.Minute).Should(Succeed())

			// Get the updated DaemonSet
			updatedDs := &appsv1.DaemonSet{}
			f.ExpectResourceExists(pmi.Name, testNs, updatedDs)

			// Verify the secret hash annotation has changed
			var updatedHashAnnotation string
			for key, value := range updatedDs.Spec.Template.Annotations {
				if key == fmt.Sprintf("powermonitor.sustainable.computing.io/secret-tls-hash-%s", secretName) {
					updatedHashAnnotation = value
					break
				}
			}
			Expect(updatedHashAnnotation).NotTo(BeEmpty(), "Should have updated secret hash annotation")
			Expect(updatedHashAnnotation).NotTo(Equal(initialHashAnnotation),
				"Secret hash annotation should change when secret content changes")

			GinkgoWriter.Printf("Updated secret hash annotation: %s\n", updatedHashAnnotation)

			// Verify DaemonSet generation has incremented (indicating an update)
			Expect(updatedDs.Generation).To(BeNumerically(">", initialGeneration),
				"DaemonSet generation should increment when pod template annotations change")

			GinkgoWriter.Printf("DaemonSet generation updated from %d to %d\n", initialGeneration, updatedDs.Generation)

			// Verify the secret is still properly mounted with updated content
			containers := updatedDs.Spec.Template.Spec.Containers
			keplerContainer, err := f.ContainerWithName(containers, pmi.DaemonsetName())
			Expect(err).NotTo(HaveOccurred(), "Should find the kepler container")

			secretMountFound := false
			for _, mount := range keplerContainer.VolumeMounts {
				if mount.Name == secretName {
					secretMountFound = true
					Expect(mount.MountPath).To(Equal("/etc/kepler/configs"), "Mount path should remain unchanged")
					Expect(mount.ReadOnly).To(BeTrue(), "Secret should still be mounted read-only")
					break
				}
			}
			Expect(secretMountFound).To(BeTrue(), "Secret mount should still be present after update")

			// Verify PowerMonitorInternal remains healthy after secret change
			f.ExpectPowerMonitorInternalStatus(pmi.Name, utils.Timeout(2*time.Minute))

			GinkgoWriter.Println("Secret change test completed successfully: DaemonSet was redeployed with new secret hash")
		})
	})
})
