set -e -u -o pipefail

trap cleanup INT

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
declare -r PROJECT_ROOT

# shellcheck source=/dev/null
source "$PROJECT_ROOT/tests/lib/utils.bash"

declare -r SCC_CRD_YAML="$PROJECT_ROOT/tests/lib/scc-crd.yaml"


declare CI_MODE=false
declare NO_DEPLOY=false
declare NO_BUILDS=false
declare SHOW_USAGE=false
declare LOGS_DIR="tmp/e2e"
declare OPERATORS_NS="operators"
declare TEST_TIMEOUT="15m"
declare BUNDLE_IMG="quay.io/sustainable_computing_io/kepler-operator-bundle"
declare OPERATOR_IMG="quay.io/sustainable_computing_io/kepler-operator"

print_usage() {
	local scr
	scr="$(basename "$0")"

	read -r -d '' help <<-EOF_HELP || true
		Usage:
		  $scr
		  $scr  --no-deploy
		  $scr  -h|--help


		Options:
		  -h|--help        show this help
		  --ci             run in CI mode
		  --no-deploy      do not build and deploy 0b0, useful for rerunning tests
		  --no-builds      skip building operator images, useful when operator image is already
		                   built and pushed
		  --ns NAMESPACE   namespace to deploy operators (default: $OPERATORS_NS)
		                   For running against openshift use --ns openshift-operators


	EOF_HELP

	echo -e "$help"
	return 0
}



tools_install(){

	# if [ $# != 3 ]; then
    # 	echo "Usage: tests/run-e2e.sh tools_install $(uname -m) $(uname)"
	# 	return 0
	# fi
	

	# echo $("$1" | awk '{print tolower($0)}')
	# echo $("$2" | awk '{print tolower($0)}')

	OS=$(uname | awk '{print tolower($0)}')

	case $(uname -m) in 
	 	x86_64) 
			ARCH="amd64"
			;;
		aarch64) 
			ARCH="arm64"
			;;
		*) 
			echo -n $(uname -m) 
			;;
	esac

	OPERATOR_SDK_DL_URL=https://github.com/operator-framework/operator-sdk/releases/download/${OPERATOR_SDK_VERSION}
    curl -LO ${OPERATOR_SDK_DL_URL}/operator-sdk_${OS}_${ARCH}
    chmod +x operator-sdk_${OS}_${ARCH} && sudo mv operator-sdk_${OS}_${ARCH} /usr/local/bin/operator-sdk

	operator-sdk version
	operator-sdk olm install --verbose
	kubectl apply -f ${SCC_CRD_YAML}

}
  




delete_olm_subscription() {
	header "Delete Old Deployments"

	$CI_MODE && {
		ok "skipping deletion of old deployment in CI mode"
		return 0
	}

	kubectl delete -n "$OPERATORS_NS" csv --all || true
	kubectl delete -n "$OPERATORS_NS" installplan -l operators.coreos.com/kepler-operator.operators= || true
}

build() {
	header "Build Operator Images"

	$NO_BUILDS && {
		info "skipping building of images"
		return 0
	}

	make operator-build OPERATOR_IMG=${OPERATOR_IMG}:latest
    make bundle bundle-build OPERATOR_IMG=${OPERATOR_IMG}:latest BUNDLE_IMG=${BUNDLE_IMG}:latest

	# docker inspect ${OPERATOR_IMAGE}
    # docker tag ${OPERATOR_IMAGE} ${OPERATOR_IMG}:ci-build

    # docker inspect ${BUNDLE_IMG}
    # docker tag ${BUNDLE_IMG} ${BUNDLE_IMG}:ci-build 

	# docker save -o /tmp/image.tar ${OPERATOR_IMG}
	# docker save -o /tmp/bundle-image.tar ${BUNDLE_IMG}


}

push(){
	make operator-push OPERATOR_IMG=${OPERATOR_IMG}:latest
	make bundle-push BUNDLE_IMG=${BUNDLE_IMG}:latest
}

deploy(){
	header "Deploy kepler-operator-bundle"

	operator-sdk run bundle ${BUNDLE_IMG}:latest  --install-mode AllNamespaces --namespace=operators
	kubectl apply -k config/samples/

}

bundle_cleanup(){
	operator-sdk cleanup kepler-operator --delete-all --delete-crds
	delete_olm_subscription
}

test_bundle(){
	wait_for_operators_ready ${OPERATORS_NS}
}

test(){
	echo "Test done"
}

# To-do test operator metrics. test kepler metrics
"$@"