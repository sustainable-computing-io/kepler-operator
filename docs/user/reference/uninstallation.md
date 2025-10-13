# Uninstalling Kepler Operator

This guide provides step-by-step instructions for completely removing Kepler Operator from your cluster.

## Pre-Uninstallation Checklist

Before uninstalling, consider:

1. **Backup PowerMonitor resources** (if you might reinstall later):

   ```bash
   kubectl get powermonitor power-monitor -o yaml > powermonitor-backup.yaml
   ```

2. **Export Grafana dashboards** (if customized):

   Export dashboards through Grafana UI or API

3. **Archive metrics** (if needed for historical analysis):

   Use Prometheus federation or remote write to preserve metrics

4. **Notify stakeholders**: Inform teams that energy monitoring will stop

## Uninstalling via Helm (Kubernetes)

### Step 1: Delete PowerMonitor Resources

Delete the PowerMonitor instance first:

```bash
# Check if PowerMonitor exists
kubectl get powermonitor power-monitor

# Delete the PowerMonitor
kubectl delete powermonitor power-monitor
```

This automatically removes Kepler DaemonSets and related resources.

### Step 2: Uninstall the Operator

**Using Makefile** (from source repository):

```bash
make helm-uninstall
```

**Using Helm directly**:

```bash
helm uninstall kepler-operator --namespace kepler-operator
```

### Step 3: Remove CRDs (Optional)

**Warning**: Deleting CRDs will permanently delete all PowerMonitor resources.

```bash
kubectl delete crd powermonitors.kepler.system.sustainable.computing.io
kubectl delete crd powermonitorinternals.kepler.system.sustainable.computing.io
```

### Step 4: Clean Up Namespaces

Delete the operator namespace if no longer needed:

```bash
kubectl delete namespace kepler-operator
```

### Step 5: Remove Webhooks (if not auto-deleted)

Check for remaining webhook configurations:

```bash
kubectl get validatingwebhookconfiguration | grep kepler
kubectl get mutatingwebhookconfiguration | grep kepler
```

Delete if present:

```bash
kubectl delete validatingwebhookconfiguration <webhook-name>
kubectl delete mutatingwebhookconfiguration <webhook-name>
```

### Verify Complete Removal

Ensure all resources are removed:

```bash
# Check for remaining operator resources
kubectl get all -n kepler-operator

# Check for CRDs
kubectl get crd | grep kepler

# Check for webhooks
kubectl get validatingwebhookconfiguration | grep kepler
kubectl get mutatingwebhookconfiguration | grep kepler

# Check for ServiceMonitors (if using prometheus-operator)
kubectl get servicemonitor -A | grep power-monitor
```

## Uninstalling via OLM (OpenShift)

### Method 1: Using Uninstall Script (Recommended)

The operator repository includes a comprehensive uninstall script:

```bash
# Clone repository if not already done
git clone https://github.com/sustainable-computing-io/kepler-operator.git
cd kepler-operator

# Dry run to see what will be deleted
./hack/uninstall-operator.sh

# Actually delete resources
./hack/uninstall-operator.sh --delete
```

The script automatically:

- Lists all Kepler-related resources
- Deletes PowerMonitor instances
- Removes CSV (ClusterServiceVersion)
- Deletes Subscription
- Removes operator resources
- Cleans up CRDs (with confirmation)

### Method 2: Manual Uninstallation via Web Console

1. **Delete PowerMonitor Resources**:

   - Navigate to **Search** → Select `PowerMonitor` kind
   - Delete the PowerMonitor instance

   Or via CLI:

   ```bash
   oc delete powermonitor power-monitor
   ```

2. **Uninstall Operator**:

   - Navigate to **Operators** → **Installed Operators**
   - Find **Kepler Operator**
   - Click the three-dot menu → **Uninstall Operator**
   - Confirm deletion

   <!-- TODO: Add screenshot of uninstall confirmation dialog -->

3. **Verify Removal**:

   - Check **Installed Operators** list
   - Verify Kepler Operator no longer appears

### Method 3: Manual Uninstallation via CLI

#### Delete PowerMonitor Resources

```bash
oc delete powermonitor power-monitor
```

Wait for the Kepler DaemonSet to be removed.

#### Step 2: Delete the Subscription

```bash
oc delete subscription kepler-operator -n openshift-operators
```

#### Step 3: Delete the ClusterServiceVersion (CSV)

Find the CSV:

```bash
oc get csv -n openshift-operators | grep kepler
```

Delete it:

```bash
oc delete csv <kepler-operator-csv-name> -n openshift-operators
```

#### Step 4: Delete Custom Resource Definitions (Optional)

**Warning**: This permanently deletes all PowerMonitor resources.

```bash
oc delete crd powermonitors.kepler.system.sustainable.computing.io
oc delete crd powermonitorinternals.kepler.system.sustainable.computing.io
```

#### Step 5: Clean Up Additional Resources

Remove any remaining resources:

```bash
# Check for operator deployments
oc get deployment -n openshift-operators | grep kepler

# Delete if present
oc delete deployment <deployment-name> -n openshift-operators

# Check for service accounts
oc get sa -n openshift-operators | grep kepler

# Delete if present
oc delete sa <sa-name> -n openshift-operators

# Check for cluster roles and bindings
oc get clusterrole | grep kepler
oc get clusterrolebinding | grep kepler

# Delete if present
oc delete clusterrole <clusterrole-name>
oc delete clusterrolebinding <clusterrolebinding-name>
```

