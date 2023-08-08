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

source "$PROJECT_ROOT/hack/utils.bash"

main() {
	local operator="kepler-operator"
	local ns="openshift-operators"
	local version="${1:-v0.5.0}"

	kubectl get csv "${operator}.$version" -n "$ns" || {
		info "failed to find $version of $operator."
		line 50
		kubectl get csv | grep -E "$operator|NAME"
		info "\n$operator version found are ☝️"
		line 50
		return 1
	}

	info "Going to delete the following"
	line 50 heavy
	local label="operators.coreos.com/${operator}.$ns="
	run kubectl get ns kepler || true
	run kubectl get kepler -A
	run kubectl get -n "$ns" olm -l "$label"
	run kubectl get operator,crd,clusterrole,clusterrolebinding -l "$label" -A
	line 50 heavy

	run kubectl delete kepler -A --all
	run kubectl delete ns kepler || true
	run kubectl delete -n "$ns" olm -l "$label"
	run kubectl delete operator,crd,clusterrole,clusterrolebinding -l "$label" -A
	ok "$operator version has been successfully uninstalled."
}

main "$@"
