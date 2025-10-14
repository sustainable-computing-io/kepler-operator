# Experimental: Redfish BMC Power Monitoring

⚠️ **EXPERIMENTAL FEATURE WARNING**

This feature is experimental and has no stability guarantees:

- Configuration format may change in future versions
- Metrics names and labels may evolve
- Suitable for controlled environments and testing only
- Not recommended for production use without thorough testing

## Overview

Enable platform-level power monitoring by configuring Kepler to collect power data from your BMCs via Redfish API. Platform metrics (`kepler_platform_watts`) are exposed separately from workload attribution metrics.

## Prerequisites

- **PowerMonitor deployed** - Kepler operator and PowerMonitor resource running
- **BMC Access** - Network connectivity to BMC endpoints from Kepler pods
- **BMC Credentials** - Username and password with read access to power metrics

## Setup Guide

### Step 1: Deploy PowerMonitor with Redfish Configuration

Deploy your PowerMonitor configured to use Redfish. The PowerMonitor references a ConfigMap and Secret that you'll create in the next steps:

```yaml
# File: hack/examples/redfish-powermonitor.yaml
apiVersion: kepler.system.sustainable.computing.io/v1alpha1
kind: PowerMonitor
metadata:
  name: power-monitor
spec:
  kepler:
    deployment:
      # Deploy to all Linux nodes
      nodeSelector:
        kubernetes.io/os: linux

      # Tolerate all taints to run on all nodes
      tolerations:
      - operator: Exists

      # Mount the Secret containing BMC credentials (will be created after namespace exists)
      secrets:
      - name: redfish-secret
        mountPath: /etc/kepler/secrets/redfish
        readOnly: true

    config:
      # Standard Kepler configuration
      logLevel: info
      sampleRate: 5s
      metricLevels:
      - node
      - pod

      # Reference the ConfigMap that enables Redfish (will be created after namespace exists)
      additionalConfigMaps:
      - name: enable-redfish
```

**Apply the PowerMonitor:**

```bash
kubectl apply -f hack/examples/redfish-powermonitor.yaml
```

The Kepler operator will create the `power-monitor` namespace and begin reconciling the PowerMonitor.

**Expected State:** The PowerMonitor conditions will show `Reconciled: False` and `Available: False` at this point because the referenced ConfigMap and Secret don't exist yet. This is normal and expected.

### Step 2: Wait for Namespace Creation

Wait for the operator to create the `power-monitor` namespace:

```bash
kubectl wait --for=jsonpath='{.status.phase}'=Active namespace/power-monitor --timeout=60s
```

### Step 3: Create Kepler Configuration ConfigMap

Create the ConfigMap that enables Redfish in Kepler:

```yaml
# File: hack/examples/redfish-configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: enable-redfish
  namespace: power-monitor
data:
  config.yaml: |
    experimental:
      platform:
        redfish:
          # Enable Redfish BMC power monitoring
          enabled: true

          # Node identifier - auto-resolves from Kubernetes if not specified
          # Must match a key in the 'nodes' section of redfish.yaml
          nodeName: ""  # Leave empty to auto-detect from Kubernetes node name

          # Path to BMC configuration file (mounted from Secret)
          configFile: /etc/kepler/secrets/redfish/redfish.yaml

          # HTTP timeout for BMC requests
          httpTimeout: 5s
```

**Apply the ConfigMap:**

```bash
kubectl apply -f hack/examples/redfish-configmap.yaml
```

### Step 4: Create BMC Credentials Secret

Create the Secret containing your BMC configuration:

```yaml
# File: hack/examples/redfish-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: redfish-secret
  namespace: power-monitor
type: Opaque
stringData:
  redfish.yaml: |
    # Node to BMC mapping
    # Key: Node identifier (must match Kubernetes node name or PowerMonitor nodeName)
    # Value: BMC identifier that references an entry in the 'bmcs' section
    nodes:
      worker-1: bmc-worker-1
      worker-2: bmc-worker-2
      worker-3: bmc-worker-3

    # BMC connection details
    bmcs:
      bmc-worker-1:
        endpoint: https://bmc-worker-1.example.com
        username: monitoring-user
        password: "SecurePassword123!"
        insecure: false  # Set to true only for development/testing

      bmc-worker-2:
        endpoint: https://192.168.1.101
        username: monitoring-user
        password: "SecurePassword456!"
        insecure: false

      bmc-worker-3:
        endpoint: https://192.168.1.102:8443  # Custom port
        username: admin
        password: "SecurePassword789!"
        insecure: false
```

