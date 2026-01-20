// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"flag"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sustainable.computing.io/kepler-operator/internal/controller"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
)

const (
	keplerImage        = `quay.io/sustainable_computing_io/kepler:v0.11.3`
	kubeRbacProxyImage = `quay.io/brancz/kube-rbac-proxy:v0.19.0`

	// Default timeouts for async operations
	defaultEventuallyTimeout  = 2 * time.Minute
	defaultEventuallyInterval = 5 * time.Second
)

var (
	Cluster                k8s.Cluster = k8s.Kubernetes
	testKeplerImage        string
	testKubeRbacProxyImage string
	runningOnVM            bool
)

func init() {
	// Register flags before test run
	flag.BoolVar(&runningOnVM, "running-on-vm", false, "Enable VM test environment")
	flag.StringVar(&testKeplerImage, "kepler-image", keplerImage, "Kepler image to use when running Internal tests")
	flag.StringVar(&testKubeRbacProxyImage, "kube-rbac-proxy-image", kubeRbacProxyImage, "Kube Rbac Proxy image to use when running mode rbac tests")
	flag.StringVar(&controller.PowerMonitorDeploymentNS, "deployment-namespace", controller.PowerMonitorDeploymentNS,
		"Namespace where kepler and its components are deployed.")
}

func TestE2E(t *testing.T) {
	// Parse any custom flags for OpenShift
	openshift := flag.Bool("openshift", false, "Indicate if tests are run against an OpenShift cluster.")
	if !flag.Parsed() {
		flag.Parse()
	}

	if *openshift {
		Cluster = k8s.OpenShift
	}

	RegisterFailHandler(Fail)

	// Set default timeouts for Eventually/Consistently
	SetDefaultEventuallyTimeout(defaultEventuallyTimeout)
	SetDefaultEventuallyPollingInterval(defaultEventuallyInterval)

	RunSpecs(t, "E2E Suite")
}
