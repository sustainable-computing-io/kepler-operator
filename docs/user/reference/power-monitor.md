# Creating and Configuring PowerMonitor Resources

This guide explains how to create and configure PowerMonitor custom resources to deploy Kepler on your cluster.

## What is a PowerMonitor?

A PowerMonitor is a Kubernetes custom resource that defines how Kepler should be deployed on your cluster. When you create a PowerMonitor resource, the Kepler Operator automatically:

1. Creates a DaemonSet to run Kepler on selected nodes
2. Configures ServiceMonitor for Prometheus integration (if enabled)
3. Sets up RBAC permissions
4. Manages Kepler configuration

**Important**: The PowerMonitor resource must be named `power-monitor`. The operator enforces this naming constraint through admission webhooks. Attempting to create a PowerMonitor with any other name will be rejected with an error.

## Deployment Namespace

By default, the operator deploys Kepler components (DaemonSet, Services, etc.) to the `power-monitor` namespace. This namespace is automatically created by the operator when you create a PowerMonitor resource.

**Note**: The deployment namespace is configurable via the operator's `--deployment-namespace` flag. If your operator was deployed with a custom namespace, all commands in this guide that reference `power-monitor` namespace should be updated to use your configured namespace.

**Check your deployment namespace:**

```bash
kubectl get deployment kepler-operator-controller -n kepler-operator -o yaml | grep deployment-namespace
```

The output shows the namespace where Kepler will be deployed (e.g., `--deployment-namespace=power-monitor`).

## Singleton Resource

The Kepler operator supports only one PowerMonitor resource per cluster, which must be named `power-monitor`. To modify your Kepler deployment configuration, update the existing PowerMonitor resource rather than creating a new one:

```bash
# Update existing PowerMonitor
kubectl patch powermonitor power-monitor --type=merge -p '{"spec":{...}}'

# Don't create new PowerMonitor with different name - this will fail
```

## Quick Start

### Basic PowerMonitor

The simplest PowerMonitor configuration deploys Kepler to all Linux nodes:

```yaml
apiVersion: kepler.system.sustainable.computing.io/v1alpha1
kind: PowerMonitor
metadata:
  name: power-monitor  # Must be "power-monitor" - enforced by operator
spec:
  kepler:
    deployment:
      nodeSelector:
        kubernetes.io/os: linux
    config:
      logLevel: info
```

Apply it to your cluster:

```bash
kubectl apply -f power-monitor.yaml
```

### Verify Deployment

Check that Kepler pods are running:

```bash
# Check PowerMonitor status
kubectl get powermonitor power-monitor

# Check Kepler DaemonSet
kubectl get daemonset -n power-monitor

# Check Kepler pods
kubectl get pods -n power-monitor
```

Expected output shows Kepler DaemonSet and pods:

```text
# DaemonSet output
NAME            DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR            AGE
power-monitor   1         1         1       1            1           kubernetes.io/os=linux   10m

# Pods output
NAME                  READY   STATUS    RESTARTS   AGE
power-monitor-xxxxx   1/1     Running   0          10m
```

## PowerMonitor Specification

### Top-Level Structure

```yaml
apiVersion: kepler.system.sustainable.computing.io/v1alpha1
kind: PowerMonitor
metadata:
  name: power-monitor
spec:
  kepler:
    deployment:  # Deployment configuration
      # ... deployment options ...
    config:      # Kepler configuration
      # ... Kepler-specific settings ...
```

### Deployment Configuration

The `spec.kepler.deployment` section controls where and how Kepler pods are deployed.

#### Node Selection

Use `nodeSelector` to target specific nodes:

```yaml
spec:
  kepler:
    deployment:
      nodeSelector:
        kubernetes.io/os: linux
        node-role.kubernetes.io/worker: ""
```

This deploys Kepler only on Linux worker nodes.

#### Tolerations

To deploy Kepler on nodes with taints:

```yaml
spec:
  kepler:
    deployment:
      tolerations:
      - operator: Exists  # Tolerate all taints
```

Or target specific taints:

```yaml
spec:
  kepler:
    deployment:
      tolerations:
      - key: "dedicated"
        operator: "Equal"
        value: "power-monitoring"
        effect: "NoSchedule"
```

