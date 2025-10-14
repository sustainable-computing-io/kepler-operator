# Installing Kepler Operator on OpenShift

This guide walks you through installing the Kepler Operator on OpenShift using OperatorHub or the Community Catalog.

## Prerequisites

- **OpenShift cluster** running version 4.x or higher
- **oc CLI** or access to the OpenShift Web Console
- Appropriate cluster permissions to install operators

### Built-in OpenShift Features

Unlike vanilla Kubernetes, OpenShift includes several features out-of-the-box that simplify Kepler deployment:

- **Service Serving Certificates**: Automatic TLS certificate management (no cert-manager required)
- **Prometheus Operator**: Built-in monitoring stack
- **User Workload Monitoring**: Optional monitoring for user applications

## Installation via OpenShift Web Console (Recommended)

### Step 1: Navigate to OperatorHub

1. Log in to the OpenShift Web Console
2. Navigate to **Operators** → **OperatorHub**
3. In the search box, type "Kepler Operator"

<!-- TODO: Add screenshot of OperatorHub catalog showing Kepler Operator -->

### Step 2: Install the Operator

1. Click on the **Kepler Operator** tile
2. Click **Install**
3. Configure installation options:
   - **Update channel**: Select `alpha` (or the latest stable channel)
   - **Installation mode**: Choose one of:
     - **All namespaces on the cluster** (recommended) - Operator can manage PowerMonitors in any namespace
     - **A specific namespace** - Operator only manages resources in selected namespace
   - **Installed Namespace**: Select `openshift-operators` (recommended) or create a custom namespace
   - **Update approval**: Choose automatic or manual updates

<!-- TODO: Add screenshot of installation configuration page -->

1. Click **Install**

### Step 3: Verify Installation

1. Navigate to **Operators** → **Installed Operators**
2. Ensure the namespace filter shows your selected namespace (e.g., `openshift-operators`)
3. Verify **Kepler Operator** appears with status **Succeeded**

<!-- TODO: Add screenshot of installed operators page showing Kepler Operator -->

The operator is now ready to manage PowerMonitor resources.

## Installation via CLI

### Using operator-sdk

If you prefer command-line installation, you can use the `operator-sdk` tool:

```bash
# Install the operator-sdk if not already installed
# See https://sdk.operatorframework.io/docs/installation/

# Run the operator bundle
operator-sdk run bundle \
  quay.io/sustainable_computing_io/kepler-operator-bundle:latest \
  --install-mode AllNamespaces \
  --namespace openshift-operators
```

### Using OLM Subscription (Advanced)

Create a Subscription resource manually:

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: kepler-operator
  namespace: openshift-operators
spec:
  channel: alpha
  name: kepler-operator
  source: community-operators
  sourceNamespace: openshift-marketplace
  installPlanApproval: Automatic
```

Apply the subscription:

```bash
oc apply -f kepler-operator-subscription.yaml
```

## Verification

### Check Operator Status

Verify the operator is running:

```bash
oc get pods -n openshift-operators | grep kepler-operator
```

Expected output:

```text
kepler-operator-controller-xxxxx-yyyyy  2/2   Running  0  1m
```

### Check ClusterServiceVersion (CSV)

```bash
oc get csv -n openshift-operators | grep kepler
```

You should see the Kepler Operator CSV with `Succeeded` phase:

```text
kepler-operator.v0.21.0   Kepler Operator   0.21.0   Succeeded
```

### Check Custom Resource Definitions

Verify the PowerMonitor CRD is installed:

```bash
oc get crd powermonitors.kepler.system.sustainable.computing.io
```

## OpenShift-Specific Features

### Security Context Constraints (SCC)

Kepler requires privileged access to read hardware sensors. The operator automatically configures appropriate Security Context Constraints (SCC) for Kepler pods.

You can verify the SCC configuration:

```bash
oc get scc | grep kepler
```

### Integration with OpenShift Monitoring

Kepler automatically integrates with OpenShift's built-in monitoring stack through ServiceMonitor resources. No additional configuration is needed for basic metrics collection.

To enable User Workload Monitoring (if not already enabled):

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

## Next Steps

Now that the operator is installed:

1. **[Create a PowerMonitor resource](../reference/power-monitor.md)** to deploy Kepler
2. **[Configure PowerMonitor](../reference/power-monitor.md)** for your cluster
3. **[Set up Grafana dashboards](../guides/grafana-dashboard.md)** to visualize energy metrics

## Troubleshooting

### Operator Not Appearing in OperatorHub

If Kepler Operator doesn't appear in OperatorHub:

1. Check if the Community Operators catalog source is enabled:

   ```bash
   oc get catalogsource -n openshift-marketplace
   ```

2. Look for `community-operators` in the list. If missing, contact your cluster administrator.

### Installation Stuck in "Installing" State

Check the install plan status:

```bash
oc get installplan -n openshift-operators
oc describe installplan <install-plan-name> -n openshift-operators
```

### CSV Failures

View CSV details to understand failures:

```bash
oc describe csv kepler-operator.v<version> -n openshift-operators
```

Check operator logs:

```bash
oc logs -n openshift-operators deployment/kepler-operator-controller
```

For more detailed troubleshooting, see the [Troubleshooting Guide](../guides/troubleshooting.md#openshift-specific-issues).

## Upgrading

OpenShift operators can be upgraded automatically or manually based on your installation configuration.

See the [Upgrading Guide](../guides/upgrading.md#upgrading-on-openshift-olm) for details.

## Uninstallation

To uninstall Kepler Operator from OpenShift, see the [Uninstallation Guide](../reference/uninstallation.md#uninstalling-via-olm-openshift).
