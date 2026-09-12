{{/*
Expand the name of the chart.
*/}}
{{- define "garm-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "garm-operator.fullname" -}}
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
{{- define "garm-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "garm-operator.labels" -}}
helm.sh/chart: {{ include "garm-operator.chart" . }}
{{ include "garm-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: garm-operator
{{- end }}

{{/*
Selector labels
*/}}
{{- define "garm-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "garm-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
control-plane: controller-manager
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "garm-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (printf "%s-controller-manager" (include "garm-operator.fullname" .)) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the GARM secret to use
*/}}
{{- define "garm-operator.secretName" -}}
{{- if .Values.garm.existingSecret }}
{{- .Values.garm.existingSecret }}
{{- else }}
{{- printf "%s-garm-secret" (include "garm-operator.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Create the name of the webhook service
*/}}
{{- define "garm-operator.webhookServiceName" -}}
{{- printf "%s-webhook-service" (include "garm-operator.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Aliases for compatibility
*/}}
{{- define "chart.name" -}}
{{- include "garm-operator.name" . }}
{{- end }}

{{- define "chart.fullname" -}}
{{- include "garm-operator.fullname" . }}
{{- end }}

{{- define "chart.labels" -}}
{{- include "garm-operator.labels" . }}
{{- end }}

{{- define "chart.selectorLabels" -}}
{{- include "garm-operator.selectorLabels" . }}
{{- end }}

{{- define "chart.serviceAccountName" -}}
{{- include "garm-operator.serviceAccountName" . }}
{{- end }}
