// SPDX-FileCopyrightText: 2025 The Kepler Authors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"time"

	"github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"
)

// Config holds configuration shared across all controllers. This struct
// should be initialized in main

var Config = struct {
	KubeRbacProxyImage   string
	Image                string
	Cluster              k8s.Cluster
	TokenRefreshInterval time.Duration
	TokenTTL             time.Duration
}{
	KubeRbacProxyImage:   "quay.io/brancz/kube-rbac-proxy:v0.19.0",
	Image:                "",
	Cluster:              k8s.Kubernetes,
	TokenRefreshInterval: 24 * time.Hour,
	TokenTTL:             168 * time.Hour,
}