#### Security Mode

Control RBAC enforcement for Kepler metrics:

```yaml
spec:
  kepler:
    deployment:
      security:
        mode: none  # Options: "none" or "rbac"
```

- `none`: No RBAC enforcement (default on kubernetes)
- `rbac`: Enable RBAC-based access control to metrics ( default on OpenShift)

#### Secrets

Mount secrets into Kepler containers:

```yaml
spec:
  kepler:
    deployment:
      secrets:
      - name: my-tls-secret
        mountPath: /etc/kepler/secrets/tls
        readOnly: true
      - name: my-config-secret
        mountPath: /opt/secrets/config
        readOnly: true
```

**Important**: Avoid mounting secrets to critical paths:

- `/etc/kepler` - Reserved for Kepler configuration
- `/sys`, `/proc`, `/dev` - System directories
- `/usr`, `/bin`, `/sbin`, `/lib` - System binaries
- `/` - Root filesystem

Use subdirectories like `/etc/kepler/secrets/` or `/opt/secrets/`.

#### Enabling Experimental Features

Some experimental features require both secrets and configuration. For example, enabling Redfish BMC monitoring:

```yaml
apiVersion: kepler.system.sustainable.computing.io/v1alpha1
kind: PowerMonitor
metadata:
  name: power-monitor
spec:
  kepler:
    deployment:
      # Mount Secret containing BMC credentials
      secrets:
      - name: redfish-secret
        mountPath: /etc/kepler/secrets/redfish
        readOnly: true

    config:
      logLevel: info
      # Enable Redfish via ConfigMap
      additionalConfigMaps:
      - name: enable-redfish
```

The Secret contains BMC credentials, while the ConfigMap enables the feature and references the Secret path.

For complete Redfish setup instructions, see:

- **[Redfish BMC Monitoring Guide](./experimental/redfish.md)** - Complete setup and configuration
- **[Custom ConfigMaps Guide](./custom-configmaps.md)** - Using additionalConfigMaps

### Kepler Configuration

The `spec.kepler.config` section controls Kepler's runtime behavior.

#### Log Level

Control log verbosity:

```yaml
spec:
  kepler:
    config:
      logLevel: info  # Options: debug, info, warn, error
```

#### Sample Rate

Set how frequently Kepler monitors resources:

```yaml
spec:
  kepler:
    config:
      sampleRate: 5s  # Default: 5s (e.g., "10s", "1m")
```

#### Staleness Threshold

Define when power calculations are considered stale:

```yaml
spec:
  kepler:
    config:
      staleness: 500ms  # Default: 500ms (e.g., "1s", "5s")
```

#### Metric Levels

Choose which metrics to export:

```yaml
spec:
  kepler:
    config:
      metricLevels:
      - node       # Node-level metrics (always recommended)
      - pod        # Pod-level metrics
      - vm         # Virtual machine metrics
      - container  # Container-level metrics (high cardinality on kubernetes)
      - process    # Process-level metrics (high cardinality)
```

Default: `[node, pod, vm]`

**Note**: Including `process` or `container` metrics significantly increases cardinality and resource usage.

#### Terminated Workload Tracking

Control how many terminated workloads to track:

```yaml
spec:
  kepler:
    config:
      maxTerminated: 500  # Default: 500
```

- Negative values: Track unlimited terminated workloads
- Zero: Disable terminated workload tracking
- Positive values: Track top N by energy consumption

#### Additional ConfigMaps

Merge custom configuration into Kepler:

```yaml
spec:
  kepler:
    config:
      additionalConfigMaps:
      - name: custom-kepler-config
```

The ConfigMap must exist in the same namespace as PowerMonitor components.

For detailed examples and best practices on using custom ConfigMaps, see the [Custom ConfigMaps Guide](./custom-configmaps.md).

## Common Use Cases

**Note**: All examples below use the required name `power-monitor`. You cannot create multiple PowerMonitor resources with different names.

### Production Deployment

Recommended settings for production:

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
      tolerations:
      - operator: Exists
    config:
      logLevel: info
      sampleRate: 10s
      staleness: 1s
      metricLevels:
      - node
      - pod
      - vm
      maxTerminated: 500
