#!/usr/bin/env bash

set -eu -o pipefail

# config
declare -r VERSION=${VERSION:-v0.0.3}
declare -r CLUSTER_PROVIDER=${CLUSTER_PROVIDER:-kind}
declare -r GRAFANA_ENABLE=${GRAFANA_ENABLE:-true}
declare -r KIND_WORKER_NODES=${KIND_WORKER_NODES:-2}
declare -r CERTMANAGER_VERSION=${CERT_MANAGER_VERSION:-1.15.0}
declare -r OLM_VERSION=${OLM_VERSION:-v0.28.0}

# constants
PROJECT_ROOT="$(git rev-parse --show-toplevel)"
declare -r PROJECT_ROOT
declare -r TMP_DIR="$PROJECT_ROOT/tmp"
declare -r DEV_CLUSTER_DIR="$TMP_DIR/local-dev-cluster"
declare -r BIN_DIR="$TMP_DIR/bin"
declare -r CERTMANAGER_URL="https://github.com/jetstack/cert-manager/releases/download/v$CERTMANAGER_VERSION/cert-manager.yaml"

source "$PROJECT_ROOT/hack/utils.bash"

git_checkout() {

	[[ -d "$DEV_CLUSTER_DIR" ]] || {
		info "git cloning local-dev-cluster - version $VERSION"
		run git clone -b "$VERSION" \
			https://github.com/sustainable-computing-io/local-dev-cluster.git \
			"$DEV_CLUSTER_DIR"
		return $?
	}

	cd "$DEV_CLUSTER_DIR"

	# NOTE: bail out if the git status is dirty as changes will be overwritten by git reset
	git diff --shortstat --exit-code >/dev/null || {
		err "local-dev-cluster has been modified"
		info "save/discard the changes and rerun the command"
		return 1
	}

	run git fetch --tags
	if [[ "$(git cat-file -t "$VERSION")" == tag ]]; then
		run git reset --hard "$VERSION"
	else
		run git reset --hard "origin/$VERSION"
	fi
}

install_cert_manager() {
	run kubectl apply --server-side --force-conflicts -f "$CERTMANAGER_URL"
	run kubectl wait --for=condition=Established crds --all --timeout=120s
}

cluster_prereqs() {
	info "setting up SCC crd"
	kubectl apply --force -f "$PROJECT_ROOT/hack/crds"

	kubectl get catalogsource && {
		info "OLM is already installed"
		return 0
	}
	info "setup OLM"
	operator-sdk olm install --version "$OLM_VERSION" --verbose --timeout 5m

	info "Ensure openshift namespace for dashboard exists"
	run kubectl create namespace openshift-config-managed

	info "Ensure prometheus can monitor all namespaces"
	run kubectl create -f hack/monitoring/rbac

	info "Ensure cert-manager is installed"
	install_cert_manager
}

ensure_all_tools() {
	header "Ensuring all tools are installed"
	"$PROJECT_ROOT/hack/tools.sh" all
}

on_cluster_up() {
	cluster_prereqs
	info 'Next: "make run" to run operator locally'
}

on_cluster_restart() {
	on_cluster_up
}

on_cluster_down() {
	info "all done"
}

main() {
	local op="$1"
	shift
	cd "$PROJECT_ROOT"
	export PATH="$BIN_DIR:$PATH"
	mkdir -p "${TMP_DIR}"
	ensure_all_tools

	#NOTE: allow cluster_<OP> to executed without going through the entire
	# setup again. This is useful (especially) in CI which uses the kepler-action
	# to setup a cluster which does not have all the prerequisites for running
	# Operator as OLM Bundle
	declare -F | cut -f3 -d ' ' | grep -qE "^cluster_$op\$" && {
		header "Running cluster_$op"
		"cluster_$op" "$@"
		return $?
	}

	header "Running Cluster Setup Script for $op"
	git_checkout
	export CLUSTER_PROVIDER
	export GRAFANA_ENABLE
	export KIND_WORKER_NODES
	cd "$DEV_CLUSTER_DIR"
	"$DEV_CLUSTER_DIR/main.sh" "$op"

	# NOTE: take additional actions after local-dev-cluster performs the "$OP"
	cd "$PROJECT_ROOT"
	on_cluster_"$op"
}

main "$1"
