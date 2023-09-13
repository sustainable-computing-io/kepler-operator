# Kepler-Operator Test Plan

**These scenarios only cover the targeted version of OpenShift i.e 4.13**

## Installation

### Kepler Operator

#### Positive

- Kepler operator installs fine from Community Catalog

  > 👉 This is to be verified from OpenShift Console

- Kepler operator installs fine from Operator-SDK.

  > 👉 This is to be verified by Upstream/Downstream CI.

- Kepler operator should be able to deploy/handle any dependencies required.

  > 👉 This includes the ability of the operator to provision, configure, and manage resources such as pods, services, volumes, etc.

- Following resources should get deployed as part of Operator Installation:
  - `kepler-operator-controller-manager deployment`
    - `kube-rbac-proxy`
    - `manager`
  - `kepler-operator-controller-manager replicaset`
  - `kepler-operator-controller-manager-metrics-service`
  - `Kepler CRD`
  - `kepler-operator-controller-manager Service Account`
  - `roles`
  - `rolebindings`
  - `lease`
  - `servicemonitor`
  - `kepler-operator CSV`
  - `installplan`
  - `subscription`
  - `packagemanifests`
  - `catalog`

- The Operator should not enter into an infinite reconcile loop. This can be
  verified by:
	- checking the logs of the Operator.
	- checking if the `metadata.resourceVersion` keeps increasing / changing in a
	  short interval say every 2-5 seconds.

#### Negative:

- OpenShift console should log appropriate errors in case of unsuccessful installation of Operator.
  > 👉 This is to be verified from OpenShift Console.
           It can be related to any failure scenario or in case of unsuccessful installation.

### Kepler Instance

#### Positive:

- Kepler operator when installed from Community Catalog can create a Kepler instance successfully.

  > 👉 This is to be verified from OpenShift console

          Kepler operator can reconcile `kepler` CR

- Kepler operator when installed from Operator-SDK can deploy Kepler instance when Kepler template is applied.

  > 👉 This is to be verified by Upstream/Downstream CI.

- Following resources should get deployed as part of Kepler Instance Creation:
  - `openshift-kepler-operator namespace`
  - `clusterrole`
  - `scc`
  - `clusterrolebindings`
  - `kepler-exporter-ds`
  - `kepler-exporter-cm`
  - `kepler-exporter-svc`
- Kepler operator only exposes `exporter port` at the time of configuring the Kepler Instance.

  - Kepler Instance should be able to run with different port number.
    > 👉 This is to be verified from OpenShift console

- Appropriate status should be reflected for creation of Instance(Reconcillation status).

  > 👉 This includes whether Kepler Instance was deployed successfully on cluster nodes and logs any error on controller manager.

- Kepler exporter daemonsets should only be created on nodes that have taint applied

  > 👉 Kepler instance should respect toleration's if applied else it should stick to the default behavior i.e deploy on all the available nodes.

- Kepler exporter daemonsets should only be created on nodes that have label applied
  > 👉 Kepler instance should respect nodeSelector if applied else it should stick to the default behavior i.e deploy on all the available nodes.

#### Negative

- OpenShift/Kubernetes cluster log error if an invalid port value is passed in the Instance configuration.

- OpenShift/Kubernetes cluster logs an appropriate error if an invalid/wrong configuration key/value is passed.

- Kepler Operator logs an appropriate error if no nodes are available for deployment of Kepler Instance.

  > 👉 Appropriate status should be reflected in this case.

- Appropriate logging of error if an invalid name of Kepler instance is provided.

  > 👉 Kepler-Operator should only respect `kepler` as the instance name. Other than that should be considered invalid.

- Kepler controller manager logs an appropriate error if Instance creation is unable to create required resources.

  > 👉 Availability status should also be updated

- Kepler controller manager logs an appropriate error if it is unable to pull Kepler Image with a pre-configured tag.

  > 👉 Availability status should also be updated

- Appropriate logging of error in case of creating more than one Kepler Instance

  > 👉 Kepler-Operator should not allow creation of more than one Kepler Instance

- Availability status should reflect in case where Kepler-Operator is unable to deploy exporter daemonsets
  > 👉 Appropriate status and message should be reflected

## Metrics

### Kepler Operator Metrics

- Kepler controller manager should be able to expose metrics at the configured port.
- Metric HTTP endpoint should be reachable either via port-forwarding or creating a route on OpenShift.
- Following metrics should be available and updated accordingly:
  - controller_runtime_reconcile_total
  - controller_runtime_max_concurrent_reconciles
  - leader_election_master_status

### Kepler Exporter Metrics

- Kepler exporter should be able to expose metrics at port 9103
  - Metric HTTP endpoint should be reachable either via port-forwarding or creating a route on OpenShift.
- Following metrics should be available and updated accordingly:
  > 👉 These metrics are used inside our grafana dashboard.
  - `kepler_container_package_joules_total`
  - `kepler_container_dram_joules_total`
  - `kepler_container_other_host_components_joules_total`
  - `kepler_container_gpu_joules_total`

## Uninstallation

### Kepler Operator

#### Positive

- Kepler Operator should get uninstalled successfully when deployed using OpenShift console.

- Kepler Operator should get uninstalled successfully when deployed using Operator-SDK.

- Following resources should get uninstalled:
  - `kepler-operator-controller-manager deployment`
    - `kube-rbac-proxy`
    - `manager`
  - `kepler-operator-controller-manager replicaset`
  - `kepler-operator-controller-manager-metrics-service`
  - `kepler-operator-controller-manager Service Account`
  - `kepler-operator CSV`
  - `subscription`
  - `servicemonitor` -> **TBD**

#### Negative

- Appropriate error must be logged in case of unable to remove the Operator from the cluster.
  > 👉 Reconciliation should report the appropriate reason.

### Kepler Instance

#### Positive

- Kepler Instance should get uninstalled successfully using OpenShift Console.

  > 👉 This is to be verified from OpenShift console

- Kepler Operator should get uninstalled successfully using Operator-SDK.

  > 👉 This is to be verified by Upstream/Downstream CI.

- Following resources should get uninstalled:
  - `openshift-kepler-operator namespace`
  - `clusterrole`
  - `scc`
  - `clusterrolebindings`
  - `kepler-exporter-ds`
  - `kepler-exporter-cm`
  - `kepler-exporter-svc`

#### Negative

- Controller manager logs an appropriate error if it is unable to remove Kepler Instance from the cluster and availability status should be updated accordingly.
  > 👉 If something isn't able to uninstall then the related objects should be present on cluster for debugging purposes.

## Grafana Dashboard

### Positive

- `deploy-grafana.sh` should be able to deploy/configure the Grafana dashboard.

  - Configuring `user-workload-monitoring` if not already configured.
  - Install Grafana Operator inside `openshift-kepler-operator` namespace.
  - Configure tokens
  - Configure Service monitors
  - Add Grafana dashboard source.

- Grafana dashboard should not show CO2 carbon emission panels when deployed. (**For Dev Preview**)

- Grafana dashboard should be able to query data from `user-workload-monitoring` Prometheus.

- Grafana dashboard should show tooltips with relevant information on each panel.

- All the panels inside the dashboard should update accordingly.

- `deploy-grafana.sh` should be able to re-load the new dashboard json config-map when re-run.

### Negative

- `deploy-grafana.sh` should log an appropriate error in case of any failure.
