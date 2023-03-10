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


# To-do: write a git action to create bundle on every pull request merge and make a commit to new branch

# Add below to bundle/metadata/annotations.yaml
##Annotattions for OpenShift version
#   com.redhat.openshift.versions: "v4.9-v4.12"




## Uncomment below line if running this script as hack/bundle.sh
# export VERSION=$1
OPERATOR_IMAGE=quay.io/sustainable_computing_io/kepler-operator

make
make docker-build IMG=$OPERATOR_IMAGE:$VERSION
make generate
make manifests

tree config/manifests


kustomize build config/manifests | operator-sdk generate bundle --version $VERSION

tree bundle/

operator-sdk bundle validate ./bundle --select-optional name=operatorhub --optional-values=k8s-version=1.25 --select-optional suite=operatorframework

mv $(pwd)/bundle.Dockerfile bundle/

FILE=bundle/ci.yaml

if [ -f "$FILE" ]; then
    rm -rf $FILE
fi

cat <<EOF >>$FILE
---
reviewers:
    - sustainable-computing-io
    - husky-parul
    - KaiyiLiu1234
updateGraph: replaces-mode
EOF



