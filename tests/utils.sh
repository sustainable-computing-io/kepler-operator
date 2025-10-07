#!/usr/bin/env bash
# Shared test utilities for e2e tests
# This file contains common functions used by both run-e2e.sh and helm.sh

# Ensure PROJECT_ROOT is set
if [[ -z "${PROJECT_ROOT:-}" ]]; then
	PROJECT_ROOT="$(git rev-parse --show-toplevel)"
	declare -r PROJECT_ROOT
fi

# Source basic utilities
source "$PROJECT_ROOT/hack/utils.bash"

# Common test variables
declare -r LOCAL_BIN="${LOCAL_BIN:-$PROJECT_ROOT/tmp/bin}"
declare -r OPERATOR_DEPLOY_NAME="${OPERATOR_DEPLOY_NAME:-kepler-operator-controller}"

# Initialize logs directory
# Creates a new logs directory and moves the old one to <dir>-prev
init_logs_dir() {
	local logs_dir="${1:-tmp/e2e}"

	rm -rf "$logs_dir-prev"
	mv "$logs_dir" "$logs_dir-prev" 2>/dev/null || true
	mkdir -p "$logs_dir"
}

# Load a docker image into kind cluster
kind_load_image() {
	local img="$1"

	# Check if image exists locally first
	if ! docker image inspect "$img" &>/dev/null; then
		# Image not local, try to pull it
		run docker pull "$img"
	fi

	run kind load docker-image "$img"
}

# Log kubernetes events for a namespace
# Usage: log_events <namespace> <log-file>
log_events() {
	local ns="$1"
	local log_file="${2:-events.log}"

	kubectl get events -w \
		-o custom-columns=FirstSeen:.firstTimestamp,LastSeen:.lastTimestamp,Count:.count,From:.source.component,Type:.type,Reason:.reason,Message:.message \
		-n "$ns" | tee "$log_file"
}

# Wait for operator deployment to be ready
# Works for both OLM and Helm deployments
# Usage: wait_for_operator <namespace> [deployment-name]
wait_for_operator() {
	local ns="$1"
	local deploy_name="${2:-$OPERATOR_DEPLOY_NAME}"
	local deployment="deploy/$deploy_name"

	header "Waiting for Kepler Operator ($ns) to be Ready"

	wait_until 30 10 "operator to run" \
		kubectl -n "$ns" rollout status "$deployment"

	run kubectl wait -n "$ns" --for=condition=Available \
		--timeout=300s "$deployment"

	ok "Operator up and running"
}

# Wait for PowerMonitor to be available
# Usage: wait_for_powermonitor <powermonitor-name>
wait_for_powermonitor() {
	local pm_name="${1:-power-monitor}"

	header "Waiting for PowerMonitor to be ready"
	wait_until 10 10 "powermonitor to be available" condition_check "True" kubectl get powermonitor "$pm_name" \
		-o jsonpath="{.status.conditions[?(@.type=='Available')].status}" || {
		fail "PowerMonitor is not ready"
		return 1
	}
	ok "PowerMonitor is ready"
	return 0
}

# Create ConfigMap to enable fake CPU meter for testing
# Usage: create_fake_cpu_configmap <namespace> [configmap-name]
create_fake_cpu_configmap() {
	local ns="$1"
	local cm_name="${2:-power-monitor-config}"

	info "Creating fake CPU meter ConfigMap in namespace $ns"
	kubectl create namespace "$ns" 2>/dev/null || true
	kubectl apply -n "$ns" -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: $cm_name
data:
  config.yaml: |
    dev:
      fake-cpu-meter:
        enabled: true
EOF
}

# Cleanup background jobs
cleanup_jobs() {
	info "Cleaning up background jobs..."
	# shellcheck disable=SC2046
	[[ -z "$(jobs -p)" ]] || kill $(jobs -p) 2>/dev/null || true
	return 0
}

# Update CRDs
# Usage: update_crds
update_crds() {
	info "Updating CRDs..."
	run kubectl apply --server-side --force-conflicts -k config/crd
	run kubectl wait --for=condition=Established crds --all --timeout=120s
	return 0
}

# Gather cluster state for debugging
# Usage: gather_cluster_state <output-dir> <namespace>
gather_cluster_state() {
	local output_dir="$1"
	local ns="${2:-}"

	mkdir -p "$output_dir"

	info "Gathering cluster state to $output_dir"

	# All resources
	kubectl get all -A >"$output_dir/all-resources.txt" 2>&1 || true

	# Events
	kubectl get events -A >"$output_dir/events.txt" 2>&1 || true

	# PowerMonitor resources
	kubectl get powermonitor -o yaml >"$output_dir/powermonitor.yaml" 2>&1 || true
	kubectl get powermonitorinternal -o yaml >"$output_dir/powermonitorinternal.yaml" 2>&1 || true

	# Operator logs if namespace provided
	if [[ -n "$ns" ]]; then
		kubectl logs -n "$ns" -l app.kubernetes.io/component=manager --tail=200 \
			>"$output_dir/operator-logs.txt" 2>&1 || true
		kubectl describe deployment -n "$ns" "$OPERATOR_DEPLOY_NAME" \
			>"$output_dir/operator-deployment.txt" 2>&1 || true
	fi

	ok "Cluster state gathered"
}
