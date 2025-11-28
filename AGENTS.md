<!-- SPDX-FileCopyrightText: 2025 The Kepler Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# AGENTS.md

> **AI Agent Guide for Kepler Operator** - Essential information for AI coding assistants (GitHub Copilot, Claude, Codex) contributing to this project.

## 🚀 Quick Reference Card

```bash
# Essential Commands
make fmt vet test    # Run before any PR
make cluster-up      # Setup local cluster
make run            # Run operator locally
git commit -s       # DCO sign-off required

# Key Rules
✓ SPDX headers in all Go files
✓ Use testify for tests
✓ Follow conventional commits
✓ Keep docs synchronized across files
✗ Never commit without human approval
✗ Never force push to main

# Documentation Commands
make docs            # Regenerate API docs after CRD changes
grep -r "version"    # Check version consistency
```

## 📋 Table of Contents

1. [Project Overview](#project-overview)
2. [AI Agent Permissions](#ai-agent-permissions)
3. [Code Standards](#code-standards)
4. [Testing Requirements](#testing-requirements)
5. [Contribution Workflow](#contribution-workflow)
6. [Documentation Standards](#documentation-standards)
7. [Getting Help](#getting-help)

---

## Project Overview

**Kepler Operator** automates deployment of [Kepler](https://github.com/sustainable-computing-io/kepler) (Kubernetes Efficient Power Level Exporter) on Kubernetes/OpenShift clusters for energy consumption monitoring.

**Tech Stack**: Go 1.24.0+, Operator SDK, controller-runtime, testify

**Project Structure**:

```text
kepler-operator/
├── api/v1alpha1/    # CRD definitions
├── internal/        # Private implementation
├── pkg/            # Public packages
├── config/         # Kustomize configs
└── tests/          # E2E tests
```

**Architecture**: See [docs/developer/architecture.md](docs/developer/architecture.md)

## AI Agent Permissions

### ✅ Allowed (No Approval Needed)

- Read any files, run quality checks (`make fmt`, `make vet`, `make test`)
- Create/update tests and documentation
- Refactor code following existing patterns
- Fix linter errors and test failures

### ⚠️ Requires Human Approval

- Git operations (commit, push, branch)
- Dependency changes (`go.mod`, `go.sum`)
- Infrastructure changes (CI/CD, Makefile)
- Deployments and releases
- Architectural or API changes

### 🚫 Never Allowed

- Force push to main/master
- Skip security checks
- Unsigned commits (missing DCO)
- Breaking changes without migration guide

## Code Standards

### Go Requirements

- **SPDX Headers**: All Go files must include:

  ```go
  // SPDX-FileCopyrightText: 2025 The Kepler Authors
  // SPDX-License-Identifier: Apache-2.0
  ```

- **Style**: Follow [Effective Go](https://go.dev/doc/effective_go) and idiomatic patterns
- **Testing**: Use `testify` framework (assert/require)
- **Functions**: Keep under 50-60 lines, single responsibility
- **Documentation**: GoDoc comments for all exported items

### Pre-commit Checks

- Format: `make fmt` (Go + shell scripts)
- Lint: `make vet` + golangci-lint
- Test: `make test` (maintain >80% coverage)
- Headers: SPDX compliance via reuse-lint-file

**Details**: See [docs/developer/pre-commit-hooks.md](docs/developer/pre-commit-hooks.md)

## Testing Requirements

### Standards

- **Coverage**: Maintain >80% code coverage
- **Framework**: Use `testify` (assert/require)
- **Pattern**: Table-driven tests for multiple scenarios
- **E2E Tests**: Run with `./tests/run-e2e.sh e2e`

### Commands

```bash
make test           # Run unit tests
make coverage       # Generate HTML report
go test -v ./...    # Run with verbose output
```

### Example Pattern

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {name: "valid input", input: "test", expected: "TEST"},
        {name: "empty input", input: "", expected: "", wantErr: true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := MyFunction(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

**Full Examples**: See existing tests in `internal/` and `pkg/`

## Contribution Workflow

### Commit Standards

Use [Conventional Commits](https://www.conventionalcommits.org/) with DCO sign-off:

```bash
# Required format with -s flag for DCO
git commit -s -m "type(scope): description"

# Types: feat, fix, docs, style, refactor, test, chore, ci, perf
# Examples:
git commit -s -m "feat(controller): add status conditions"
git commit -s -m "fix: resolve nil pointer in webhook"
git commit -s -m "docs: update installation guide"
```

### PR Requirements

1. **Before Opening PR**: Run `make fmt vet test`
2. **Title**: Use conventional commit format
3. **Scope**: One logical change per PR
4. **Checks**: Must pass all CI checks (see `.github/workflows/pr-checks.yaml`)

### Important Commands

```bash
# Regenerate after API changes
make manifests generate bundle docs

# Validate everything
make fmt vet test
pre-commit run --all-files

# Helm operations
make helm-validate helm-sync-crds
```

### Common Gotchas

- Don't edit auto-generated files (`bundle/`, `docs/user/reference/api.md`)
- Always sign commits with `-s` flag
- Run `make manifests generate bundle` after CRD changes
- Check [docs/developer/](docs/developer/) for detailed workflows

## Documentation Standards

### Documentation Consistency Principles

Maintaining accurate and consistent documentation across the repository is critical for both AI agents and human contributors. Documentation serves as the single source of truth for project operations.

#### Key Documentation Files

1. **AGENTS.md** - AI agent-specific guidelines and quick reference
2. **README.md** - Project overview and quick start for all users
3. **docs/user/** - End-user documentation for installation and usage
4. **docs/developer/** - Developer guides and architecture details
5. **CONTRIBUTING.md** - Contribution guidelines for all contributors
6. **API docs** - Auto-generated from CRD definitions (`docs/user/reference/api.md`)

#### Consistency Requirements

- **Cross-Reference Accuracy**: All links between documents must be valid and point to current content
- **Version Alignment**: Tech stack versions, commands, and examples must match across all docs
- **Command Synchronization**: Installation commands, make targets, and examples must be identical
- **Feature Parity**: New features documented in one place must be reflected everywhere relevant

#### Documentation Update Workflow

1. **When Changing Code**:
   - Update relevant documentation in the same PR
   - Ensure examples still work with the changes
   - Update version numbers if dependencies change

2. **Documentation Checklist**:
   - [ ] Update AGENTS.md if AI-relevant processes change
   - [ ] Update README.md for user-facing changes
   - [ ] Update user guides for feature changes
   - [ ] Update developer guides for architecture changes
   - [ ] Regenerate API docs after CRD changes: `make docs`
   - [ ] Verify all cross-document links still work
   - [ ] Ensure examples are tested and functional

3. **Common Inconsistencies to Avoid**:
   - Different Helm chart versions between README and installation guides
   - Outdated command examples that no longer work
   - Missing updates to Table of Contents when adding sections
   - Inconsistent terminology (use `PowerMonitor` for CRD kinds/Go types, `power-monitor` for file names/resource names)
   - Divergent installation steps between quick starts and detailed guides

#### Auto-Generated Documentation

Some documentation is auto-generated and should **never** be manually edited:

- `docs/user/reference/api.md` - Generated from CRD definitions via `make docs`
- `bundle/` - OLM bundle generated via `make bundle`
- Helm chart dependencies - Synced via `make helm-sync-crds`

Always regenerate these files when their sources change.

#### Bundle Generation Guidelines

The `bundle/` directory contains OLM manifests that are auto-generated from source files. Understanding this workflow prevents common mistakes:

1. **Never edit `bundle/` files directly** - They are auto-generated by operator-sdk
2. **Source files and permissions**:
   - **CRD definitions** (`api/v1alpha1/*_types.go`): AI agents can modify Go types
   - **`config/` directory**: Contains operator-sdk scaffolding (kustomize configs, webhook configs, RBAC) - **requires human approval** to modify
3. **Regeneration commands**:

   ```bash
   make manifests generate  # Generate CRDs and RBAC from Go types
   make bundle              # Regenerate OLM bundle (runs operator-sdk generate)
   ```

4. **Validation**: The `make bundle` target automatically validates the bundle using `operator-sdk bundle validate`
5. **Common mistakes to avoid**:
   - Editing `bundle/manifests/*.yaml` directly (changes will be overwritten)
   - Forgetting to run `make manifests` before `make bundle`
   - Modifying `config/` scaffolding without human approval

#### Documentation Testing

- **Link Validation**: Use markdown link checkers to ensure no broken links
- **Command Testing**: Test all command examples in documentation
- **Version Verification**: Grep for version strings to ensure consistency
- **Example Validation**: Run all code examples to ensure they work

```bash
# Check for common version inconsistencies
grep -r "v1.18.2" docs/ README.md AGENTS.md  # cert-manager version
grep -r "Go 1.24" docs/ README.md AGENTS.md   # Go version

# Test documentation commands (example)
# Extract and run bash code blocks from markdown files
```

#### AI Agent Documentation Tasks

When working with documentation:

1. **Before Making Changes**: Search all related docs to understand current state
2. **During Changes**: Update all affected documentation files together
3. **After Changes**: Verify consistency across all documentation
4. **Flag Inconsistencies**: If you notice existing inconsistencies, fix them or flag for human review

Remember: Documentation debt compounds quickly. Keeping docs synchronized prevents confusion and reduces support burden.

## Getting Help

### Resources

- **Documentation**: [User Guide](docs/user/), [Developer Guide](docs/developer/), [Contributing](CONTRIBUTING.md)
- **Community**: CNCF Slack `#kepler`, [GitHub Discussions](https://github.com/sustainable-computing-io/kepler-operator/discussions)
- **Issues/PRs**: Search [existing issues](https://github.com/sustainable-computing-io/kepler-operator/issues) and [PRs](https://github.com/sustainable-computing-io/kepler-operator/pulls)

### Quick FAQ

- **Local development**: `make cluster-up` then `make run`
- **Regenerate manifests**: `make manifests generate bundle`
- **Pre-commit failures**: Run `pre-commit run --all-files`
- **API docs**: Run `make docs` after CRD changes
- **See all commands**: `make help`

### AI-Human Collaboration

1. **AI Capabilities**: Generate code, tests, docs, refactor, analyze issues
2. **Human Review Required**: All AI output needs human verification
3. **Effective Prompts**: Be specific, provide context, reference files
4. **Conflict Resolution**: Human has final say; consult maintainers for architecture

### Standards Enforcement

- **Pre-commit hooks**: Automated local checks (see [pre-commit guide](docs/developer/pre-commit-hooks.md))
- **CI Pipeline**: Comprehensive PR validation
- **Code Review**: Human verification of quality and design
- **Metrics**: >80% coverage, clean lints, passing tests

---

**Thank you for contributing to Kepler Operator!** 🌱

For detailed information on any topic, consult the full documentation in the `docs/` directory or contact maintainers.
