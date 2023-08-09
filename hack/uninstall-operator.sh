#!/usr/bin/env bash

# Copyright 2023.
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
#

set -eu -o pipefail

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
declare -r PROJECT_ROOT

# config
declare DELETE_RESOURCES=false
declare OPERATORS_NS="openshift-operators"
declare OPERATOR_VERSION="v0.5.0"

source "$PROJECT_ROOT/hack/utils.bash"

parse_args() {
	### while there are args parse them
	while [[ -n "${1+xxx}" ]]; do
		case $1 in
		--delete)
			DELETE_RESOURCES=true
			shift
			;; # exit the loop
		--ns | -n)
			shift
			OPERATORS_NS=$1
			shift
			;;
		--version | -v)
			shift
			OPERATOR_VERSION=$1
			shift
			;;
		*) return 1 ;; # show usage on everything else
		esac
	done
	return 0
}

print_usage() {
	local scr
	scr="$(basename "$0")"

	read -r -d '' help <<-EOF_HELP || true
		Usage:
		  $scr
		  $scr  --delete
		  $scr  --version
		  $scr  --ns | -n

		 ─────────────────────────────────────────────────────────────────

		Options:
			--delete              deletes the resources listed
		  --version | -v        version of the operator to delete
			                      default: $OPERATOR_VERSION
			--ns | -n NAMESPACE   namespace where the operator is deployed
			                      default: $OPERATORS_NS


	EOF_HELP

	echo -e "$help"
	return 0
}

main() {
	local operator="kepler-operator"

	parse_args "$@" || {
		print_usage
		fail "failed to parse args"
		return 1
	}

	header "Resources of Kepler Operator - $OPERATOR_VERSION"

	kubectl get csv "${operator}.$OPERATOR_VERSION" -n "$OPERATORS_NS" || {
		info "failed to find $OPERATOR_VERSION of $operator."
		line 50
		kubectl get csv | grep -E "$operator|NAME"

		line 50
		info "$operator version found are ☝️"
		return 1
	}

	local label="operators.coreos.com/${operator}.$OPERATORS_NS="

	info "Going to delete the following"
	line 50 heavy
	run kubectl get ns kepler || true
	run kubectl get kepler -A
	run kubectl get -n "$OPERATORS_NS" olm -l "$label"
	run kubectl get operator,crd,clusterrole,clusterrolebinding -l "$label" -A
	run kubectl get leases 0d9cbc82.sustainable.computing.io -n "$OPERATORS_NS"
	line 50 heavy

	! $DELETE_RESOURCES && {
		info "Not deleting any resources, use --delete flag to force deletion"
		return 0
	}

	header "Deleting Kepler Operator Version $OPERATOR_VERSION"

	run kubectl delete kepler -A --all
	run kubectl delete ns kepler || true
	run kubectl delete -n "$OPERATORS_NS" olm -l "$label"
	run kubectl delete operator,crd,clusterrole,clusterrolebinding -l "$label" -A
	run kubectl delete leases 0d9cbc82.sustainable.computing.io -n "$OPERATORS_NS"

	ok "$operator version has been successfully uninstalled.\n"
}

main "$@"
