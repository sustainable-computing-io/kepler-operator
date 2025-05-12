# Kepler Operator

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![CI Status](https://github.com/sustainable-computing-io/kepler-operator/actions/workflows/publish-images.yaml/badge.svg)](https://github.com/sustainable-computing-io/kepler-operator/actions/workflows/publish-images.yaml)
[![Codecov](https://codecov.io/gh/sustainable-computing-io/kepler-operator/graph/badge.svg?token=036JVLMN2V)](https://codecov.io/gh/sustainable-computing-io/kepler-operator)
[![Release](https://img.shields.io/github/v/release/sustainable-computing-io/kepler-operator)](https://github.com/sustainable-computing-io/kepler-operator/releases)

Kepler Operator is a Kubernetes operator that automates the deployment and management of [Kepler](https://github.com/sustainable-computing-io/kepler) on Kubernetes and OpenShift clusters.

## 🔍 What is Kepler?

[Kepler](https://github.com/sustainable-computing-io/kepler) (Kubernetes-based Efficient Power Level Exporter) is a Prometheus
exporter. It uses eBPF to probe CPU performance counters and Linux kernel
tracepoints.

These data and stats from cgroup and sysfs can then be fed into ML models to
estimate energy consumption by Pods.

Check out the project on GitHub ➡️ [Kepler](https://github.com/sustainable-computing-io/kepler)

## 🚀 Getting Started

You'll need a Kubernetes or OpenShift cluster. For local testing, use [KIND](https://sigs.k8s.io/kind). Otherwise, connect to a remote cluster.

**Note:** The operator uses the current kubeconfig context (check with `kubectl cluster-info`).

### 💻 Using Kind Cluster

To quickly set up a local environment with Kind:

```sh
make cluster-up
```

### 🛠️ Local Development

To run the operator locally outside the cluster:

```sh
make tools
make run
kubectl apply -k config/samples/
```

### On Vanilla Kubernetes

Deploy the operator and its dependencies:

```sh
make tools
kubectl create -f https://github.com/prometheus-operator/prometheus-operator/releases/download/v0.76.0/bundle.yaml
kubectl create -f https://github.com/jetstack/cert-manager/releases/download/v1.15.3/cert-manager.yaml
make deploy
kubectl apply -k config/samples/
```

### 📦 Using Pre-published Image

You can use the pre-built image from quay.io:

```sh
make deploy OPERATOR_IMG=quay.io/sustainable_computing_io/kepler-operator:v1alpha1
kubectl apply -k config/samples/
```

Alternatively, build and use your own image:

```sh
make operator-build operator-push IMG_BASE=<your-registry>
make deploy IMG_BASE=<your-registry>/kepler-operator:<tag>
kubectl apply -k config/samples/
```

### On OpenShift

Deploy the operator on OpenShift:

```sh
make tools
make operator-build operator-push \
     bundle bundle-build bundle-push \
     IMG_BASE=<your-registry> VERSION=0.0.0-dev
./tmp/bin/operator-sdk run bundle <your-registry>/kepler-operator-bundle:0.0.0-dev \
  --install-mode AllNamespaces --namespace openshift-operators --skip-tls
```

## 🗑️ Uninstallation

To list the installed resources before deletion:

```sh
./hack/uninstall-operator.sh
```

To completely remove the operator and all related resources:

```sh
./hack/uninstall-operator.sh --delete
```

## 📚 Developer Documentation

[Developer documentation](https://github.com/sustainable-computing-io/kepler-operator/tree/v1alpha1/docs/developer) is available for those who want to contribute to the codebase or understand its internals.

## 🤝 Contributing

You can contribute by:

* Reporting [issues](https://github.com/sustainable-computing-io/kepler-operator/issues)
* Fixing issues by opening [Pull Requests](https://github.com/sustainable-computing-io/kepler-operator/pulls)
* Improving documentation
* Sharing your success stories with Kepler

## 📝 License

This project is licensed under the Apache License 2.0 - see the [LICENSES](LICENSES) for details.
