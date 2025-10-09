# Troubleshooting Guide

This guide helps you diagnose and resolve common issues with Kepler Operator on both Kubernetes and OpenShift.

## Quick Diagnostic Workflow

Follow this systematic approach to diagnose issues:

### Step 1: Check Operator Health

```bash
# Kubernetes
kubectl get pods -n kepler-operator
kubectl logs -n kepler-operator deployment/kepler-operator-controller

# OpenShift
oc get pods -n openshift-operators | grep kepler
oc logs -n openshift-operators deployment/kepler-operator-controller
```

### Step 2: Check PowerMonitor Status

```bash
kubectl get powermonitor power-monitor
kubectl describe powermonitor power-monitor
```

Look for status conditions - they should show `Reconciled: True` and `Available: True`.

### Step 3: Check Kepler DaemonSet

```bash
kubectl get daemonset -A | grep power-monitor
kubectl get pods -A -l app.kubernetes.io/name=power-monitor-exporter
```

### Step 4: Decision Tree

- **No operator pods running?** → [Operator Issues](#operator-issues)
- **Operator running, but PowerMonitor not reconciling?** → [PowerMonitor Issues](#powermonitor-issues)
- **Pods in CrashLoopBackOff?** → [Kepler Pod Issues](#kepler-pod-issues)
- **Pods running but no metrics?** → [Metrics Issues](#metrics-issues)
- **Metrics in Prometheus but not Grafana?** → [Monitoring Integration Issues](#monitoring-integration-issues)

## Common Issues (Both Platforms)

### Operator Issues

#### Operator Pod Not Starting

**Symptoms**: No operator pods in `Running` state.

**Diagnosis**:

```bash
# Check pod status
kubectl get pods -n kepler-operator
kubectl describe pod <operator-pod-name> -n kepler-operator
```

**Common causes**:

1. **Image pull failures**:

```text
   Error: ImagePullBackOff
   ```

   **Solution**: Verify image exists and credentials are configured:

   ```bash
   kubectl describe pod <operator-pod-name> -n kepler-operator | grep -A 5 "Events:"
   ```

1. **Insufficient resources**:

```text
   Error: 0/3 nodes are available: insufficient memory
   ```

   **Solution**: Increase node resources or adjust operator resource requests.

1. **RBAC permission errors**:

```text
   Error: forbidden: User cannot create resource
   ```

   **Solution**: Verify ClusterRole and ClusterRoleBinding are created:

   ```bash
   kubectl get clusterrole | grep kepler-operator
   kubectl get clusterrolebinding | grep kepler-operator
   ```

#### Operator Logs Show Errors

**Check for common error patterns**:

```bash
kubectl logs -n kepler-operator deployment/kepler-operator-controller | grep -i error
```

**Webhook errors**:

```text
Error: failed to get certificate
```

**Solution**: Verify cert-manager is running (Kubernetes) or Service Serving Certificates are enabled (OpenShift).

### PowerMonitor Issues

#### PowerMonitor Stuck in Reconciling

**Symptoms**: PowerMonitor exists but DaemonSet never deploys.

**Diagnosis**:

```bash
kubectl describe powermonitor power-monitor
```

Look at `status.conditions` for error messages.

**Common causes**:

1. **Validation errors**:

```text
   Error: spec.kepler.config.sampleRate: Invalid value
   ```

   **Solution**: Fix configuration and reapply:

   ```bash
   kubectl edit powermonitor power-monitor
   ```

1. **Missing permissions**:

   Check operator has permissions to create DaemonSets:

   ```bash
   kubectl auth can-i create daemonsets --as=system:serviceaccount:kepler-operator:kepler-operator-controller
   ```

#### DaemonSet Not Deploying

**Symptoms**: PowerMonitor created successfully but no DaemonSet or pods appear in the `power-monitor` namespace.

**Diagnosis**:

```bash
# Check if power-monitor namespace exists
kubectl get namespace power-monitor

# Check if DaemonSet exists
kubectl get daemonset -A | grep power-monitor

# Check operator logs for errors
kubectl logs -n kepler-operator deployment/kepler-operator-controller --tail=100 | grep -i error
```

**Common causes**:

1. **prometheus-operator not installed** (MOST COMMON):

```text
   ERROR controller-runtime.source.EventHandler if kind is a CRD, it should be installed before calling Start
   {"kind": "ServiceMonitor.monitoring.coreos.com", "error": "failed to get restmapping: no matches for kind \"ServiceMonitor\" in version \"monitoring.coreos.com/v1\""}
   ```

   **Problem**: Kepler Operator requires prometheus-operator because it creates ServiceMonitor resources. Without it, the operator cannot complete reconciliation and the DaemonSet is never created.

   **Solution**: Install prometheus-operator.

   See the **[Monitoring Stack Setup Guide](../installation/monitoring-stack-kubernetes.md)** for detailed installation instructions.

   **Quick check if installed**:

   ```bash
   # Check if ServiceMonitor CRD exists
   kubectl get crd servicemonitors.monitoring.coreos.com
   ```

   Once prometheus-operator is installed:

   ```bash
   # The operator will automatically reconcile and create the DaemonSet
   kubectl get daemonset -n power-monitor --watch
   ```

1. **Namespace permissions**:

   Check operator has permissions to create resources in the power-monitor namespace:

   ```bash
   kubectl auth can-i create daemonsets --as=system:serviceaccount:kepler-operator:kepler-operator-controller -n power-monitor
   ```

1. **Other reconciliation errors**:

   Check full operator logs:

   ```bash
   kubectl logs -n kepler-operator deployment/kepler-operator-controller --tail=200
   ```

### Kepler Pod Issues

#### Pods in CrashLoopBackOff

**Diagnosis**:

```bash
kubectl get pods -n <namespace> -l app.kubernetes.io/name=power-monitor-exporter
kubectl logs -n <namespace> <power-monitor-pod-name>
kubectl describe pod -n <namespace> <power-monitor-pod-name>
```

**Common causes**:

1. **Missing hardware sensors** (VMs or cloud instances):

```text
   Error: failed to read RAPL: no such file or directory
   ```

   **Solution**: Enable fake CPU meter via ConfigMap. Create a ConfigMap with the following structure:

   > ⚠️ **WARNING**: Fake CPU meter is a HACK for development/testing in VM environments only. **NOT for production use.**

   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: enable-fake-cpu
     namespace: <namespace>
   data:
     config.yaml: |
       dev:
         fake-cpu-meter:
           enabled: true
   ```

   Then reference it in your PowerMonitor:

   ```yaml
   spec:
     kepler:
       config:
         additionalConfigMaps:
         - name: enable-fake-cpu
   ```

   This is commonly needed when running Kepler in VMs or cloud environments without hardware power sensors.

1. **Permission denied errors**:

```text
   Error: failed to access /sys/class/powercap: permission denied
   ```

   **Note**: The operator automatically configures the required privileged security context for Kepler pods. If you see this error, it indicates a configuration issue with the operator deployment. Check operator logs for errors related to security context or permissions.

1. **Invalid configuration**:

```text
   Error: invalid log level: "invalid"
   ```

   **Solution**: Fix PowerMonitor configuration:

   ```bash
   kubectl edit powermonitor power-monitor
   ```

#### Pods Not Scheduled on Nodes

**Symptoms**: `DesiredNumberScheduled` doesn't match `CurrentNumberScheduled`.

**Diagnosis**:

```bash
kubectl describe daemonset power-monitor -n <namespace>
kubectl get nodes --show-labels
```

**Common causes**:

1. **nodeSelector mismatch**:

   PowerMonitor has:

   ```yaml
   nodeSelector:
     custom-label: "true"
   ```

   But nodes don't have this label.

   **Solution**: Either:
   - Add labels to nodes: `kubectl label nodes <node-name> custom-label=true`
   - Or adjust PowerMonitor `nodeSelector`

1. **Taint/Toleration issues**:

   Nodes are tainted but PowerMonitor doesn't have matching tolerations.

   **Solution**: Add tolerations to PowerMonitor:

   ```yaml
   spec:
     kepler:
       deployment:
         tolerations:
         - operator: Exists
   ```

### Metrics Issues

#### No Metrics Available

**Symptoms**: Kepler pods running but no metrics in Prometheus.

**Diagnosis**:

```bash
# Port-forward to Kepler pod
kubectl port-forward -n <namespace> <kepler-pod-name> 28282:28282

# Check if metrics endpoint works
curl http://localhost:28282/metrics | grep kepler
```

**If metrics endpoint works**:

- Problem is with Prometheus scraping
- See [Monitoring Integration Issues](#monitoring-integration-issues)

**If metrics endpoint fails**:

1. **Kepler not exposing metrics**:

   Check Kepler logs:

   ```bash
   kubectl logs -n <namespace> <power-monitor-pod-name>
   ```

1. **Wrong port**:

   Verify Service exposes correct port:

   ```bash
   kubectl get svc -n <namespace>
   kubectl describe svc power-monitor -n <namespace>
   ```

#### Missing Specific Metric Levels

**Symptoms**: Node metrics available but not pod/container metrics.

**Diagnosis**: Check PowerMonitor `metricLevels`:

```bash
kubectl get powermonitor power-monitor -o jsonpath='{.spec.kepler.config.metricLevels}'
```

**Solution**: Add missing metric levels:

```bash
kubectl patch powermonitor power-monitor --type=merge -p '
{
  "spec": {
    "kepler": {
      "config": {
        "metricLevels": ["node", "pod", "vm", "container"]
      }
    }
  }
}'
```

### Monitoring Integration Issues

#### ServiceMonitor Not Created

**Kubernetes**:

Verify ServiceMonitor is enabled in Helm values:

```bash
helm get values kepler-operator -n kepler-operator
```

If `metrics.serviceMonitor.enabled: false`, upgrade with:

```bash
helm upgrade kepler-operator ./manifests/helm/kepler-operator \
  --namespace kepler-operator \
  --set metrics.serviceMonitor.enabled=true
```

**OpenShift**:

ServiceMonitor should be created automatically. Verify:

```bash
oc get servicemonitor -n <namespace>
```

#### Prometheus Not Discovering ServiceMonitor

**Symptoms**: Kepler pods running, metrics endpoint works, ServiceMonitor exists, but Prometheus has no Kepler targets.

**Diagnosis**:

1. **Check if ServiceMonitor exists**:

   ```bash
   kubectl get servicemonitor -n power-monitor
   ```

2. **Check Prometheus targets**:

   ```bash
   # Port-forward to Prometheus
   kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090

   # Check targets in browser at http://localhost:9090/targets
   # Or query via API
   curl -s 'http://localhost:9090/api/v1/targets' | grep -i "power-monitor"
   ```

3. **Check Prometheus ServiceMonitor selector**:

   ```bash
   kubectl get prometheus -n monitoring -o yaml | grep -A 5 serviceMonitorSelector
   ```

**Common causes**:

1. **ServiceMonitor label selector mismatch (MOST COMMON with kube-prometheus-stack)**:

   **Problem**: kube-prometheus-stack configures Prometheus to only discover ServiceMonitors with the `release: prometheus` label by default. Kepler's ServiceMonitor doesn't have this label.

   **Diagnosis**:

   ```bash
   # Check Prometheus selector
   kubectl get prometheus -n monitoring -o jsonpath='{.spec.serviceMonitorSelector}'

   # Check ServiceMonitor labels
   kubectl get servicemonitor power-monitor -n power-monitor -o jsonpath='{.metadata.labels}'
   ```

   If Prometheus requires `release: prometheus` but ServiceMonitor doesn't have it, they won't match.

   **Solution Option 1** - Add the required label to ServiceMonitor (quick fix):

   ```bash
   kubectl patch servicemonitor power-monitor -n power-monitor \
     --type=merge \
     -p '{"metadata":{"labels":{"release":"prometheus"}}}'
   ```

   Wait 30-60 seconds for Prometheus to discover the target, then verify:

   ```bash
   curl -s 'http://localhost:9090/api/v1/query?query=kepler_node_cpu_watts' | grep -o '"status":"[^"]*"'
   ```

   **Solution Option 2** - Configure Prometheus to discover all ServiceMonitors (persistent fix):

   ```bash
   helm upgrade prometheus prometheus-community/kube-prometheus-stack \
     --namespace monitoring \
     --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
     --reuse-values
   ```

   This configures Prometheus to scrape all ServiceMonitors regardless of labels.

2. **ServiceMonitor in wrong namespace**:

   **Problem**: Prometheus may only watch ServiceMonitors in specific namespaces.

   **Diagnosis**:

   ```bash
   kubectl get prometheus -n monitoring -o yaml | grep -A 5 serviceMonitorNamespaceSelector
   ```

   **Solution**: Either:
   - Move ServiceMonitor to a watched namespace, or
   - Configure Prometheus to watch additional namespaces:

   ```bash
   helm upgrade prometheus prometheus-community/kube-prometheus-stack \
     --namespace monitoring \
     --set prometheus.prometheusSpec.serviceMonitorNamespaceSelector={} \
     --reuse-values
   ```

3. **ServiceMonitor endpoint configuration**:

   **Diagnosis**:

   ```bash
   kubectl get servicemonitor power-monitor -n power-monitor -o yaml
   kubectl get service power-monitor -n power-monitor -o yaml
   ```

   Ensure the `endpoints[].port` in ServiceMonitor matches a port name in the Service.

   **Solution**: ServiceMonitor and Service are both created by the operator and should match. If they don't, this indicates an operator bug - file an issue.

#### Metrics in Prometheus But Not in Grafana

**Diagnosis**:

1. **Check Grafana datasource**:

   Navigate to **Configuration** → **Data Sources** → **Prometheus** → **Test**

1. **Query metrics directly**:

   In Grafana **Explore**, run:

   ```promql
   kepler_node_cpu_joules_total
   ```

**Common causes**:

1. **Datasource misconfigured**:

   Verify datasource URL points to correct Prometheus service.

1. **Dashboard queries incorrect**:

   Check dashboard JSON for correct metric names and labels.

1. **Time range issue**:

   Ensure dashboard time range includes when Kepler was running.

## Kubernetes-Specific Issues

### cert-manager Issues

#### Webhook Certificate Errors

**Symptoms**:

```text
Error: Internal error occurred: failed calling webhook
Error: x509: certificate signed by unknown authority
```

**Diagnosis**:

```bash
kubectl get certificate -n kepler-operator
kubectl describe certificate -n kepler-operator
```

**Common causes**:

1. **cert-manager not running**:

   ```bash
   kubectl get pods -n cert-manager
   ```

   **Solution**: Install cert-manager:

   ```bash
   kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.18.2/cert-manager.yaml
   ```

1. **Certificate not ready**:

   ```bash
   kubectl get certificate -n kepler-operator
   ```

   If `READY: False`, check cert-manager logs:

   ```bash
   kubectl logs -n cert-manager deployment/cert-manager
   ```

1. **Certificate expired**:

   Certificates auto-renew, but if expired:

   ```bash
   kubectl delete certificate -n kepler-operator <cert-name>
   ```

   cert-manager will recreate it.

### Helm Issues

#### Helm Install/Upgrade Failures

**Diagnosis**:

```bash
helm list -n kepler-operator
helm status kepler-operator -n kepler-operator
helm history kepler-operator -n kepler-operator
```

**Common causes**:

1. **Helm release in failed state**:

   ```bash
   helm rollback kepler-operator -n kepler-operator
   ```

1. **CRD conflicts**:

   If CRDs already exist:

   ```bash
   kubectl get crd powermonitors.kepler.system.sustainable.computing.io
   ```

   Helm doesn't manage CRD lifecycle by default. Manually delete if needed (caution: deletes all instances).

1. **Values override not working**:

   Debug by rendering templates:

   ```bash
   helm template kepler-operator ./manifests/helm/kepler-operator \
     --namespace kepler-operator \
     --values custom-values.yaml
   ```

### Prometheus Operator Issues

#### ServiceMonitor Not Being Scraped

**Quick diagnosis**:

```bash
# Check if ServiceMonitor exists
kubectl get servicemonitor -n power-monitor

# Check if Prometheus has the target
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090
# Then visit http://localhost:9090/targets and search for "power-monitor"
```

**If ServiceMonitor exists but Prometheus isn't scraping it**, this is typically a label selector mismatch issue.

See the detailed troubleshooting steps in [Prometheus Not Discovering ServiceMonitor](#prometheus-not-discovering-servicemonitor) above for:

- How to diagnose label selector mismatches
- Two solutions (quick fix vs persistent fix)
- Namespace selector issues
- Complete verification steps

## OpenShift-Specific Issues

### OperatorHub/OLM Issues

#### Operator Not Appearing in OperatorHub

**Diagnosis**:

```bash
oc get catalogsource -n openshift-marketplace
oc get packagemanifest | grep kepler
```

**Common causes**:

1. **Community Operators catalog not enabled**:

   Contact cluster administrator to enable community catalog.

1. **Catalog source not ready**:

   ```bash
   oc describe catalogsource community-operators -n openshift-marketplace
   ```

#### CSV Installation Failures

**Diagnosis**:

```bash
oc get csv -n openshift-operators | grep kepler
oc describe csv <csv-name> -n openshift-operators
oc get installplan -n openshift-operators
```

**Common causes**:

1. **Install plan not approved** (manual approval mode):

   ```bash
   oc patch installplan <install-plan-name> \
     --namespace openshift-operators \
     --type merge \
     --patch '{"spec":{"approved":true}}'
   ```

1. **Resource conflicts**:

   Check events:

   ```bash
   oc get events -n openshift-operators --sort-by='.lastTimestamp'
   ```

### Security Context Constraints (SCC)

#### SCC Violations

**Symptoms**:

```text
Error: pods "kepler-xxx" is forbidden: unable to validate against any security context constraint
```

**Diagnosis**:

```bash
oc get scc | grep kepler
oc describe scc <kepler-scc>
```

**Solution**:

Kepler Operator automatically creates appropriate SCC. If missing:

```bash
oc adm policy add-scc-to-user privileged -z kepler -n <namespace>
```

### OpenShift Monitoring Stack

#### User Workload Monitoring Not Enabled

**Diagnosis**:

```bash
oc get pods -n openshift-user-workload-monitoring
```

If no pods exist, User Workload Monitoring is not enabled.

**Solution**:

```bash
oc apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: cluster-monitoring-config
  namespace: openshift-monitoring
data:
  config.yaml: |
    enableUserWorkload: true
EOF
```

Verify it's enabled:

```bash
oc get pods -n openshift-user-workload-monitoring
```

#### ServiceMonitor Not Scraped by Prometheus

**Diagnosis**:

Check if ServiceMonitor is in correct namespace for user workload monitoring:

```bash
oc get servicemonitor -A
```

**Solution**:

OpenShift user workload monitoring scrapes ServiceMonitors in user namespaces, not `openshift-*` namespaces.

Ensure ServiceMonitor is in the correct namespace (typically where PowerMonitor is deployed).

## Getting Help

If issues persist after troubleshooting:

1. **Collect diagnostic information**:

   ```bash
   # Operator logs
   kubectl logs -n kepler-operator deployment/kepler-operator-controller > operator-logs.txt

   # PowerMonitor status
   kubectl get powermonitor power-monitor -o yaml > powermonitor-status.yaml

   # Kepler pod logs
   kubectl logs -n <namespace> -l app.kubernetes.io/name=power-monitor-exporter > kepler-logs.txt

   # Events
   kubectl get events -A --sort-by='.lastTimestamp' > events.txt
   ```

1. **Search existing issues**:

   [GitHub Issues](https://github.com/sustainable-computing-io/kepler-operator/issues)

1. **File a bug report**:

   Include:
   - Platform (Kubernetes/OpenShift version)
   - Operator version
   - PowerMonitor configuration
   - Diagnostic logs
   - Steps to reproduce

1. **Join the community**:

   - [GitHub Discussions](https://github.com/sustainable-computing-io/kepler-operator/discussions)
   - [Kepler Community](https://sustainable-computing.io/)

## Additional Resources

- **[PowerMonitor Guide](./power-monitor.md)** - Creating and configuring PowerMonitors
- **[PowerMonitor Configuration](./power-monitor.md)** - Configuration options
- **[API Reference](../reference/api.md)** - Complete API specification
