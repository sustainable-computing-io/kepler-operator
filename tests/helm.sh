#!/usr/bin/env bash
# Helm E2E testing script
# Tests the Kepler Operator Helm chart deployment end-to-end

set -e -u -o pipefail

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
declare -r PROJECT_ROOT

# Source test utilities
source "$PROJECT_ROOT/tests/utils.sh"

# Script configuration
declare -r HELM_RELEASE_NAME="${HELM_RELEASE_NAME:-kepler-operator}"
declare -r HELM_NAMESPACE="${HELM_NAMESPACE:-kepler-operator}"
declare -r POWERMONITOR_NS="${POWERMONITOR_NS:-power-monitor}"
declare -r LOGS_DIR="${LOGS_DIR:-tmp/helm-e2e}"

# Testdata paths
declare -r TESTDATA_DIR="$PROJECT_ROOT/tests/testdata"
declare -r POWERMONITOR_VM_YAML="$TESTDATA_DIR/powermonitor-vm.yaml"
declare -r POWERMONITOR_BAREMETAL_YAML="$TESTDATA_DIR/powermonitor-baremetal.yaml"
declare -r FAKE_CPU_CONFIGMAP_YAML="$TESTDATA_DIR/fake-cpu-configmap.yaml"

# Image configuration
# NOTE: these are not readonly because it can be overridden by --flag
declare VERSION="${VERSION:-0.0.0-dev}"
declare IMG_BASE="${IMG_BASE:-localhost:5001}"
declare OPERATOR_IMG="$IMG_BASE/kepler-operator:$VERSION"

# Managed image versions (what operator deploys)
declare -r KEPLER_IMAGE="${KEPLER_IMAGE:-quay.io/sustainable_computing_io/kepler:latest}"
declare -r KUBE_RBAC_PROXY_IMAGE="${KUBE_RBAC_PROXY_IMAGE:-quay.io/brancz/kube-rbac-proxy:v0.19.0}"

# Script flags
declare NO_BUILD=false
declare NO_DEPLOY=false
declare CLEANUP=false
declare RUNNING_ON_VM=false
declare SHOW_USAGE=false

# Trap cleanup on exit
trap cleanup_on_exit INT TERM

cleanup_on_exit() {
	cleanup_jobs
	if $CLEANUP; then
		uninstall_helm || true
	fi
}

# Build operator image
build_operator() {
	header "Build Operator Image"

	$NO_BUILD && {
		info "Skipping operator image build (--no-build)"
		return 0
	}

	run make operator-build \
		VERSION="$VERSION" \
		IMG_BASE="$IMG_BASE"

	ok "Operator image built: $OPERATOR_IMG"
}

# Load operator image to kind cluster
load_operator_image() {
	header "Load Operator Image to Kind"

	$NO_BUILD && {
		info "Skipping image load (--no-build)"
		return 0
	}

	kind_load_image "$OPERATOR_IMG"

	ok "Operator image loaded to kind"
}

# Install operator via Helm
install_helm() {
	header "Install Operator via Helm"

	# Sync CRDs first
	run make helm-sync-crds

	# Install via Helm
	run helm upgrade --install "$HELM_RELEASE_NAME" \
		manifests/helm/kepler-operator \
		--namespace "$HELM_NAMESPACE" \
		--create-namespace \
		--set operator.image="$OPERATOR_IMG" \
		--set kepler.image="$KEPLER_IMAGE" \
		--set kube-rbac-proxy.image="$KUBE_RBAC_PROXY_IMAGE" \
		--timeout=5m \
		--wait

	ok "Operator installed via Helm"
}

# Wait for webhook certificate to be ready
wait_for_webhook_cert() {
	header "Waiting for Webhook Certificate"

	info "Waiting for webhook certificate to be issued..."
	run kubectl wait --for=condition=Ready --timeout=300s \
		-n "$HELM_NAMESPACE" certificate/kepler-operator-serving-cert

	# Give webhook time to start with the certificate
	sleep 10

	ok "Webhook certificate ready"
}

# Deploy PowerMonitor on VM with fake CPU meter
deploy_pm_on_vm() {
	# Deploy PowerMonitor CR first (operator will create namespace)
	info "Creating PowerMonitor resource with fake CPU meter"
	kubectl apply -f "$POWERMONITOR_VM_YAML"

	# Wait for operator to create the namespace
	info "Waiting for operator to create namespace $POWERMONITOR_NS"
	kubectl wait --for=jsonpath='{.status.phase}'=Active \
		--timeout=60s namespace/"$POWERMONITOR_NS" 2>/dev/null || {
		# Namespace might not exist yet, wait for it to be created
		local retries=30
		while [[ $retries -gt 0 ]]; do
			if kubectl get namespace "$POWERMONITOR_NS" >/dev/null 2>&1; then
				break
			fi
			sleep 2
			((retries--))
		done
	}

	# Create fake CPU meter ConfigMap after namespace exists
	info "Creating fake CPU meter ConfigMap"
	kubectl apply -n "$POWERMONITOR_NS" -f "$FAKE_CPU_CONFIGMAP_YAML"
}

# Deploy PowerMonitor on bare metal with hardware sensors
deploy_pm_on_baremetal() {
	info "Creating PowerMonitor resource (using hardware sensors)"
	kubectl apply -f "$POWERMONITOR_BAREMETAL_YAML"
}

# Deploy PowerMonitor
deploy_powermonitor() {
	header "Deploy PowerMonitor"

	if $RUNNING_ON_VM; then
		deploy_pm_on_vm
	else
		deploy_pm_on_baremetal
	fi

	# Wait for PowerMonitor to be ready
	wait_for_powermonitor power-monitor

	ok "PowerMonitor deployed successfully"
}

