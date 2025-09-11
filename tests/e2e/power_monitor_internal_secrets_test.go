// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/internal/controller"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"github.com/sustainable.computing.io/kepler-operator/tests/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

func TestPowerMonitorInternal_SecretsMount_SingleSecret(t *testing.T) {
	f := utils.NewFramework(t)
	name := "e2e-pmi-single-secret"
	testNs := controller.PowerMonitorDeploymentNS
	secretName := "test-secret-1"

	// Pre-condition: Verify PowerMonitorInternal doesn't exist
	f.AssertNoResourceExists(name, "", &v1alpha1.PowerMonitorInternal{}, utils.ShortWait())

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
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pmi.DaemonsetName(), testNs, &ds, utils.Timeout(1*time.Minute))

	// Wait for PowerMonitorInternal to reach degraded state due to missing secret
	pmi = f.WaitUntilPowerMonitorInternalCondition(name, v1alpha1.Available, v1alpha1.ConditionDegraded, utils.Timeout(5*time.Second))

	// Assert that the degraded condition is specifically due to missing secret
	availableCondition, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Available)
	assert.NoError(t, err, "Should find Available condition")
	assert.Equal(t, v1alpha1.SecretNotFound, availableCondition.Reason, "PowerMonitorInternal should be degraded due to missing secret")
	assert.Contains(t, availableCondition.Message, secretName, "Error message should mention the missing secret")
	assert.Contains(t, availableCondition.Message, testNs, "Error message should mention the namespace")
	t.Logf("PowerMonitorInternal is correctly marked as degraded due to missing secret: %s", availableCondition.Message)

	// Verify namespace was created despite missing secret
	ns := corev1.Namespace{}
	f.AssertResourceExists(testNs, "", &ns)

	// Now create the missing secret
	f.CreateTestSecret(secretName, testNs, map[string]string{
		"config.yaml": "database: mysql\nuser: kepler",
		"token.txt":   "secret-token-123",
	})

	// Wait for PowerMonitorInternal to be reconciled successfully after secret creation
	pmi = f.WaitUntilPowerMonitorInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)

	// Verify secret volume is added
	secretVolumes := 0
	for _, vol := range ds.Spec.Template.Spec.Volumes {
		if vol.Secret != nil && vol.Secret.SecretName == secretName {
			secretVolumes++
			assert.Equal(t, secretName, vol.Name, "Volume name should match secret name")
		}
	}
	assert.Equal(t, 1, secretVolumes, "Should have exactly 1 secret volume")

	// Verify secret volume mount in main container
	containers := ds.Spec.Template.Spec.Containers
	assert.True(t, len(containers) >= 1, "Should have at least 1 container")

	keplerCntr, err := f.ContainerWithName(containers, pmi.DaemonsetName())
	assert.NoError(t, err, "Should find the kepler container")
	secretMounts := 0
	for _, mount := range keplerCntr.VolumeMounts {
		if mount.Name == secretName {
			secretMounts++
			assert.Equal(t, "/etc/kepler/subdir", mount.MountPath, "Mount path should match specification")
			assert.True(t, mount.ReadOnly, "Secret should be mounted read-only")
		}
	}
	assert.Equal(t, 1, secretMounts, "Should have exactly 1 secret mount")

	// Verify PowerMonitorInternal final status is healthy (no longer degraded)
	f.AssertPowerMonitorInternalStatus(pmi.Name, utils.Timeout(2*time.Minute))

	// Assert that the Available condition is no longer showing SecretNotFound
	pmi = &v1alpha1.PowerMonitorInternal{}
	f.AssertResourceExists(name, "", pmi)
	finalCondition, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Available)
	assert.NoError(t, err, "Should find Available condition")
	assert.NotEqual(t, v1alpha1.SecretNotFound, finalCondition.Reason,
		"Available condition should no longer have SecretNotFound reason after secret creation")
	t.Logf("Final PowerMonitorInternal condition: status=%s, reason=%s",
		finalCondition.Status, finalCondition.Reason)
}

