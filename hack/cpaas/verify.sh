#!/usr/bin/env bash

set -eu -o pipefail

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
declare -r PROJECT_ROOT
declare OPERATOR_NS=${OPERATOR_NS:-"openshift-operators"}
declare OPERATOR_DEPLOY_NAME=${OPERATOR_DEPLOY_NAME:-"kepler-operator-controller"}

declare -r LOGS_DIR="tmp/verify"
declare -r IMAGE_COUNT=2

declare RELEASED=false
declare UNRELEASED=false
declare SHOW_HELP=false

source "$PROJECT_ROOT/hack/utils.bash"

validate_podman() {
	header "validating podman"

	# check if podman command is available
	command -v podman >/dev/null 2>&1 || {
		fail "No podman found"
		info "Please install podman or make sure its running"
		return 1
	}

	# check if podman is up and running
	! podman info >/dev/null 2>&1 && {
		fail "Podman is not running or not properly configured"
		return 1
	}
	return 0
}

init_logs_dir() {
	rm -rf "$LOGS_DIR-prev" || true
	mv "$LOGS_DIR" "$LOGS_DIR-prev" || true
	mkdir -p "$LOGS_DIR"
}

verify_images() {
	header "verifying images of Power Monitoring Operator"

	local powermon_images=""

	powermon_images=$(oc get deployment "$OPERATOR_DEPLOY_NAME" -n "$OPERATOR_NS" -o yaml | grep -o "registry.redhat.io/.*")

	$RELEASED && {
		info "verifying released images"
		powermon_images=$(echo "$powermon_images" | sort | uniq)
	}
	$UNRELEASED && {
		info "verifying unreleased images"
		powermon_images=$(echo "$powermon_images" |
			sed "s/registry.redhat.io/registry-proxy.engineering.redhat.com\/rh-osbs/" | sort | uniq)
	}
	[[ $(echo "$powermon_images" | wc -l) -ne "$IMAGE_COUNT" ]] && {
		err "expected $IMAGE_COUNT, images for Power Monitoring Operator. Found: \n $powermon_images"
		return 1
	}
	ok "found all $IMAGE_COUNT images for Power Monitoring Operator"

	info "pulling the images..."

	for img in $powermon_images; do
		local sanitize_img=""
		podman pull "$img" --tls-verify=false --arch=amd64 -q || {
			err "cannot pull the image: $img"
			return 1
		}
		sanitize_img=$(echo "$img" | sed 's/\//_/g')
		podman image inspect "$img" >"$LOGS_DIR/$sanitize_img-inspect.json"
	done

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
		--release)
			RELEASED=true
			shift
			;; # exit the loop
		--unrelease)
			UNRELEASED=true
			shift
			;;
		--namespace)
			shift
			OPERATOR_NS="$1"
			shift
			;;
		*)
			SHOW_HELP=true
			return 1
			;; # show usage on everything else
		esac
	done
	return 0
}

print_usage() {
	local scr
	scr="$(basename "$0")"

	read -r -d '' help <<-EOF_HELP || true
		Usage:
		  $scr  --namespace
		  $scr  --release
		  $scr  --unrelease
		 ─────────────────────────────────────────────────────────────────
		Options:
		  --namespace            namespace where operator is installed
		                         default: $OPERATOR_NS
		  --release              verify images of released version of Power Monitoring Operator
		  --unrelease            verify images of unreleased version of Power Monitoring Operator
	EOF_HELP

	echo -e "$help"
	return 0
}

main() {
	[[ -z "$*" ]] && {
		fail "No argument provided. Expect at least one of the arguments"
		print_usage
		return 1
	}

	parse_args "$@" || {
		print_usage
		fail "failed to parse the args"
		return 1
	}

	$SHOW_HELP && {
		print_usage
		return 0
	}

	init_logs_dir
	validate_podman || die "validating podman failed"

	verify_images || {
		fail "failed to verify images"
		return 1
	}
	ok "🎉 All images verified and pulled successfully. 🎉"
	info "Image details are saved in: $LOGS_DIR"

}
main "$@"
