#!/usr/bin/env bash
set -eu -o pipefail

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
declare -r PROJECT_ROOT
declare -r powermon_bundle_package_name="power-monitoring-operator-bundle-container"
declare -r OCP_VERSION=${OCP_VERSION:-'v4.13'}

source "$PROJECT_ROOT/hack/utils.bash"

function check_prereq {
  header "checking pre requisites..."
  if ! command -v jq &> /dev/null
  then
      err "jq could not be found. Please install jq first."
      exit
  fi
  if ! command -v oc &> /dev/null
  then
      err "oc could not be found. Please install oc first."
      exit
  fi
  if ! command -v podman &> /dev/null
  then
      err "podman could not be found. Please install podman first."
      exit
  fi
  ok "all prereqs ok.."
}

function get_index_image {
  local index_image="noimage"
  local package=$1
  local requested_ocp_version=$2
  messages=$(curl -s 'https://datagrepper.engineering.redhat.com/raw?topic=/topic/VirtualTopic.eng.ci.redhat-container-image.index.built&delta=824000&contains='$package)

  local i=0
  local bundle=""
  while true
  do
    message=$(echo $messages | jq --argjson i $i '.raw_messages[$i]')
    bundle=$(echo $message | jq -r '.msg.index.added_bundle_images[0]')
    version=$(echo $bundle| awk -F':' '{print $2}')
	ocp_version=$(echo $message | jq -r '.msg.index.ocp_version')
    if [[ $version == 0.1* ]] && [[ $ocp_version == "$requested_ocp_version" ]];
    then
      index_image=$(echo $message | jq -r '.msg.index.index_image')
      break
    else
      i=$(expr $i + 1)
    fi
  done
  ok "index image found: $index_image"
  brew_index_image=$(echo $index_image | awk -F':' '{print "brew.registry.redhat.io/rh-osbs/iib:"$2}')
  echo $brew_index_image
}

function add_brew_registry_credentials(){
  header "Getting credentials for brew registry"
  info "getting token from employee-token-manager"
  local tokens=$(curl --negotiate -u : https://employee-token-manager.registry.redhat.com/v1/tokens -s)
  if [[ "$?" != "0" ]] || [[ "$tokens" == "null" ]] || [[ "$tokens" == "" ]];
  then
	err "could not get token. \n\
    please use the following command to create a token and retry the script. \n\
    curl --negotiate -u : -X POST -H 'Content-Type: application/json' --data '{\"description\":\"for testing cpaas built powermon images on openshift 4 cluster\"}' https://employee-token-manager.registry.redhat.com/v1/tokens -s
	"
	exit -1
  fi
  ok "found token..."
  local username=$(echo $tokens | jq 'last' | jq -r  '.credentials.username')
  local password=$(echo $tokens | jq 'last' | jq -r  '.credentials.password')
  #info "username: $username, password: $password"

  info "getting auth from cluster"
  run oc get secret/pull-secret -n openshift-config -o json | jq -r '.data.".dockerconfigjson"' | base64 -d > authfile

  info "logging to brew registry"
  run podman login --authfile authfile --username "$username" --password "$password" brew.registry.redhat.io

  info "set auth to cluster"
  run oc set data secret/pull-secret -n openshift-config --from-file=.dockerconfigjson=authfile
}


function patch_operator_hub(){
  header "Patching Operator Hub to disable default sources.."
  run oc patch OperatorHub cluster --type json -p '[{"op": "add", "path": "/spec/disableAllDefaultSources", "value": true}]'
}

function create_icsp(){
  header "Creating ImageContentSourcePolicy to mirror images.."
  run cat <<EOF | oc apply -f -
  apiVersion: operator.openshift.io/v1alpha1
  kind: ImageContentSourcePolicy
  metadata:
    name: brew-registry
  spec:
    repositoryDigestMirrors:
    - mirrors:
      - brew.registry.redhat.io
      source: registry.redhat.io
    - mirrors:
      - brew.registry.redhat.io
      source: registry.stage.redhat.io
    - mirrors:
      - brew.registry.redhat.io
      source: registry-proxy.engineering.redhat.com
EOF
}

function add_catalog_source(){
  header "Adding CatalogSource for power monitoring with index Image..."
  local index_image=$1
  run cat <<EOF | oc apply -f -
  apiVersion: operators.coreos.com/v1alpha1
  kind: CatalogSource
  metadata:
    name: rc-powermon-operator-catalog
    namespace: openshift-marketplace
  spec:
    sourceType: grpc
    image: $index_image
    displayName: Openshift Power Monitoring
    publisher: Power Mon RC Images
EOF
}


main(){

  check_prereq
  header "Getting index image from CVP builds"
  local index_image="noimage"
  index_image=$(get_index_image $powermon_bundle_package_name $OCP_VERSION)
  ok "using index image: ${index_image}"
  add_brew_registry_credentials
  patch_operator_hub
  create_icsp
  add_catalog_source $index_image

  header "All Done"

  ok "Wait for few minutes and use OperatorHub in cluster to install Power Monitoring Operator."
}

main "$@"
