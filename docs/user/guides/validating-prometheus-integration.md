# Validating Prometheus Integration

This guide walks you through verifying that Prometheus is successfully scraping Kepler metrics after deploying a PowerMonitor resource.

## Prerequisites

Before validating Prometheus integration, ensure you have:

- PowerMonitor deployed and Kepler pods running
- Prometheus installed (via kube-prometheus-stack or manual installation)
- `kubectl` access to your cluster

## Validation Steps

### Step 1: Verify ServiceMonitor Creation

When you create a PowerMonitor resource, the Kepler Operator automatically creates a ServiceMonitor in the same namespace where Kepler is deployed.

**Check ServiceMonitor exists:**

```bash
kubectl get servicemonitor -n power-monitor
```

**Expected output:**

```text
NAME            AGE
power-monitor   5m
```

**Inspect ServiceMonitor details:**

```bash
kubectl describe servicemonitor power-monitor -n power-monitor
```

**Key fields to verify:**

```yaml
Spec:
  Endpoints:
    Port: http          # Should be "http" (port 28282)
    Scheme: http
  Selector:
    Match Labels:
      app.kubernetes.io/component: exporter
      app.kubernetes.io/managed-by: kepler-operator
```

**Troubleshooting:**

- If ServiceMonitor doesn't exist, check operator logs:

  ```bash
  kubectl logs -n kepler-operator deployment/kepler-operator-controller
  ```

- Verify prometheus-operator CRDs are installed:

  ```bash
  kubectl get crd servicemonitors.monitoring.coreos.com
  ```

---

### Step 2: Verify Kepler Service

The ServiceMonitor references a Kubernetes Service. Verify it exists and is configured correctly:

```bash
kubectl get svc -n power-monitor
```

**Expected output:**

```text
NAME            TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)     AGE
power-monitor   ClusterIP   None         <none>        28282/TCP   5m
```

**Check service endpoints:**

```bash
kubectl get endpoints power-monitor -n power-monitor
```

**Expected output should show Kepler pod IPs:**

```text
NAME            ENDPOINTS           AGE
power-monitor   10.244.0.10:28282   5m
```

**Troubleshooting:**

- If no endpoints, check Kepler pods are running:

  ```bash
  kubectl get pods -n power-monitor -l app.kubernetes.io/name=power-monitor-exporter
  ```

- Verify pod labels match the service selector:

  ```bash
  kubectl get pods -n power-monitor -l app.kubernetes.io/name=power-monitor-exporter --show-labels
  ```

---

### Step 3: Verify Prometheus Discovery

Check if Prometheus has discovered the ServiceMonitor.

#### Method 1: Check Prometheus Logs

```bash
# Get Prometheus pod name (adjust namespace if different)
PROM_POD=$(kubectl get pods -n monitoring -l app.kubernetes.io/name=prometheus -o jsonpath='{.items[0].metadata.name}')

# Check logs for ServiceMonitor discovery
kubectl logs -n monitoring $PROM_POD prometheus | grep power-monitor
```

**Expected output:**

```text
level=info msg="Using pod service account via in-cluster config" discovery=kubernetes config=serviceMonitor/power-monitor/power-monitor/0
```

#### Method 2: Query Prometheus Targets API

Port-forward to Prometheus:

```bash
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090
```

In another terminal, query the targets API:

```bash
curl -s http://localhost:9090/api/v1/targets | \
  jq '.data.activeTargets[] | select(.labels.job | contains("power-monitor"))'
```

**Expected output:**

```json
{
  "discoveredLabels": {
    "__address__": "10.244.0.10:28282",
    "__meta_kubernetes_endpoint_port_name": "http",
    "__meta_kubernetes_namespace": "power-monitor",
    "__meta_kubernetes_pod_name": "power-monitor-xxxxx",
    "__meta_kubernetes_service_name": "power-monitor",
    "job": "power-monitor"
  },
  "labels": {
    "instance": "kind-control-plane",
    "job": "power-monitor",
    "namespace": "power-monitor"
  },
  "scrapePool": "serviceMonitor/power-monitor/power-monitor/0",
  "scrapeUrl": "http://10.244.0.10:28282/metrics",
  "lastError": "",
  "lastScrape": "2025-10-09T01:45:00.000Z",
  "lastScrapeDuration": 0.012345678,
  "health": "up"
}
```

**Key indicators of success:**

