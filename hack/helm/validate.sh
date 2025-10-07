#!/usr/bin/env bash
# Copyright 2025 The Kepler Contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
HELM_CHART_DIR="$PROJECT_ROOT/manifests/helm/kepler-operator"
CRD_SOURCE_DIR="$PROJECT_ROOT/config/crd/bases"
CRD_DEST_DIR="$HELM_CHART_DIR/crds"

# Image versions for validation
OPERATOR_IMAGE="${OPERATOR_IMAGE:-quay.io/sustainable_computing_io/kepler-operator:0.21.0}"
KEPLER_IMAGE="${KEPLER_IMAGE:-quay.io/sustainable_computing_io/kepler:latest}"
KUBE_RBAC_PROXY_IMAGE="${KUBE_RBAC_PROXY_IMAGE:-quay.io/brancz/kube-rbac-proxy:v0.18.1}"

# shellcheck source=hack/utils.bash
source "$SCRIPT_DIR/../utils.bash"

# Validate that required tools are available
check_tools() {
	local bin_dir="$PROJECT_ROOT/tmp/bin"
	local tools=("helm" "kustomize" "yq")
	for tool in "${tools[@]}"; do
		if [[ ! -x "$bin_dir/$tool" ]]; then
			fail "$tool is not installed. Please run 'make $tool' to install it."
			return 1
		fi
	done
}

# Use project-local tools
export PATH="$PROJECT_ROOT/tmp/bin:$PATH"

# Render Helm templates with standard test values
render_helm_template() {
	helm template kepler-operator "$HELM_CHART_DIR" \
		--namespace kepler-operator \
		--set operator.image="$OPERATOR_IMAGE" \
		--set kepler.image="$KEPLER_IMAGE" \
		--set kube-rbac-proxy.image="$KUBE_RBAC_PROXY_IMAGE" \
		--set metrics.serviceMonitor.enabled=true
}

# Validate Helm chart syntax
validate_helm_syntax() {
	info "Validating Helm chart syntax..."

	helm lint "$HELM_CHART_DIR" >/dev/null 2>&1 || {
		fail "Helm chart syntax validation failed"
		helm lint "$HELM_CHART_DIR"
		return 1

	}
	ok "Helm chart syntax is valid"
	return 0
}

# Validate that templates render successfully
validate_helm_template() {
	info "Validating Helm templates render successfully..."

	render_helm_template >/dev/null 2>&1 || {
		fail "Helm template rendering failed"
		render_helm_template
		return 1
	}
	ok "Helm templates render successfully"
	return 0
}

# Validate CRD sync status
validate_crd_sync() {
	info "Validating CRD sync status..."
	local all_synced=true

	for crd_file in "$CRD_SOURCE_DIR"/*.yaml; do
		local crd_name
		crd_name=$(basename "$crd_file")
		local dest_file="$CRD_DEST_DIR/$crd_name"

		[[ -f "$dest_file" ]] || {
			fail "CRD $crd_name not found in Helm chart crds/ directory"
			all_synced=false
			continue
		}

		diff -q "$crd_file" "$dest_file" >/dev/null 2>&1 || {
			fail "CRD $crd_name is out of sync. Run 'make helm-sync-crds' to sync."
			all_synced=false
			continue
		}
	done

	[[ "$all_synced" == "true" ]] || return 1
	ok "All CRDs are synced"
	return 0
}

# Validate that all expected resources are present
validate_resources() {
	info "Validating expected resources are present..."
	local expected_resources=(
		"ServiceAccount"
		"Role"
		"ClusterRole"
		"RoleBinding"
		"ClusterRoleBinding"
		"Service"
		"Deployment"
		"Certificate"
		"Issuer"
		"MutatingWebhookConfiguration"
		"ValidatingWebhookConfiguration"
		"ServiceMonitor"
	)

	local rendered
	rendered=$(render_helm_template)

	local all_found=true
	for resource in "${expected_resources[@]}"; do
		echo "$rendered" | grep -q "^kind: $resource$" || {
			fail "Expected resource $resource not found in rendered templates"
			all_found=false
		}
	done

	[[ "$all_found" == "true" ]] || return 1
	ok "All expected resources are present"
	return 0
}

main() {
	info "Starting Helm chart validation..."

	check_tools
	validate_helm_syntax
	validate_helm_template
	validate_crd_sync
	validate_resources

	ok "Helm chart validation completed successfully"
}

main "$@"
