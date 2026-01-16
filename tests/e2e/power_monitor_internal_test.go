// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
	"github.com/sustainable.computing.io/kepler-operator/internal/controller"
	powermonitor "github.com/sustainable.computing.io/kepler-operator/pkg/components/power-monitor"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
	"github.com/sustainable.computing.io/kepler-operator/tests/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("PowerMonitorInternal", func() {
	var f *utils.Framework

	BeforeEach(func() {
		f = utils.NewFramework()
	})

	Describe("Reconciliation", func() {
		Context("when creating a basic PowerMonitorInternal", func() {
			It("should reconcile and create DaemonSet", func() {
				name := "e2e-pmi"
				testNs := controller.PowerMonitorDeploymentNS

				By("verifying PowerMonitorInternal doesn't exist initially")
				f.ExpectNoResourceExists(name, "", &v1alpha1.PowerMonitorInternal{})

				By("creating PowerMonitorInternal")
				f.CreateTestPowerMonitorInternal(name, testNs, runningOnVM, testKeplerImage, testKubeRbacProxyImage, Cluster, v1alpha1.SecurityModeNone, []string{})

				By("verifying namespace is created")
				f.ExpectResourceExists(testNs, "", &corev1.Namespace{})
				ds := &appsv1.DaemonSet{}

				By("waiting for PowerMonitorInternal to be reconciled")
				pmi := f.ExpectPowerMonitorInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)

				By("verifying DaemonSet exists and has correct configuration")
				f.ExpectResourceExists(pmi.Name, testNs, ds)
				containers := ds.Spec.Template.Spec.Containers
				Expect(containers).To(HaveLen(1), "should have exactly 1 container")
				Expect(containers[0].Ports).To(HaveLen(1), "should have exactly 1 port")
				Expect(containers[0].Ports[0].ContainerPort).To(BeEquivalentTo(28282))

				By("verifying PowerMonitorInternal status")
				f.ExpectPowerMonitorInternalStatus(pmi.Name, utils.Timeout(5*time.Minute))
			})
		})
	})

	Describe("RBAC Reconciliation", func() {
		Context("when using RBAC security mode", func() {
			It("should reconcile with TLS certificates and access control", func() {
				name := "e2e-pmi"
				// test namespace must be the deployment namespace for controller
				// to watch the deployments / daemonsets etc
				testNs := controller.PowerMonitorDeploymentNS

				By("verifying PowerMonitorInternal doesn't exist initially")
				f.ExpectNoResourceExists(name, "", &v1alpha1.PowerMonitorInternal{})

				By("creating PowerMonitorInternal with RBAC security mode")
				pmi := f.CreateTestPowerMonitorInternal(name, testNs, runningOnVM, testKeplerImage, testKubeRbacProxyImage, Cluster, v1alpha1.SecurityModeRBAC, []string{
					"successful-test-namespace:successful-test-curl-sa",
				})

				tlsCertSecretName := powermonitor.SecretTLSCertName
				var caCertSource string

				if Cluster == k8s.Kubernetes {
					By("deploying cert-manager certificates for Kubernetes cluster")
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
					By("waiting for OpenShift certificates")
					f.WaitUntilPowerMonitorInternalCondition(pmi.Name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)
					f.WaitForOpenshiftCerts(pmi.Name, testNs, tlsCertSecretName)
					caCertSource = tlsCertSecretName
				}

				By("verifying namespace and DaemonSet exist")
				f.ExpectResourceExists(testNs, "", &corev1.Namespace{})
				ds := &appsv1.DaemonSet{}
				f.ExpectResourceExists(pmi.Name, testNs, ds)

				By("verifying PowerMonitorInternal is reconciled")
				retrievedPmi := f.ExpectPowerMonitorInternalCondition(name, v1alpha1.Reconciled, v1alpha1.ConditionTrue)
				Expect(retrievedPmi.Spec.Kepler.Deployment.Tolerations).To(Equal([]corev1.Toleration{{Operator: "Exists"}}))
				reconciled, err := k8s.FindCondition(retrievedPmi.Status.Conditions, v1alpha1.Reconciled)
				Expect(err).NotTo(HaveOccurred(), "unable to get reconciled condition")
				Expect(reconciled.ObservedGeneration).To(Equal(retrievedPmi.Generation))
				Expect(reconciled.Status).To(Equal(v1alpha1.ConditionTrue))

				By("verifying PowerMonitorInternal is available")
				retrievedPmi = f.ExpectPowerMonitorInternalCondition(name, v1alpha1.Available, v1alpha1.ConditionTrue)
				available, err := k8s.FindCondition(retrievedPmi.Status.Conditions, v1alpha1.Available)
				Expect(err).NotTo(HaveOccurred(), "unable to get available condition")
				Expect(available.ObservedGeneration).To(Equal(retrievedPmi.Generation))
				Expect(available.Status).To(Equal(v1alpha1.ConditionTrue))

				audience := fmt.Sprintf("%s.%s.svc", pmi.Name, testNs)
				serviceURL := fmt.Sprintf(
					"https://%s.%s.svc:%d/metrics",
					pmi.Name,
					testNs,
					powermonitor.SecurePort,
				)

				By("verifying TLS secrets are created")
				tlsSecret := &corev1.Secret{}
				f.ExpectResourceExists(
					tlsCertSecretName,
					testNs,
					tlsSecret,
					utils.Timeout(5*time.Minute),
				)
				Expect(tlsSecret.Data["tls.crt"]).NotTo(BeEmpty(), "TLS cert should be present")
				Expect(tlsSecret.Data["tls.key"]).NotTo(BeEmpty(), "TLS key should be present")

				By("deploying curl job with allowed service account")
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
				Expect(jobLogs).To(ContainSubstring("HTTP/2 200"),
					"expected %s to successfully access (200) the secure endpoint", successfulJobName)

				By("deploying curl job with blocked service account")
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
				Expect(jobLogs).To(ContainSubstring("HTTP/2 403"),
					"expected %s to receive forbidden error (403) when accessing secure endpoint", failedJobname)
			})
		})
	})
})
