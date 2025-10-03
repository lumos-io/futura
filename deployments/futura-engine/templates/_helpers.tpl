{{/*
Expand the name of the chart.
*/}}
{{- define "futura-engine.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "futura-engine.fullname" -}}
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
{{- define "futura-engine.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "futura-engine.labels" -}}
helm.sh/chart: {{ include "futura-engine.chart" . }}
{{ include "futura-engine.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: futura
{{- end }}

{{/*
Selector labels for all services mode
*/}}
{{- define "futura-engine.selectorLabels" -}}
app.kubernetes.io/name: {{ include "futura-engine.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: all-services
{{- end }}

{{/*
Selector labels for recommendation service
*/}}
{{- define "futura-engine.recommendationSelectorLabels" -}}
app.kubernetes.io/name: {{ include "futura-engine.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: recommendation
{{- end }}

{{/*
Selector labels for RL server
*/}}
{{- define "futura-engine.rlSelectorLabels" -}}
app.kubernetes.io/name: {{ include "futura-engine.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: rl-server
{{- end }}

{{/*
Selector labels for agent coordinator
*/}}
{{- define "futura-engine.agentCoordinatorSelectorLabels" -}}
app.kubernetes.io/name: {{ include "futura-engine.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: agent-coordinator
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "futura-engine.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "futura-engine.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
