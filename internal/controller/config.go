/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package controller

import "github.com/sustainable.computing.io/kepler-operator/pkg/utils/k8s"

// Config holds configuration shared across all controllers. This struct
// should be initialized in main
var (
	Config = struct {
		Image   string
		Cluster k8s.Cluster
	}{
		Image:   "",
		Cluster: k8s.Kubernetes,
	}

	InternalConfig = struct {
		ModelServerImage string
		EstimatorImage   string
	}{
		ModelServerImage: "",
		EstimatorImage:   "",
	}
)
