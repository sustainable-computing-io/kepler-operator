# Upgrading Kepler Operator

This guide explains how to upgrade Kepler Operator on both Kubernetes (using Helm) and OpenShift (using OLM).

## Pre-Upgrade Checklist

Before upgrading, ensure you:

1. **Review release notes** for breaking changes and new features
2. **Backup PowerMonitor resources**:

   ```bash
   kubectl get powermonitor power-monitor -o yaml > powermonitor-backup.yaml
   ```

3. **Verify current version**:

   ```bash
   # Kubernetes (Helm)
   helm list -n kepler-operator

   # OpenShift (OLM)
   oc get csv -n openshift-operators | grep kepler
   ```

4. **Review current configuration** to ensure it's compatible with the new version

## Upgrading on Kubernetes (Helm)

### Check Available Versions

**From Helm repository** (if using a chart repository):

```bash
helm repo update
helm search repo kepler-operator --versions
```

**From packaged chart**:

Check available chart versions in releases or your chart storage.

### Upgrade Procedure

#### Option 1: Using Makefile (from source)

```bash
# Ensure you're on the desired version branch or tag
git fetch --all --tags
git checkout v<version>  # e.g., v0.21.0

# Upgrade
make helm-install
```

The `helm-install` target uses `helm upgrade --install`, which upgrades if already installed.

#### Option 2: Using Helm Directly

```bash
# Upgrade to specific version
helm upgrade kepler-operator ./manifests/helm/kepler-operator \
  --namespace kepler-operator \
  --timeout 5m \
  --wait
```

**With custom values**:

```bash
helm upgrade kepler-operator ./manifests/helm/kepler-operator \
  --namespace kepler-operator \
  --values custom-values.yaml \
  --timeout 5m \
  --wait
```

**From packaged chart**:

```bash
helm upgrade kepler-operator kepler-operator-<new-version>.tgz \
  --namespace kepler-operator \
  --timeout 5m \
  --wait
```

### Monitor the Upgrade

Watch the operator deployment:

```bash
kubectl rollout status deployment/kepler-operator-controller -n kepler-operator
```

Check operator pods:

```bash
kubectl get pods -n kepler-operator
```

### Verify Upgrade

1. **Check Helm release version**:

   ```bash
   helm list -n kepler-operator
   ```

2. **Verify operator version**:

   ```bash
   kubectl get deployment kepler-operator-controller -n kepler-operator -o jsonpath='{.spec.template.spec.containers[0].image}'
   ```

3. **Check PowerMonitor status**:

   ```bash
   kubectl get powermonitor
   ```

4. **Verify Kepler DaemonSet**:

   ```bash
   kubectl get daemonset -A | grep power-monitor
   kubectl rollout status daemonset/power-monitor -n <namespace>
   ```

### Rollback (if needed)

If the upgrade fails or causes issues:

```bash
# List release history
helm history kepler-operator -n kepler-operator

# Rollback to previous version
helm rollback kepler-operator -n kepler-operator
```

Or rollback to a specific revision:

```bash
helm rollback kepler-operator <revision-number> -n kepler-operator
```

## Upgrading on OpenShift (OLM)

OpenShift uses Operator Lifecycle Manager (OLM) for upgrades.

### Automatic vs Manual Approval

Check your current approval strategy:

```bash
oc get subscription kepler-operator -n openshift-operators -o jsonpath='{.spec.installPlanApproval}'
```

- `Automatic`: Upgrades happen automatically when new versions are available
- `Manual`: Upgrades require manual approval

### Automatic Upgrades

If `installPlanApproval: Automatic`, upgrades happen automatically:

1. OLM detects new operator version in catalog
2. Creates an InstallPlan
3. Automatically approves and executes the upgrade
4. Operator is upgraded with no manual intervention

**Monitor automatic upgrade**:

```bash
# Watch CSV status
oc get csv -n openshift-operators -w | grep kepler

# Check install plans
oc get installplan -n openshift-operators
```

### Manual Upgrades

If `installPlanApproval: Manual`, you must approve upgrades:

#### Via OpenShift Web Console

1. Navigate to **Operators** → **Installed Operators**
2. Select **Kepler Operator**
3. Look for **Upgrade available** notification
4. Click **Upgrade** or **Preview InstallPlan**
5. Review changes
6. Click **Approve** to start the upgrade

<!-- TODO: Add screenshot of upgrade approval in OpenShift console -->

#### Via CLI

1. **List pending install plans**:

   ```bash
   oc get installplan -n openshift-operators
   ```

   Look for plans with `APPROVED: false`.

2. **Describe the install plan** to review changes:

   ```bash
   oc describe installplan <install-plan-name> -n openshift-operators
   ```

3. **Approve the install plan**:

   ```bash
   oc patch installplan <install-plan-name> \
     --namespace openshift-operators \
     --type merge \
     --patch '{"spec":{"approved":true}}'
   ```

4. **Monitor the upgrade**:

   ```bash
   oc get csv -n openshift-operators -w | grep kepler
   ```

### Change Approval Strategy