func TestPowerMonitorInternal_SecretsMount_MultipleSecrets(t *testing.T) {
	f := utils.NewFramework(t)
	name := "e2e-pmi-multiple-secrets"
	testNs := controller.PowerMonitorDeploymentNS

	// Pre-condition: Verify PowerMonitorInternal doesn't exist
	f.AssertNoResourceExists(name, "", &v1alpha1.PowerMonitorInternal{})

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
	pmi = f.WaitUntilPowerMonitorInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)

	// Verify daemonset exists
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pmi.Name, testNs, &ds)

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
				assert.Equal(t, vol.Secret.SecretName, vol.Name, "Volume name should match secret name")
			}
		}
	}
	assert.Equal(t, 3, secretVolumes, "Should have exactly 3 secret volumes")

	// Verify all secrets were found
	for secretName, found := range expectedSecrets {
		assert.True(t, found, "Secret volume %s should be present", secretName)
	}

	// Verify secret volume mounts in main container
	containers := ds.Spec.Template.Spec.Containers
	assert.True(t, len(containers) >= 1, "Should have at least 1 container")

	keplerContainer, err := f.ContainerWithName(containers, pmi.DaemonsetName())
	assert.NoError(t, err, "Should find the main Kepler container")
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
			assert.Equal(t, expectedMount.mountPath, mount.MountPath, "Mount path should match specification for %s", mount.Name)
			assert.Equal(t, expectedMount.readOnly, mount.ReadOnly, "ReadOnly setting should match specification for %s", mount.Name)
		}
	}
	assert.Equal(t, 3, secretMounts, "Should have exactly 3 secret mounts")

	// Verify PowerMonitorInternal status
	f.AssertPowerMonitorInternalStatus(pmi.Name, utils.Timeout(5*time.Minute))
}

func TestPowerMonitorInternal_SecretsMount_ValidationFailure(t *testing.T) {
	f := utils.NewFramework(t)
	name := "e2e-pmi-secret-validation"
	testNs := controller.PowerMonitorDeploymentNS
	nonExistentSecretName := "non-existent-secret"

	// Pre-condition: Verify PowerMonitorInternal doesn't exist
	f.AssertNoResourceExists(name, "", &v1alpha1.PowerMonitorInternal{})

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
	pmi = f.WaitUntilPowerMonitorInternalCondition(name, v1alpha1.Available, v1alpha1.ConditionDegraded, utils.Timeout(3*time.Minute))

	// Assert that the degraded condition is specifically due to missing secret
	availableCondition, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Available)
	assert.NoError(t, err, "Should find Available condition")
	assert.Equal(t, v1alpha1.SecretNotFound, availableCondition.Reason, "PowerMonitorInternal should be degraded due to missing secret")
	assert.Contains(t, availableCondition.Message, nonExistentSecretName, "Error message should mention the missing secret")
	assert.Contains(t, availableCondition.Message, testNs, "Error message should mention the namespace")
	t.Logf("PowerMonitorInternal is correctly marked as degraded due to missing secret: %s", availableCondition.Message)

	// Verify that despite the missing secret, some resources are still created (namespace, etc.)
	// The reconciliation should continue but mark the resource as degraded
}

