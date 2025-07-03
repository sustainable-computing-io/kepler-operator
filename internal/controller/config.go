// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package controller

import "github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"

// Config holds configuration shared across all controllers. This struct
// should be initialized in main

var Config = struct {
	KubeRbacProxyImage string
	RebootImage        string
	Image              string
	Cluster            k8s.Cluster
}{
	KubeRbacProxyImage: "quay.io/brancz/kube-rbac-proxy:v0.19.0",
	RebootImage:        "quay.io/sustainable_computing_io/kepler-reboot:v0.0.10",
	Image:              "",
	Cluster:            k8s.Kubernetes,
}
