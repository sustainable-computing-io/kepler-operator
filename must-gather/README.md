# Power Monitoring Must-Gather

The Power Monitoring `must-gather` tool is designed to collect
information about power monitoring components within an OpenShift cluster.
This tool extends the functionality of [OpenShift must-gather](https://github.com/openshift/must-gather)
to specifically target and retrieve data related to power monitoring,
including support for both the upstream Kepler Operator and the
Power Monitoring Operator.

## Usage

To run the must-gather, use one of the following
commands, depending on the operator and namespace where it is deployed

### Using the image from the Operator deployment

```sh
oc adm must-gather --image=$(oc -n <namespace> get deployment.apps/kepler-operator-controller -o jsonpath='{.spec.template.spec.containers[?(@.name == "manager")].image}') -- /usr/bin/gather --operator <operator-name> --ns <namespace>
```

Replace `<namespace>` with the namespace where the operator is deployed, and
`<operator-name>` with the name of the operator(e.g. `kepler-operator` or `power-monitoring-operator`).

### Using a specific image

```sh
oc adm must-gather --image=quay.io/sustainable_computing_io/kepler-operator:v1alpha1 -- /usr/bin/gather --operator <operator-name> --ns <namespace>

```

Running these commands will collect and store information in a newly created directory, based on the specified operator and namespace.