# Verify deployment
verify_deployment() {
	header "Verify Deployment"

	# Check operator deployment
	info "Verifying operator deployment..."
	kubectl get deployment -n "$HELM_NAMESPACE" kepler-operator-controller

	# Check PowerMonitor DaemonSet
	info "Verifying PowerMonitor DaemonSet..."
	kubectl get daemonset -n "$POWERMONITOR_NS" power-monitor

	# Check pods are running
	info "Checking PowerMonitor pods..."
	kubectl get pods -n "$POWERMONITOR_NS"

	ok "All components verified"
}

# Uninstall Helm release
uninstall_helm() {
	header "Uninstall Helm Release"

	# Delete PowerMonitor first
	kubectl delete powermonitor power-monitor --ignore-not-found=true || true
	sleep 5

	# Uninstall Helm release
	run helm uninstall "$HELM_RELEASE_NAME" \
		--namespace "$HELM_NAMESPACE" || true

	ok "Helm release uninstalled"
}

# Parse command line arguments
parse_args() {
	while [[ $# -gt 0 ]]; do
		case $1 in
		-h | --help)
			SHOW_USAGE=true
			return 0
			;;
		--no-build)
			NO_BUILD=true
			shift
			;;
		--no-deploy)
			NO_DEPLOY=true
			shift
			;;
		--cleanup)
			CLEANUP=true
			shift
			;;
		--running-on-vm)
			RUNNING_ON_VM=true
			shift
			;;
		--version)
			shift
			VERSION="$1"
			OPERATOR_IMG="$IMG_BASE/kepler-operator:$VERSION"
			shift
			;;
		--version=*)
			VERSION="${1#*=}"
			OPERATOR_IMG="$IMG_BASE/kepler-operator:$VERSION"
			shift
			;;
		*)
			err "Unknown option: $1"
			SHOW_USAGE=true
			return 1
			;;
		esac
	done
	return 0
}

# Show usage
show_usage() {
	local scr
	scr="$(basename "$0")"

	cat <<-EOF
		🔆 Usage:
		  $scr [OPTIONS]

		📋 Description:
		  Run Helm E2E tests for the Kepler Operator

		💡 Examples:
		  # Full flow: build, load, deploy, verify
		  ❯ $scr

		  # Run in CI/VM environment (enables fake CPU meter)
		  ❯ $scr --running-on-vm

		  # Use existing image (skip build)
		  ❯ $scr --no-build --version=0.21.0

		  # Quick iteration (skip deploy, just verify)
		  ❯ $scr --no-deploy

		  # Full flow with cleanup
		  ❯ $scr --cleanup

		⚙️ Options:
		  -h, --help          Show this help
		  --no-build          Skip building operator image
		  --no-deploy         Skip deployment (assumes operator already installed)
		  --cleanup           Uninstall Helm release after test
		  --running-on-vm     Enable fake CPU meter (for VMs without hardware sensors)
		  --version VER       Operator version to test (default: $VERSION)

		📝 Prerequisites:
		  - Kubernetes cluster running (kind recommended)
		  - cert-manager installed (run 'make cluster-up')
		  - helm, kubectl, docker available

		📂 Logs:
		  Test logs are saved to: $LOGS_DIR
	EOF

	return 0
}

# Print test configuration
print_config() {
	header "Test Configuration"
	cat <<-EOF
		  Operator Image:      $OPERATOR_IMG
		  Kepler Image:        $KEPLER_IMAGE
		  Kube RBAC Proxy:     $KUBE_RBAC_PROXY_IMAGE
		  Helm Release:        $HELM_RELEASE_NAME
		  Helm Namespace:      $HELM_NAMESPACE
		  PowerMonitor NS:     $POWERMONITOR_NS
		  Skip Build:          $NO_BUILD
		  Skip Deploy:         $NO_DEPLOY
		  Running on VM:       $RUNNING_ON_VM
		  Cleanup After:       $CLEANUP
		  Logs Directory:      $LOGS_DIR

	EOF
	line 50
}

# Main test flow
main() {
	export PATH="$LOCAL_BIN:$PATH"

	# Parse arguments
	parse_args "$@" || {
		show_usage
		return 1
	}

	if $SHOW_USAGE; then
		show_usage
		return 0
	fi

	cd "$PROJECT_ROOT"

	# Initialize logs directory
	init_logs_dir "$LOGS_DIR"

	# Print configuration
	print_config

	# Start background event logging
	log_events "$HELM_NAMESPACE" "$LOGS_DIR/operator-events.log" &
	log_events "$POWERMONITOR_NS" "$LOGS_DIR/powermonitor-events.log" &

	local ret=0

	# Run test flow
	if ! $NO_DEPLOY; then
		build_operator || ret=$?
		[[ $ret -ne 0 ]] && return $ret

		load_operator_image || ret=$?
		[[ $ret -ne 0 ]] && return $ret

		install_helm || ret=$?
		[[ $ret -ne 0 ]] && return $ret

		wait_for_webhook_cert || ret=$?
		[[ $ret -ne 0 ]] && return $ret

		wait_for_operator "$HELM_NAMESPACE" "kepler-operator-controller" || ret=$?
		[[ $ret -ne 0 ]] && return $ret

		deploy_powermonitor || ret=$?
		[[ $ret -ne 0 ]] && return $ret
	fi

	verify_deployment || ret=$?

	# Cleanup background jobs
	cleanup_jobs

	# Always gather cluster state after test run (for debugging)
	gather_cluster_state "$LOGS_DIR" "$HELM_NAMESPACE"

	if [[ $ret -eq 0 ]]; then
		ok "✅ Helm E2E Tests Passed"
	else
		fail "❌ Helm E2E Tests Failed"
		info "Check logs in: $LOGS_DIR"
	fi

	return $ret
}

main "$@"