func TestPowerMonitorInternal_Secrets_Lifecycle_CRUD(t *testing.T) {
	f := utils.NewFramework(t)
	name := "e2e-pmi-secrets-lifecycle"
	testNs := controller.PowerMonitorDeploymentNS
	secretName := "lifecycle-test-secret"

	// Pre-condition: Verify PowerMonitorInternal doesn't exist
	f.AssertNoResourceExists(name, "", &v1alpha1.PowerMonitorInternal{})

	// Step 1: Create PowerMonitorInternal without any secrets initially
	pmi := f.CreateTestPowerMonitorInternal(name, testNs, runningOnVM, testKeplerImage, testKubeRbacProxyImage, Cluster, v1alpha1.SecurityModeNone, []string{})

	// Wait for namespace to be created
	f.WaitForNamespace(testNs, utils.Timeout(2*time.Minute))

	// Wait for PowerMonitorInternal to be reconciled successfully (no secrets = no issues)
	pmi = f.WaitUntilPowerMonitorInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)
	pmi = f.WaitUntilPowerMonitorInternalCondition(name, v1alpha1.Available, v1alpha1.ConditionTrue)

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
		assert.NoError(t, err, "Should be able to update PMI with secret reference")
	}

	// Step 3: Wait for PowerMonitorInternal to reach degraded state due to missing secret
	pmi = f.WaitUntilPowerMonitorInternalCondition(name, v1alpha1.Available, v1alpha1.ConditionDegraded, utils.Timeout(2*time.Minute))

	// Assert that the degraded condition is specifically due to missing secret
	availableCondition, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Available)
	assert.NoError(t, err, "Should find Available condition")
	assert.Equal(t, v1alpha1.SecretNotFound, availableCondition.Reason, "PowerMonitorInternal should be degraded due to missing secret")
	assert.Contains(t, availableCondition.Message, secretName, "Error message should mention the missing secret")
	assert.Contains(t, availableCondition.Message, testNs, "Error message should mention the namespace")
	t.Logf("PowerMonitorInternal correctly degraded due to missing secret: %s", availableCondition.Message)

	// Step 4: Create the missing secret
	f.CreateTestSecret(secretName, testNs, map[string]string{
		"redfish.yaml": "database: lifecycle-test\nmode: testing",
		"app.conf":     "lifecycle=enabled\ntesting=true",
	})

	// Wait for PowerMonitorInternal to recover and become available
	pmi = f.WaitUntilPowerMonitorInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)
	pmi = f.WaitUntilPowerMonitorInternalCondition(name, v1alpha1.Available, v1alpha1.ConditionTrue)

	// Verify the secret is now properly mounted
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pmi.Name, testNs, &ds)

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
	keplerCntr, err := f.ContainerWithName(containers, pmi.DaemonsetName())
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
	recoveredCondition, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Available)
	assert.NoError(t, err, "Should find Available condition")
	assert.NotEqual(t, v1alpha1.SecretNotFound, recoveredCondition.Reason,
		"Available condition should no longer have SecretNotFound reason after secret creation")
	assert.Equal(t, v1alpha1.ConditionTrue, recoveredCondition.Status, "PowerMonitorInternal should be available")
	t.Logf("PowerMonitorInternal successfully recovered: status=%s, reason=%s", recoveredCondition.Status, recoveredCondition.Reason)

	// Step 5: Delete the secret to trigger degradation again
	f.DeleteTestSecret(secretName, testNs)

	// Wait for PowerMonitorInternal to become degraded again due to missing secret
	pmi = f.WaitUntilPowerMonitorInternalCondition(name, v1alpha1.Available, v1alpha1.ConditionDegraded, utils.Timeout(2*time.Minute))

	// Assert that the degraded condition is again due to missing secret
	degradedAgainCondition, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Available)
	assert.NoError(t, err, "Should find Available condition")
	assert.Equal(t, v1alpha1.SecretNotFound, degradedAgainCondition.Reason, "PowerMonitorInternal should be degraded again due to missing secret")
	assert.Contains(t, degradedAgainCondition.Message, secretName, "Error message should mention the missing secret")
	t.Logf("PowerMonitorInternal correctly degraded again after secret deletion: %s", degradedAgainCondition.Message)

	// Step 6: Remove the secret reference from PMI spec using DeepCopy
	pmi = f.GetPowerMonitorInternal(name)
	{
		t.Logf("Remove the secret reference from PMI")
		removePatchPmi := pmi.DeepCopy()
		removePatchPmi.Spec.Kepler.Deployment.Secrets = []v1alpha1.SecretRef{} // Empty slice to remove secrets
		err = f.Patch(removePatchPmi)
		assert.NoError(t, err, "Should be able to update PMI to remove secret reference")
	}

	// Wait for PowerMonitorInternal to become available again (no secret requirements)
	pmi = f.WaitUntilPowerMonitorInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)
	pmi = f.WaitUntilPowerMonitorInternalCondition(name, v1alpha1.Available, v1alpha1.ConditionTrue)

	// Assert that the Available condition is healthy again
	finalCondition, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Available)
	assert.NoError(t, err, "Should find Available condition")
	assert.NotEqual(t, v1alpha1.SecretNotFound, finalCondition.Reason,
		"Available condition should no longer have SecretNotFound reason after removing secret reference")
	assert.Equal(t, v1alpha1.ConditionTrue, finalCondition.Status, "PowerMonitorInternal should be available")
	t.Logf("PowerMonitorInternal final recovery after removing secret reference: status=%s, reason=%s", finalCondition.Status, finalCondition.Reason)

	// Verify that the secret volume is no longer present in the DaemonSet
	finalDs := appsv1.DaemonSet{}
	f.AssertResourceExists(pmi.Name, testNs, &finalDs)

	secretVolumesAfterRemoval := 0
	for _, vol := range finalDs.Spec.Template.Spec.Volumes {
		if vol.Secret != nil && vol.Secret.SecretName == secretName {
			secretVolumesAfterRemoval++
		}
	}
	assert.Equal(t, 0, secretVolumesAfterRemoval, "Should have no secret volumes after removing secret reference")

	// Verify that the secret volume mount is no longer present in the container
	finalContainers := finalDs.Spec.Template.Spec.Containers
	finalKeplerCntr, err := f.ContainerWithName(finalContainers, pmi.DaemonsetName())
	assert.NoError(t, err, "Should find the kepler container")

	secretMountsAfterRemoval := 0
	for _, mount := range finalKeplerCntr.VolumeMounts {
		if mount.Name == secretName {
			secretMountsAfterRemoval++
		}
	}
	assert.Equal(t, 0, secretMountsAfterRemoval, "Should have no secret mounts after removing secret reference")

	t.Logf("Lifecycle test completed successfully: PMI went through all expected state transitions")
}