#### Verify Complete Removal on OpenShift

```bash
# Check CSV is gone
oc get csv -n openshift-operators | grep kepler

# Check subscription is gone
oc get subscription -n openshift-operators | grep kepler

# Check for remaining resources
oc get all -A -l "app.kubernetes.io/part-of=kepler-operator"

# Check CRDs
oc get crd | grep kepler

# Check ServiceMonitors
oc get servicemonitor -A | grep power-monitor
```

## Removing Monitoring Components (Optional)

### Remove ServiceMonitors

If using prometheus-operator:

```bash
kubectl delete servicemonitor -A -l "app.kubernetes.io/part-of=kepler-operator"
```

### Remove Grafana Dashboards

**OpenShift**:

If you deployed Grafana using the deployment script:

```bash
oc delete deployment grafana -n kepler-operator
oc delete svc grafana -n kepler-operator
oc delete route grafana -n kepler-operator
oc delete configmap grafana-datasources -n kepler-operator
oc delete configmap grafana-dashboards -n kepler-operator
```

**Kubernetes**:

Delete dashboards via Grafana UI or remove dashboard ConfigMaps.

### Preserve Historical Metrics (Optional)

If you want to keep historical energy data:

1. **Prometheus Federation**: Set up a remote Prometheus to federate data
2. **Remote Write**: Configure Prometheus to remote write to long-term storage
3. **Snapshot**: Take Prometheus snapshots before uninstalling

## Troubleshooting Uninstallation

### Resources Not Deleting

If resources remain after deletion:

1. **Check for finalizers**:

   ```bash
   kubectl get powermonitor power-monitor -o yaml | grep finalizers
   ```

   Remove finalizers if stuck:

   ```bash
   kubectl patch powermonitor power-monitor \
     --type json \
     --patch='[{"op": "remove", "path": "/metadata/finalizers"}]'
   ```

2. **Force delete**:

   ```bash
   kubectl delete powermonitor power-monitor --grace-period=0 --force
   ```

### Namespace Stuck in Terminating

If the `kepler-operator` namespace is stuck:

1. **Check for remaining resources**:

   ```bash
   kubectl api-resources --verbs=list --namespaced -o name | xargs -n 1 kubectl get --show-kind --ignore-not-found -n kepler-operator
   ```

2. **Remove finalizers from namespace**:

   ```bash
   kubectl get namespace kepler-operator -o json | \
     jq '.spec.finalizers = []' | \
     kubectl replace --raw "/api/v1/namespaces/kepler-operator/finalize" -f -
   ```

### CRDs Cannot Be Deleted

If CRDs refuse to delete:

1. **Delete the instance first**:

   ```bash
   kubectl delete powermonitor power-monitor
   ```

2. **Remove finalizers from CRD**:

   ```bash
   kubectl patch crd powermonitors.kepler.system.sustainable.computing.io \
     --type json \
     --patch='[{"op": "remove", "path": "/metadata/finalizers"}]'
   ```

3. **Force delete**:

   ```bash
   kubectl delete crd powermonitors.kepler.system.sustainable.computing.io --grace-period=0 --force
   ```

### OLM Resources Remain on OpenShift

If OLM resources persist after CSV deletion:

1. **Check InstallPlan**:

   ```bash
   oc get installplan -n openshift-operators | grep kepler
   ```

   Delete if present:

   ```bash
   oc delete installplan <installplan-name> -n openshift-operators
   ```

2. **Check OperatorGroup** (only if no other operators in namespace):

   ```bash
   oc get operatorgroup -n openshift-operators
   ```

## Post-Uninstallation

After complete removal:

1. **Verify no Kepler metrics** in Prometheus:

   Query Prometheus for Kepler metrics - should return no results:

   ```promql
   {__name__=~"kepler_.*"}
   ```

2. **Check system resources** are freed:

   ```bash
   kubectl top nodes
   ```

   CPU/memory usage should reflect Kepler removal.

3. **Optional: Remove monitoring stack** (if no longer needed):

   **Kubernetes**:

   ```bash
   # If using kube-prometheus-stack
   helm uninstall prometheus -n monitoring
   helm uninstall grafana -n monitoring
   ```

   **OpenShift**:

   OpenShift monitoring is cluster-managed - do not remove.

4. **Optional: Remove cert-manager** (Kubernetes only, if no other users):

   **Warning**: Only remove if no other operators/services use cert-manager.

   ```bash
   kubectl delete -f https://github.com/cert-manager/cert-manager/releases/download/v1.18.2/cert-manager.yaml
   ```

## Reinstallation

If you want to reinstall Kepler Operator later:

1. Follow the appropriate installation guide:
   - [Kubernetes Installation](../installation/kubernetes.md)
   - [OpenShift Installation](../installation/openshift.md)

2. Restore PowerMonitor resources from backup:

   ```bash
   kubectl apply -f powermonitor-backup.yaml
   ```

3. Restore Grafana dashboards (if backed up)

## Next Steps

- **[Kubernetes Installation](../installation/kubernetes.md)** - Reinstall on Kubernetes
- **[OpenShift Installation](../installation/openshift.md)** - Reinstall on OpenShift
- **[Getting Help](https://github.com/sustainable-computing-io/kepler-operator/issues)** - Report issues
