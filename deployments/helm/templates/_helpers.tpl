{{/*
Expand the name of the chart.
*/}}
{{- define "swit.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "swit.fullname" -}}
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
{{- define "swit.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "swit.labels" -}}
helm.sh/chart: {{ include "swit.chart" . }}
{{ include "swit.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: swit-platform
{{- end }}

{{/*
Selector labels
*/}}
{{- define "swit.selectorLabels" -}}
app.kubernetes.io/name: {{ include "swit.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "swit.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "swit.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
MySQL 服务名称
*/}}
{{- define "swit.mysql.fullname" -}}
{{- printf "%s-mysql" (include "swit.fullname" .) }}
{{- end }}

{{/*
MySQL 服务标签
*/}}
{{- define "swit.mysql.labels" -}}
{{ include "swit.labels" . }}
app.kubernetes.io/component: database
{{- end }}

{{/*
MySQL 选择器标签
*/}}
{{- define "swit.mysql.selectorLabels" -}}
{{ include "swit.selectorLabels" . }}
app.kubernetes.io/component: database
{{- end }}

{{/*
Consul 服务名称
*/}}
{{- define "swit.consul.fullname" -}}
{{- printf "%s-consul" (include "swit.fullname" .) }}
{{- end }}

{{/*
Consul 服务标签
*/}}
{{- define "swit.consul.labels" -}}
{{ include "swit.labels" . }}
app.kubernetes.io/component: service-discovery
{{- end }}

{{/*
Consul 选择器标签
*/}}
{{- define "swit.consul.selectorLabels" -}}
{{ include "swit.selectorLabels" . }}
app.kubernetes.io/component: service-discovery
{{- end }}

{{/*
Swit Auth 服务名称
*/}}
{{- define "swit.auth.fullname" -}}
{{- printf "%s-auth" (include "swit.fullname" .) }}
{{- end }}

{{/*
Swit Auth 服务标签
*/}}
{{- define "swit.auth.labels" -}}
{{ include "swit.labels" . }}
app.kubernetes.io/component: authentication
{{- end }}

{{/*
Swit Auth 选择器标签
*/}}
{{- define "swit.auth.selectorLabels" -}}
{{ include "swit.selectorLabels" . }}
app.kubernetes.io/component: authentication
{{- end }}

{{/*
Swit Serve 服务名称
*/}}
{{- define "swit.serve.fullname" -}}
{{- printf "%s-serve" (include "swit.fullname" .) }}
{{- end }}

{{/*
Swit Serve 服务标签
*/}}
{{- define "swit.serve.labels" -}}
{{ include "swit.labels" . }}
app.kubernetes.io/component: api-server
{{- end }}

{{/*
Swit Serve 选择器标签
*/}}
{{- define "swit.serve.selectorLabels" -}}
{{ include "swit.selectorLabels" . }}
app.kubernetes.io/component: api-server
{{- end }}

{{/*
获取命名空间
*/}}
{{- define "swit.namespace" -}}
{{- default .Release.Namespace .Values.namespaceOverride }}
{{- end }}

{{/*
获取镜像
*/}}
{{- define "swit.image" -}}
{{- $registryName := .imageRoot.registry -}}
{{- $repositoryName := .imageRoot.repository -}}
{{- $tag := .imageRoot.tag | toString -}}
{{- if .global }}
    {{- if .global.imageRegistry }}
        {{- $registryName = .global.imageRegistry -}}
    {{- end -}}
{{- end -}}
{{- if $registryName }}
{{- printf "%s/%s:%s" $registryName $repositoryName $tag -}}
{{- else -}}
{{- printf "%s:%s" $repositoryName $tag -}}
{{- end -}}
{{- end }}

{{/*
生成镜像拉取密钥
*/}}
{{- define "swit.imagePullSecrets" -}}
{{- $pullSecrets := list }}
{{- if .Values.global }}
  {{- range .Values.global.imagePullSecrets -}}
    {{- $pullSecrets = append $pullSecrets . -}}
  {{- end -}}
{{- end -}}
{{- range .Values.imagePullSecrets -}}
  {{- $pullSecrets = append $pullSecrets . -}}
{{- end -}}
{{- if (not (empty $pullSecrets)) }}
imagePullSecrets:
{{- range $pullSecrets }}
  - name: {{ . }}
{{- end }}
{{- end }}
{{- end }}

{{/*
生成存储类名称
*/}}
{{- define "swit.storageClass" -}}
{{- if .persistence.storageClass }}
  {{- if (eq "-" .persistence.storageClass) }}
storageClassName: ""
  {{- else }}
storageClassName: {{ .persistence.storageClass | quote }}
  {{- end }}
{{- else if .global.storageClass }}
  {{- if (eq "-" .global.storageClass) }}
storageClassName: ""
  {{- else }}
storageClassName: {{ .global.storageClass | quote }}
  {{- end }}
{{- end }}
{{- end }}
