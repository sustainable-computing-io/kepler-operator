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
# Copyright 2023 The Kepler Contributors
#

set -eu -o pipefail

# constants
PROJECT_ROOT="$(git rev-parse --show-toplevel)"
GOOS="$(go env GOOS)"
GOARCH="$(go env GOARCH)"

declare -r PROJECT_ROOT GOOS GOARCH
declare -r LOCAL_BIN="$PROJECT_ROOT/tmp/bin"

# versions
declare -r KUSTOMIZE_VERSION=${KUSTOMIZE_VERSION:-v3.8.7}
declare -r CONTROLLER_TOOLS_VERSION=${CONTROLLER_TOOLS_VERSION:-v0.13.0}
declare -r OPERATOR_SDK_VERSION=${OPERATOR_SDK_VERSION:-v1.35.0}
declare -r YQ_VERSION=${YQ_VERSION:-v4.34.2}
declare -r CRDOC_VERSION=${CRDOC_VERSION:-v0.6.2}
declare -r OC_VERSION=${OC_VERSION:-4.13.0}
declare -r KUBECTL_VERSION=${KUBECTL_VERSION:-v1.28.4}
declare -r SHFMT_VERSION=${SHFMT_VERSION:-v3.7.0}
declare -r JQ_VERSION=${JQ_VERSION:-1.7}

# install
declare -r KUSTOMIZE_INSTALL_SCRIPT="https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
declare -r OPERATOR_SDK_INSTALL="https://github.com/operator-framework/operator-sdk/releases/download/$OPERATOR_SDK_VERSION/operator-sdk_${GOOS}_${GOARCH}"
declare -r YQ_INSTALL="https://github.com/mikefarah/yq/releases/download/$YQ_VERSION/yq_${GOOS}_${GOARCH}"
declare -r OC_URL="https://mirror.openshift.com/pub/openshift-v4/clients/ocp/$OC_VERSION"
declare -r JQ_INSTALL_URL="https://github.com/jqlang/jq/releases/download/jq-$JQ_VERSION"

source "$PROJECT_ROOT/hack/utils.bash"

go_install() {
	local pkg="$1"
	local version="$2"
	shift 2

	info "installing $pkg version: $version"

	GOBIN=$LOCAL_BIN go install "$pkg@$version" || {
		fail "failed to install $pkg - $version"
		return 1
	}
	ok "$pkg - $version was installed successfully"
	return 0
}

validate_version() {
	local cmd="$1"
	local version_arg="$2"
	local version_regex="$3"
	shift 3

	command -v "$cmd" >/dev/null 2>&1 || return 1

	[[ "$(eval "$cmd $version_arg" | grep -o "$version_regex")" =~ $version_regex ]] || {
		return 1
	}

	ok "$cmd matching $version_regex already installed"
}

version_kubectl() {
	kubectl version --client
}

install_kubectl() {
	local version_regex="Client Version: $KUBECTL_VERSION"

	validate_version kubectl "version --client" "$version_regex" && return 0

	info "installing kubectl version: $KUBECTL_VERSION"
	local install_url="https://dl.k8s.io/release/$KUBECTL_VERSION/bin/$GOOS/$GOARCH/kubectl"

	curl -Lo "$LOCAL_BIN/kubectl" "$install_url" || {
		fail "failed to install kubectl"
		return 1
	}
	chmod +x "$LOCAL_BIN/kubectl"
	ok "kubectl - $KUBECTL_VERSION was installed successfully"
}

version_kustomize() {
	kustomize version
}

install_kustomize() {
	validate_version kustomize version "$KUSTOMIZE_VERSION" && return 0

	info "installing kustomize version: $KUSTOMIZE_VERSION"
	(
		# NOTE: this handles softlinks properly
		cd "$LOCAL_BIN"
		curl -Ss $KUSTOMIZE_INSTALL_SCRIPT | bash -s -- "${KUSTOMIZE_VERSION:1}" .
	) || {
		fail "failed to install kustomize"
		return 1
	}
	ok "kustomize was installed successfully"
}

version_controller-gen() {
	controller-gen --version
}

install_controller-gen() {
	local version_regex="Version: $CONTROLLER_TOOLS_VERSION"
	validate_version controller-gen --version "$version_regex" && return 0
	go_install sigs.k8s.io/controller-tools/cmd/controller-gen "$CONTROLLER_TOOLS_VERSION"
}

