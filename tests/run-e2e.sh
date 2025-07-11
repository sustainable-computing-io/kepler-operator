#!/usr/bin/env bash
set -e -u -o pipefail

trap cleanup INT

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
declare -r PROJECT_ROOT

source "$PROJECT_ROOT/hack/utils.bash"

declare -r LOCAL_BIN="$PROJECT_ROOT/tmp/bin"
declare -r OPERATOR="kepler-operator"
declare -r OLM_CATALOG="kepler-operator-catalog"
declare -r VERSION=${VERSION:-"0.0.0-e2e"}
declare -r OPERATOR_DEPLOY_YAML="config/manager/base/manager.yaml"
declare -r KEPLER_CR="config/samples/kepler.system_v1alpha1_kepler.yaml"
declare -r OPERATOR_CSV="bundle/manifests/$OPERATOR.clusterserviceversion.yaml"
declare -r OPERATOR_DEPLOY_NAME="kepler-operator-controller"
declare -r OPERATOR_RELEASED_BUNDLE="quay.io/sustainable_computing_io/$OPERATOR-bundle"
declare -r TEST_IMAGES_YAML="tests/images.yaml"

declare IMG_BASE="${IMG_BASE:-localhost:5001/$OPERATOR}"
# NOTE: this vars are initialized in init_operator_img
declare OPERATOR_IMG=""
declare BUNDLE_IMG=""

declare CI_MODE=false
declare NO_DEPLOY=false
declare NO_BUILDS=false
declare SHOW_USAGE=false
declare LOGS_DIR="tmp/e2e"
declare POWERMONITOR_RELEASED_CR="tmp/power-monitor-released.yaml"
declare OPERATORS_NS="operators"
declare POWERMONITOR_NS="power-monitor"
declare TEST_TIMEOUT="15m"
# declare -a PORT_FORWARDED_PIDS=()

cleanup() {
	info "Cleaning up ..."
	# shell check  ignore word splitting when using jobs -p
	# shellcheck disable=SC2046
	[[ -z "$(jobs -p)" ]] || kill $(jobs -p) || true

	return 0
}

delete_olm_subscription() {
	header "Delete Old Deployments"

	$CI_MODE && {
		ok "skipping deletion of old deployment in CI mode"
		return 0
	}

	kubectl delete -n "$OPERATORS_NS" csv --all || true
	kubectl delete -n "$OPERATORS_NS" installplan,subscriptions,catalogsource \
		-l operators.coreos.com/$OPERATOR.operators= || true
	kubectl delete -n "$OPERATORS_NS" installplan,subscriptions,catalogsource \
		-l operators.coreos.com/$OPERATOR.openshift-operators= || true
	kubectl delete -n "$OPERATORS_NS" catalogsource $OLM_CATALOG || true
}

build_bundle() {
	header "Build Operator Bundle"

	$NO_BUILDS && {
		info "skipping building of images"
		return 0
	}

	run make operator-build bundle bundle-build \
		OPERATOR_IMG="$OPERATOR_IMG" \
		BUNDLE_IMG="$BUNDLE_IMG" \
		VERSION="$VERSION"
}

push_bundle() {
	header "Push Operator Bundle Images"
	$NO_BUILDS && {
		info "skipping pushing images"
		return 0
	}

	run make operator-push bundle-push \
		OPERATOR_IMG="$OPERATOR_IMG" \
		BUNDLE_IMG="$BUNDLE_IMG" \
		VERSION="$VERSION"

}

gather_olm() {
	header "Gather OLM resources"

	for x in $(kubectl api-resources --api-group=operators.coreos.com -o name); do
		run kubectl get "$x" -n "$OPERATORS_NS" -o yaml | tee "$LOGS_DIR/$x.yaml"
	done
}

cmd_upgrade() {
	header "Running Bundle Upgrade"
	kind_load_images
	delete_olm_subscription || true
	build_bundle
	push_bundle

	local -i ret=0

	local replaced_version=""
	replaced_version=$(yq ".spec.replaces| sub(\"$OPERATOR.v\"; \"\")" "$OPERATOR_CSV")
	replaced_version=$(echo "$replaced_version" | tr -d '"')

	local released_bundle="$OPERATOR_RELEASED_BUNDLE:$replaced_version"

	info "Running Released Bundle - $replaced_version"
	run operator-sdk run bundle "$released_bundle" \
		--install-mode AllNamespaces --namespace "$OPERATORS_NS" || {
		ret=$?
		line 50 heavy
		gather_olm || true
		fail "Running Released Bundle failed"
		return $ret
	}

	info "Creating a new Kepler CR"
	run kubectl apply -f "$KEPLER_CR"

	info "Creating new PowerMonitor CR"
	create_power_monitor "$replaced_version"

	wait_for_kepler || return 1
	wait_for_power_monitor || return 1

	info "Running Upgrade to new bundle"
	run operator-sdk run bundle-upgrade "$BUNDLE_IMG" \
		--namespace "$OPERATORS_NS" --use-http || {
		ret=$?
		line 50 heavy
		gather_olm || true
		fail "Running Bundle Upgrade failed"
		return $ret
	}

	wait_for_operator "$OPERATORS_NS"

	wait_until 10 10 "kepler images to be up to date" check_images

	wait_for_kepler || return 1
	wait_for_power_monitor || return 1

	return 0
}