```

### Development/Testing

Enable verbose logging for troubleshooting:

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
      logLevel: debug
      sampleRate: 5s
      metricLevels:
      - node
      - pod
      - container
```

### High-Cardinality Metrics

For detailed analysis with process-level metrics:

```yaml
apiVersion: kepler.system.sustainable.computing.io/v1alpha1
kind: PowerMonitor
metadata:
  name: power-monitor
spec:
  kepler:
    deployment:
      nodeSelector:
        node-role.kubernetes.io/monitoring: "true"
    config:
      logLevel: info
      sampleRate: 30s  # Reduce sampling frequency
      metricLevels:
      - node
      - pod
      - process
      maxTerminated: 100  # Limit terminated tracking
```

**Warning**: Process-level metrics can generate thousands of time series. Use cautiously.

## Understanding PowerMonitor Status

Check PowerMonitor status:

```bash
kubectl get powermonitor power-monitor -o yaml
```

Key status fields:

```yaml
status:
  conditions:
  - type: Reconciled
    status: "True"
    reason: ReconcileSuccess
    message: "PowerMonitor reconciled successfully"
  - type: Available
    status: "True"
    reason: DaemonSetAvailable
    message: "Kepler DaemonSet is available"
  kepler:
    desiredNumberScheduled: 3   # Nodes that should run Kepler
    currentNumberScheduled: 3    # Nodes currently running Kepler
    numberReady: 3               # Kepler pods in Ready state
    numberAvailable: 3           # Kepler pods available
    updatedNumberScheduled: 3    # Nodes with updated Kepler
```

## Updating PowerMonitor

To update a PowerMonitor configuration:

1. Edit your PowerMonitor YAML
2. Apply the changes:

   ```bash
   kubectl apply -f power-monitor.yaml
   ```

3. The operator automatically updates the Kepler DaemonSet
4. Kepler pods are recreated with new configuration

### Rolling Update Example

Change log level from `info` to `debug`:

```bash
kubectl patch powermonitor power-monitor --type=merge -p '{"spec":{"kepler":{"config":{"logLevel":"debug"}}}}'
```

Monitor the rollout:

```bash
kubectl rollout status daemonset/power-monitor -n power-monitor
```

## Deleting PowerMonitor

To remove Kepler from your cluster:

```bash
kubectl delete powermonitor power-monitor
```

This automatically:

- Deletes the power-monitor DaemonSet
- Removes Kepler pods from all nodes
- Cleans up associated resources

## Best Practices

1. **Start with defaults** - Begin with default configuration and adjust based on observability needs
2. **Monitor cardinality** - Use Prometheus to track metric cardinality before enabling container/process metrics
3. **Test changes** - Test configuration changes in development before applying to production
4. **Use nodeSelector** - Target specific nodes for detailed monitoring rather than all nodes
5. **Balance sample rate** - Lower sample rates provide more accurate data but increase resource usage
6. **Limit terminated tracking** - Set `maxTerminated` based on your workload churn rate
7. **Security mode** - Enable `rbac` mode if you need to restrict metrics access

## Troubleshooting

### PowerMonitor Not Reconciling

Check operator logs:

```bash
kubectl logs -n kepler-operator deployment/kepler-operator-controller
```

### Kepler Pods Not Starting

Check DaemonSet status:

```bash
kubectl describe daemonset power-monitor -n power-monitor
```

Check pod logs:

```bash
kubectl logs -n power-monitor daemonset/power-monitor
```

### No Kepler Pods on Certain Nodes

Verify node labels match `nodeSelector`:

```bash
kubectl get nodes --show-labels
```

Check node taints:

```bash
kubectl get nodes -o json | jq '.items[].spec.taints'
```

For more troubleshooting guidance, see the [Troubleshooting Guide](./troubleshooting.md).

## Next Steps

- **[Custom ConfigMaps Guide](./custom-configmaps.md)** - Advanced Kepler configuration with custom ConfigMaps
- **[Grafana Dashboards](./grafana-dashboard.md)** - Visualize Kepler metrics
- **[Troubleshooting Guide](./troubleshooting.md)** - Diagnose common issues
- **[API Reference](../reference/api.md)** - Complete PowerMonitor API specification
