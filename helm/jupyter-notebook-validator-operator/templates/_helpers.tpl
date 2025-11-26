{{/*
Expand the name of the chart.
*/}}
{{- define "jupyter-notebook-validator-operator.name" -}}
{{- default "notebook-validator" .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "jupyter-notebook-validator-operator.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default "notebook-validator" .Values.nameOverride }}
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
{{- define "jupyter-notebook-validator-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "jupyter-notebook-validator-operator.labels" -}}
helm.sh/chart: {{ include "jupyter-notebook-validator-operator.chart" . }}
{{ include "jupyter-notebook-validator-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "jupyter-notebook-validator-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "jupyter-notebook-validator-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
control-plane: controller-manager
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "jupyter-notebook-validator-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "jupyter-notebook-validator-operator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the validation runner service account
*/}}
{{- define "jupyter-notebook-validator-operator.validationRunnerServiceAccountName" -}}
{{- if .Values.validationRunner.serviceAccount.create }}
{{- default "validation-runner" .Values.validationRunner.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.validationRunner.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Return the proper image name
*/}}
{{- define "jupyter-notebook-validator-operator.image" -}}
{{- $registryName := .Values.image.repository }}
{{- $tag := .Values.image.tag | default .Chart.AppVersion }}
{{- printf "%s:%s" $registryName $tag }}
{{- end }}

{{/*
Return the proper kube-rbac-proxy image name
*/}}
{{- define "jupyter-notebook-validator-operator.rbacProxyImage" -}}
{{- $registryName := .Values.operator.metrics.authProxy.image.repository }}
{{- $tag := .Values.operator.metrics.authProxy.image.tag }}
{{- printf "%s:%s" $registryName $tag }}
{{- end }}

{{/*
Return the namespace
*/}}
{{- define "jupyter-notebook-validator-operator.namespace" -}}
{{- default .Release.Namespace .Values.namespace }}
{{- end }}

{{/*
Return true if CRDs should be installed
*/}}
{{- define "jupyter-notebook-validator-operator.installCRDs" -}}
{{- if .Values.crds.install }}
{{- true }}
{{- end }}
{{- end }}

{{/*
Return true if running on OpenShift
*/}}
{{- define "jupyter-notebook-validator-operator.isOpenShift" -}}
{{- if .Values.openshift.enabled }}
{{- true }}
{{- end }}
{{- end }}

{{/*
Return the metrics service name
*/}}
{{- define "jupyter-notebook-validator-operator.metricsServiceName" -}}
{{- printf "%s-metrics" (include "jupyter-notebook-validator-operator.fullname" .) }}
{{- end }}

{{/*
Return the webhook service name
*/}}
{{- define "jupyter-notebook-validator-operator.webhookServiceName" -}}
{{- printf "%s-webhook" (include "jupyter-notebook-validator-operator.fullname" .) }}
{{- end }}

{{/*
Return the leader election role name
*/}}
{{- define "jupyter-notebook-validator-operator.leaderElectionRoleName" -}}
{{- printf "%s-leader-election" (include "jupyter-notebook-validator-operator.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Return the manager role name
*/}}
{{- define "jupyter-notebook-validator-operator.managerRoleName" -}}
{{- printf "%s-manager" (include "jupyter-notebook-validator-operator.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Return the validation runner role name
*/}}
{{- define "jupyter-notebook-validator-operator.validationRunnerRoleName" -}}
{{- printf "%s-validation-runner" (include "jupyter-notebook-validator-operator.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Return the auth proxy role name
*/}}
{{- define "jupyter-notebook-validator-operator.authProxyRoleName" -}}
{{- printf "%s-auth-proxy" (include "jupyter-notebook-validator-operator.fullname" .) }}
{{- end }}

{{/*
Return the metrics reader role name
*/}}
{{- define "jupyter-notebook-validator-operator.metricsReaderRoleName" -}}
{{- printf "%s-metrics-reader" (include "jupyter-notebook-validator-operator.fullname" .) }}
{{- end }}

{{/*
Return the ServiceMonitor name
*/}}
{{- define "jupyter-notebook-validator-operator.serviceMonitorName" -}}
{{- printf "%s-metrics" (include "jupyter-notebook-validator-operator.fullname" .) }}
{{- end }}

{{/*
Return the ConfigMap name
*/}}
{{- define "jupyter-notebook-validator-operator.configMapName" -}}
{{- printf "%s-config" (include "jupyter-notebook-validator-operator.fullname" .) }}
{{- end }}

{{/*
Create the args for the operator container
*/}}
{{- define "jupyter-notebook-validator-operator.operatorArgs" -}}
{{- if .Values.operator.leaderElection.enabled }}
- --leader-elect
{{- end }}
{{- if .Values.operator.metrics.enabled }}
- --metrics-bind-address={{ .Values.operator.metrics.bindAddress }}
{{- end }}
- --health-probe-bind-address={{ .Values.operator.health.bindAddress }}
{{- end }}

{{/*
Create environment variables for the operator
*/}}
{{- define "jupyter-notebook-validator-operator.env" -}}
{{- if .Values.env }}
{{- toYaml .Values.env }}
{{- end }}
{{- end }}

{{/*
Return true if Tekton is enabled
*/}}
{{- define "jupyter-notebook-validator-operator.tektonEnabled" -}}
{{- if .Values.tekton.enabled }}
{{- true }}
{{- end }}
{{- end }}

{{/*
Return true if Prometheus ServiceMonitor should be created
*/}}
{{- define "jupyter-notebook-validator-operator.createServiceMonitor" -}}
{{- if and .Values.prometheus.enabled .Values.operator.metrics.enabled }}
{{- true }}
{{- end }}
{{- end }}