To change from manual to automatic (or vice versa):

```bash
# Switch to automatic
oc patch subscription kepler-operator \
  --namespace openshift-operators \
  --type merge \
  --patch '{"spec":{"installPlanApproval":"Automatic"}}'

# Switch to manual
oc patch subscription kepler-operator \
  --namespace openshift-operators \
  --type merge \
  --patch '{"spec":{"installPlanApproval":"Manual"}}'
```

### Verify Upgrade on OpenShift

1. **Check CSV version**:

   ```bash
   oc get csv -n openshift-operators | grep kepler
   ```

   The new version should show `Succeeded` status.

2. **Verify operator pod**:

   ```bash
   oc get pods -n openshift-operators | grep kepler-operator
   ```

3. **Check PowerMonitor status**:

   ```bash
   oc get powermonitor
   ```

4. **Verify Kepler DaemonSet**:

   ```bash
   oc get daemonset -A | grep power-monitor
   ```

## Post-Upgrade Tasks

After upgrading on either platform:

### Verify Operator Health

```bash
# Check operator logs
kubectl logs -n kepler-operator deployment/kepler-operator-controller --tail=50

# Or on OpenShift
oc logs -n openshift-operators deployment/kepler-operator-controller --tail=50
```

Look for errors or warnings.

### Verify PowerMonitor Status

```bash
kubectl get powermonitor power-monitor -o wide
```

Check that the PowerMonitor shows healthy status:

```bash
kubectl describe powermonitor power-monitor
```

Look for:

- `Reconciled: True`
- `Available: True`
- No error conditions

### Verify Kepler DaemonSet

```bash
kubectl get daemonset -A | grep power-monitor
```

Ensure:

- `DESIRED` = `CURRENT` = `READY` = `UP-TO-DATE`
- All Kepler pods are running

### Test Metrics Collection

Verify metrics are still being collected:

```bash
# Port-forward to a Kepler pod
kubectl port-forward -n <namespace> daemonset/power-monitor 28282:28282

# Query metrics
curl http://localhost:28282/metrics | grep kepler_node
```

Or check in Prometheus/Grafana for recent metrics.

### Update PowerMonitor (if needed)

If the new operator version requires PowerMonitor configuration changes:

```bash
kubectl edit powermonitor power-monitor
```

Or apply updated YAML:

```bash
kubectl apply -f updated-powermonitor.yaml
```

## Troubleshooting Upgrades

### Upgrade Stuck or Failing

**Kubernetes (Helm)**:

1. Check Helm release status:

   ```bash
   helm status kepler-operator -n kepler-operator
   ```

2. View Helm history:

   ```bash
   helm history kepler-operator -n kepler-operator
   ```

3. Check operator logs:

   ```bash
   kubectl logs -n kepler-operator deployment/kepler-operator-controller
   ```

4. If stuck, try forcing upgrade:

   ```bash
   helm upgrade kepler-operator ./manifests/helm/kepler-operator \
     --namespace kepler-operator \
     --force \
     --wait
   ```

**OpenShift (OLM)**:

1. Check CSV status:

   ```bash
   oc describe csv <csv-name> -n openshift-operators
   ```

2. Check install plan:

   ```bash
   oc get installplan -n openshift-operators
   oc describe installplan <install-plan-name> -n openshift-operators
   ```

3. Check operator logs:

   ```bash
   oc logs -n openshift-operators deployment/kepler-operator-controller
   ```

### PowerMonitor Not Updating

If PowerMonitor resources don't update after operator upgrade:

1. Check operator is running new version
2. Manually trigger reconciliation by adding an annotation:

   ```bash
   kubectl annotate powermonitor power-monitor \
     reconcile-trigger="$(date +%s)"
   ```

3. Check operator logs for reconciliation errors

### Kepler Pods Not Updating

If Kepler DaemonSet pods don't update:

1. Manually restart DaemonSet:

   ```bash
   kubectl rollout restart daemonset/power-monitor -n <namespace>
   ```

2. Check DaemonSet status:

   ```bash
   kubectl describe daemonset power-monitor -n <namespace>
   ```

3. Check pod status:

   ```bash
   kubectl describe pods -l app.kubernetes.io/name=power-monitor-exporter -n <namespace>
   ```

For more troubleshooting, see the [Troubleshooting Guide](./troubleshooting.md).

## Best Practices

1. **Test in non-production first**: Upgrade in development/staging before production
2. **Review release notes carefully**: Understand breaking changes and new features
3. **Backup before upgrading**: Save PowerMonitor configurations
4. **Monitor during upgrade**: Watch logs and status during the upgrade process
5. **Upgrade during maintenance window**: Schedule upgrades during low-traffic periods
6. **Keep Helm/OLM up to date**: Ensure your tooling is current
7. **Document your process**: Keep notes on customizations and configurations

## Next Steps

- **[Troubleshooting Guide](./troubleshooting.md)** - Diagnose upgrade issues
- **[PowerMonitor Configuration](../reference/power-monitor.md)** - Review configuration options for new versions
