// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"flag"
	"os"
	"testing"

	"github.com/sustainable.computing.io/kepler-operator/internal/controller"
	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
)

const (
	keplerImage        = `quay.io/sustainable_computing_io/kepler:v0.10.2`
	kubeRbacProxyImage = `quay.io/brancz/kube-rbac-proxy:v0.19.0`
)

var (
	Cluster                k8s.Cluster = k8s.Kubernetes
	testKeplerImage        string
	testKubeRbacProxyImage string
	skipKeplerTests        bool
	runningOnVM            bool
)

func TestMain(m *testing.M) {
	openshift := flag.Bool("openshift", false, "Indicate if tests are run aginast an OpenShift cluster.")
	flag.StringVar(&controller.PowerMonitorDeploymentNS, "deployment-namespace", controller.PowerMonitorDeploymentNS,
		"Namespace where kepler and its components are deployed.")
	flag.StringVar(&testKeplerImage, "kepler-image", keplerImage, "Kepler image to use when running Internal tests")
	flag.StringVar(&testKubeRbacProxyImage, "kube-rbac-proxy-image", kubeRbacProxyImage, "Kube Rbac Proxy image to use when running mode rbac tests")
	flag.BoolVar(&skipKeplerTests, "skip-kepler-tests", false, "Skip Kepler tests")
	flag.BoolVar(&runningOnVM, "running-on-vm", false, "Enable VM test environment")
	flag.Parse()

	if *openshift {
		Cluster = k8s.OpenShift
	}

	os.Exit(m.Run())
}
