<!-- SPDX-FileCopyrightText: 2025 The Kepler Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# E2E Tests

End-to-end tests for the Kepler Operator, validating the full operator lifecycle against a real Kubernetes or OpenShift cluster.

## Running Tests

### Prerequisites

- A running Kubernetes (Kind) or OpenShift cluster
- The operator deployed via OLM (`tests/run-e2e.sh` handles this)
- `kubectl` configured to access the cluster

### Full E2E Suite

The `run-e2e.sh` script builds the operator, deploys it via OLM, and runs all E2E tests:

```bash
# Build, deploy, and run all E2E tests
./tests/run-e2e.sh e2e

# Run on a VM environment (enables fake CPU meter)
./tests/run-e2e.sh e2e --ci -- -running-on-vm

# Run upgrade tests
./tests/run-e2e.sh upgrade
```

### Skip Build/Deploy

When the operator is already deployed and you want to iterate on test code:

```bash
./tests/run-e2e.sh e2e --no-deploy
```

### Run Specific Tests

Pass Ginkgo flags after `--` to filter specs by name:

```bash
# Run only specs matching "Reconciliation"
./tests/run-e2e.sh e2e --no-deploy -- --ginkgo.focus "Reconciliation"

# Run only PowerMonitor specs
./tests/run-e2e.sh e2e --no-deploy -- --ginkgo.focus "PowerMonitor"

# Skip slow RBAC tests
./tests/run-e2e.sh e2e --no-deploy -- --ginkgo.skip "RBAC"
```

### Run Against OpenShift

```bash
./tests/run-e2e.sh e2e --ns openshift-operators -- -openshift
```

### Script Options

| Flag | Description |
|------|-------------|
| `--ci` | CI mode (skips interactive checks) |
| `--no-deploy` | Skip build and deploy; reuse existing operator deployment |
| `--no-builds` | Skip image builds; useful when images are pre-built |
| `--ns NAMESPACE` | Operator namespace (default: `operators`) |
| `--image-base URL` | Image registry base (default: `localhost:5001`) |

### Test Flags

These flags are passed after `--`:

| Flag | Default | Description |
|------|---------|-------------|
| `-openshift` | `false` | Run against an OpenShift cluster |
| `-running-on-vm` | `false` | Enable VM test mode (fake CPU meter) |
| `-kepler-image` | `quay.io/sustainable_computing_io/kepler:v0.11.3` | Kepler image to use |
| `-deployment-namespace` | `power-monitor` | Namespace for Kepler components |

### Test Output

Logs are written to `tmp/e2e/`:

- `e2e.log` -- full test output
- `*-events.log` -- Kubernetes events during the test run
- `operator-errors.log` -- operator error logs (on failure)

## Test Architecture

