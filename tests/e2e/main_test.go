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
	keplerImage       = `quay.io/sustainable_computing_io/kepler:release-0.7.12`
	keplerRebootImage = `quay.io/sustainable_computing_io/kepler-reboot:v0.0.9`
)

var (
	Cluster               k8s.Cluster = k8s.Kubernetes
	testKeplerImage       string
	testKeplerRebootImage string
	skipKeplerTests       bool
	runningOnVM           bool
)

func TestMain(m *testing.M) {
	openshift := flag.Bool("openshift", true, "Indicate if tests are run aginast an OpenShift cluster.")
	flag.StringVar(&controller.KeplerDeploymentNS, "deployment-namespace", controller.KeplerDeploymentNS,
		"Namespace where kepler and its components are deployed.")
	flag.StringVar(&testKeplerImage, "kepler-image", keplerImage, "Kepler image to use when running Internal tests")
	flag.StringVar(&testKeplerRebootImage, "kepler-reboot-image", keplerRebootImage, "Kepler image to use when running PowerMonitorInternal tests")
	flag.BoolVar(&skipKeplerTests, "skip-kepler-tests", false, "Skip Kepler tests")
	flag.BoolVar(&runningOnVM, "running-on-vm", false, "Enable VM test environment")
	flag.Parse()

	if *openshift {
		Cluster = k8s.OpenShift
	}

	os.Exit(m.Run())
}
