// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package controller

import "github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"

// Config holds configuration shared across all controllers. This struct
// should be initialized in main

var Config = struct {
	RebootImage string
	Image       string
	Cluster     k8s.Cluster
}{
	RebootImage: "quay.io/sustainable_computing_io/kepler-reboot:v0.0.9",
	Image:       "",
	Cluster:     k8s.Kubernetes,
}