func TestPowerMonitorInternal_SecretsMount_DuplicateMountPath(t *testing.T) {
	f := utils.NewFramework(t)
	name := "e2e-pmi-duplicate-mount"
	testNs := controller.PowerMonitorDeploymentNS

	// Pre-condition: Verify PowerMonitorInternal doesn't exist
	f.AssertNoResourceExists(name, "", &v1alpha1.PowerMonitorInternal{})

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
	pmi = f.WaitUntilPowerMonitorInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionFalse, utils.Timeout(30*time.Second))
	f.AssertNoResourceExists(pmi.DaemonsetName(), testNs, &appsv1.DaemonSet{}, utils.ShortWait())

	// Assert that the degraded condition mentions the duplicate mount path issue
	reconciledCondition, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Reconciled)
	assert.NoError(t, err, "Should find Reconciled condition")
	// The error should mention duplicate mount paths (Kubernetes validation error)
	t.Logf("reconciledCondition  message: %s", reconciledCondition.Message)
	assert.Contains(t, reconciledCondition.Message, "duplicate entries for key",
		"Error message should mention duplicate mount path validation failure")

	availableCondition, err := k8s.FindCondition(pmi.Status.Conditions, v1alpha1.Available)
	assert.NoError(t, err, "Should find Available condition")
	assert.Equal(t, v1alpha1.ConditionFalse, availableCondition.Status, "Available condition should be false")
	t.Logf("availableCondition  message: %s", availableCondition.Message)
	assert.Contains(t, availableCondition.Message, `DaemonSet.apps "e2e-pmi-duplicate-mount" not found`,
		"Available condition should mention DaemonSet not found")

	t.Logf("PowerMonitorInternal correctly failed due to duplicate mount path: %s", reconciledCondition.Message)
}

