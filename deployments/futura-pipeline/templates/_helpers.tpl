{{/*
Expand the name of the chart.
*/}}
{{- define "futura-pipeline.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "futura-pipeline.fullname" -}}
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
{{- define "futura-pipeline.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "futura-pipeline.labels" -}}
helm.sh/chart: {{ include "futura-pipeline.chart" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: futura
{{- end }}

{{/*
Selector labels for collect service
*/}}
{{- define "futura-pipeline.collectSelectorLabels" -}}
app.kubernetes.io/name: {{ include "futura-pipeline.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: collect
{{- end }}

{{/*
Selector labels for validate service
*/}}
{{- define "futura-pipeline.validateSelectorLabels" -}}
app.kubernetes.io/name: {{ include "futura-pipeline.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: validate
{{- end }}

{{/*
Selector labels for enrich service
*/}}
{{- define "futura-pipeline.enrichSelectorLabels" -}}
app.kubernetes.io/name: {{ include "futura-pipeline.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: enrich
{{- end }}

{{/*
Selector labels for store service
*/}}
{{- define "futura-pipeline.storeSelectorLabels" -}}
app.kubernetes.io/name: {{ include "futura-pipeline.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: store
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "futura-pipeline.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "futura-pipeline.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
