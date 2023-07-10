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

set -e
set -o pipefail
set -x

function prepareKubeConfig() {
    mkdir -p /tmp/.kube
    docker ps -a
    if [ "$CLUSTER_PROVIDER" == "microshift" ]
    then
        docker tag quay.io/sustainable_computing_io/kepler-operator:ci-build localhost:5001/kepler-operator:ci-build
        docker push localhost:5001/kepler-operator:ci-build
        docker exec -i microshift cat /var/lib/microshift/resources/kubeadmin/kubeconfig > /tmp/.kube/config
        IMG=registry:5000/kepler-operator:ci-build
    else
        docker tag quay.io/sustainable_computing_io/kepler-operator:ci-build localhost:5001/kepler-operator:ci-build
        docker push localhost:5001/kepler-operator:ci-build
        kind get kubeconfig --name=kind > /tmp/.kube/config
        IMG=localhost:5001/kepler-operator:ci-build
    fi
	cd config/manager && kustomize edit set image controller=${IMG}
}

function main() {
    prepareKubeConfig
}

main
