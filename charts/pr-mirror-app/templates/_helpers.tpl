{{- /* SPDX-License-Identifier: Apache-2.0 */ -}}
{{/*
Expand the name of the chart.
*/}}
{{- define "pr-mirror-app.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "pr-mirror-app.fullname" -}}
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
{{- define "pr-mirror-app.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "pr-mirror-app.labels" -}}
helm.sh/chart: {{ include "pr-mirror-app.chart" . }}
{{ include "pr-mirror-app.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/component: server
app.kubernetes.io/part-of: {{ include "pr-mirror-app.name" . }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "pr-mirror-app.selectorLabels" -}}
app.kubernetes.io/name: {{ include "pr-mirror-app.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "pr-mirror-app.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "pr-mirror-app.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Resolve the container image reference. Falls back to .Chart.AppVersion when image.tag is empty.
*/}}
{{- define "pr-mirror-app.image" -}}
{{- $tag := default .Chart.AppVersion .Values.image.tag -}}
{{- printf "%s:%s" .Values.image.repository $tag -}}
{{- end }}

{{/*
Validate a seccompProfile block. Fails the install with a readable error when the profile is misconfigured.
Usage: {{- include "pr-mirror-app.validateSeccompProfile" (dict "profile" .Values.podSecurityContext.seccompProfile "scope" "podSecurityContext") -}}
*/}}
{{- define "pr-mirror-app.validateSeccompProfile" -}}
{{- $profile := .profile -}}
{{- $scope := .scope -}}
{{- if $profile -}}
{{- $type := $profile.type | default "" -}}
{{- if not (has $type (list "RuntimeDefault" "Localhost" "Unconfined")) -}}
{{- fail (printf "%s.seccompProfile.type must be one of RuntimeDefault, Localhost, Unconfined (got %q)" $scope $type) -}}
{{- end -}}
{{- if and (eq $type "Localhost") (not $profile.localhostProfile) -}}
{{- fail (printf "%s.seccompProfile.type=Localhost requires .localhostProfile to be set" $scope) -}}
{{- end -}}
{{- if and (ne $type "Localhost") $profile.localhostProfile -}}
{{- fail (printf "%s.seccompProfile.localhostProfile is only valid when type=Localhost (got type=%q)" $scope $type) -}}
{{- end -}}
{{- end -}}
{{- end }}

{{/*
Render the chart-managed Secret name (used for envFrom wiring when secret.create=true).
*/}}
{{- define "pr-mirror-app.secretName" -}}
{{- printf "%s-secret" (include "pr-mirror-app.fullname" .) -}}
{{- end }}

{{/*
Render the chart-managed ConfigMap name.
*/}}
{{- define "pr-mirror-app.configmapName" -}}
{{- printf "%s-config" (include "pr-mirror-app.fullname" .) -}}
{{- end }}