The E2E tests use [Ginkgo v2](https://onsi.github.io/ginkgo/) with [Gomega](https://onsi.github.io/gomega/) matchers, following [Operator SDK testing recommendations](https://sdk.operatorframework.io/docs/building-operators/golang/testing/). Tests run against a real cluster (Kind in CI, any cluster locally) with the operator deployed via OLM.

### Directory Structure

```bash
tests/
├── e2e/                          # Test specs
│   ├── suite_test.go             # Ginkgo suite bootstrap and flags
│   ├── power_monitor_test.go     # PowerMonitor CR tests
│   ├── power_monitor_internal_test.go        # PowerMonitorInternal tests
│   └── power_monitor_internal_secrets_test.go # Secret mount and lifecycle tests
├── utils/                        # Test framework
│   ├── assertions.go             # Polling helpers and assertion functions
│   ├── framework.go              # Framework: resource CRUD, cleanup, cert management
│   └── power_monitor_internal_builder.go # Builder for PowerMonitorInternal specs
├── testdata/                     # Test fixtures
├── run-e2e.sh                    # Main entry point for running E2E tests
└── helm.sh                       # Helm-specific E2E tests
```

### Framework

| File | Purpose |
|------|---------|
| `tests/e2e/suite_test.go` | Ginkgo suite bootstrap, flag registration, default timeouts |
| `tests/utils/framework.go` | `Framework` struct: Kubernetes client, resource CRUD, cleanup, cert-manager helpers |
| `tests/utils/assertions.go` | Polling helpers (`WaitUntil`, `ExpectResourceExists`, etc.) using `Eventually` |
| `tests/utils/power_monitor_internal_builder.go` | Builder pattern for constructing `PowerMonitorInternal` specs |

The `Framework` wraps a controller-runtime `client.Client` and provides helpers for:

- **Resource lifecycle**: `CreatePowerMonitor`, `DeletePowerMonitor`, `CreateTestSecret`, etc. Each create method registers a `DeferCleanup` callback so resources are automatically cleaned up when the spec completes.
- **Polling assertions**: `ExpectResourceExists`, `ExpectPowerMonitorCondition`, etc. use Gomega's `Eventually` to poll the cluster until a condition is met or the timeout expires.
- **Node operations**: `AddResourceLabels`, `TaintNode` -- modify node state with automatic cleanup via `DeferCleanup`.
- **Cert-manager integration**: `DeployOpenshiftCerts`, `CreateSelfSignedClusterIssuer`, etc. for TLS certificate tests.

### Test Structure

Tests are organized using Ginkgo's `Describe`/`Context`/`It` hierarchy:

```go
Describe("PowerMonitor")           # Top-level grouping by CRD
  Describe("Reconciliation")       # Feature area
    It("should reconcile ...")     # Specific behavior
  Describe("NodeSelector")
    It("should deploy only ...")
    It("should remain ...")
```

### Conventions

- All top-level `Describe` blocks use the `Serial` decorator to prevent parallel execution. These tests create cluster-scoped resources (PowerMonitor CRs, namespaces, node labels/taints) that would conflict if run concurrently.
- Each `BeforeEach` creates a fresh `Framework` instance. The `Framework` creates a new controller-runtime client each time, but this is lightweight since it only establishes a REST config connection.
- `By()` annotations mark major logical steps in longer tests. This improves Ginkgo's output when a test fails, showing exactly which step was in progress.
- `DeferCleanup` handles all resource teardown. You do not need to write explicit cleanup in `AfterEach`.

### Test Files

| File | What it tests |
|------|---------------|
| `power_monitor_test.go` | PowerMonitor CR: reconciliation, deletion, node selectors, taints/tolerations, config maps, RBAC mode, config fields, security context, secret lifecycle |
| `power_monitor_internal_test.go` | PowerMonitorInternal CR: basic reconciliation, RBAC with TLS certificates |
| `power_monitor_internal_secrets_test.go` | Secret mounting: single/multiple secrets, validation failures, lifecycle CRUD, duplicate mount paths, auto-redeploy on secret change |

## Writing a New Test

### 1. Choose the right file

Add your test to the existing file that matches the CRD or feature you're testing. Create a new file only if testing a completely new CRD or cross-cutting concern.

### 2. Follow the pattern

```go
Describe("YourFeature", func() {
    It("should do something specific", func() {
        // Pre-condition
        f.ExpectNoResourceExists("power-monitor", "", &v1alpha1.PowerMonitor{})

        // Action
        f.CreateTestPowerMonitor("power-monitor", runningOnVM)

        // Assertion (uses Eventually internally)
        pm := f.ExpectPowerMonitorCondition("power-monitor",
            v1alpha1.Available, v1alpha1.ConditionTrue)

        // Verify specific fields
        ds := &appsv1.DaemonSet{}
        f.ExpectResourceExists(pm.Name, controller.PowerMonitorDeploymentNS, ds)
        Expect(ds.Spec.Template.Spec.HostPID).To(BeTrue())
    })
})
```

### 3. Key guidelines

- **No manual cleanup**: Use `f.CreateTestPowerMonitor` / `f.CreateTestSecret` / etc. which register `DeferCleanup` automatically. If you create resources with raw client calls, wrap cleanup in `DeferCleanup`.
- **Use Eventually for async state**: Never assert on cluster state with a bare `Expect`. Use `f.ExpectResourceExists`, `f.ExpectPowerMonitorCondition`, or wrap in `Eventually`.
- **Use custom timeouts when needed**: Default timeout is 2 minutes. For operations that may take longer (cert provisioning, RBAC setup), pass `utils.Timeout(5*time.Minute)`.
- **Guard node access**: Always check `Expect(nodes).NotTo(BeEmpty())` before accessing `nodes[0]`.
- **Support both Kubernetes and OpenShift**: Use `if Cluster == k8s.Kubernetes { ... } else { ... }` when behavior differs between platforms (e.g., cert-manager vs OpenShift service certificates).

## CI Integration

E2E tests run on every PR via GitHub Actions (`.github/workflows/pr-checks.yaml`):

| CI Job | Command | What it does |
|--------|---------|--------------|
| `e2e` | `./tests/run-e2e.sh e2e --ci -- -running-on-vm` | Full E2E suite on Kind with 2 worker nodes |
| `operator-upgrade` | `./tests/run-e2e.sh upgrade` | Tests OLM bundle upgrade from previous release |
| `helm-e2e` | `.github/helm-e2e` action | Helm chart installation and validation |

The `run-e2e.sh` script:

1. Builds the operator and bundle images
2. Pushes them to the Kind local registry
3. Deploys the operator via `operator-sdk run bundle`
4. Waits for the operator to be ready (including webhook validation)
5. Runs `go test ./e2e/...` with the configured flags
6. Collects event logs and operator error logs on failure

### Test Artifacts

On failure, CI archives the `tmp/e2e/` directory containing:

- `e2e.log` -- full Ginkgo test output
- `*-events.log` -- Kubernetes events from operator and PowerMonitor namespaces
- `operator-errors.log` -- filtered operator error logs