**Apply the Secret:**

```bash
kubectl apply -f hack/examples/redfish-secret.yaml
```

**Security Note:** Never commit BMC credentials to git. Use proper secret management (Sealed Secrets, External Secrets Operator, Vault).

**What happens next:** Creating the Secret triggers the operator's reconcile loop. The operator will redeploy Kepler with the Redfish configuration. The PowerMonitor status conditions will update:

- **Reconciled**: Changes from `False` to `True`
- **Available**: Changes from `False` to `True`

### Step 5: Verify Deployment

**Check PowerMonitor status conditions:**

```bash
# Check Reconciled condition
kubectl get powermonitor power-monitor -o jsonpath='{.status.conditions[?(@.type=="Reconciled")].status}'

# Check Available condition
kubectl get powermonitor power-monitor -o jsonpath='{.status.conditions[?(@.type=="Available")].status}'
```

**Expected output:** Both should return `True`

**Check Kepler pods are running:**

```bash
kubectl get pods -n power-monitor
```

**Check Kepler logs for Redfish initialization:**

```bash
kubectl logs -n power-monitor daemonset/power-monitor | grep -i redfish
```

**Expected log output:**

```text
level=INFO msg="BMC configuration loaded" service=experimental.redfish node_name=kind-worker bmc_id=mock-bmc endpoint=http://172.18.0.1:28001
level=INFO msg="Successfully initialized power reader" service=experimental.redfish node_name=kind-worker
level=INFO msg="Successfully connected to BMC" service=experimental.redfish node_name=kind-worker
```

## Configuration Reference

### Main Configuration (ConfigMap)

```yaml
experimental:
  platform:
    redfish:
      enabled: true                    # Enable Redfish monitoring
      nodeName: ""                     # Auto-resolves from Kubernetes
      configFile: "/path/to/redfish.yaml"  # Path to mounted Secret
      httpTimeout: 5s                  # BMC request timeout
```

**Configuration Options:**

- **enabled**: Enable/disable Redfish monitoring (default: `false`)
- **nodeName**: Node identifier for BMC mapping (auto-resolves if empty)
  - Priority: CLI flag → Kubernetes node name → hostname
- **configFile**: Path to BMC configuration YAML (must point to mounted Secret)
- **httpTimeout**: Maximum time to wait for BMC responses (default: `5s`)

### BMC Configuration (Secret)

```yaml
nodes:
  <node-id>: <bmc-id>  # Map node to BMC

bmcs:
  <bmc-id>:
    endpoint: "https://bmc.example.com"
    username: "monitoring-user"
    password: "secure-password"
    insecure: false  # TLS verification
```

**Node-to-BMC Mapping:**

- **One-to-One**: Each bare metal node has its own BMC

  ```yaml
  nodes:
    worker-1: bmc-worker-1
    worker-2: bmc-worker-2
  ```

- **Many-to-One**: Multiple VMs share the same bare metal BMC

  ```yaml
  nodes:
    vm-worker-1: shared-bmc
    vm-worker-2: shared-bmc
    vm-worker-3: shared-bmc
  ```

- **Blade Servers**: Multiple blades sharing chassis BMC

  ```yaml
  nodes:
    blade-1: chassis-bmc-1
    blade-2: chassis-bmc-1
    blade-3: chassis-bmc-1
  ```

## Metrics

