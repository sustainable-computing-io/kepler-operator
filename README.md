# Kepler Operator

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![CI Status](https://github.com/sustainable-computing-io/kepler-operator/actions/workflows/push.yaml/badge.svg)](https://github.com/sustainable-computing-io/kepler-operator/actions/workflows/push.yaml)
[![Codecov](https://codecov.io/gh/sustainable-computing-io/kepler-operator/graph/badge.svg?token=036JVLMN2V)](https://codecov.io/gh/sustainable-computing-io/kepler-operator)
[![Release](https://img.shields.io/github/v/release/sustainable-computing-io/kepler-operator)](https://github.com/sustainable-computing-io/kepler-operator/releases)

Kepler Operator is a Kubernetes operator that automates the deployment and management of [Kepler](https://github.com/sustainable-computing-io/kepler) on Kubernetes and OpenShift clusters.

## 🔍 What is Kepler?

[Kepler](https://github.com/sustainable-computing-io/kepler) (Kubernetes-based Efficient Power Level Exporter) is a Prometheus exporter
that measures energy consumption metrics at the container, pod, and node level in
Kubernetes clusters.

Check out the project on GitHub ➡️ [Kepler](https://github.com/sustainable-computing-io/kepler)

## 🚀 Getting Started

### For Users

#### Quick Start (Kubernetes with Helm)

Get Kepler up and running in minutes. For comprehensive installation instructions, prerequisites, configuration options, and troubleshooting, see the **[Kubernetes Installation Guide](docs/user/installation/kubernetes.md)**.

```sh
# 1. Install cert-manager (required for operator webhooks)
# For the latest version, see: https://cert-manager.io/docs/installation/
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.18.2/cert-manager.yaml

# Wait for cert-manager to be ready
kubectl wait --for=condition=available --timeout=120s deployment -n cert-manager --all

# 2. Install Prometheus Operator (required for ServiceMonitor support)
# This installs prometheus-operator + Prometheus + Grafana
# If you only need prometheus-operator, see the monitoring stack guide
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false

# Wait for monitoring stack to be ready
kubectl wait --for=condition=ready --timeout=180s pod -n monitoring --all

# 3. Install Kepler Operator
helm install kepler-operator \
  oci://quay.io/sustainable_computing_io/charts/kepler-operator \
  --namespace kepler-operator \
  --create-namespace

# Wait for operator to be ready
kubectl wait --for=condition=available --timeout=120s deployment -n kepler-operator --all

# 4. Deploy Kepler
# Note: PowerMonitor must be named "power-monitor" (enforced by operator)
kubectl apply -f https://raw.githubusercontent.com/sustainable-computing-io/kepler-operator/main/config/samples/kepler.system_v1alpha1_powermonitor.yaml

# Wait for Kepler pods to be running
kubectl wait --for=condition=ready --timeout=120s pod -n power-monitor --all

# 5. Verify installation
kubectl get pods -n power-monitor
```

**Next Steps:**

To ensure Kepler is working correctly, follow these validation guides:

- **[Validate Prometheus Integration](docs/user/guides/validating-prometheus-integration.md)** - Verify metrics are being collected (recommended)
- **[Setup Grafana Dashboards](docs/user/guides/grafana-dashboard.md)** - Visualize power consumption metrics
- [Configuration Options](docs/user/guides/power-monitor.md) - Customize Kepler deployment

**Need Help?**

- [Kubernetes Installation Guide](docs/user/installation/kubernetes.md) - Detailed prerequisites, configuration options, and installation steps
- [Validate Prometheus Integration](docs/user/guides/validating-prometheus-integration.md) - If Kepler is running but metrics aren't appearing
- [Troubleshooting Guide](docs/user/guides/troubleshooting.md) - Common issues and solutions

#### Quick Start (OpenShift)

Install from OperatorHub via the OpenShift Web Console. See the [OpenShift Installation Guide](docs/user/installation/openshift.md) for details.

#### User Documentation

For detailed installation, configuration, and usage instructions, see the [User Guides](docs/user/README.md):

- **Installation**:
  - [Kubernetes Installation (Helm)](docs/user/installation/kubernetes.md)
  - [Monitoring Stack Installation](docs/user/installation/monitoring-stack-kubernetes.md)
  - [OpenShift Installation (OperatorHub)](docs/user/installation/openshift.md)
- **Usage**:
  - [Creating PowerMonitor Resources](docs/user/guides/power-monitor.md)
  - [Configuring PowerMonitor](docs/user/guides/power-monitor.md)
  - [Setting up Grafana Dashboards](docs/user/guides/grafana-dashboard.md)
  - [Validating Prometheus Integration](docs/user/guides/validating-prometheus-integration.md)
  - [Upgrading](docs/user/guides/upgrading.md)
- **Reference**:
  - [API Reference](docs/user/reference/api.md)
  - [Troubleshooting](docs/user/guides/troubleshooting.md)
  - [Uninstallation](docs/user/reference/uninstallation.md)

### For Developers

#### Quick Start

```sh
# Setup local Kind cluster with prerequisites
make cluster-up

# Run operator locally
make run

# In another terminal, create a PowerMonitor
kubectl apply -k config/samples/
```

#### Developer Documentation

For contribution guidelines, architecture details, and development workflows, see the [Developer Documentation](docs/developer/README.md)

## 🤝 Contributing

You can contribute by:

- Reporting [issues](https://github.com/sustainable-computing-io/kepler-operator/issues)
- Fixing issues by opening [Pull Requests](https://github.com/sustainable-computing-io/kepler-operator/pulls)
- Improving documentation
- Sharing your success stories with Kepler

## 📝 License

This project is licensed under the Apache License 2.0 - see the [LICENSES](LICENSES) for details.
