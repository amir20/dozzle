{{- define "dozzle.fullname" -}}
{{- if contains .Chart.Name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "dozzle.selectorLabels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "dozzle.labels" -}}
{{ include "dozzle.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
{{- end -}}

{{- define "dozzle.serviceAccountName" -}}
{{- default (include "dozzle.fullname" .) .Values.serviceAccount.name -}}
{{- end -}}

{{- /* ClusterRole and ClusterRoleBinding are cluster-scoped, so the same release name in two namespaces would collide. */ -}}
{{- define "dozzle.clusterName" -}}
{{- printf "%s-%s" (include "dozzle.fullname" .) .Release.Namespace | trunc 63 | trimSuffix "-" -}}
{{- end -}}