# creating power monitor with fake cpu meter enabled for testing Operator Upgrade and invalid PowerMonitor scenarios
create_power_monitor() {
	local released_version="$1"
	shift 1

	local operator_csv_name="$OPERATOR.v$released_version"
	run kubectl get csv "$operator_csv_name"

	info "Getting PowerMonitor example from CSV for previous released version: $released_version"
	kubectl get csv "$operator_csv_name" -o yaml |
		yq -P '.metadata.annotations."alm-examples" | fromjson | .[] | select(.kind == "PowerMonitor")' |
		tee "$POWERMONITOR_RELEASED_CR"

	info "Adding addionalConfigMaps to enable fake cpu meter for upgrade tests"
	yq eval -i '.spec.kepler.config.additionalConfigMaps = [{"name": "power-monitor-config"}]' \
		"$POWERMONITOR_RELEASED_CR"

	info "Setting the Security mode as none for tests"
	yq eval -i '.spec.kepler.deployment.security.mode = "none"' "$POWERMONITOR_RELEASED_CR"

	cat "$POWERMONITOR_RELEASED_CR"

	run kubectl apply -f "$POWERMONITOR_RELEASED_CR"

	info "Creating PowerMonitor ConfigMap"
	run kubectl create -n "$POWERMONITOR_NS" -f - <<EOF
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: power-monitor-config
    data:
      config.yaml: |
        dev:
          fake-cpu-meter:
            enabled: true
EOF
}

wait_for_kepler() {
	header "Waiting for Kepler to be ready"
	wait_until 10 10 "kepler to be available" condition_check "Degraded" oc get kepler kepler \
		-o jsonpath="{.status.conditions[?(@.type=='Available')].status}" || {
		fail "Kepler is not ready"
		return 1
	}
	ok "Kepler is ready"
	return 0
}

wait_for_power_monitor() {
	header "Waiting for PowerMonitor to be ready"
	wait_until 10 10 "powermonitor to be available" condition_check "True" oc get powermonitor power-monitor \
		-o jsonpath="{.status.conditions[?(@.type=='Available')].status}" || {
		fail "PowerMonitor is not ready"
		return 1
	}
	ok "PowerMonitor is ready"
	return 0
}

check_images() {
	header "Checking Kepler Images"
	local actual_image=""
	local expected_image=""

	info "Checking PowerMonitor CR image"
	actual_image=$(kubectl get powermonitorinternals -o \
		jsonpath="{.items[*].spec.kepler.deployment.image}")
	expected_image=$(yq -r .spec.relatedImages[0].image "$OPERATOR_CSV")
	[[ "$actual_image" != "$expected_image" ]] && {
		fail "Kepler images are not up to date: actual: $actual_image != $expected_image"
		return 1
	}

	ok "Kepler images are up to date"
	return 0
}

run_bundle() {
	header "Running Bundle"
	local -i ret=0

	run operator-sdk run bundle "$BUNDLE_IMG" \
		--namespace "$OPERATORS_NS" --use-http || {
		ret=$?
		line 50 heavy
		gather_olm || true
		fail "Running Bundle failed"
		return $ret

	}
	return 0
}

log_events() {
	local ns="$1"
	shift
	kubectl get events -w \
		-o custom-columns=FirstSeen:.firstTimestamp,LastSeen:.lastTimestamp,Count:.count,From:.source.component,Type:.type,Reason:.reason,Message:.message \
		-n "$ns" | tee "$LOGS_DIR/$ns-events.log"
}

watch_operator_errors() {
	local err_log="$1"
	shift

	kubectl logs -f -n "$OPERATORS_NS" "deploy/$OPERATOR_DEPLOY_NAME" |
		grep -i error | tee "$err_log"
}

