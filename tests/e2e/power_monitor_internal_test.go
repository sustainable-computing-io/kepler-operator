// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

// import (
// 	"fmt"
// 	"strconv"
// 	"strings"
// 	"testing"
// 	"time"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/sustainable.computing.io/kepler-operator/api/v1alpha1"
// 	"github.com/sustainable.computing.io/kepler-operator/internal/controller"
// 	powermonitor "github.com/sustainable.computing.io/kepler-operator/pkg/components/power-monitor"
// 	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/test"
// 	appsv1 "k8s.io/api/apps/v1"
// 	corev1 "k8s.io/api/core/v1"
// )

// func TestPowerMonitorInternal_Reconciliation(t *testing.T) {
// 	f := test.NewFramework(t)
// 	name := "e2e-pmi"
// 	// test namespace must be the deployment namespace for controller
// 	// to watch the deployments / daemonsets etc
// 	testNs := controller.PowerMonitorDeploymentNS

// 	// pre-condition
// 	f.AssertNoResourceExists(name, "", &v1alpha1.PowerMonitorInternal{})

// 	// when
// 	b := test.PowerMonitorInternalBuilder{}
// 	pmi := f.CreatePowerMonitorInternal(name,
// 		b.WithNamespace(testNs),
// 		b.WithKeplerImage(testKeplerRebootImage),
// 		b.WithCluster(Cluster),
// 		b.WithAnnotation(vmAnnotationKey, strconv.FormatBool(enableVMTest)),
// 	)

// 	// then the following resources will be created
// 	f.AssertResourceExists(testNs, "", &corev1.Namespace{})

// 	ds := appsv1.DaemonSet{}
// 	f.AssertResourceExists(pmi.Name, testNs, &ds)
// 	containers := ds.Spec.Template.Spec.Containers
// 	assert.Equal(t, 1, len(containers))
// 	assert.Equal(t, 1, len(containers[0].Ports))
// 	assert.EqualValues(t, 28282, containers[0].Ports[0].ContainerPort)
// 	// test expected status (PowerMonitor)
// 	f.AssertPowerMonitorInternalStatus(pmi.Name, test.Timeout(5*time.Minute))
// 	/*
// 	   f.DeletePowerMonitorInternal(name)
// 	   f.AssertNoResourceExists(testNs, "", &corev1.Namespace{})
// 	   f.AssertNoResourceExists(ds.Name, ds.Namespace, &ds)
// 	*/
// }

// func TestPowerMonitorInternal_RBAC_Reconciliation(t *testing.T) {
// 	f := test.NewFramework(t)
// 	name := "e2e-pmi"
// 	// test namespace must be the deployment namespace for controller
// 	// to watch the deployments / daemonsets etc
// 	testNs := controller.PowerMonitorDeploymentNS

// 	// pre-condition
// 	f.AssertNoResourceExists(name, "", &v1alpha1.PowerMonitorInternal{})

// 	// when
// 	b := test.PowerMonitorInternalBuilder{}
// 	pmi := f.CreatePowerMonitorInternal(name,
// 		b.WithNamespace(testNs),
// 		b.WithKeplerImage(testKeplerRebootImage),
// 		b.WithCluster(Cluster),
// 		b.WithAnnotation(vmAnnotationKey, strconv.FormatBool(enableVMTest)),
// 		b.WithModeRBAC([]string{
// 			"system:serviceaccount:successful-test-namespace:successful-test-curl-sa",
// 		}),
// 	)

// 	// then the following resources will be created
// 	f.AssertResourceExists(testNs, "", &corev1.Namespace{})

// 	// generate missing certs required in openshift
// 	clusterIssuerName := "selfsigned-cluster-issuer"
// 	caCertName := "power-monitor-ca"
// 	caCertSecretName := "power-monitor-ca-secret"
// 	pmIssuerName := "power-monitor-ca-issuer"
// 	tlsCertName := powermonitor.SecretTLSCertName
// 	tlsCertSecretName := powermonitor.SecretTLSCertName
// 	f.DeployOpenshiftCerts(name, testNs, clusterIssuerName, caCertName, caCertSecretName, pmIssuerName, tlsCertName, tlsCertSecretName)
// 	ds := appsv1.DaemonSet{}
// 	f.AssertResourceExists(pmi.Name, testNs, &ds)
// 	containers := ds.Spec.Template.Spec.Containers
// 	assert.Equal(t, 2, len(containers))
// 	// test expected status (PowerMonitor)
// 	f.AssertPowerMonitorInternalStatus(pmi.Name, test.Timeout(5*time.Minute))
// 	// wait for reconciliation to be ready
// 	time.Sleep(60 * time.Second)

// 	audience := fmt.Sprintf("%s.%s.svc", name, testNs)
// 	serviceURL := fmt.Sprintf("https://%s.%s.svc:%d/metrics", name, testNs, powermonitor.SecurePort)

// 	// deploy successful curl job
// 	successfulJobName := "successful-test-curl"
// 	successfulTestSAName := "successful-test-curl-sa"
// 	successfulTestCurlNs := "successful-test-namespace"
// 	logs := f.CreateCurlPowerMonitorTestSuite(successfulJobName, successfulTestSAName, successfulTestCurlNs, audience, serviceURL, caCertSecretName, testNs)
// 	assert.True(t, strings.Contains(logs, "HTTP/2 200"), fmt.Sprintf("expected %s to successfully access (200) the secure endpoint but it did not", successfulJobName))

// 	// deploy blocked curl job
// 	failedJobname := "failed-test-curl"
// 	failedTestSAName := "failed-test-curl-sa"
// 	failedTestCurlNs := "failed-test-namespace"
// 	logs = f.CreateCurlPowerMonitorTestSuite(failedJobname, failedTestSAName, failedTestCurlNs, audience, serviceURL, caCertSecretName, testNs)
// 	assert.True(t, strings.Contains(logs, "HTTP/2 403"), fmt.Sprintf("expected %s to receive a forbidden error (403) when attempting to access secure endpoint but did not", failedJobname))
// 	/*
// 	   f.DeletePowerMonitorInternal(name)
// 	   f.AssertNoResourceExists(testNs, "", &corev1.Namespace{})
// 	   f.AssertNoResourceExists(ds.Name, ds.Namespace, &ds)
// 	*/
// }
