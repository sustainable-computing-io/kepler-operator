# Setting up Monitoring Stack on Kubernetes

This guide helps you set up Prometheus and Grafana on vanilla Kubernetes to visualize Kepler metrics.

## Overview

Kepler exports metrics in Prometheus format. To visualize these metrics with Grafana dashboards, you need:

1. **Prometheus** - To collect and store Kepler metrics
2. **prometheus-operator** - To enable ServiceMonitor-based discovery (optional but recommended)
3. **Grafana** - To visualize metrics with dashboards

**Note**: This is a prerequisite guide, not part of Kepler Operator itself. Kepler Operator assumes you already have a monitoring solution if you want metrics visualization.

## When Do You Need This?

You need a monitoring stack if you want to:

- Visualize energy consumption metrics in Grafana dashboards
- Query historical power consumption data
- Set up alerts based on energy usage
- Integrate with existing monitoring infrastructure

You can install Kepler Operator without a monitoring stack, but you won't be able to visualize metrics.

## Quick Start: kube-prometheus-stack (Recommended)

The easiest way to set up monitoring on Kubernetes is using the `kube-prometheus-stack` Helm chart, which includes:

- Prometheus Operator
- Prometheus instance
- Grafana
- AlertManager
- Pre-configured dashboards and alerts

### Install kube-prometheus-stack

```bash
# Add Helm repository
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

# Install the stack
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false

# Wait for all monitoring components to be ready
kubectl wait --for=condition=ready --timeout=180s pod -n monitoring --all
```

**Important**: The `serviceMonitorSelectorNilUsesHelmValues=false` setting allows Prometheus to discover ServiceMonitors from all namespaces, not just those created by this Helm release.

### Verify Installation

```bash
# Check pods are running
kubectl get pods -n monitoring

# Check Prometheus is accessible
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090
# Open http://localhost:9090 in your browser

# Check Grafana is accessible
kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80
# Open http://localhost:3000 in your browser
# Default credentials: admin / prom-operator
```

### Configure Kepler Integration

When installing Kepler Operator via Helm, enable ServiceMonitor creation:

```bash
helm install kepler-operator ./manifests/helm/kepler-operator \
  --namespace kepler-operator \
  --create-namespace \
  --set metrics.serviceMonitor.enabled=true
```

This creates a ServiceMonitor that Prometheus will automatically discover.

## Alternative: Manual Setup

If you prefer not to use the kube-prometheus-stack, you can install components individually.

### Install prometheus-operator

```bash
kubectl create -f https://github.com/prometheus-operator/prometheus-operator/releases/download/v0.76.0/bundle.yaml
```

### Install Prometheus Instance

Create a Prometheus instance that watches for ServiceMonitors:

```yaml
# prometheus-instance.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: prometheus
  namespace: monitoring
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: prometheus
rules:
- apiGroups: [""]
  resources:
  - nodes
  - nodes/metrics
  - services
  - endpoints
  - pods
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources:
  - configmaps
  verbs: ["get"]
- nonResourceURLs: ["/metrics"]
  verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: prometheus
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: prometheus
subjects:
- kind: ServiceAccount
  name: prometheus
  namespace: monitoring
---
apiVersion: monitoring.coreos.com/v1
kind: Prometheus
metadata:
  name: prometheus
  namespace: monitoring
spec:
  serviceAccountName: prometheus
  serviceMonitorSelector: {}  # Select all ServiceMonitors
  resources:
    requests:
      memory: 400Mi
  enableAdminAPI: false
```

Apply the configuration:

```bash
kubectl create namespace monitoring
kubectl apply -f prometheus-instance.yaml
```

### Install Grafana

Follow the official Grafana documentation to install Grafana on Kubernetes:

- [Grafana Installation Guide](https://grafana.com/docs/grafana/latest/setup-grafana/installation/kubernetes/)

Or use the Grafana Helm chart:

```bash
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update

helm install grafana grafana/grafana \
  --namespace monitoring \
  --set persistence.enabled=true \
  --set adminPassword=admin
```

### Configure Prometheus as Grafana Data Source

1. Access Grafana (default port 3000)
2. Navigate to **Configuration** → **Data Sources**
3. Click **Add data source**
4. Select **Prometheus**
5. Set URL to: `http://prometheus-operated.monitoring.svc:9090`
6. Click **Save & Test**

## Verification

### Check Prometheus is Scraping Kepler

After installing Kepler Operator and creating a PowerMonitor:

1. Access Prometheus UI (port-forward to port 9090)
2. Navigate to **Status** → **Targets**
3. Look for `kepler` targets - they should be in "UP" state

Alternatively, query for Kepler metrics:

```promql
kepler_node_cpu_joules_total
```

If you see results, Prometheus is successfully scraping Kepler metrics.

### Check ServiceMonitor

Verify the ServiceMonitor was created:

```bash
kubectl get servicemonitor -A | grep power-monitor
```

## Next Steps

Now that your monitoring stack is ready:

1. **[Install Kepler Operator](./kubernetes.md)** with ServiceMonitor enabled
2. **[Create a PowerMonitor](../guides/power-monitor.md)** to deploy Kepler
3. **[Import Grafana dashboards](../guides/grafana-dashboard.md)** to visualize metrics

## Troubleshooting

### Prometheus Not Discovering ServiceMonitor

If Prometheus isn't discovering the Kepler ServiceMonitor:

1. Check if prometheus-operator is running:

   ```bash
   kubectl get pods -n monitoring | grep prometheus-operator
   ```

2. Check Prometheus configuration for ServiceMonitor selector:

   ```bash
   kubectl get prometheus -n monitoring -o yaml | grep -A5 serviceMonitorSelector
   ```

   If it has specific label selectors, your ServiceMonitor must match those labels.

3. Check ServiceMonitor labels:

   ```bash
   kubectl get servicemonitor power-monitor -n power-monitor -o yaml
   ```

### No Kepler Metrics in Prometheus

If Kepler targets appear in Prometheus but no metrics are available:

1. Check Kepler pods are running:

   ```bash
   kubectl get pods -n power-monitor -l app.kubernetes.io/name=power-monitor-exporter
   ```

2. Verify Kepler is exposing metrics:

   ```bash
   kubectl port-forward -n power-monitor daemonset/power-monitor 28282:28282
   curl http://localhost:28282/metrics | grep kepler
   ```

3. Check ServiceMonitor configuration matches Kepler service:

   ```bash
   kubectl get svc power-monitor -n power-monitor
   kubectl get servicemonitor power-monitor -n power-monitor -o yaml
   ```

## External Resources

- [kube-prometheus-stack Helm Chart](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack)
- [Prometheus Operator Documentation](https://prometheus-operator.dev/)
- [Grafana Documentation](https://grafana.com/docs/)
- [Prometheus Documentation](https://prometheus.io/docs/)
