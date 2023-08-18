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

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
declare -r PROJECT_ROOT
declare -r LOCAL_BIN="$PROJECT_ROOT/tmp/bin"
declare -r CSV_FILE=bundle/manifests/kepler-operator.clusterserviceversion.yaml

source "$PROJECT_ROOT/hack/utils.bash"

# TODO: write a git action to create bundle on every pull request merge and make a commit to new branch
# TODO: Add below to bundle/metadata/annotations.yaml
#Annotattions for OpenShift version
#   com.redhat.openshift.versions: "v4.9-v4.12"

OPERATOR_IMG=${OPERATOR_IMG:-quay.io/sustainable_computing_io/kepler-operator}
VERSION=${VERSION:-"$(cat "$PROJECT_ROOT/VERSION")"}
BUNDLE_GEN_FLAGS=${BUNDLE_GEN_FLAGS:-}

declare -r OPERATOR_IMG VERSION BUNDLE_GEN_FLAGS

main() {
	cd "$PROJECT_ROOT"
	export PATH="$LOCAL_BIN:$PATH"

	# NOTE: get the current version in the bundle csv, which is the version that
	# this generation replaces in normal case

	local old_version=""
	old_version="$(yq -r .spec.version "$CSV_FILE")"
	local old_bundle_version="kepler-operator.v$old_version"

	# NOTE: if this is just a regeneration, then use the old replaces itself
	[[ "$old_version" == "$VERSION" ]] &&
		old_bundle_version=$(yq .spec.replaces "$CSV_FILE")

	info "Found old version: $old_bundle_version"

	info "Building bundle version $VERSION"
	run operator-sdk generate kustomize manifests --apis-dir=./pkg/api --verbose

	local gen_opts=()
	read -r -a gen_opts <<<"$BUNDLE_GEN_FLAGS"
	kustomize build config/manifests |
		sed \
			-e "s|<OPERATOR_IMG>|$OPERATOR_IMG|g" \
			-e "s|<OLD_BUNDLE_VERSION>|$old_bundle_version|g" |
		tee tmp/pre-bundle.yaml |
		operator-sdk generate bundle "${gen_opts[@]}"

	[[ "$old_version" != "$VERSION" ]] && {
		info "Replacing old version $old_version ->  $VERSION"
		sed \
			-e "s|replaces: .*|replaces: $old_bundle_version|g" \
			"$CSV_FILE" >"$CSV_FILE.tmp"
		mv "$CSV_FILE.tmp" "$CSV_FILE"
	}
	run tree bundle/

	info "updating ci/reviewers"
	cat <<-EOF >bundle/ci.yaml
		---
		reviewers:
		    - sustainable-computing-io
		    - husky-parul
		    - KaiyiLiu1234
		    - sthaha
		updateGraph: replaces-mode
	EOF

	info "Adding additional metadata annotations"
	cat <<-EOF >>bundle/metadata/annotations.yaml
		# Annotations for OpenShift version
		  com.redhat.openshift.versions: "v4.11-v4.14"
	EOF

	run operator-sdk bundle validate ./bundle \
		--select-optional name=operatorhub \
		--optional-values=k8s-version=1.25 \
		--select-optional suite=operatorframework
}

main "$@"
