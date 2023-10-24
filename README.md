# kepler-operator

Kepler Operator installs Kepler and all required manifests on Kubernetes/OpenShift

## Description

[Kepler][kepler] (Kubernetes-based Efficient Power Level Exporter) is a Prometheus
exporter. It uses eBPF to probe CPU performance counters and Linux kernel
tracepoints.

These data and stats from cgroup and sysfs can then be fed into ML models to
estimate energy consumption by Pods.

Check out the project on GitHub ➡️ [Kepler][kepler]

## Getting Started

You’ll need a Kubernetes/OpenShift cluster to run against. You can use
[KIND](https://sigs.k8s.io/kind) or microshift to get a local cluster for
testing, or run against a remote cluster.

**Note:** Your controller will automatically use the current context in your
kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).


### To run a kind cluster locally

```sh
make cluster-up
```

### To run a microshift cluster locally

```sh
make cluster-up CLUSTER_PROVIDER=microshift
```

### Run kepler-operator locally out of cluster

```sh
make tools
make run
kubectl apply -k config/samples/
```

### Run using pre-published image

You can use the image from [quay.io](https://quay.io/repository/sustainable_computing_io/kepler-operator?tab=tags) to deploy kepler-operator.

```sh
make deploy OPERATOR_IMG=quay.io/sustainable_computing_io/kepler-operator:latest
kubectl apply -k config/samples/
```

Alternatively, if you like to build and use your own image,

```sh
make operator-build operator-push IMG_BASE=<some-registry>
make deploy IMG=<some-registry>/kepler-operator:tag
kubectl apply -k config/samples/
```

### Uninstall the operator

List the installed version and the releated resources that will be
deleted before uninstalling by running the uninstall script.
```sh
./hack/uninstall-operator.sh
```
Once the above is verified, uninstall the operator and all the related
resources by specifying the `--delete` flag.

```sh
./hack/uninstall-operator.sh  --delete

```

## Developer Docs
[Developer Docs][dev-docs] can be found under [docs/developer][dev-docs]

### Automated development environment

If don't have a `go` development environment, or you just want a reproducible
environment to start fresh, you can use [Docker Desktop](https://www.docker.com/products/docker-desktop/),
[Visual Studio Code](https://code.visualstudio.com), and the [Dev Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
extension to bring up an environment with `go`, `docker`, `kind`, `kubectl`,
`kustomize`, and `oc`.

To activate this, open the project from the command line
with `code .` and then press the **Reopen in Container** button when prompted.
(See the [Developing inside a Container](https://code.visualstudio.com/docs/devcontainers/containers)
documentation for more details).

## Contributing

You can contribute by:
* Raising [issues](https://github.com/sustainable-computing-io/kepler-operator/issues) related to kepler-operator
* Fixing issues by opening [Pull Requests](https://github.com/sustainable-computing-io/kepler-operator/pulls)

## License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

[kepler]: https://github.com/sustainable-computing-io/kepler
[dev-docs]: https://github.com/sustainable-computing-io/kepler-operator/tree/v1alpha1/docs/developer
