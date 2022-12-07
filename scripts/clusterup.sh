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

function _deploy_kepler() {
    git clone https://github.com/sustainable-computing-io/kepler;
    kubectl apply -f kepler/manifests/kubernetes/deployment.yaml;
    _wait_containers_ready kepler;
    kubectl apply -f  kepler/manifests/kubernetes/keplerExporter-serviceMonitor.yaml;
    rm -rf kepler/
}

function _deploy_carbon_intensity_exporter(){
    kubectl apply -f https://github.com/sustainable-computing-io/carbon-aware-scaling-poc/blob/main/deploy/carbon-intensity-exporter.yaml
    _wait_containers_ready carbon_intensity_exporter;

}



_deploy_prometheus_operator;
#_deploy_kepler;
#_deploy_carbon_intensity_exporter;

#pip install kopf;
