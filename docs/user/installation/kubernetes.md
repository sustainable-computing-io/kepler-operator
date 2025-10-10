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

> **Important distinction**:
>
> - **prometheus-operator** (required) - Provides ServiceMonitor CRD that Kepler Operator uses to define how metrics should be scraped
> - **Prometheus** (optional but recommended) - The actual monitoring server that scrapes and stores metrics
> - **Grafana** (optional) - Visualization tool for creating dashboards from Prometheus metrics
>
> You need prometheus-operator to install Kepler Operator, but you don't strictly need a Prometheus instance unless you want to collect and query the metrics.

**Check if prometheus-operator is already installed:**

```bash
kubectl get crd servicemonitors.monitoring.coreos.com
```

If the CRD exists, prometheus-operator is installed. If not, continue with installation below.

**Installation using kube-prometheus-stack (Recommended)**:

The easiest way to install prometheus-operator along with Prometheus and Grafana is using the [kube-prometheus-stack](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack) Helm chart:

> **Note**: These commands may become outdated as the kube-prometheus-stack chart evolves. If the installation fails or behaves unexpectedly, please:
>
> - Check the [official kube-prometheus-stack documentation](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack)
> - [File an issue](https://github.com/sustainable-computing-io/kepler-operator/issues/new) or submit a PR to update this guide

```bash
# Add the prometheus-community Helm repository
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

# Install kube-prometheus-stack
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --wait \
  --timeout=10m
```

**Verify prometheus-operator is running:**

```bash
kubectl wait --for=condition=Ready pods -l app=kube-prometheus-stack-operator -n monitoring --timeout=300s
kubectl get crd servicemonitors.monitoring.coreos.com
```

**Alternative**: If you only need prometheus-operator without Prometheus/Grafana, follow the [prometheus-operator installation guide](https://prometheus-operator.dev/docs/getting-started/installation/).

### Validate Prerequisites

Before installing Kepler Operator, verify all required components are ready:

```bash
# Check Kubernetes version (must be >= 1.24.0)
kubectl version --short

# Check cert-manager is ready
kubectl get pods -n cert-manager
kubectl get crd certificates.cert-manager.io

# Check prometheus-operator is ready
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

## Prometheus Integration

### ServiceMonitor Label Requirements

When you create a PowerMonitor resource, the Kepler Operator automatically creates a ServiceMonitor to enable Prometheus scraping. However, **Prometheus must be configured to discover this ServiceMonitor**.

Most Prometheus installations (including kube-prometheus-stack) use label selectors to control which ServiceMonitors they scrape. If Prometheus is not scraping Kepler metrics, you may need to add the appropriate labels.

**Check Prometheus ServiceMonitor selector:**

```bash
# For kube-prometheus-stack
kubectl get prometheus -n monitoring -o yaml | grep -A 5 serviceMonitorSelector
```

Common configurations:

1. **kube-prometheus-stack default** - Requires `release: prometheus` label:

   ```yaml
   serviceMonitorSelector:
     matchLabels:
       release: prometheus
   ```

   **Fix**: Add the label to Kepler's ServiceMonitor:

   ```bash
   kubectl patch servicemonitor power-monitor -n power-monitor \
     --type=merge \
     -p '{"metadata":{"labels":{"release":"prometheus"}}}'
   ```

2. **Prometheus configured to scrape all ServiceMonitors** - No label required:

   ```yaml
   serviceMonitorSelector: {}
   ```

   This is achieved by setting:

   ```bash
   helm upgrade prometheus prometheus-community/kube-prometheus-stack \
     --namespace monitoring \
     --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
     --reuse-values
   ```

**Verify Prometheus is scraping Kepler:**

```bash
# Port-forward to Prometheus
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090

# Query for Kepler metrics (in another terminal)
curl -s 'http://localhost:9090/api/v1/query?query=kepler_node_cpu_watts' | grep -o '"status":"[^"]*"'
```

You should see `"status":"success"` with non-empty results.

For more details, see [Validating Prometheus Integration](../guides/validating-prometheus-integration.md).

## Next Steps

Now that the operator is installed, you can:

1. **[Create a PowerMonitor resource](../guides/power-monitor.md)** to deploy Kepler
2. **[Configure PowerMonitor](../configuring-kepler.md)** for your specific needs
3. **[Validate Prometheus Integration](../guides/validating-prometheus-integration.md)** to ensure metrics are being scraped
4. **[Set up Grafana dashboards](../guides/grafana-dashboard.md)** to visualize metrics

## Troubleshooting

If you encounter issues:

- **Operator pod not starting**: Check logs with `kubectl logs -n kepler-operator deployment/kepler-operator-controller-manager`
- **Webhook errors**: Ensure cert-manager is running and healthy
- **CRD not found**: Run `make helm-sync-crds` if installing from source

For more detailed troubleshooting, see the [Troubleshooting Guide](../guides/troubleshooting.md).

## Uninstallation

To uninstall Kepler Operator, see the [Uninstallation Guide](../reference/uninstallation.md).
