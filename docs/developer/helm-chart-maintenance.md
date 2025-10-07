# ⎈ Helm Chart Maintenance Guide

This guide explains how to maintain and update the Helm chart for the Kepler Operator.

---

## 📋 Overview

The Helm chart uses a **hybrid automation approach**:

- **Manual**: Templates are hand-crafted for full control and customization
- **Automated**: CRDs are automatically synced from `config/crd/bases/`
- **Validated**: Automated checks ensure consistency with kustomize deployment

This approach balances maintainability with flexibility.

---

## 🗂️ Chart Structure

```text
manifests/helm/kepler-operator/
├── Chart.yaml              # Chart metadata (version, appVersion)
├── values.yaml             # Default configuration values
├── README.md              # User-facing installation guide
├── .helmignore            # Files excluded from packaging
├── crds/                  # CRDs (auto-synced from config/crd/bases/)
│   ├── kepler.system...powermonitors.yaml
│   └── kepler.system...powermonitorinternals.yaml
└── templates/
    ├── _helpers.tpl       # Template helper functions
    ├── NOTES.txt          # Post-install instructions
    ├── serviceaccount.yaml
    ├── rbac.yaml          # All RBAC resources
    ├── deployment.yaml
    ├── services.yaml      # Metrics + webhook services
    ├── certificate.yaml   # cert-manager resources (conditional)
    ├── webhooks.yaml      # Webhook configurations (conditional)
    └── servicemonitor.yaml # Prometheus ServiceMonitor (conditional)
```

---

## 🔄 When to Update the Helm Chart

| Change Type | Action Required | Files to Update |
|-------------|----------------|-----------------|
| **CRD Modified** | Run `make helm-sync-crds` | Auto-synced to `crds/` |
| **RBAC Changed** | Manual template update | `templates/rbac.yaml` |
| **Deployment Changed** | Manual template update | `templates/deployment.yaml` |
| **New Resource Added** | Create new template | `templates/<resource>.yaml` |
| **Config Option Added** | Update values & templates | `values.yaml` + relevant template |
| **Version Bump** | Update chart metadata | `Chart.yaml` (version, appVersion) |

---

## 🛠️ Update Workflow

### 1. Make Changes

```bash
# If CRDs changed, sync them
make helm-sync-crds

# If templates changed, edit manually
vim manifests/helm/kepler-operator/templates/<file>.yaml

# If configuration changed, update values
vim manifests/helm/kepler-operator/values.yaml
```

### 2. Validate Changes

```bash
# Run all validation tests (recommended)
make helm-validate  # Full validation (syntax, templates, CRD sync, resources)

# Or preview rendered manifests:
make helm-template  # Preview rendered manifests
```

### 3. Test Locally (Optional)

```bash
# Full end-to-end test (recommended)
./tests/helm.sh

# Or manual testing:
make helm-install         # Install to cluster
kubectl get all -n kepler-operator  # Verify deployment
make helm-uninstall       # Clean up

# Advanced: test with existing image
./tests/helm.sh --no-build --version=0.21.0
```

---

## ✍️ Creating/Updating Templates

### Use Kustomize as Reference

**Important**: Always use `config/default/k8s` as your source of truth, NOT `config/manifests`.

```bash
# Generate reference manifest
make manifests
kustomize build config/default/k8s > /tmp/kustomize-ref.yaml

# Extract specific resources
./tmp/bin/yq 'select(.kind == "Deployment")' /tmp/kustomize-ref.yaml
./tmp/bin/yq 'select(.kind == "Service")' /tmp/kustomize-ref.yaml
```

**Why `config/default/k8s`?**

- `config/default/k8s`: Standard Kubernetes deployment (matches Helm use case)
- `config/manifests`: OLM-specific with ClusterServiceVersion (different model)

### Template Creation Steps

1. Extract resource from kustomize output
2. Replace hardcoded values with template helpers:
   - Names: `{{ include "kepler-operator.fullname" . }}-<suffix>`
   - Namespace: `{{ include "kepler-operator.namespace" . }}`
   - Labels: `{{ include "kepler-operator.labels" . | nindent 4 }}`
   - Images: `{{ include "kepler-operator.image" . }}`
3. Add conditional rendering if needed:

   ```yaml
   {{- if .Values.feature.enabled }}
   # resource definition
   {{- end }}
   ```

4. Use values from `values.yaml`:

   ```yaml
   replicas: {{ .Values.replicaCount }}
   resources:
     {{- toYaml .Values.resources | nindent 12 }}
   ```

### Helper Function Reference

Common helpers available in `templates/_helpers.tpl`:

```yaml
# Chart name
{{ include "kepler-operator.name" . }}

# Full name (release-name + chart-name)
{{ include "kepler-operator.fullname" . }}

# Namespace
{{ include "kepler-operator.namespace" . }}

# Standard labels
{{ include "kepler-operator.labels" . | nindent 4 }}

# Selector labels (stable, for pod selectors)
{{ include "kepler-operator.managerLabels" . | nindent 6 }}

# Image references
{{ include "kepler-operator.image" . }}                    # Operator image
{{ include "kepler-operator.keplerImage" . }}              # Kepler image
{{ include "kepler-operator.kubeRbacProxyImage" . }}       # Kube RBAC Proxy image

# Service account name
{{ include "kepler-operator.serviceAccountName" . }}
```

---

