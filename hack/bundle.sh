#!/usr/bin/env bash

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
VERSION_REPLACED=${VERSION_REPLACED:-}
VERSION=${VERSION:-"$(cat "$PROJECT_ROOT/VERSION")"}
BUNDLE_GEN_FLAGS=${BUNDLE_GEN_FLAGS:-}

declare -r OPERATOR_IMG VERSION BUNDLE_GEN_FLAGS

main() {
	cd "$PROJECT_ROOT"
	export PATH="$LOCAL_BIN:$PATH"

	# NOTE: get the current version in the bundle csv, which is the version that
	# this generation replaces in normal case

	local version_replaced="$VERSION_REPLACED"
	[[ -z "$version_replaced" ]] && {
		version_replaced="$(yq -r .spec.version "$CSV_FILE")"
	}

	local old_bundle_version="kepler-operator.v$version_replaced"
	# NOTE: if this is just a regeneration, then use the old replaces itself
	[[ "$version_replaced" == "$VERSION" ]] &&
		old_bundle_version=$(yq .spec.replaces "$CSV_FILE")

	info "Found old version: $old_bundle_version"

	info "Building bundle version $VERSION"
	run operator-sdk generate kustomize manifests --verbose

	local gen_opts=()
	read -r -a gen_opts <<<"$BUNDLE_GEN_FLAGS"
	kustomize build config/manifests |
		sed \
			-e "s|<OPERATOR_IMG>|$OPERATOR_IMG|g" \
			-e "s|<KEPLER_IMG>|$KEPLER_IMG|g" \
			-e "s|<KEPLER_REBOOT_IMG>|$KEPLER_REBOOT_IMG|g" \
			-e "s|<KUBE_RBAC_PROXY_IMG>|$KUBE_RBAC_PROXY_IMG|g" \
			-e "s|<OLD_BUNDLE_VERSION>|$old_bundle_version|g" |
		tee tmp/pre-bundle.yaml |
		operator-sdk generate bundle "${gen_opts[@]}"

	[[ "$version_replaced" != "$VERSION" ]] && {
		info "Replacing old version $version_replaced ->  $VERSION"
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
		    - vprashar2929
		    - KaiyiLiu1234
		    - sthaha
		    - vimalk78
		updateGraph: replaces-mode
	EOF

	info "Adding additional metadata annotations"
	cat <<-EOF >>bundle/metadata/annotations.yaml
		# Annotations for OpenShift version
		  com.redhat.openshift.versions: "v4.16-v4.19"
	EOF

	run operator-sdk bundle validate ./bundle \
		--select-optional name=operatorhub \
		--optional-values=k8s-version=1.25 \
		--select-optional suite=operatorframework

	local csv_file=bundle/manifests/kepler-operator.clusterserviceversion.yaml

	if git diff --ignore-matching-lines='createdAt:' --exit-code "$csv_file" >/dev/null; then
		info "no changes to $(basename $csv_file) detected; resetting it"
		run git checkout -- "$csv_file"
	fi

}

main "$@"
