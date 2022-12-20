set -ex pipefail

function _wait_containers_ready {
    echo "Waiting for all containers to become ready ..."
    namespace=$1
    kubectl wait --for=condition=Ready pod --all -n $namespace --timeout 12m
}

function _deploy_prometheus_operator() {
    git clone https://github.com/prometheus-operator/kube-prometheus;
    kubectl apply --server-side -f  kube-prometheus/manifests/setup;
    sleep 5;
    
    for file in $(ls kube-prometheus/manifests/prometheusOperator-*); do
        kubectl apply -f $file
    done
    for file in $(ls kube-prometheus/manifests/prometheus-*); do
        kubectl apply -f $file
    done
    for file in $(ls kube-prometheus/manifests/grafana-*); do
            kubectl apply -f $file
    done
    
    _wait_containers_ready monitoring
    rm -rf kube-prometheus/
}

# function _deploy_kepler() {
#     git clone https://github.com/sustainable-computing-io/kepler;
#     kubectl apply -f kepler/manifests/kubernetes/deployment.yaml;
#     _wait_containers_ready kepler;
#     kubectl apply -f  kepler/manifests/kubernetes/keplerExporter-serviceMonitor.yaml;
#     rm -rf kepler/
# }


# function _fetch_kind() {
#     CLUSTER_PROVIDER="kind"
#     CONFIG_OUT_DIR=${CONFIG_OUT_DIR:-"_output/manifests/${CLUSTER_PROVIDER}/generated"}
#     rm -rf ${CONFIG_OUT_DIR}
#     mkdir -p ${CONFIG_OUT_DIR}

#     # check CPU arch
#     PLATFORM=$(uname -m)
#     case ${PLATFORM} in
#     x86_64* | i?86_64* | amd64*)
#         ARCH="amd64"
#         ;;
#     ppc64le)
#         ARCH="ppc64le"
#         ;;
#     aarch64* | arm64*)
#         ARCH="arm64"
#         ;;
#     *)
#         echo "invalid Arch, only support x86_64, ppc64le, aarch64"
#         exit 1
#         ;;
#     esac

#     mkdir -p ${CONFIG_OUT_DIR}
#     KIND="${CONFIG_OUT_DIR}"/.kind
    
#     if [[ "$OSTYPE" == "darwin"* ]]; then
#             curl -LSs https://github.com/kubernetes-sigs/kind/releases/download/v$KIND_VERSION/kind-darwin-${ARCH} -o "$KIND"
#         else
#             curl -LSs https://github.com/kubernetes-sigs/kind/releases/download/v$KIND_VERSION/kind-linux-${ARCH} -o "$KIND"
#         fi
#         chmod +x "$KIND"
# }

# function _wait_kind_up {
#     echo "Waiting for kind to be ready ..."
    
#     while [ -z "$($CTR_CMD exec --privileged ${CLUSTER_NAME}-control-plane kubectl --kubeconfig=/etc/kubernetes/admin.conf get nodes -o=jsonpath='{.items..status.conditions[-1:].status}' | grep True)" ]; do
#         echo "Waiting for kind to be ready ..."
#         sleep 10
#     done
#     echo "Waiting for dns to be ready ..."
#     kubectl wait -n kube-system --timeout=12m --for=condition=Ready -l k8s-app=kube-dns pods
# }



_deploy_prometheus_operator;