func TestPowerMonitorInternal_SecretsMount_AutoRedeployOnSecretChange(t *testing.T) {
	f := utils.NewFramework(t)
	name := "e2e-pmi-secret-redeploy"
	testNs := controller.PowerMonitorDeploymentNS
	secretName := "changing-secret"

	// Pre-condition: Verify PowerMonitorInternal doesn't exist
	f.AssertNoResourceExists(name, "", &v1alpha1.PowerMonitorInternal{}, utils.ShortWait())

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
	pmi = f.WaitUntilPowerMonitorInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)
	pmi = f.WaitUntilPowerMonitorInternalCondition(name, v1alpha1.Available, v1alpha1.ConditionTrue)

	// Get the initial DaemonSet and capture its initial hash annotation
	initialDs := appsv1.DaemonSet{}
	f.AssertResourceExists(pmi.Name, testNs, &initialDs)

	// Find the initial secret hash annotation
	var initialHashAnnotation string
	for key, value := range initialDs.Spec.Template.Annotations {
		if key == fmt.Sprintf("powermonitor.sustainable.computing.io/secret-tls-hash-%s", secretName) {
			initialHashAnnotation = value
			break
		}
	}
	assert.NotEmpty(t, initialHashAnnotation, "Should have initial secret hash annotation")
	t.Logf("Initial secret hash annotation: %s", initialHashAnnotation)

	// Capture initial DaemonSet generation for comparison
	initialGeneration := initialDs.Generation
	t.Logf("Initial DaemonSet generation: %d", initialGeneration)

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
	f.WaitUntil("DaemonSet to be updated with new secret hash", func(ctx context.Context) (bool, error) {
		updatedDs := appsv1.DaemonSet{}
		err := f.Client().Get(ctx, types.NamespacedName{Name: pmi.Name, Namespace: testNs}, &updatedDs)
		if err != nil {
			return false, err
		}

		// Check if the secret hash annotation has changed
		for key, value := range updatedDs.Spec.Template.Annotations {
			if key == fmt.Sprintf("powermonitor.sustainable.computing.io/secret-tls-hash-%s", secretName) {
				return value != initialHashAnnotation, nil
			}
		}
		return false, nil
	}, utils.Timeout(2*time.Minute))

	// Get the updated DaemonSet
	updatedDs := appsv1.DaemonSet{}
	f.AssertResourceExists(pmi.Name, testNs, &updatedDs)

	// Verify the secret hash annotation has changed
	var updatedHashAnnotation string
	for key, value := range updatedDs.Spec.Template.Annotations {
		if key == fmt.Sprintf("powermonitor.sustainable.computing.io/secret-tls-hash-%s", secretName) {
			updatedHashAnnotation = value
			break
		}
	}
	assert.NotEmpty(t, updatedHashAnnotation, "Should have updated secret hash annotation")
	assert.NotEqual(t, initialHashAnnotation, updatedHashAnnotation,
		"Secret hash annotation should change when secret content changes")

	t.Logf("Updated secret hash annotation: %s", updatedHashAnnotation)

	// Verify DaemonSet generation has incremented (indicating an update)
	assert.Greater(t, updatedDs.Generation, initialGeneration,
		"DaemonSet generation should increment when pod template annotations change")

	t.Logf("DaemonSet generation updated from %d to %d", initialGeneration, updatedDs.Generation)

	// Verify the secret is still properly mounted with updated content
	containers := updatedDs.Spec.Template.Spec.Containers
	keplerContainer, err := f.ContainerWithName(containers, pmi.DaemonsetName())
	assert.NoError(t, err, "Should find the kepler container")

	secretMountFound := false
	for _, mount := range keplerContainer.VolumeMounts {
		if mount.Name == secretName {
			secretMountFound = true
			assert.Equal(t, "/etc/kepler/configs", mount.MountPath, "Mount path should remain unchanged")
			assert.True(t, mount.ReadOnly, "Secret should still be mounted read-only")
			break
		}
	}
	assert.True(t, secretMountFound, "Secret mount should still be present after update")

	// Verify PowerMonitorInternal remains healthy after secret change
	f.AssertPowerMonitorInternalStatus(pmi.Name, utils.Timeout(2*time.Minute))

	t.Logf("Secret change test completed successfully: DaemonSet was redeployed with new secret hash")
}