Redfish monitoring exposes `kepler_platform_watts` metrics with platform-level power consumption from BMCs. For complete metric documentation, see the [Kepler Metrics Reference](https://sustainable-computing.io/kepler/design/metrics/).

## Validation

### Step 1: Check Metrics Endpoint

Verify Kepler is exposing platform metrics:

**Method 1: From inside the pod (recommended for testing):**

```bash
kubectl exec -n power-monitor daemonset/power-monitor -- curl -s http://localhost:28282/metrics | grep kepler_platform_watts
```

**Method 2: Using port-forward:**

```bash
kubectl port-forward -n power-monitor daemonset/power-monitor 28282:28282
# In another terminal:
curl http://localhost:28282/metrics | grep kepler_platform_watts
```

**Expected output:**

```text
# HELP kepler_platform_watts Current platform power in watts from BMC (PowerSubsystem or deprecated Power API)
# TYPE kepler_platform_watts gauge
kepler_platform_watts{bmc_id="mock-bmc",chassis_id="1U",node_name="kind-worker",source="redfish",source_id="PS1",source_name="Power Supply 1",source_type="PowerSupply"} 245
kepler_platform_watts{bmc_id="mock-bmc",chassis_id="2U",node_name="kind-worker",source="redfish",source_id="PS1",source_name="Power Supply 1",source_type="PowerSupply"} 345
```

### Step 2: Query Prometheus

If using Prometheus, verify metrics are being scraped:

```bash
kubectl port-forward -n monitoring svc/prometheus 9090:9090
```

Navigate to `http://localhost:9090` and query:

```promql
kepler_platform_watts
```

## Security Notes

**Important:** Never commit BMC credentials to git in plaintext. Use Kubernetes Secrets or external secret management solutions (Sealed Secrets, External Secrets Operator, Vault) to protect BMC credentials.

## Troubleshooting

### PowerMonitor Shows Degraded Status

**Symptom:** PowerMonitor conditions show `Reconciled: False` and `Available: False` after initial deployment

**Cause:** This is **expected behavior** when the PowerMonitor is created before the ConfigMap and Secret exist.

**Solution:** Continue with steps 3 and 4 to create the ConfigMap and Secret. Both conditions will automatically transition to `True` once both resources are created and Kepler is successfully redeployed.

**Verify status transition:**

```bash
# Watch the status change
kubectl get powermonitor power-monitor -w

# Check both conditions
kubectl get powermonitor power-monitor -o jsonpath='{.status.conditions[?(@.type=="Reconciled")]}'
kubectl get powermonitor power-monitor -o jsonpath='{.status.conditions[?(@.type=="Available")]}'
```

### Kepler Pods Crashing

**Symptom:** Kepler pods show `CrashLoopBackOff` status

**Diagnosis:**

Check pod logs for initialization errors:

```bash
kubectl logs -n power-monitor daemonset/power-monitor --tail=50 | grep -i "error\|fatal"
```

**Common Causes:**

1. **BMC connectivity issues** - Pods fail to connect to BMC during initialization
   - Error: `failed to connect to BMC at http://...`
   - Solution: See [DNS Resolution Failures](#dns-resolution-failures) or [BMC Connection Errors](#bmc-connection-errors)

2. **Missing or invalid Secret** - Secret not mounted or has invalid YAML
   - Error: `failed to load BMC configuration`
   - Solution: Verify Secret exists and has correct structure

3. **Node name mismatch** - Node name doesn't match any entry in Secret's `nodes` section
   - Error: `node not found in redfish configuration`
   - Solution: See [Node Not Found in Configuration](#node-not-found-in-configuration)

### No Platform Metrics

**Symptom:** `kepler_platform_watts` metrics not present

**Solutions:**

1. **Check Kepler logs:**

   ```bash
   kubectl logs -n power-monitor daemonset/power-monitor | grep -i redfish
   ```

2. **Verify Secret is mounted:**

   ```bash
   kubectl exec -n power-monitor daemonset/power-monitor -- ls -la /etc/kepler/secrets/redfish/
   ```

3. **Verify configuration:**

   ```bash
   kubectl exec -n power-monitor daemonset/power-monitor -- cat /etc/kepler/secrets/redfish/redfish.yaml
   ```

4. **Check metrics from inside pod:**

   ```bash
   kubectl exec -n power-monitor daemonset/power-monitor -- curl -s http://localhost:28282/metrics | grep kepler_platform_watts
   ```

### BMC Connection Errors

**Symptom:** Logs show "connection refused" or "timeout"

**Solutions:**

1. **Test BMC connectivity from pod:**

   ```bash
   kubectl exec -n power-monitor daemonset/power-monitor -- curl -k https://<bmc-endpoint>/redfish/v1/
   ```

2. **Check network policies:**

   ```bash
   kubectl get networkpolicy -n power-monitor
   ```

3. **Verify BMC endpoint configuration:**
   - Ensure endpoint URL is correct
   - Check port is accessible
   - Verify DNS resolution if using hostnames

### DNS Resolution Failures

**Symptom:** Logs show "dial tcp: lookup host.docker.internal: no such host" or similar DNS errors

**Cause:** DNS names like `host.docker.internal` may not resolve inside Kubernetes pods

**Solutions:**

1. **For Kind clusters with mock BMC server:**

   Use the Kind network gateway IP instead of `host.docker.internal`:

   ```bash
   # Get Kind network gateway IP
   docker network inspect kind | jq '.[0].IPAM.Config[0].Gateway'
   ```

   Update your Secret to use the gateway IP:

   ```yaml
   bmcs:
     mock-bmc:
       endpoint: http://172.18.0.1:28001  # Use gateway IP, not host.docker.internal
       username: ""
       password: ""
       insecure: true
   ```

2. **For production BMCs:**

   - Use IP addresses instead of hostnames if DNS is unreliable
   - Verify DNS resolution from within the pod:

     ```bash
     kubectl exec -n power-monitor daemonset/power-monitor -- nslookup <bmc-hostname>
     ```

### Authentication Errors

**Symptom:** Logs show "401 Unauthorized" or "403 Forbidden"

**Solutions:**

1. **Test credentials manually:**

   ```bash
   curl -k -u username:password https://<bmc-endpoint>/redfish/v1/Chassis
   ```

2. **Verify Secret content:**

   ```bash
   kubectl get secret redfish-secret -n power-monitor -o jsonpath='{.data.redfish\.yaml}' | base64 -d
   ```

3. **Check BMC user permissions:**
   - Ensure user has permission to read power metrics
   - Some BMCs require specific roles (e.g., "Operator" on Dell iDRAC)

### Node Not Found in Configuration

**Symptom:** Logs show "node not found in redfish configuration"

**Solutions:**

1. **Check node name matches:**

   ```bash
   kubectl get nodes
   kubectl get powermonitor power-monitor -o jsonpath='{.spec.kepler.config.experimental.platform.redfish.nodeName}'
   ```

2. **Verify nodes section in Secret:**

   ```bash
   kubectl get secret redfish-secret -n power-monitor -o jsonpath='{.data.redfish\.yaml}' | base64 -d | grep -A 10 "nodes:"
   ```

### Secret or ConfigMap Not Found

**Symptom:** Logs show "configmap not found" or "secret not found"

**Cause:** Secret or ConfigMap created in wrong namespace

**Solution:**

1. **Verify resources exist in power-monitor namespace:**

   ```bash
   kubectl get configmap enable-redfish -n power-monitor
   kubectl get secret redfish-secret -n power-monitor
   ```

2. **If in wrong namespace, delete and recreate:**

   ```bash
   # Delete from wrong namespace
   kubectl delete configmap enable-redfish -n <wrong-namespace>
   kubectl delete secret redfish-secret -n <wrong-namespace>

   # Recreate in correct namespace
   kubectl apply -f hack/examples/redfish-configmap.yaml
   kubectl apply -f hack/examples/redfish-secret.yaml
   ```

**Important:** All resources (Secret, ConfigMap, PowerMonitor) must be in the same namespace where Kepler is deployed (`power-monitor` by default).

## Limitations

Current experimental implementation has these limitations:

- **Power-only metrics**: No energy counters (`_joules_total`) due to on-demand collection
- **Basic caching**: Simple staleness-based cache (could be enhanced)
- **Synchronous collection**: BMC queries happen during Prometheus scrape (mitigated by caching)
- **No resilience patterns**: Basic retry logic only (no circuit breaker)

## Future Enhancements

Planned improvements for future versions:

- Background collection with enhanced caching
- Circuit breaker patterns for BMC failures
- Integration with external secret stores (Vault, AWS Secrets Manager)
- Energy counter derivation from power readings
- Sub-component power zones (fans, storage, network)
- Additional authentication methods

## Related Documentation

- **[PowerMonitor Guide](../../reference/power-monitor.md)** - Complete PowerMonitor configuration
- **[Custom ConfigMaps Guide](../../reference/custom-configmaps.md)** - Using additionalConfigMaps
- **[Kepler Redfish Proposal](https://github.com/sustainable-computing-io/kepler/blob/main/doc/dev/EP_001-redfish-support.md)** - Technical design document

## Getting Help

- **Issues**: [GitHub Issues](https://github.com/sustainable-computing-io/kepler-operator/issues)
- **Discussions**: [GitHub Discussions](https://github.com/sustainable-computing-io/kepler-operator/discussions)
- **Kepler Docs**: [Kepler Documentation](https://sustainable-computing.io/)
