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
	keplerImage = `quay.io/sustainable_computing_io/kepler:release-0.7.12`
)

var (
	Cluster         k8s.Cluster = k8s.Kubernetes
	testKeplerImage string
)

func TestMain(m *testing.M) {
	openshift := flag.Bool("openshift", true, "Indicate if tests are run aginast an OpenShift cluster.")
	flag.StringVar(&controller.KeplerDeploymentNS, "deployment-namespace", controller.KeplerDeploymentNS,
		"Namespace where kepler and its components are deployed.")
	flag.StringVar(&testKeplerImage, "kepler-image", keplerImage, "Kepler image to use when running Internal tests")
	flag.Parse()

	if *openshift {
		Cluster = k8s.OpenShift
	}

	os.Exit(m.Run())
}
