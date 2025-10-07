# Kepler Operator Helm Chart

Helm chart for deploying the Kepler Operator on Kubernetes.

> **Note**: This guide provides both `make` targets (for developers working from source) and direct `helm` commands (for users installing from packaged charts).

## Prerequisites

- Kubernetes >=1.24.0
- Helm >=3.0.0
- cert-manager >=1.18.0 (for webhook certificates)

## Installation

### Install cert-manager (if not already installed)

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.18.2/cert-manager.yaml
```

### Install Kepler Operator

**From source repository:**

```bash
make helm-install
```

**Using Helm directly:**

```bash
helm install kepler-operator ./manifests/helm/kepler-operator \
  --namespace kepler-operator \
  --create-namespace
```

**From packaged chart:**

```bash
helm install kepler-operator kepler-operator-0.21.0.tgz \
  --namespace kepler-operator \
  --create-namespace
```

### Install with custom values

```bash
helm install kepler-operator ./manifests/helm/kepler-operator \
  --namespace kepler-operator \
  --create-namespace \
  --set operator.image=quay.io/sustainable_computing_io/kepler-operator:v0.21.0 \
  --set kepler.image=quay.io/sustainable_computing_io/kepler:v0.11.0 \
  --set metrics.serviceMonitor.enabled=true
```

Or create a custom `values.yaml` and install:

```bash
helm install kepler-operator ./manifests/helm/kepler-operator \
  --namespace kepler-operator \
  --create-namespace \
  --values custom-values.yaml
```

## Configuration

Key configuration values:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `operator.image` | Operator image (full path with tag) | `quay.io/sustainable_computing_io/kepler-operator:0.21.0` |
| `operator.pullPolicy` | Image pull policy | `IfNotPresent` |
| `kepler.image` | Kepler image (full path with tag) | `quay.io/sustainable_computing_io/kepler:v0.11.0` |
| `kube-rbac-proxy.image` | Kube RBAC Proxy image (full path with tag) | `quay.io/brancz/kube-rbac-proxy:v0.19.0` |
| `replicaCount` | Number of operator replicas | `1` |
| `namespace` | Operator namespace | `kepler-operator` |
| `webhooks.enabled` | Enable admission webhooks | `true` |
| `webhooks.certManager.enabled` | Use cert-manager for webhook certificates | `true` |
| `metrics.serviceMonitor.enabled` | Enable Prometheus ServiceMonitor | `false` |

See [values.yaml](values.yaml) for complete list of configuration options.

## Creating a PowerMonitor Resource

After installing the operator, create a PowerMonitor resource:

```yaml
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
```

## Upgrading

**From source repository:**

```bash
make helm-install  # Uses helm upgrade --install
```

**Using Helm directly:**

```bash
helm upgrade kepler-operator ./manifests/helm/kepler-operator \
  --namespace kepler-operator
```

## Uninstalling

**From source repository:**

```bash
make helm-uninstall
```

**Using Helm directly:**

```bash
helm uninstall kepler-operator --namespace kepler-operator
```

## Development

For contributors working on the Helm chart, see the [Helm Chart Maintenance Guide](../../../docs/developer/helm-chart-maintenance.md).

### Testing

**Static validation:**

```bash
make helm-validate   # Run all validation tests (syntax, templates, CRD sync, resources)
make helm-template   # Preview rendered manifests
```

**End-to-end testing:**

```bash
# Full e2e test (requires cluster with cert-manager)
./tests/helm.sh

# See all options
./tests/helm.sh --help
```

### Syncing CRDs

```bash
make helm-sync-crds
```

## License

Apache License 2.0