version_operator-sdk() {
	operator-sdk version
}

install_operator-sdk() {
	local version_regex="operator-sdk version: \"$OPERATOR_SDK_VERSION\""

	validate_version operator-sdk version "$version_regex" && return 0

	info "installing operator-sdk with version: $OPERATOR_SDK_VERSION"
	curl -sSLo "$LOCAL_BIN/operator-sdk" "$OPERATOR_SDK_INSTALL" || {
		fail "failed to install operator-sdk"
		return 1
	}
	chmod +x "$LOCAL_BIN/operator-sdk"
	ok "operator-sdk was installed successfully"
}

version_govulncheck() {
	warn govulncheck - latest
}

install_govulncheck() {

	# NOTE: govulncheck does not have a -version flag, so checking
	# if it is available is "good" enough
	command -v govulncheck >/dev/null 2>&1 && {
		ok "govulncheck is already installed"
		return 0
	}
	go_install golang.org/x/vuln/cmd/govulncheck latest
}

version_yq() {
	yq --version
}

install_yq() {
	local version_regex="version ${YQ_VERSION}"

	validate_version yq --version "$version_regex" && return 0

	info "installing yq with version: $YQ_VERSION"
	curl -sSLo "$LOCAL_BIN/yq" "$YQ_INSTALL" || {
		fail "failed to install yq"
		return 1
	}
	chmod +x "$LOCAL_BIN/yq"
	ok "yq was installed successfully"
}

version_shfmt() {
	shfmt --version
}

install_shfmt() {
	validate_version shfmt --version "$SHFMT_VERSION" && return 0
	go_install mvdan.cc/sh/v3/cmd/shfmt "$SHFMT_VERSION"
}

version_crdoc() {
	crdoc --version
}

install_crdoc() {
	local version_regex="version $CRDOC_VERSION"

	validate_version crdoc --version "$version_regex" && return 0
	go_install fybrik.io/crdoc "$CRDOC_VERSION"
}

version_oc() {
	oc version --client
}

install_oc() {
	local version_regex="Client Version: $OC_VERSION"

	validate_version oc "version --client" "$version_regex" && return 0

	info "installing oc version: $OC_VERSION"
	local os="$GOOS"
	[[ $os == "darwin" ]] && os="mac"

	local install="$OC_URL/openshift-client-$os.tar.gz"
	# NOTE: tar should be extracted to a tmp dir since it also contains kubectl
	# which overwrites kubectl installed by install_kubectl above
	local oc_tmp="$LOCAL_BIN/tmp-oc"
	mkdir -p "$oc_tmp"
	curl -sNL "$install" | tar -xzf - -C "$oc_tmp" || {
		fail "failed to install oc"
		return 1
	}
	mv "$oc_tmp/oc" "$LOCAL_BIN/"
	chmod +x "$LOCAL_BIN/oc"
	rm -rf "$LOCAL_BIN/tmp-oc/"
	ok "oc was installed successfully"

}
version_jq() {
	jq --version
}
install_jq() {
	validate_version jq --version "$JQ_VERSION" && {
		return 0
	}
	local os="$GOOS"
	[[ $os == "darwin" ]] && os="macos"

	curl -sSLo "$LOCAL_BIN/jq" "$JQ_INSTALL_URL/jq-$os-$GOARCH" || {
		fail "failed to install jq"
	}
	chmod +x "$LOCAL_BIN/jq"
	ok "jq was installed successfully"
}

install_all() {
	info "installing all tools ..."
	local ret=0
	for tool in $(declare -F | cut -f3 -d ' ' | grep install_ | grep -v 'install_all'); do
		"$tool" || ret=1
	done
	return $ret
}

version_all() {

	header "Versions"
	for version_tool in $(declare -F | cut -f3 -d ' ' | grep version_ | grep -v 'version_all'); do
		local tool="${version_tool#version_}"
		local location=""
		location="$(command -v "$tool")"
		info "$tool -> $location"
		"$version_tool"
		echo
	done
	line "50"
}

main() {
	local op="${1:-all}"
	shift || true

	mkdir -p "$LOCAL_BIN"
	export PATH="$LOCAL_BIN:$PATH"

	# NOTE: skip installation if invocation is tools.sh version
	if [[ "$op" == "version" ]]; then
		version_all
		return $?
	fi

	install_"$op"
	version_"$op"
}

main "$@"