- `"health": "up"` - Target is healthy
- `"lastError": ""` - No scraping errors
- `scrapeUrl` shows correct IP and port (28282)

---

### Step 4: Query Kepler Metrics

Verify that Kepler metrics are actually available in Prometheus.

#### List All Kepler Metrics

With Prometheus port-forward still active:

```bash
curl -s http://localhost:9090/api/v1/label/__name__/values | \
  jq -r '.data[]' | grep ^kepler_
```

**Expected output (11+ metrics):**

```text
kepler_build_info
kepler_node_cpu_active_joules_total
kepler_node_cpu_active_watts
kepler_node_cpu_idle_joules_total
kepler_node_cpu_idle_watts
kepler_node_cpu_info
kepler_node_cpu_joules_total
kepler_node_cpu_usage_ratio
kepler_node_cpu_watts
kepler_pod_cpu_joules_total
kepler_pod_cpu_watts
```

**NOTE:** The specific metric names shown above are examples and may vary depending on your Kepler version
and PowerMonitor configuration. The key validation is that you see multiple metrics starting
with `kepler_` prefix. The number and types of metrics depend on your PowerMonitor
`metricLevels` configuration (e.g., node, pod, container, process).

#### Query Specific Metrics

**Query node power consumption:**

```bash
curl -s 'http://localhost:9090/api/v1/query?query=kepler_node_cpu_active_watts' | jq .
```

**Expected response:**

```json
{
  "status": "success",
  "data": {
    "resultType": "vector",
    "result": [
      {
        "metric": {
          "__name__": "kepler_node_cpu_active_watts",
          "instance": "node-name",
          "job": "power-monitor",
          "node_name": "node-name",
          "path": "/host/sys/class/powercap/intel-rapl:0",
          "zone": "package"
        },
        "value": [1728456789.123, "0.262"]
      }
    ]
  }
}
```

The `value` array contains `[timestamp, "metric_value"]`. In this example, the node is consuming 0.262 watts of active CPU power.

**Query pod power consumption:**

```bash
curl -s 'http://localhost:9090/api/v1/query?query=kepler_pod_cpu_joules_total' | jq .
```

---

### Step 5: Verify Prometheus Configuration

Check if Prometheus is configured to discover ServiceMonitors from your namespace.

```bash
kubectl get prometheus -n monitoring -o yaml
```

**Check these configuration fields:**

```yaml
spec:
  # Should allow discovering ServiceMonitors from all namespaces
  # or specifically include "power-monitor"
  serviceMonitorNamespaceSelector: {}  # Empty means all namespaces

  # Should allow discovering ServiceMonitors with any labels
  # or specifically match Kepler's ServiceMonitor labels
  serviceMonitorSelector: {}  # Empty means all ServiceMonitors
```

**If you see specific selectors:**

```yaml
serviceMonitorNamespaceSelector:
  matchLabels:
    monitoring: enabled  # Only discovers from namespaces with this label
```

**Solution:** Add the required label to your namespace:

```bash
kubectl label namespace power-monitor monitoring=enabled
```

Or configure Prometheus to use empty selectors (discover from all namespaces):

```bash
helm upgrade prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false
```

---

## Validation Checklist

Use this checklist to verify complete Prometheus integration:

- [ ] **ServiceMonitor exists** in power-monitor namespace
- [ ] **Service exists** with port 28282
- [ ] **Service has endpoints** pointing to Kepler pods
- [ ] **Prometheus logs** show ServiceMonitor discovery
- [ ] **Prometheus target** shows health="up"
- [ ] **No scraping errors** in target lastError field
- [ ] **Kepler metrics available** when querying Prometheus
- [ ] **Metric values are realistic** (not all zeros or NaN)

---

## Common Issues and Solutions

### Issue 1: ServiceMonitor Not Created

**Symptoms:**

- `kubectl get servicemonitor -n power-monitor` returns no resources

**Diagnosis:**

```bash
# Check operator logs
kubectl logs -n kepler-operator deployment/kepler-operator-controller | grep -i servicemonitor

# Check if prometheus-operator CRDs exist
kubectl get crd servicemonitors.monitoring.coreos.com
```

**Solution:**

If CRD doesn't exist, prometheus-operator is not installed.

See the **[Monitoring Stack Setup Guide](../installation/monitoring-stack-kubernetes.md)** for detailed installation instructions.

After installing prometheus-operator, the operator will automatically create the ServiceMonitor.

---

### Issue 2: Target Shows "Down" Status

**Symptoms:**

