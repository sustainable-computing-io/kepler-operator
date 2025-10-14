# Installing Kepler Operator on Kubernetes

This guide walks you through installing the Kepler Operator on vanilla Kubernetes using Helm.

## Prerequisites

Before installing Kepler Operator, ensure you have:

- **Kubernetes cluster** running version 1.24.0 or higher
- **kubectl** configured to communicate with your cluster
- **Helm** version 3.0.0 or higher installed
- **cert-manager** version 1.18.0 or higher (for webhook certificates)
- **prometheus-operator** (for ServiceMonitor support - required)

### Required: cert-manager

Kepler Operator uses webhooks for validating PowerMonitor resources. These webhooks require TLS certificates, which are managed by cert-manager.

**Check if cert-manager is already installed:**

```bash
kubectl get pods -n cert-manager
```

**If cert-manager is not installed, install it:**

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.18.2/cert-manager.yaml
```

**Verify cert-manager is running:**

```bash
kubectl wait --for=condition=Ready pods --all -n cert-manager --timeout=300s
```

### Required: prometheus-operator

Kepler Operator requires prometheus-operator to be installed because it creates ServiceMonitor resources for metrics collection.

> **Important**: You only need **prometheus-operator** (for the ServiceMonitor CRD). A full monitoring stack with Prometheus and Grafana is optional.

**Check if prometheus-operator is already installed:**

```bash
kubectl get crd servicemonitors.monitoring.coreos.com
```

If the CRD exists, prometheus-operator is already installed and you can skip to [Installation Methods](#installation-methods).

**If not installed**, see the detailed guide:

📖 **[Setting up Monitoring Stack on Kubernetes](./monitoring-stack-kubernetes.md)**

This guide covers:

- Installing prometheus-operator (required)
- Optional: Installing Prometheus and Grafana for metrics visualization
- Configuring ServiceMonitor discovery
- Troubleshooting monitoring integration

### Validate Prerequisites

Before installing Kepler Operator, verify all required components are ready:

```bash
# Check Kubernetes version (must be >= 1.24.0)
kubectl version --short

# Check cert-manager is ready
kubectl get pods -n cert-manager
kubectl get crd certificates.cert-manager.io

# Check prometheus-operator CRD exists
kubectl get crd servicemonitors.monitoring.coreos.com
```

All commands should complete successfully before proceeding with installation.

## Installation Methods

You can install Kepler Operator using one of these methods:

### Method 1: From Source Repository (Recommended for Development)

If you have cloned the Kepler Operator repository:

```bash
make helm-install
```

This command automatically:

- Syncs CRDs to the Helm chart
- Installs the operator in the `kepler-operator` namespace
- Waits for the deployment to be ready

### Method 2: Using Helm with Local Chart

If you have access to the chart directory:

```bash
helm install kepler-operator ./manifests/helm/kepler-operator \
  --namespace kepler-operator \
  --create-namespace
```

### Method 3: From Packaged Helm Chart

If you have a packaged `.tgz` chart file:

```bash
helm install kepler-operator kepler-operator-<version>.tgz \
  --namespace kepler-operator \
  --create-namespace
```

Replace `<version>` with the actual version number (e.g., `0.21.0`).

## Custom Configuration

### Using Command-Line Flags

You can customize the installation using `--set` flags:

```bash
helm install kepler-operator ./manifests/helm/kepler-operator \
  --namespace kepler-operator \
  --create-namespace \
  --set operator.image=quay.io/sustainable_computing_io/kepler-operator:v0.21.0 \
  --set kepler.image=quay.io/sustainable_computing_io/kepler:v0.11.0 \
  --set metrics.serviceMonitor.enabled=true
```

### Using a Values File

For more complex configurations, create a custom `values.yaml`:

```yaml
# custom-values.yaml
operator:
  image: quay.io/sustainable_computing_io/kepler-operator:v0.21.0
  pullPolicy: IfNotPresent

kepler:
  image: quay.io/sustainable_computing_io/kepler:v0.11.0

metrics:
  serviceMonitor:
    enabled: true  # Enable if prometheus-operator is installed

webhooks:
  enabled: true
  certManager:
    enabled: true
```

Then install with:

```bash
helm install kepler-operator ./manifests/helm/kepler-operator \
  --namespace kepler-operator \
  --create-namespace \
  --values custom-values.yaml
```

### Key Configuration Options

| Parameter | Description | Default |
|-----------|-------------|---------|
| `operator.image` | Operator image with tag | `quay.io/sustainable_computing_io/kepler-operator:<version>` |
| `kepler.image` | Kepler image with tag | `quay.io/sustainable_computing_io/kepler:<version>` |
| `kube-rbac-proxy.image` | Kube RBAC Proxy image | `quay.io/brancz/kube-rbac-proxy:<version>` |
| `metrics.serviceMonitor.enabled` | Create ServiceMonitor for Prometheus | `false` |
| `webhooks.enabled` | Enable admission webhooks | `true` |
| `webhooks.certManager.enabled` | Use cert-manager for certificates | `true` |

For the complete list of configuration options, see the [Helm Chart README](../../../manifests/helm/kepler-operator/README.md).

## Verification

### Check Operator Deployment

Verify the operator is running:

```bash
kubectl get pods -n kepler-operator
```

You should see output similar to:

```text
NAME                                            READY   STATUS    RESTARTS   AGE
kepler-operator-controller-xxxxx-yyyyy          1/1     Running   0          1m
```

### Check Webhook Configuration

Verify webhooks are configured:

```bash
kubectl get validatingwebhookconfiguration | grep kepler
kubectl get mutatingwebhookconfiguration | grep kepler
```

### Check CRDs

Verify PowerMonitor CRD is installed:

```bash
kubectl get crd powermonitors.kepler.system.sustainable.computing.io
```

## Prometheus Integration (Optional)

If you installed Prometheus for metrics visualization, you need to ensure it can discover and scrape Kepler's ServiceMonitor.

For complete setup and troubleshooting instructions, see:

- 📖 **[Setting up Monitoring Stack on Kubernetes](./monitoring-stack-kubernetes.md)** - Initial setup
- 📖 **[Validating Prometheus Integration](../guides/validating-prometheus-integration.md)** - Verify metrics are being scraped

## Next Steps

Now that the operator is installed, you can:

1. **[Create a PowerMonitor resource](../reference/power-monitor.md)** to deploy Kepler
2. **[Custom ConfigMaps](../reference/custom-configmaps.md)** for advanced Kepler configuration
3. **[Set up Monitoring Stack](./monitoring-stack-kubernetes.md)** (optional) - If you want to visualize metrics
4. **[Validate Prometheus Integration](../guides/validating-prometheus-integration.md)** (if using Prometheus) - Ensure metrics are being scraped
5. **[Set up Grafana dashboards](../guides/grafana-dashboard.md)** (if using Grafana) - Visualize metrics

## Troubleshooting

If you encounter issues:

- **Operator pod not starting**: Check logs with `kubectl logs -n kepler-operator deployment/kepler-operator-controller`
- **Webhook errors**: Ensure cert-manager is running and healthy
- **CRD not found**: Run `make helm-sync-crds` if installing from source

For more detailed troubleshooting, see the [Troubleshooting Guide](../guides/troubleshooting.md).

## Uninstallation

To uninstall Kepler Operator, see the [Uninstallation Guide](../reference/uninstallation.md).
