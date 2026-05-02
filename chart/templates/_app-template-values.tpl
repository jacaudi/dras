{{/*
Build the values bag passed to bjw-s/app-template.

Responsibilities of this helper:
  1. Strip the renderer controller/service in standard mode.
  2. Inject container image refs from the top-level image: block, defaulting empty tag to .Chart.AppVersion.
  3. Inject Pushover envFrom on the dras container (always required).
  4. Append STATION_IDS / INTERVAL envs to the dras container.
  5. (Advanced mode only) Append RENDERER_URL on dras + S3_BUCKET/AWS_REGION on renderer. — added in Task 2
  6. (Schema enforcement) Fail clauses for required values. — added in Task 4

Implementation note: the .Values | toYaml | fromYaml round-trip converts
common.Values typed nested maps into plain map[string]interface{} so that
sprig set/unset/dig operate reliably on all nested keys.
*/}}
{{- define "dras.appTemplateValues" -}}
{{- /* Round-trip through YAML to convert common.Values to plain maps */ -}}
{{- $v := .Values | toYaml | fromYaml -}}

{{- /* Inject container image refs from the top-level .Values.image.{dras,renderer}.{repository,tag,pullPolicy}
     surface, defaulting empty tag to .Chart.AppVersion. This is the operator-facing image-overrides path:
     `helm install --set image.dras.tag=v2.7.0` pins the dras image. */ -}}
{{- $drasImg := .Values.image.dras -}}
{{- $drasTag := default .Chart.AppVersion $drasImg.tag -}}
{{- $_ := set $v.controllers.dras.containers.app "image" (dict
      "repository" $drasImg.repository
      "tag" $drasTag
      "pullPolicy" $drasImg.pullPolicy) -}}

{{- if hasKey $v.controllers "renderer" -}}
  {{- $rendImg := .Values.image.renderer -}}
  {{- $rendTag := default .Chart.AppVersion $rendImg.tag -}}
  {{- $_ := set $v.controllers.renderer.containers.app "image" (dict
        "repository" $rendImg.repository
        "tag" $rendTag
        "pullPolicy" $rendImg.pullPolicy) -}}
{{- end -}}

{{- /* Strip renderer pieces in standard mode */ -}}
{{- if eq .Values.mode "standard" -}}
  {{- $_ := unset $v.controllers "renderer" -}}
  {{- $_ := unset $v.service "renderer" -}}
{{- end -}}

{{- /* Append Pushover envFrom on dras */ -}}
{{- $appContainer := $v.controllers.dras.containers.app -}}
{{- $envFrom := default (list) $appContainer.envFrom -}}
{{- $envFrom = append $envFrom (dict "secretRef" (dict "name" .Values.dras.pushover.existingSecret)) -}}
{{- $_ := set $appContainer "envFrom" $envFrom -}}

{{- /*
     RESERVED CONTAINER ENV NAMES — managed by this chart, do not set in values.

     dras container:
       STATION_IDS, INTERVAL — appended below.
       PUSHOVER_API_TOKEN, PUSHOVER_USER_KEY — sourced via envFrom secretRef above.
       RENDERER_URL — appended in advanced mode (Task 2).

     renderer container (advanced mode):
       S3_BUCKET, AWS_REGION — appended in Task 2.

     Task 4 will add fail clauses that reject operator-supplied values for
     these keys. For now, last write wins (chart-managed entry is appended
     after operator entries, so it shadows them in Kubernetes' env merge).
*/ -}}
{{- /* Append STATION_IDS and INTERVAL envs on dras */ -}}
{{- $existing := $appContainer.env -}}
{{- $envs := list -}}
{{- /* env in values.yaml is a map ({}); convert to list if needed */ -}}
{{- if kindIs "map" $existing -}}
  {{- range $k, $val := $existing -}}
    {{- $envs = append $envs (dict "name" $k "value" $val) -}}
  {{- end -}}
{{- else if $existing -}}
  {{- $envs = $existing -}}
{{- end -}}
{{- $envs = append $envs (dict "name" "STATION_IDS" "value" .Values.dras.stationIds) -}}
{{- $envs = append $envs (dict "name" "INTERVAL" "value" (toString .Values.dras.interval)) -}}
{{- $_ := set $appContainer "env" $envs -}}

{{- /* Advanced-mode-only wiring. Standard mode falls through to plain dras. */ -}}
{{- if eq .Values.mode "advanced" -}}

  {{- /* dras: RENDERER_URL pointing at the renderer Service */ -}}
  {{- $rendURL := printf "http://%s-renderer.%s.svc.cluster.local:%v"
        .Release.Name .Release.Namespace .Values.renderer.service.port -}}
  {{- $drasEnv := $appContainer.env -}}
  {{- $drasEnv = append $drasEnv (dict "name" "RENDERER_URL" "value" $rendURL) -}}
  {{- $_ := set $appContainer "env" $drasEnv -}}

  {{- /* renderer: S3_BUCKET, AWS_REGION envs */ -}}
  {{- $rendContainer := $v.controllers.renderer.containers.app -}}
  {{- $rendEnv := default (list) $rendContainer.env -}}
  {{- $rendEnv = append $rendEnv (dict "name" "S3_BUCKET" "value" .Values.renderer.s3.bucket) -}}
  {{- $rendEnv = append $rendEnv (dict "name" "AWS_REGION" "value" .Values.renderer.s3.region) -}}
  {{- $_ := set $rendContainer "env" $rendEnv -}}

{{- end -}}

{{ toYaml $v }}
{{- end -}}