## 🧪 Validation Details

The `make helm-validate` command runs three layers of checks:

### Layer 1: Syntax Validation

```bash
helm lint manifests/helm/kepler-operator
```

- Validates Chart.yaml structure
- Checks template syntax
- Verifies values.yaml schema

### Layer 2: Template Rendering

```bash
helm template kepler-operator manifests/helm/kepler-operator \
  --set metrics.serviceMonitor.enabled=true
```

- Ensures templates render without errors
- Tests value substitution
- Validates conditional logic

### Layer 3: Consistency Checks

```bash
./hack/helm/validate.sh
```

- Verifies CRD sync status (CRDs match `config/crd/bases/`)
- Validates all expected resources present
- Checks project-local tools available

---

## 💡 Common Patterns

### Conditional Resources

Use feature flags in `values.yaml`:

```yaml
# values.yaml
webhooks:
  enabled: true
  certManager:
    enabled: true
```

Then wrap entire templates:

```yaml
# templates/certificate.yaml
{{- if .Values.webhooks.certManager.enabled }}
# Certificate and Issuer resources
{{- end }}
```

### Multi-Resource Templates

Group related resources in single file with `---` separator:

```yaml
# templates/rbac.yaml
# Role
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
...
---
# RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
...
```

### Image Configuration

Use full image paths for simplicity:

```yaml
# values.yaml
operator:
  image: quay.io/sustainable_computing_io/kepler-operator:0.21.0
  pullPolicy: IfNotPresent

kepler:
  image: quay.io/sustainable_computing_io/kepler:v0.11.0

kube-rbac-proxy:
  image: quay.io/brancz/kube-rbac-proxy:v0.19.0

# _helpers.tpl
{{- define "kepler-operator.image" -}}
{{- .Values.operator.image }}
{{- end }}

{{- define "kepler-operator.keplerImage" -}}
{{- .Values.kepler.image }}
{{- end }}

{{- define "kepler-operator.kubeRbacProxyImage" -}}
{{- index .Values "kube-rbac-proxy" "image" }}
{{- end }}
```

This approach is simpler and allows overriding with:

```bash
helm install kepler-operator ./chart \
  --set operator.image=localhost:5001/kepler-operator:dev
```

---

## ⚠️ Common Pitfalls

### ❌ Wrong Kustomize Overlay

```bash
kustomize build config/manifests  # OLM-specific, wrong!
```

✅ Use:

```bash
kustomize build config/default/k8s  # Vanilla K8s, correct!
```

### ❌ Hardcoded Names

```yaml
name: kepler-operator-controller
namespace: kepler-operator
```

✅ Use helpers:

```yaml
name: {{ include "kepler-operator.fullname" . }}-controller
namespace: {{ include "kepler-operator.namespace" . }}
```

### ❌ Validation Without Optional Resources

```bash
helm template kepler-operator manifests/helm/kepler-operator
# ServiceMonitor missing!
```

✅ Enable all optionals:

```bash
helm template kepler-operator manifests/helm/kepler-operator \
  --set metrics.serviceMonitor.enabled=true
```

### ❌ Mutable Selector Labels

```yaml
selector:
  matchLabels:
    {{- include "kepler-operator.labels" . | nindent 4 }}
    # Includes version, breaks on upgrade!
```

✅ Use stable selectors:

```yaml
selector:
  matchLabels:
    {{- include "kepler-operator.managerLabels" . | nindent 4 }}
```

### ❌ Namespace Template + --create-namespace Flag

```yaml
# templates/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: {{ include "kepler-operator.namespace" . }}
```

AND using `--create-namespace` flag causes conflict:

```text
Error: namespaces "kepler-operator" already exists
```

✅ Use **only** `--create-namespace` flag (standard Helm practice):

```bash
helm install kepler-operator ./chart \
  --namespace kepler-operator \
  --create-namespace  # Let Helm create namespace
```

**Rationale**: The `--create-namespace` flag is simpler and follows standard Helm conventions. Template-based namespace creation adds unnecessary complexity and potential conflicts.

---

## 📦 Release Process

When releasing a new version:

1. **Update Chart.yaml**:

   ```yaml
   version: 0.22.0        # Bump chart version
   appVersion: 0.22.0     # Match operator version
   ```

2. **Sync CRDs** (if changed):

   ```bash
   make helm-sync-crds
   ```

3. **Validate**:

   ```bash
   make helm-validate  # Runs syntax, template, CRD sync, and resource validation
   ```

4. **Package** (optional):

   ```bash
   make helm-package
   ```

5. **Commit changes**:

   ```bash
   git add manifests/helm/kepler-operator/
   git commit -m "chore(helm): bump chart version to 0.22.0"
   ```

---

## 📚 Additional Resources

- **Helm Best Practices**: <https://helm.sh/docs/chart_best_practices/>
- **Knowledge Base**: `tmp/agents/knowledge/helm-deployment.md`
- **Chart README**: `manifests/helm/kepler-operator/README.md` (user guide)
- **Kustomize Docs**: <https://kubectl.docs.kubernetes.io/references/kustomize/>

---

## 🤝 Getting Help

- Review existing templates for patterns
- Check validation errors: `make helm-validate` provides specific guidance
- See knowledge base for detailed explanations: `tmp/agents/knowledge/helm-deployment.md`
- Ask in project discussions or issues

Happy charting! ⛵
