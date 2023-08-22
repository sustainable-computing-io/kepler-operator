#!/usr/bin/env bash
#
# This file is part of the Kepler project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2022 The Kepler Contributors
#

set -eu -o pipefail

# config
declare -r VERSION=${VERSION:-v0.0.3}
declare -r CLUSTER_PROVIDER=${CLUSTER_PROVIDER:-kind}
declare -r GRAFANA_ENABLE=${GRAFANA_ENABLE:-true}
declare -r KIND_WORKER_NODES=${KIND_WORKER_NODES:-2}

# constants
PROJECT_ROOT="$(git rev-parse --show-toplevel)"
declare -r PROJECT_ROOT
declare -r TMP_DIR="$PROJECT_ROOT/tmp"
declare -r DEV_CLUSTER_DIR="$TMP_DIR/local-dev-cluster"
declare -r BIN_DIR="$TMP_DIR/bin"
declare -r OPERATOR_SDK_VERSION=${OPERATOR_SDK_VERSION:-v1.27.0}

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

cluster_prereqs() {
	info "setting up SCC crd"
	kubectl apply --force -f "$PROJECT_ROOT/hack/crds"

	kubectl get catalogsource && {
		info "OLM is already installed"
		return 0
	}
	info "setup OLM"
	operator-sdk olm install --verbose

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
