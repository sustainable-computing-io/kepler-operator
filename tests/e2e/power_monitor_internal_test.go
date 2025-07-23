// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/internal/controller"
	powermonitor "github.com/sustainable.computing.io/kepler-operator/pkg/components/power-monitor"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"github.com/sustainable.computing.io/kepler-operator/tests/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestPowerMonitorInternal_Reconciliation(t *testing.T) {
	f := utils.NewFramework(t)
	name := "e2e-pmi"
	testNs := controller.PowerMonitorDeploymentNS

	// Pre-condition: Verify PowerMonitorInternal doesn't exist
	f.AssertNoResourceExists(name, "", &v1alpha1.PowerMonitorInternal{})

	// Create PowerMonitorInternal
	f.CreateTestPowerMonitorInternal(name, testNs, runningOnVM, testKeplerImage, testKubeRbacProxyImage, Cluster, v1alpha1.SecurityModeNone, []string{})

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
	f.AssertPowerMonitorInternalStatus(pmi.Name, utils.Timeout(5*time.Minute))
}

func TestPowerMonitorInternal_RBAC_Reconciliation(t *testing.T) {
	f := utils.NewFramework(t)
	name := "e2e-pmi"
	// test namespace must be the deployment namespace for controller
	// to watch the deployments / daemonsets etc
	testNs := controller.PowerMonitorDeploymentNS

	// pre-condition
	f.AssertNoResourceExists(name, "", &v1alpha1.PowerMonitorInternal{})

	// Create PowerMonitorInternal with RBAC security mode
	pmi := f.CreateTestPowerMonitorInternal(name, testNs, runningOnVM, testKeplerImage, testKubeRbacProxyImage, Cluster, v1alpha1.SecurityModeRBAC, []string{
		"successful-test-namespace:successful-test-curl-sa",
	})

	tlsCertSecretName := powermonitor.SecretTLSCertName
	var caCertSource string

	if Cluster == k8s.Kubernetes {
		// For Kubernetes clusters, deploy cert-manager and dependencies
		clusterIssuerName := "selfsigned-cluster-issuer"
		caCertName := "power-monitor-ca"
		caCertSecretName := "power-monitor-ca-secret"
		pmIssuerName := "power-monitor-ca-issuer"
		f.DeployOpenshiftCerts(
			pmi.Name,
			testNs,
			clusterIssuerName,
			caCertName,
			caCertSecretName,
			pmIssuerName,
			tlsCertSecretName,
			tlsCertSecretName,
		)
		caCertSource = caCertSecretName
	} else {
		f.WaitUntilPowerMonitorInternalCondition(pmi.Name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)
		f.WaitForOpenshiftCerts(pmi.Name, testNs, tlsCertSecretName)
		caCertSource = tlsCertSecretName
	}

	// then
	f.AssertResourceExists(testNs, "", &corev1.Namespace{})
	ds := appsv1.DaemonSet{}
	f.AssertResourceExists(pmi.Name, testNs, &ds)

	retrievedPmi := f.WaitUntilPowerMonitorInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)
	// default toleration
	assert.Equal(t, []corev1.Toleration{{Operator: "Exists"}}, retrievedPmi.Spec.Kepler.Deployment.Tolerations)
	reconciled, err := k8s.FindCondition(retrievedPmi.Status.Conditions, v1alpha1.Reconciled)
	assert.NoError(t, err, "unable to get reconciled condition")
	assert.Equal(t, reconciled.ObservedGeneration, retrievedPmi.Generation)
	assert.Equal(t, reconciled.Status, v1alpha1.ConditionTrue)

	retrievedPmi = f.WaitUntilPowerMonitorInternalCondition(name, v1alpha1.Available, v1alpha1.ConditionTrue)
	available, err := k8s.FindCondition(retrievedPmi.Status.Conditions, v1alpha1.Available)
	assert.NoError(t, err, "unable to get available condition")
	assert.Equal(t, available.ObservedGeneration, retrievedPmi.Generation)
	assert.Equal(t, available.Status, v1alpha1.ConditionTrue)

	audience := fmt.Sprintf("%s.%s.svc", pmi.Name, testNs)
	serviceURL := fmt.Sprintf(
		"https://%s.%s.svc:%d/metrics",
		pmi.Name,
		testNs,
		powermonitor.SecurePort,
	)

	// wait for relevant secrets to be created
	tlsSecret := corev1.Secret{}
	f.AssertResourceExists(
		tlsCertSecretName,
		testNs,
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
			testNs,
		)
	} else {
		jobLogs = f.CreateCurlPowerMonitorTestSuiteForOpenShift(
			successfulJobName,
			successfulTestSAName,
			successfulTestCurlNs,
			audience,
			serviceURL,
			caCertSource,
			testNs,
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
			testNs,
		)
	} else {
		jobLogs = f.CreateCurlPowerMonitorTestSuiteForOpenShift(
			failedJobname,
			failedTestSAName,
			failedTestCurlNs,
			audience,
			serviceURL,
			caCertSource,
			testNs,
		)
	}
	assert.True(t, strings.Contains(jobLogs, "HTTP/2 403"), fmt.Sprintf("expected %s to receive a forbidden error (403) when attempting to access secure endpoint but did not", failedJobname))
}