# run_e2e takes an optional set of args to be passed to go test
run_e2e() {
	header "Running e2e tests"

	local error_log="$LOGS_DIR/operator-errors.log"

	log_events "$OPERATORS_NS" &
	log_events "kepler-operator" &
	log_events "power-monitor" &
	watch_operator_errors "$error_log" &

	local ret=0
	run go test -v -failfast -timeout $TEST_TIMEOUT \
		./tests/e2e/... "$@" \
		2>&1 | tee "$LOGS_DIR/e2e.log" || ret=1

	# terminate both log_events
	{ jobs -p | xargs -I {} -- pkill -TERM -P {}; } || true
	wait
	sleep 1 #  wait for the Terminated logs to be written

	if [[ "$ret" -ne 0 ]]; then
		# logging of errors may not be immediate, so it is better to read logs again
		# than dumping the $error_log file
		sleep 2
		info "Operator Error Logs"
		line 50
		kubectl logs -n "$OPERATORS_NS" "deploy/$OPERATOR_DEPLOY_NAME" |
			grep -i error | tee "$error_log"
		line 50
	fi

	return $ret
}

# NOTE: ARGS_PARSED will be set by parse_args to the number of args
# it was able to parse
declare -i ARGS_PARSED=0

parse_args() {
	### while there are args parse them
	while [[ -n "${1+xxx}" ]]; do
		ARGS_PARSED+=1
		case $1 in
		-h | --help)
			SHOW_USAGE=true
			break
			;; # exit the loop
		--) break ;;
		--no-deploy)
			NO_DEPLOY=true
			shift
			;;
		--no-builds)
			NO_BUILDS=true
			shift
			;;
		--ci)
			CI_MODE=true
			shift
			;;
		--image-base)
			shift
			IMG_BASE="$1"
			shift
			ARGS_PARSED+=1
			;;
		--ns)
			shift
			OPERATORS_NS=$1
			shift
			ARGS_PARSED+=1
			;;
		*)
			return 1
			;; # show usage on everything else
		esac
	done

	return 0
}

init_operator_img() {
	OPERATOR_IMG="$IMG_BASE/$OPERATOR:$VERSION"
	BUNDLE_IMG="$IMG_BASE/$OPERATOR-bundle:$VERSION"
	declare -r OPERATOR_IMG BUNDLE_IMG
}

cmd_help() {
	local scr
	scr="$(basename "$0")"

	read -r -d '' help <<-EOF_HELP || true
		🔆 Usage:
		  $scr <command> [OPTIONS] -- [GO TEST ARGS]
		  $scr -h|--help

		📋 Commands:
		  e2e             run end-to-end tests
		  upgrade         run bundle upgrade tests

		💡 Examples:
		  # run e2e tests
		  ❯   $scr e2e

		  # run upgrade tests
		  ❯   $scr upgrade

		  # run only invalid test with e2e
		  ❯   $scr e2e -- -run TestInvalid

		  # do not redeploy operator and run only invalid test
		  ❯   $scr e2e --no-deploy -- -run TestInvalid

		⚙️ Options:
		  -h|--help        show this help
		  --ci             run in CI mode
		  --no-deploy      do not build and deploy Operator; useful for rerunning tests
		  --no-builds      skip building operator images; useful when operator image is already
		                   built and pushed
		  --ns NAMESPACE   namespace to deploy operators (default: $OPERATORS_NS)
		                   E.g. running against openshift use --ns openshift-operators
	EOF_HELP

	echo -e "$help"
	return 0
}

init_logs_dir() {
	rm -rf "$LOGS_DIR-prev"
	mv "$LOGS_DIR" "$LOGS_DIR-prev" || true
	mkdir -p "$LOGS_DIR"
}

restart_operator() {
	header "Restart Operator deployment"

	ensure_deploy_img_is_always_pulled || return 1
	local deployment="deployment/$OPERATOR_DEPLOY_NAME"

	info "scale down Operator"
	run kubectl scale -n "$OPERATORS_NS" --replicas=0 "$deployment"
	run kubectl wait -n "$OPERATORS_NS" --for=delete \
		pods -l app.kubernetes.io/component=manager --timeout=60s

	update_crds

	info "scale up"
	kubectl scale -n "$OPERATORS_NS" --replicas=1 "$deployment"
	wait_for_operator "$OPERATORS_NS"

	ok "Operator deployment restarted"

}

update_crds() {
	# try replacing any installed crds; failure is often because the
	# CRDs are absent and in that case, try creating and fail if that fails

	run kubectl apply --server-side --force-conflicts -k config/crd
	run kubectl wait --for=condition=Established crds --all --timeout=120s

	return 0
}

kind_load_image() {
	local img="$1"

	run docker pull "$img"
	run kind load docker-image "$img"
	return 0
}

kind_load_images() {
	header "Load Images"
	while read -r img; do
		kind_load_image "$img"
	done < <(yq -r .spec.relatedImages[].image "$OPERATOR_CSV")

	info "loading additional images from $TEST_IMAGES_YAML"
	while read -r img; do
		kind_load_image "$img"
	done < <(yq -r .images[].image "$TEST_IMAGES_YAML")

	return 0
}

