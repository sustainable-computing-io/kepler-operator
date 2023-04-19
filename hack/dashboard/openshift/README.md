# Enabling Dashboard for Kepler on OpenShift

The following cmd will:
- Enable OpenShift User Workload Monitoring
- Deploy Grafana operator
- Create and configure Grafana instance for Kepler
- Define Prometheus datasource
- Define Grafana dashboard

```bash
$(pwd)/deploy-grafana.sh
```


# Test if `kepler-exporter` metrics are scraped by OpenShift user workload monitoring

```bash
kubectl exec -ti -n openshift-user-workload-monitoring prometheus-user-workload-0 -- bash -c 'curl "localhost:9090/api/v1/query?query=kepler_container_package_joules_total[5s]"' 
```