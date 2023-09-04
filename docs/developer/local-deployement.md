# Local Deployment of Kepler with Kepler-Operator on Kubernetes/OpenShift

This guide will walk you through deploying Kepler with the Kepler-Operator 
on your local Kubernetes or OpenShift cluster.

## Prerequisites

Before you begin, ensure you have the following prerequisites:

- A running Kubernetes or OpenShift cluster.
- `kubectl` or `oc` (OpenShift CLI) configured to access the cluster.
- Helm installed on your local machine.

## Steps

### 1. Clone the Kepler-Operator Repository

Clone the Kepler-Operator repository from GitHub:

```bash
git clone https://github.com/sustainable-computing-io/kepler-operator.git
# clones the git repo 
cd kepler-operator
# move to the kepler-operator repository in your local system
```

### 2. Create the development env
```bash
make   #  runs `make fresh` and is the default target 
```

### 3. Install the Kepler-Operator using Helm

Install the Kepler-Operator using Helm. Ensure you specify the 
keplerOperator.enabled option to enable Kepler in your deployment:

```bash
helm install kepler-operator ./helm/kepler-operator --set keplerOperator.enabled=true
#helm command used to install helm operator kepler-operator
```

### 4. Create a Custom Resource (CR):

Define a Custom Resource (CR) to specify your Kepler deployment configuration. 
You can create a YAML file for your CR or use one of the sample CRs provided 
in the `config/samples`` directory. For example:

```bash
kubectl apply -f config/samples/kepler_v1alpha1_kepler.yaml
# running this kubectl apply command will create or update a Kubernetes resource 
# based on the definition provided in the kepler_v1alpha1_kepler.yaml file
```
Make sure to customize the CR according to your requirements, including 
the probes,  energy data collection, and Prometheus configuration.

### 5. Monitor Deployment:

Monitor the deployment of Kepler by checking the status of the custom resource:

```bash
kubectl get keplers
# This is a command for interacting with a Kubernetes cluster. 
# It's used to retrieve information about Kubernetes resources
```
Wait until the Kepler deployment is running and healthy.


### 6. Access Kepler Metrics:

Kepler exports energy-related system metrics as Prometheus metrics. 
You can access these metrics by setting up Prometheus and Grafana, 
which can be deployed separately or as part of your Kubernetes/OpenShift cluster.

- ### Prometheus: 
Deploy Prometheus on your cluster and configure it to scrape metrics from Kepler. 
You can set up Prometheus manually or use Helm charts for Prometheus.

- ### Grafana: 
If you prefer a visualization dashboard, you can also deploy Grafana and 
configure it to visualize Prometheus metrics. Grafana provides powerful visualization 
and alerting capabilities.

### 7. Clean Up:

When you're finished, you can clean up your deployment:

- Delete the Kepler CR:

```bash
kubectl delete -f config/samples/kepler_v1alpha1_kepler.yaml
# kubectl delete is a command for interacting with a Kubernetes cluster. 
# It's used to delete Kubernetes resources.
```
- Uninstall the Kepler-Operator:

```bash 
helm uninstall kepler-operator
# After running this command, the kepler-operator and its associated resources 
# should be removed from your Kubernetes cluster, effectively 
# uninstalling the Kepler-Operator and any Kepler instances it was managing.
```
This will remove the Kepler deployment and the Kepler-Operator from your cluster.

By following these steps, you can deploy Kepler with the Kepler-Operator on your 
Kubernetes or OpenShift cluster, collect energy-related metrics, and visualize them 
using tools like Prometheus and Grafana. Remember to customize the Kepler CR and 
Prometheus/Grafana configurations to suit your specific monitoring needs.