deploy_operator() {
	header "Build and Deploy Operator"
	ensure_imgpullpolicy_always_in_yaml
	kind_load_images
	delete_olm_subscription || true
	build_bundle
	push_bundle
	run_bundle
	wait_for_operator "$OPERATORS_NS"
}

ensure_imgpullpolicy_always_in_yaml() {
	$CI_MODE && {
		ok "skipping check of imagePullPolicy in deployment yaml"
		return 0
	}

	local pull_policy
	pull_policy=$(grep '\s\+imagePullPolicy:' "$OPERATOR_DEPLOY_YAML" | tr -d ' ' | cut -f2 -d:)

	[[ "$pull_policy" != "Always" ]] && {
		info "Modify $OPERATOR_DEPLOY_YAML imagePullPolicy -> Always"
		info "  ❯ sed -e 's|imagePullPolicy: .*|imagePullPolicy: Always|g' -i $OPERATOR_DEPLOY_YAML"
		warn "Deployment's imagePullPolicy must be Always instead of $pull_policy"
		return 1
	}

	ok "Operator deployment yaml imagePullPolicy is Always"
}

ensure_deploy_img_is_always_pulled() {
	$CI_MODE && {
		ok "skipping check of imagePullPolicy of Operator deployment"
		return 0
	}

	local pull_policy
	pull_policy=$(kubectl get deploy/$OPERATOR_DEPLOY_NAME \
		-n "$OPERATORS_NS" \
		-ojsonpath='{.spec.template.spec.containers[0].imagePullPolicy}')

	if [[ "$pull_policy" != "Always" ]]; then
		info "Edit $OPERATOR_DEPLOY_YAML imagePullPolicy and redeploy"
		info "  ❯ sed -e 's|imagePullPolicy: .*|imagePullPolicy: Always|g' -i $OPERATOR_DEPLOY_YAML"
		warn "Deployment's imagePullPolicy must be Always instead of $pull_policy"
		return 1
	fi
	ok "Operator deployment imagePullPolicy is Always"
}

reject_invalid() {
	local invalid_kepler='invalid-pre-test-kepler'

	# ensure that applying an invalid kepler will be rejected by the webhook
	{
		# NOTE: || true ignores pipefail so that non-zero exit code will not be reported
		# when kubectl apply fails (as expected)
		sed -e "s|name: kepler$|name: $invalid_kepler|g" "$KEPLER_CR" |
			kubectl apply -f- 2>&1 || true

	} | tee /dev/stderr |
		grep -q "BadRequest.* admission webhook .* denied the request" && return 0

	kubectl delete kepler "$invalid_kepler" || true
	return 1
}

# wait_for_operator requires the namespace where the operator is installed
wait_for_operator() {
	header "Waiting  for Kepler Operator ($OPERATORS_NS) to be Ready"
	local deployment="deploy/$OPERATOR_DEPLOY_NAME"

	wait_until 30 10 "operator to run" \
		kubectl -n "$OPERATORS_NS" rollout status "$deployment"

	run kubectl wait -n "$OPERATORS_NS" --for=condition=Available \
		--timeout=300s "$deployment"

	# NOTE: ensure that operator is actually ready by creating an invalid kepler
	# and wait until the operator is able to reconcile
	info "Ensure that webhooks are installed and working"

	wait_until 30 10 "webhooks to be ready" reject_invalid

	ok "Operator up and running"
}

print_config() {
	header "Test Configuration"
	cat <<-EOF
		  image base:      $IMG_BASE
		  operator image:  $OPERATOR_IMG
		  bundle:          $BUNDLE_IMG
		  CI Mode:         $CI_MODE
		  Skip Builds:     $NO_BUILDS
		  Skip Deploy:     $NO_DEPLOY
		  Operator namespace: $OPERATORS_NS
		  Logs directory: $LOGS_DIR

	EOF
	line 50
}

cmd_e2e() {
	if $NO_DEPLOY; then
		restart_operator || die "restarting operator failed 🤕"
	else
		deploy_operator
	fi

	local -i ret=0
	run_e2e "$@" || ret=$?

	info "e2e test - exit code: $ret"
	line 50 heavy
	return $ret
}

main() {
	export PATH="$LOCAL_BIN:$PATH"

	local fn=${1:-''}
	shift

	parse_args "$@" || die "parse args failed"
	# eat up all the parsed args so that the rest can be passed to go test
	shift $ARGS_PARSED
	$SHOW_USAGE && {
		cmd_help
		exit 0
	}

	cd "$PROJECT_ROOT"

	local cmd_fn="cmd_$fn"
	if ! is_fn "$cmd_fn"; then
		err "unknown command: $fn"
		cmd_help
		return 1
	fi

	init_operator_img
	init_logs_dir
	print_config

	$cmd_fn "$@" || return 1

	return 0
}

main "$@"
