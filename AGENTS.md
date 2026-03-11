<!-- SPDX-FileCopyrightText: 2025 The Kepler Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# AGENTS.md

> AI Agent Guide for **Kepler Operator** — a Kubernetes operator that automates
> deployment of [Kepler](https://github.com/sustainable-computing-io/kepler)
> (Kubernetes Efficient Power Level Exporter) for energy consumption monitoring
> on Kubernetes/OpenShift clusters.

## Quick Reference

```bash
# Before any PR
make fmt vet test

# After CRD/API changes
make manifests generate bundle docs helm-sync-crds

# Local development
make cluster-up     # Setup KinD cluster
make run            # Run operator locally

# See all targets
make help
```

## Agent Behavioral Rules

These rules are **non-negotiable** and apply to all AI agents working on this
codebase.

1. **Verify before claiming.** Do not invent facts about the codebase — read
   the code before making claims or changes.
2. **Do not over-engineer.** Implement exactly what is asked, nothing more. No
   speculative features, unnecessary abstractions, or unsolicited refactoring.
3. **Ask when unclear.** If requirements are ambiguous, ask the user rather than
   guessing.
4. **Report failing tests honestly.** If a test fails unexpectedly, report the
   failure. Do not modify the test to make it pass without understanding why it
   failed.
5. **Use `make` targets.** Always prefer `make` targets over raw `go`
   commands — only run raw commands if no `make` target exists for that
   operation.
6. **Never force push to `main`.** No exceptions.
7. **Never edit auto-generated files.** This includes `bundle/`,
   `docs/user/reference/api.md`, and `zz_generated.deepcopy.go`. Regenerate
   them using `make` targets instead.
8. **Keep docs in sync with code.** If a change affects behavior, APIs, or
   workflows, update the relevant documentation in the same PR. Do not leave
   documentation debt behind.
9. **Sign all commits.** Use `git commit -s` for DCO sign-off. Never commit
   without human approval.

## Project Overview

**Tech Stack**: Go 1.24.0+, Operator SDK, controller-runtime, testify

```text
kepler-operator/
├── api/v1alpha1/       # CRD definitions (PowerMonitor, PowerMonitorInternal)
├── cmd/                # Operator entrypoint
├── internal/
│   ├── config/         # Configuration management and builder
│   └── controller/     # Reconciler logic
├── pkg/
│   ├── components/     # Kubernetes resource builders (deployments, services)
│   ├── reconciler/     # Generic reconciliation framework (Action pattern)
│   ├── utils/          # Kubernetes and general utilities
│   └── version/        # Version info
├── config/             # Kustomize configs (operator-sdk scaffolding)
├── manifests/helm/     # Helm chart
├── bundle/             # Auto-generated OLM bundle — do not edit
├── tests/              # E2E tests (separate Go module)
├── hack/               # Development scripts
└── docs/               # User and developer documentation
```

**Architecture**: See [docs/developer/architecture.md](docs/developer/architecture.md)

## Permissions

### Allowed (No Approval Needed)

- Read any files
- Run quality checks: `make fmt`, `make vet`, `make test`
- Create/update tests and documentation
- Refactor code following existing patterns
- Fix linter errors and test failures

### Requires Human Approval

- Git operations (commit, push, branch)
- Dependency changes (`go.mod`, `go.sum`)
- Infrastructure changes (CI/CD, Makefile, `config/` directory)
- Deployments and releases
- Architectural or API changes

### Never Allowed

- Force push to `main`
- Skip pre-commit hooks or security checks
- Unsigned commits (missing DCO)
- Breaking API changes without migration guide
- Editing auto-generated files directly

## Code Standards

### Go

- **SPDX Headers** required in all `.go` files:

  ```go
  // SPDX-FileCopyrightText: 2025 The Kepler Authors
  // SPDX-License-Identifier: Apache-2.0
  ```

- **Idiomatic Go**: Follow [Effective Go](https://go.dev/doc/effective_go). Use
  short, focused functions (under 50-60 lines), meaningful names, and proper
  error handling.
- **Testing**: Use `testify` (assert/require). Table-driven tests for multiple
  scenarios. Maintain >80% coverage.
- **GoDoc**: Add comments to all exported types, constants, and fields —
  especially CRD types in `api/v1alpha1/*_types.go`.
- **Terminology**: Use `PowerMonitor` for CRD kinds and Go types,
  `power-monitor` for file names and resource names.

### Pre-commit Checks

The project uses pre-commit hooks
(see [docs/developer/pre-commit-hooks.md](docs/developer/pre-commit-hooks.md)):

- `make fmt` — Format Go code and shell scripts
- `make vet` — Go vet analysis
- `make test` — Unit tests with coverage
- golangci-lint, yamllint, shellcheck, markdownlint, codespell
- SPDX header compliance via reuse-lint-file
- Commit message validation via commitlint (conventional commits)

## Testing

```bash
make test           # Run unit tests
make coverage       # Generate HTML coverage report
```

- **Framework**: `testify` (assert/require)
- **Pattern**: Table-driven tests for multiple scenarios
- **Coverage**: >80% required
- **E2E Tests**: `./tests/run-e2e.sh e2e` (separate Go module in `tests/`)

See existing tests in `internal/` and `pkg/` for patterns.

## Contribution Workflow

### Commit Standards

Use [Conventional Commits](https://www.conventionalcommits.org/) with DCO
sign-off:

```bash
git commit -s -m "type(scope): description"
# Types: feat, fix, docs, style, refactor, test, chore, ci, perf
```

### PR Checklist

1. Run `make fmt vet test`
2. If CRD types changed: `make manifests generate bundle docs helm-sync-crds`
3. Title uses conventional commit format
4. One logical change per PR
5. All CI checks pass (see `.github/workflows/pr-checks.yaml`)

### Auto-Generated Files

These files are regenerated from source — never edit them directly:

| File/Directory | Regenerate with |
|----------------|-----------------|
| `bundle/` | `make bundle` |
| `docs/user/reference/api.md` | `make docs` |
| `zz_generated.deepcopy.go` | `make generate` |
| Helm CRDs | `make helm-sync-crds` |

### Documentation

When changing code, update relevant docs in the same PR:

- **README.md** — Project overview and quick start
- **docs/user/** — Installation and usage guides
- **docs/developer/** — Architecture, development workflows
- **CONTRIBUTING.md** — Contribution guidelines
- **CRD types** — GoDoc comments in `api/v1alpha1/*_types.go` (source for
  auto-generated API docs)

Do not edit `docs/user/reference/api.md` directly — it is generated from CRD
type comments via `make docs`.

## Getting Help

- **All make targets**: `make help`
- **Developer docs**: [docs/developer/](docs/developer/)
- **User docs**: [docs/user/](docs/user/)
- **Contributing guide**: [CONTRIBUTING.md](CONTRIBUTING.md)
- **Community**: CNCF Slack `#kepler`
- **Issues**: [GitHub Issues](https://github.com/sustainable-computing-io/kepler-operator/issues)
