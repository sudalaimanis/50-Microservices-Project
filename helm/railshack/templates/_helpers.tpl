{{/* RailShack · 50 Microservices Project by DevOps Shack · Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack · SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work. */}}
{{/* Common labels for one microservice. Call with (dict "name" $name "lang" $lang "root" $) */}}
{{- define "railshack.labels" -}}
app.kubernetes.io/name: {{ .name }}
app.kubernetes.io/instance: {{ .root.Release.Name }}
app.kubernetes.io/part-of: railshack
app.kubernetes.io/version: {{ .root.Values.image.tag | quote }}
app.kubernetes.io/managed-by: {{ .root.Release.Service }}
helm.sh/chart: {{ .root.Chart.Name }}-{{ .root.Chart.Version }}
railshack.devopsshack.com/language: {{ .lang | default "unknown" }}
{{- end }}

{{- define "railshack.selector" -}}
app.kubernetes.io/name: {{ .name }}
app.kubernetes.io/instance: {{ .root.Release.Name }}
{{- end }}

{{/* Merged config for one service: defaults <- service overrides */}}
{{- define "railshack.cfg" -}}
{{- $cfg := mergeOverwrite (deepCopy .root.Values.defaults) (deepCopy .svc) -}}
{{- toYaml $cfg -}}
{{- end }}