- Prometheus target health is "down"
- `lastError` field shows scraping errors

**Common Errors:**

#### Error: "context deadline exceeded"

```json
{
  "health": "down",
  "lastError": "Get \"http://10.244.0.10:28282/metrics\": context deadline exceeded"
}
```

**Cause:** Network connectivity issues or Kepler pod not responding

**Solution:**

```bash
# Verify Kepler pod is running
kubectl get pods -n power-monitor

# Check Kepler pod logs for errors
kubectl logs -n power-monitor daemonset/power-monitor

# Verify metrics endpoint is accessible from within cluster
kubectl exec -n power-monitor daemonset/power-monitor -- \
  curl -s http://localhost:28282/metrics | head -20
```

#### Error: "connection refused"

**Cause:** Kepler not listening on port 28282 or wrong port configured

**Solution:**

```bash
# Verify Kepler is listening on port 28282
kubectl exec -n power-monitor daemonset/power-monitor -- netstat -tlnp | grep 28282

# Check service port configuration
kubectl get svc power-monitor -n power-monitor -o yaml
```

---

### Issue 3: No Metrics Returned

**Symptoms:**

- Prometheus target is "up"
- Queries return empty results: `"result": []`

**Diagnosis:**

```bash
# Check if metrics are actually exposed by Kepler
kubectl exec -n power-monitor daemonset/power-monitor -- \
  curl -s http://localhost:28282/metrics | grep kepler_
```

**Solution:**

If Kepler is not exposing metrics:

1. **Check metric levels configuration:**

   ```bash
   kubectl get powermonitor power-monitor -o jsonpath='{.spec.kepler.config.metricLevels}'
   ```

   Ensure at least `[node, pod]` are configured.

2. **Check Kepler pod logs for errors:**

   ```bash
   kubectl logs -n power-monitor daemonset/power-monitor | grep -i error
   ```

3. **Verify hardware support:**

   On VMs or cloud instances without RAPL support, enable fake CPU meter:

   > ⚠️ **WARNING**: Fake CPU meter is a HACK for development/testing in VM environments only. **NOT for production use.**

   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: enable-fake-cpu
     namespace: power-monitor
   data:
     config.yaml: |
       dev:
         fake-cpu-meter:
           enabled: true
   ---
   apiVersion: kepler.system.sustainable.computing.io/v1alpha1
   kind: PowerMonitor
   metadata:
     name: power-monitor
   spec:
     kepler:
       config:
         additionalConfigMaps:
         - name: enable-fake-cpu
   ```

---

### Issue 4: Prometheus Not Discovering ServiceMonitor

**Symptoms:**

- ServiceMonitor exists
- Prometheus logs don't mention power-monitor
- Target doesn't appear in Prometheus

**Diagnosis:**

```bash
# Check Prometheus namespace selector
kubectl get prometheus -n monitoring -o jsonpath='{.items[0].spec.serviceMonitorNamespaceSelector}'

# Check Prometheus ServiceMonitor selector
kubectl get prometheus -n monitoring -o jsonpath='{.items[0].spec.serviceMonitorSelector}'
```

**Solution:**

If selectors are restrictive, either:

**Option 1:** Add required labels to namespace:

```bash
# Example: if Prometheus requires label "monitoring: enabled"
kubectl label namespace power-monitor monitoring=enabled
```

**Option 2:** Configure Prometheus to discover all ServiceMonitors:

```bash
helm upgrade prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --reuse-values \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
  --set prometheus.prometheusSpec.serviceMonitorNamespaceSelector={}
```

---

## Next Steps

Once you've validated that Prometheus is scraping Kepler metrics:

- **[Set up Grafana Dashboards](./grafana-dashboard.md)** to visualize the metrics
- **[Custom ConfigMaps](./custom-configmaps.md)** to adjust metric levels and collection settings
- **[Set up Alerts](https://prometheus.io/docs/alerting/latest/overview/)** for abnormal power consumption

---

## Additional Resources

- **[Prometheus Operator Documentation](https://prometheus-operator.dev/)** - ServiceMonitor configuration
- **[PromQL Basics](https://prometheus.io/docs/prometheus/latest/querying/basics/)** - Learn to query metrics
- **[Kepler Metrics Documentation](https://sustainable-computing.io/kepler/design/metrics/)** - Complete list of Kepler metrics
- **[Troubleshooting Guide](./troubleshooting.md)** - General troubleshooting for Kepler Operator
