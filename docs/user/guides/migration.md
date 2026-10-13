# Migrating Legacy Kepler Resources to PowerMonitor

The legacy `Kepler` API was removed in operator v0.21.0. Migrate existing
deployments while an operator retaining both APIs, such as v0.20.0, is still
installed. That version handles legacy cleanup but does not recreate the
legacy exporter. Current operators no longer reconcile the legacy API. Pause
automatic OLM upgrades if necessary while planning the migration.

PowerMonitor deploys the rewritten Kepler exporter. This changes configuration,
metric names, and monitoring resources; review the migration in a test cluster
and plan a maintenance window. Parallel deployments can help comparison on a
compatible operator, but do not guarantee uninterrupted metrics.

## Assess and Back Up the Existing Deployment

```bash
kubectl get keplers.kepler.system.sustainable.computing.io
kubectl get keplers.kepler.system.sustainable.computing.io kepler -o yaml > kepler-backup.yaml
kubectl get keplers.kepler.system.sustainable.computing.io kepler -o json > kepler-backup.json
kubectl get daemonset -n kepler -o yaml > kepler-daemonset-backup.yaml
kubectl get servicemonitor -n kepler -o yaml > kepler-servicemonitor-backup.yaml
```

Substitute the actual legacy resource name and component namespace. Record the
operator/exporter versions, node selection, tolerations, Redfish settings,
custom ConfigMaps and Secrets, dashboard queries, and alert rules. Also check
whether a PowerMonitor already exists: only one named `power-monitor` is
supported per cluster. Update that resource rather than creating a second one.

## Map the Supported Configuration

| Legacy Kepler field | PowerMonitor destination |
| --- | --- |
| `spec.exporter.deployment.nodeSelector` | `spec.kepler.deployment.nodeSelector` |
| `spec.exporter.deployment.tolerations` | `spec.kepler.deployment.tolerations` |
| `spec.exporter.deployment.port` | No configurable equivalent; raw exporter port is `28282`, RBAC Service HTTPS port is `8443` |
| `spec.exporter.redfish` | Manual conversion using the [Redfish guide](experimental/redfish.md) and mounted Secrets/ConfigMaps |
| Exporter log settings or environment overrides | Review separately; the legacy API has no `spec.exporter.config.logLevel` field |

PowerMonitor adds options such as `logLevel`, `metricLevels`, `sampleRate`,
`staleness`, `maxTerminated`, and metrics-access security. These are settings
to choose for the new deployment, not automatic mappings from legacy fields.
See [PowerMonitor configuration](../reference/power-monitor.md). Leave security
unspecified to retain platform defaults: RBAC on OpenShift and none on
Kubernetes.

Check the API of the operator version used for migration. For example,
v0.20.0 defaults `maxTerminated` to `500` and does not expose
`spec.kepler.deployment.secrets`; current versions default tracking to `0`
and support those Secret mounts. The linked Redfish guide uses the current
API. If you depend on Redfish, plan and validate its conversion for the target
operator version before retiring the legacy deployment.

## Prepare and Review the New Manifest

This `jq` example converts only the shared scheduling fields from the JSON
backup. It does not apply resources or convert Redfish and custom configuration:

```bash
jq '{
  apiVersion: "kepler.system.sustainable.computing.io/v1alpha1",
  kind: "PowerMonitor",
  metadata: {name: "power-monitor"},
  spec: {kepler: {deployment: (
    (.spec.exporter.deployment // {}) |
    with_entries(select(.key == "nodeSelector" or .key == "tolerations"))
  )}}
}' kepler-backup.json > power-monitor-migration.json
```

JSON is accepted by `kubectl`. Review the resulting manifest, add any required
runtime settings, and resolve Redfish/custom configuration manually. Retain
backups until verification and your rollback window are complete.

```bash
kubectl apply --dry-run=server -f power-monitor-migration.json
kubectl apply -f power-monitor-migration.json
pm_generation=$(kubectl get powermonitor power-monitor -o jsonpath='{.metadata.generation}')
kubectl wait \
  --for=jsonpath='{.status.conditions[?(@.type=="Reconciled")].observedGeneration}'="$pm_generation" \
  powermonitor/power-monitor --timeout=180s
kubectl wait --for=condition=Reconciled powermonitor/power-monitor --timeout=180s
kubectl rollout status daemonset/power-monitor -n power-monitor --timeout=180s
kubectl get powermonitor power-monitor -o yaml
kubectl get pods -n power-monitor -o wide
```

The component namespace defaults to `power-monitor`. Substitute a customized
`--deployment-namespace` value. If readiness fails, inspect conditions and logs
before proceeding.

## Verify Metrics and Update Monitoring

For local diagnostics, forward the raw exporter port:

```bash
kubectl port-forward -n power-monitor daemonset/power-monitor 28282:28282
```

In another terminal, check `http://localhost:28282/metrics`. The RBAC-protected
Service uses HTTPS port `8443`; do not disable security to use this diagnostic
forward.

The operator manages the new `power-monitor` ServiceMonitor. Verify its target
is being scraped using the [Prometheus integration guide](validating-prometheus-integration.md)
rather than copying a legacy ServiceMonitor definition. Update dashboard and
alert expressions to current metrics, for example:

```promql
kepler_node_cpu_watts{job="power-monitor"}
kepler_pod_cpu_watts{job="power-monitor"}
rate(kepler_node_cpu_joules_total{job="power-monitor"}[5m])
```

These CPU metrics are not a direct whole-node-power replacement for legacy
metrics. Review their zones and labels, and use separate platform/GPU metrics
where appropriate. Confirm node coverage, recent samples, dashboards, and
alerts before removing the old deployment.

## Cleanup and Rollback

Before deleting the legacy resource, inspect its actual deployment namespace:

```bash
kubectl get keplerinternals.kepler.system.sustainable.computing.io kepler \
  -o jsonpath='{.spec.exporter.deployment.namespace}{"\n"}'
```

In v0.20.0, deleting the legacy Kepler resource also causes the operator to
delete that namespace. Continue only if it is dedicated to the legacy
deployment and contains no operator installation, new PowerMonitor deployment,
or unrelated resources.
If the namespace is shared, stop and plan version-specific cleanup with the
maintainers; the delete command below is unsafe in that case.

After verification, delete the backed-up legacy custom resource using its
actual name while the compatible operator can still process its finalizers:

```bash
kubectl delete keplers.kepler.system.sustainable.computing.io kepler
```

Check that its managed resources were removed. Do not delete a CRD as a cleanup
shortcut. Complete this step before upgrading beyond the
legacy-compatible operator, and update deployment automation and runbooks.

If validation fails before cleanup, keep the legacy deployment and restore
monitoring to it. After cleanup, applying the custom-resource backup alone does
not restore the exporter under v0.20.0, and v0.21.0+ cannot reconcile the legacy
API. Recovery requires a tested operator version that deploys the legacy
exporter or a reviewed restoration of the backed-up deployment resources.
Operator downgrade and OLM recovery depend on your installation method and
should be planned and tested beforehand. Remove a new PowerMonitor only after
the fallback deployment and monitoring have been verified.

For diagnosis, see [troubleshooting](troubleshooting.md). For the subsequent
operator upgrade, see [upgrading](upgrading.md).

For the migration-compatible operator's API and installation instructions,
see the upstream [v0.20.0 API reference](https://github.com/sustainable-computing-io/kepler-operator/blob/3418c41e2262df4504d200c544ae1e64cd6935b3/docs/api.md#kepler)
and [README](https://github.com/sustainable-computing-io/kepler-operator/blob/3418c41e2262df4504d200c544ae1e64cd6935b3/README.md).
