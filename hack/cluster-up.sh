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

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
declare -r PROJECT_ROOT
declare -r TMP_DIR="$PROJECT_ROOT/tmp"
declare -r CLUSTER_PROVIDER=${CLUSTER_PROVIDER:-kind}
declare -r VERSION=${VERSION:-v0.0.1}

function main() {

    if [ ! -d "${TMP_DIR}" ]
    then
        mkdir "${TMP_DIR}"
    fi

    cd "${TMP_DIR}"

    if [ -d local-dev-cluster ]
    then
        echo "using local local-dev-cluster"
    else
        echo "downloading local-dev-cluster with version ${VERSION}"
        git clone -b "${VERSION}" https://github.com/sustainable-computing-io/local-dev-cluster.git --depth=1
    fi

    cd local-dev-cluster
    ./main.sh up
}

main
