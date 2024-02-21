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
declare OPERATOR_VERSION=""
declare OPERATOR=""
declare OPERATOR_CSV=""
declare SHOW_HELP=false

source "$PROJECT_ROOT/hack/utils.bash"

init() {
	OPERATOR=$(kubectl get operators -o name | grep -E 'power|kepler' | awk -F '[/.]' '{print $(NF-1)}') || {
		fail "No operator found! Is it installed?"
		return 1
	}
	ok "found $OPERATOR installed"

	OPERATOR_CSV=$(kubectl get csv -n "$OPERATORS_NS" -o name | grep -E "$OPERATOR\.v") || {
		warn "No csv found for $OPERATOR! Is it installed?"
		return 1
	}
	ok "found $OPERATOR csv: $OPERATOR_CSV"

	[[ -z "$OPERATOR_VERSION" ]] && {
		info "No operator version specified; finding the installed version"
		local version=""
		version=$(kubectl get -n "$OPERATORS_NS" "$OPERATOR_CSV" -o jsonpath="{.spec.version}")
		[[ -z "$version" ]] && {
			fail "Failed to find version of $OPERATOR - $OPERATOR_CSV"
			return 1
		}
		OPERATOR_VERSION="v$version"
	}
	ok "$OPERATOR version: $OPERATOR_VERSION"

	return 0
}

parse_args() {
	### while there are args parse them
	while [[ -n "${1+xxx}" ]]; do
		case $1 in
		--help | -h)
			shift
			SHOW_HELP=true
			return 0
			;;
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
			[[ "${1:0:1}" != "v" ]] && OPERATOR_VERSION="v$1"
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
		  --delete                deletes the resources listed
		  --version VERSION | -v  specify version of the operator to delete
		  --ns | -n NAMESPACE     namespace where the operator is deployed
			                          default: $OPERATORS_NS


	EOF_HELP

	echo -e "$help"
	return 0
}

list_olm_resources() {
	header "Listing Resources of $OPERATOR"

	info_run kubectl get csv -n "$OPERATORS_NS"
	kubectl get csv -n "$OPERATORS_NS" | grep -E "$OPERATOR|NAME" || true

	run kubectl get olm -n "$OPERATORS_NS" -o wide || true

}

main() {
	parse_args "$@" || {
		print_usage
		fail "failed to parse args"
		return 1
	}

	$SHOW_HELP && {
		print_usage
		return 0
	}

	init || {
		err "cannot gather operator details"
		return 1
	}

	header "Resources of $OPERATOR - $OPERATOR_VERSION"

	kubectl get csv "${OPERATOR}.$OPERATOR_VERSION" -n "$OPERATORS_NS" || {
		kubectl get csv -n "$OPERATORS_NS" | grep -E "$OPERATOR|NAME" || true
		line 50
		info "$OPERATOR version found are ☝️"

		fail "failed to find version $OPERATOR_VERSION of $OPERATOR."
		list_olm_resources "$OPERATOR"
		return 1
	}

	local label="operators.coreos.com/${OPERATOR}.$OPERATORS_NS="

	header "Going to delete the following"
	run kubectl get ns kepler || true
	run kubectl get kepler -A
	run kubectl get -n "$OPERATORS_NS" olm -l "$label"
	run kubectl get crd,clusterrole,clusterrolebinding -l "$label" -A
	run kubectl get operators "$OPERATOR.$OPERATORS_NS"
	run kubectl get leases 0d9cbc82.sustainable.computing.io -n "$OPERATORS_NS" || true
	run kubectl get catalogsource kepler-operator-catalog -n "$OPERATORS_NS" || true
	line 50 heavy

	! $DELETE_RESOURCES && {
		info "To delete all resources listed above, rerun with --delete option added. \n"
		info_run "   $0 $* --delete"
		return 0
	}

	header "Deleting $OPERATOR Version $OPERATOR_VERSION"

	run kubectl delete kepler -A --all || true
	run kubectl delete ns kepler || true
	run kubectl delete -n "$OPERATORS_NS" olm -l "$label"
	run kubectl delete operators,crd,clusterrole,clusterrolebinding -l "$label" -A || true
	run kubectl delete operators "$OPERATOR.$OPERATORS_NS" || true
	run kubectl delete leases 0d9cbc82.sustainable.computing.io -n "$OPERATORS_NS" || true
	run kubectl delete catalogsource kepler-operator-catalog -n "$OPERATORS_NS" || true

	ok "$OPERATOR version has been successfully uninstalled.\n"
}

main "$@"
