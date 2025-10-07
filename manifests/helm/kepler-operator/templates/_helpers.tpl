{{/*
Expand the name of the chart.
*/}}
{{- define "kepler-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kepler-operator.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "kepler-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "kepler-operator.labels" -}}
helm.sh/chart: {{ include "kepler-operator.chart" . }}
{{ include "kepler-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: kepler-operator
{{- end }}

{{/*
Selector labels
*/}}
{{- define "kepler-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kepler-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Manager labels (for deployment and pod)
*/}}
{{- define "kepler-operator.managerLabels" -}}
{{ include "kepler-operator.selectorLabels" . }}
app.kubernetes.io/component: manager
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "kepler-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default "kepler-operator-controller-manager" .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the namespace to use
*/}}
{{- define "kepler-operator.namespace" -}}
{{- default "kepler-operator-system" .Values.namespace }}
{{- end }}

{{/*
Operator image
*/}}
{{- define "kepler-operator.image" -}}
{{- .Values.operator.image }}
{{- end }}

{{/*
Kepler image (managed by operator)
*/}}
{{- define "kepler-operator.keplerImage" -}}
{{- .Values.kepler.image }}
{{- end }}

{{/*
Kube RBAC Proxy image (managed by operator)
*/}}
{{- define "kepler-operator.kubeRbacProxyImage" -}}
{{- index .Values "kube-rbac-proxy" "image" }}
{{- end }}

{{/*
Deployment namespace for power monitoring components
Defaults to "power-monitor" (the operator's code default) if not specified
*/}}
{{- define "kepler-operator.deploymentNamespace" -}}
{{- default "power-monitor" .Values.operator.deploymentNamespace }}
{{- end }}
