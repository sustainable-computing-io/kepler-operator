# Kepler Operator User Guides

Welcome to the Kepler Operator user documentation. This guide will help you install, configure, and operate Kepler on your Kubernetes or OpenShift cluster.

## What is Kepler?

[Kepler](https://github.com/sustainable-computing-io/kepler) (Kubernetes-based Efficient Power Level Exporter) is a Prometheus exporter that measures energy consumption metrics at the container, pod, and node level in Kubernetes clusters. This enables you to monitor and optimize your cluster's energy efficiency.

## What is Kepler Operator?

Kepler Operator is a Kubernetes operator that automates the deployment and lifecycle management of Kepler. It simplifies installation, configuration, and upgrades across both vanilla Kubernetes and OpenShift platforms.

## Getting Started

Choose your platform to begin:

### Installation Guides

- **[Kubernetes Installation (Helm)](installation/kubernetes.md)** - Install on vanilla Kubernetes using Helm
  - Includes prerequisites: cert-manager and prometheus-operator
  - Optional: Prometheus and Grafana for metrics visualization
- **[OpenShift Installation (OperatorHub)](installation/openshift.md)** - Install via OperatorHub/OLM
  - Uses OpenShift's built-in certificate management
  - Integrates with OpenShift monitoring stack

### Prerequisites

- **[Setting up Monitoring Stack on Kubernetes](installation/monitoring-stack-kubernetes.md)** - Guide for prometheus-operator, Prometheus, and Grafana
  - prometheus-operator is **REQUIRED** (Kepler Operator creates ServiceMonitor resources)
  - Prometheus and Grafana are optional (needed only for metrics visualization)

## Guides

Step-by-step tutorials for common tasks:

- **[Validating Prometheus Integration](guides/validating-prometheus-integration.md)** - Verify Prometheus is scraping Kepler metrics
- **[Setting up Grafana Dashboards](guides/grafana-dashboard.md)** - Visualize energy metrics
- **[Upgrading](guides/upgrading.md)** - Upgrade the operator (Helm and OLM)
- **[Troubleshooting Guide](guides/troubleshooting.md)** - Common issues and platform-specific solutions

### Experimental Features

⚠️ These features are experimental and may change in future versions:

- **[Redfish BMC Power Monitoring](guides/experimental/redfish.md)** - Platform-level power consumption via Baseboard Management Controllers

## Reference Documentation

Detailed feature documentation and specifications:

- **[PowerMonitor Resources](reference/power-monitor.md)** - Complete PowerMonitor CR specification and configuration options
- **[Custom ConfigMaps](reference/custom-configmaps.md)** - Advanced Kepler configuration using additionalConfigMaps
- **[API Reference](reference/api.md)** - Complete API specification
- **[Uninstallation](reference/uninstallation.md)** - Clean removal procedures

## Developer Documentation

If you want to contribute to Kepler Operator or understand its internals, see the [Developer Documentation](../developer/README.md).

## Quick Reference

### Kubernetes Quick Start

```bash
# 1. Install cert-manager (required)
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.18.2/cert-manager.yaml

# 2. Install prometheus-operator (required)
# Using kube-prometheus-stack (includes Prometheus + Grafana)
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false

# 3. Install Kepler Operator via Helm
helm install kepler-operator ./manifests/helm/kepler-operator \
  --namespace kepler-operator \
  --create-namespace

# 4. Create PowerMonitor
kubectl apply -f - <<EOF
apiVersion: kepler.system.sustainable.computing.io/v1alpha1
kind: PowerMonitor
metadata:
  name: power-monitor
spec:
  kepler:
    deployment:
      nodeSelector:
        kubernetes.io/os: linux
    config:
      logLevel: info
EOF
```

### OpenShift Quick Start

```bash
# Install from OperatorHub (UI)
# 1. Navigate to OperatorHub in OpenShift Console
# 2. Search for "Kepler Operator"
# 3. Click Install
# 4. Follow installation wizard

# Or install via CLI
# Follow the OpenShift Installation guide for detailed steps
```

## Getting Help

- **Issues**: [GitHub Issues](https://github.com/sustainable-computing-io/kepler-operator/issues)
- **Discussions**: [GitHub Discussions](https://github.com/sustainable-computing-io/kepler-operator/discussions)
- **Kepler Documentation**: [Kepler Docs](https://sustainable-computing.io/)

## Contributing

We welcome contributions! See the [main README](../../README.md) for contribution guidelines.

## License

Kepler Operator is licensed under the Apache License 2.0. See [LICENSE](../../LICENSES) for details.
