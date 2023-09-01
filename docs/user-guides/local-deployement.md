# Local Deployment of Kepler with Kepler-Operator on Kubernetes/OpenShift

This guide will walk you through deploying Kepler with the Kepler-Operator on your local Kubernetes or OpenShift cluster.

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
cd kepler-operator
```

### 2. Install the Kepler-Operator using Helm

Install the Kepler-Operator using Helm. Ensure you specify the keplerOperator.enabled option to enable Kepler in your deployment:

```bash
helm install kepler-operator ./helm/kepler-operator --set keplerOperator.enabled=true
```

### 3. Create a Custom Resource (CR):

Define a Custom Resource (CR) to specify your Kepler deployment configuration. You can create a YAML file for your CR or use one of the sample CRs provided in the `config/samples`` directory. For example:

```bash
kubectl apply -f config/samples/kepler_v1alpha1_kepler.yaml
```
Make sure to customize the CR according to your requirements, including the probes, energy data collection, and Prometheus configuration.

### 4. Monitor Deployment:

Monitor the deployment of Kepler by checking the status of the custom resource:

```bash
kubectl get keplers
```
Wait until the Kepler deployment is running and healthy.


### 5. Access Kepler Metrics:

Kepler exports energy-related system metrics as Prometheus metrics. You can access these metrics by setting up Prometheus and Grafana, which can be deployed separately or as part of your Kubernetes/OpenShift cluster.

- ### Prometheus: 
Deploy Prometheus on your cluster and configure it to scrape metrics from Kepler. You can set up Prometheus manually or use Helm charts for Prometheus.

- ### Grafana: 
If you prefer a visualization dashboard, you can also deploy Grafana and configure it to visualize Prometheus metrics. Grafana provides powerful visualization and alerting capabilities.

### 6. Clean Up:

When you're finished, you can clean up your deployment:

- Delete the Kepler CR:

```bash
kubectl delete -f config/samples/kepler_v1alpha1_kepler.yaml
```
- Uninstall the Kepler-Operator:

```bash 
helm uninstall kepler-operator
```
This will remove the Kepler deployment and the Kepler-Operator from your cluster.

By following these steps, you can deploy Kepler with the Kepler-Operator on your Kubernetes or OpenShift cluster, collect energy-related metrics, and visualize them using tools like Prometheus and Grafana. Remember to customize the Kepler CR and Prometheus/Grafana configurations to suit your specific monitoring needs.




