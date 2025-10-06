# Kepler Operator Must-Gather

`kepler-operator-must-gather` is a tool for collecting diagnostic information about the Kepler Operator and Power Monitoring components within an OpenShift cluster. This tool extends the functionality of [OpenShift must-gather](https://github.com/openshift/must-gather) to specifically target and retrieve data related to power monitoring deployments.

## About

The Kepler Operator must-gather tool collects information about:

- Operator deployment and configuration
- OLM (Operator Lifecycle Manager) resources
- Power monitoring instances (Kepler deployments)
- Prometheus metrics and monitoring configuration
- Pod logs and runtime information
- Hardware information (kernel version, RAPL data)

This tool supports both the upstream **Kepler Operator** and the downstream **Power Monitoring Operator**.

## Prerequisites

- OpenShift CLI (`oc`) installed and configured
- Access to an OpenShift cluster where the operator is deployed
- Appropriate RBAC permissions to collect cluster resources

## Usage

### Using the image from the operator deployment

To run must-gather using the operator's current image:

```sh
oc adm must-gather \
  --image=$(oc -n <namespace> get deployment.apps/kepler-operator-controller \
  -o jsonpath='{.spec.template.spec.containers[?(@.name == "manager")].image}') \
  -- /usr/bin/gather --operator <operator-name> --ns <namespace>
```

**Parameters:**

- `<namespace>`: The namespace where the operator is deployed (e.g., `openshift-operators`)
- `<operator-name>`: The name of the operator (e.g., `kepler-operator` or `power-monitoring-operator`)

**Example:**

```sh
oc adm must-gather \
  --image=$(oc -n openshift-operators get deployment.apps/kepler-operator-controller \
  -o jsonpath='{.spec.template.spec.containers[?(@.name == "manager")].image}') \
  -- /usr/bin/gather --operator kepler-operator --ns openshift-operators
```

### Using a specific image

To use a specific must-gather image:

```sh
oc adm must-gather \
  --image=quay.io/sustainable_computing_io/kepler-operator:v1alpha1 \
  -- /usr/bin/gather --operator <operator-name> --ns <namespace>
```

**Example:**

```sh
oc adm must-gather \
  --image=quay.io/sustainable_computing_io/kepler-operator:v1alpha1 \
  -- /usr/bin/gather --operator kepler-operator --ns openshift-operators
```

### Command-line options

The `gather` script supports the following options:

```sh
Options:
  --operator | -o    Specify the name of the operator that is deployed
                     Default: kepler-operator

  --ns | -n          Namespace where the operator is deployed
                     Default: openshift-operators

  --dest-dir | -d    Gather collection path
                     Default: /must-gather

  --help | -h        Display help information
```

## Collected Information

### Operator Information

- Catalog source, subscription, and install plan
- ClusterServiceVersion (CSV)
- Operator deployment configuration
- Operator pod details and logs

### OLM Resources

- All OLM-managed resources related to the operator
- Summary of operator lifecycle management state

### Power Monitor Components

- PowerMonitor custom resource and internals
- DaemonSet configuration
- ConfigMap settings
- ServiceAccount and SecurityContextConstraints
- Events in the power monitoring namespace

### Pod-Level Information

For each Kepler pod:

- Pod specification and status
- Container logs
- Kernel version information
- RAPL (Running Average Power Limit) capabilities
- Hardware power monitoring capabilities

### Monitoring Information (if available)

- Prometheus rules and configuration
- Active targets
- Time-series database (TSDB) status
- Runtime information from Prometheus replicas

## Troubleshooting

### Common issues

**Issue:** `Error from server (NotFound): deployments.apps "kepler-operator-controller" not found`

**Solution:** Verify the operator is installed and the namespace is correct:

```sh
oc get deployment -n <namespace>
```

**Issue:** `error: insufficient permissions`

**Solution:** Ensure you have cluster-admin privileges or appropriate RBAC permissions to run must-gather.

**Issue:** `cannot gather UWM details; skipping gathering monitoring info`

**Solution:** This is informational. The must-gather will still collect other resources. User Workload Monitoring may not be configured in your cluster.

### Viewing must-gather logs

Check the `gather-debug.log` file in the output directory for detailed information about the collection process:

```sh
cat must-gather.local.<timestamp>/gather-debug.log
```

## Development

### Collection Scripts

Data collection scripts are located in:

- `gather` - Main collection script
- `utils` - Utility functions for logging and command execution

### Testing Changes

To test must-gather changes locally:

1. Build the operator image with your changes:

   ```sh
   make operator-build operator-push IMG_BASE=<your-registry>
   ```

2. Run must-gather with your custom image:

   ```sh
   oc adm must-gather \
     --image=<your-registry>/kepler-operator:<tag> \
     -- /usr/bin/gather --operator kepler-operator --ns openshift-operators
   ```

3. Verify the collected data in the output directory

## Additional Resources

- [OpenShift must-gather documentation](https://docs.openshift.com/container-platform/latest/support/gathering-cluster-data.html)

## Contributing

For issues or improvements related to must-gather:

- Report issues: [GitHub Issues](https://github.com/sustainable-computing-io/kepler-operator/issues)
- Submit pull requests: [GitHub Pull Requests](https://github.com/sustainable-computing-io/kepler-operator/pulls)
