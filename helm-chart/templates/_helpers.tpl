{{/*
Expand the name of the chart.
*/}}
{{- define "binderhub-container-registry-helper.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "binderhub-container-registry-helper.fullname" -}}
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
{{- define "binderhub-container-registry-helper.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "binderhub-container-registry-helper.labels" -}}
helm.sh/chart: {{ include "binderhub-container-registry-helper.chart" . }}
{{ include "binderhub-container-registry-helper.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "binderhub-container-registry-helper.selectorLabels" -}}
app.kubernetes.io/name: {{ include "binderhub-container-registry-helper.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "binderhub-container-registry-helper.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "binderhub-container-registry-helper.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Generate a random auth_token if one doesn't already exist
https://github.com/jupyterhub/zero-to-jupyterhub-k8s/blob/78bec62d591b7a97560c37edd79c7a15186fcfe9/jupyterhub/templates/hub/_helpers-passwords.tpl#L34-L47
*/}}
{{- define "binderhub-container-registry-helper.auth_token" -}}
  {{- if .Values.auth_token }}
    {{- .Values.auth_token }}
  {{- else }}
    {{- $k8s_state := lookup "v1" "Secret" .Release.Namespace (include "binderhub-container-registry-helper.fullname" .) | default (dict "data" (dict)) }}
    {{- if hasKey $k8s_state.data "auth_token" }}
        {{- index $k8s_state.data "auth_token" | b64dec }}
    {{- else }}
        {{- randAlphaNum 64 }}
    {{- end }}
  {{- end }}
{{- end }}
