# ⚙️ Configuring Kepler with Custom ConfigMaps

This guide explains how to customize Kepler's configuration using the `additionalConfigMaps` feature in the PowerMonitor custom resource.

## 📋 Overview

The Kepler Operator allows you to customize Kepler's behavior by providing additional configuration through Kubernetes ConfigMaps. When you reference ConfigMaps in your PowerMonitor CR, the operator automatically:

1. Merges your custom configuration with the default Kepler configuration
2. Updates the Kepler DaemonSet with the new configuration
3. Automatically triggers a rollout when the ConfigMap changes

## ✅ Prerequisites

- Kepler Operator installed in your cluster
- PowerMonitor CR deployed
- Access to create ConfigMaps in the namespace where Kepler is deployed (default: `power-monitor`)

## 🚀 Step-by-Step Configuration

### 📝 Step 1: Create a ConfigMap with Your Custom Configuration

Create a ConfigMap containing your Kepler configuration in a file named `config.yaml`. The ConfigMap must exist in the same namespace as the PowerMonitor deployment.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-kepler-config
  namespace: power-monitor
data:
  config.yaml: |
    dev:
      fake-cpu-meter:
        enabled: false
```

Apply the ConfigMap:

```bash
kubectl apply -f my-kepler-config.yaml
```

### 🔗 Step 2: Reference the ConfigMap in Your PowerMonitor CR

Update your PowerMonitor custom resource to reference the ConfigMap you created:

```yaml
apiVersion: kepler.system.sustainable.computing.io/v1alpha1
kind: PowerMonitor
metadata:
  name: power-monitor
spec:
  kepler:
    config:
      logLevel: info
      additionalConfigMaps:
        - name: my-kepler-config
    deployment:
      security:
        mode: none
```

Apply the updated PowerMonitor CR:

```bash
kubectl apply -f power-monitor.yaml
```

### ✔️ Step 3: Verify the Configuration

The operator will automatically reconcile the DaemonSet with your new configuration. You can monitor the rollout:

```bash
# Check PowerMonitor status
kubectl get powermonitor power-monitor

# Watch DaemonSet rollout
kubectl rollout status daemonset/power-monitor -n power-monitor

# Verify the ConfigMap is mounted
kubectl describe daemonset power-monitor -n power-monitor
```

## 💡 Configuration Examples

### 📤 Example 1: Enabling the Stdout Exporter

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kepler-stdout-config
  namespace: power-monitor
data:
  config.yaml: |
    exporter:
      stdout:
        enabled: true
```

### 📊 Example 2: Customizing Prometheus Exporter Settings

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kepler-prometheus-config
  namespace: power-monitor
data:
  config.yaml: |
    exporter:
      prometheus:
        enabled: true
        debugCollectors:
          - go
```

### 🔧 Example 3: Enabling Development Features

Enable fake CPU meter for testing environments:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kepler-dev-config
  namespace: power-monitor
data:
  config.yaml: |
    dev:
      fake-cpu-meter:
        enabled: true
```

### 🐛 Example 4: Enabling pprof Debug Endpoints

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kepler-pprof-config
  namespace: power-monitor
data:
  config.yaml: |
    debug:
      pprof:
        enabled: true
```

## 🗂️ Using Multiple ConfigMaps

You can reference multiple ConfigMaps to organize your configuration. The operator merges them in the order specified:

```yaml
spec:
  kepler:
    config:
      additionalConfigMaps:
        - name: my-kepler-config
        - name: kepler-stdout-config
        - name: kepler-prometheus-config
        - name: kepler-dev-config
        - name: kepler-pprof-config
```

**📌 Note:** If there are conflicting settings across multiple ConfigMaps, the later ConfigMap in the list takes precedence.

**💡 Tip:** Settings controlled by `spec.kepler.config` still override the merged ConfigMap values. Use `additionalConfigMaps` only for fields that are not exposed in the CR spec.

## 🔄 Updating Configuration

To update Kepler's configuration:

1. **Update the ConfigMap:**

   ```bash
   kubectl edit configmap my-kepler-config -n power-monitor
   ```

2. **The operator automatically detects the change** and triggers a DaemonSet rollout with the updated configuration.

3. **Monitor the rollout:**

   ```bash
   kubectl rollout status daemonset/power-monitor -n power-monitor
   ```

## 🔍 Troubleshooting

### ❌ ConfigMap Not Found

If you see an error like `configMap my-kepler-config not found in power-monitor namespace`:

1. Verify the ConfigMap exists:

   ```bash
   kubectl get configmap my-kepler-config -n power-monitor
   ```

2. Ensure the namespace matches your PowerMonitor deployment namespace

3. Check the ConfigMap name is correctly spelled in the PowerMonitor CR

### ⚠️ Configuration Not Applied

If your configuration changes aren't taking effect:

1. Check the PowerMonitor status for reconciliation errors:

   ```bash
   kubectl describe powermonitor power-monitor
   ```

2. Verify the ConfigMap has the correct structure with a `config.yaml` key:

   ```bash
   kubectl get configmap my-kepler-config -n power-monitor -o yaml
   ```

3. Check the operator logs:

   ```bash
   kubectl logs -n openshift-operators deployment/kepler-operator-controller-manager
   ```

### 🔁 DaemonSet Not Rolling Out

If the DaemonSet doesn't roll out after updating the ConfigMap:

1. Check if the ConfigMap hash annotation changed:

   ```bash
   kubectl get daemonset power-monitor -n power-monitor -o jsonpath='{.spec.template.metadata.annotations}'
   ```

2. Manually trigger a rollout:

   ```bash
   kubectl rollout restart daemonset/power-monitor -n power-monitor
   ```

## ⭐ Best Practices

1. **Use Descriptive Names:** Name your ConfigMaps descriptively (e.g., `kepler-production-config`, `kepler-dev-settings`)

2. **Test Before Production:** Test configuration changes in a development environment first

3. **Monitor Changes:** Watch the DaemonSet rollout after making configuration changes

4. **Validate YAML:** Ensure your `config.yaml` is valid YAML before creating the ConfigMap

5. **Document Changes:** Add comments or annotations to explain why specific configurations were chosen

6. **Use Multiple ConfigMaps:** Separate concerns by using different ConfigMaps for different aspects (e.g., metrics, performance, debugging)

## 📚 Related Resources

- [PowerMonitor API Reference](../api.md#powermonitor)
- [Kepler Configuration Reference](https://sustainable-computing.io/kepler/usage/configuration/)
- [Kepler Documentation](https://sustainable-computing.io/)

## 💬 Additional Support

For more information or help:

- Visit the [Kepler Operator repository](https://github.com/sustainable-computing-io/kepler-operator)
- File an issue on [GitHub](https://github.com/sustainable-computing-io/kepler-operator/issues)
