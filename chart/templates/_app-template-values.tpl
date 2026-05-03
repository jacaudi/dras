{{/*
Build the values bag passed to bjw-s/app-template.

Responsibilities of this helper:
  1. Strip the renderer init/sidecar in standard mode.
  2. Inject container image refs from the top-level image: block,
     defaulting empty tag to .Chart.AppVersion.
  3. Inject Pushover envFrom on the dras container (always required).
  4. Append STATION_IDS / INTERVAL envs to the dras container.
  5. (Advanced mode only) Append RENDERER_URL on dras + S3_BUCKET/AWS_REGION
     on the renderer sidecar.
  6. Cross-field validation that JSON schema can't express.

Implementation note: the .Values | toYaml | fromYaml round-trip converts
common.Values typed nested maps into plain map[string]interface{} so that
sprig set/unset/dig operate reliably on all nested keys.

Layout note: as of #109 the chart ships a single Pod with a `dras` main
container and a `dras-lvl2-renderer` Kubernetes sidecar (initContainer with
restartPolicy: Always, K8s ≥1.28). The sidecar pattern guarantees the
renderer reaches Ready before dras starts, eliminating the cold-start race.
Pre-#109 the renderer was a separate Deployment hedged for hypothetical
GPU isolation; Py-ART has no GPU path so the split had no payoff and was
reversed.
*/}}
{{- define "dras.appTemplateValues" -}}
{{- /* Cross-field validation. JSON schema enforces shape; these clauses cover
     conditional invariants that schema can't express cleanly. */ -}}
{{- if not .Values.dras.stationIds -}}
  {{- fail "dras.stationIds is required (e.g. \"KATX,KRAX\")" -}}
{{- end -}}
{{- if not .Values.dras.pushover.existingSecret -}}
  {{- fail "dras.pushover.existingSecret is required (name of a Secret with PUSHOVER_API_TOKEN and PUSHOVER_USER_KEY)" -}}
{{- end -}}
{{- if eq .Values.mode "advanced" -}}
  {{- if not .Values.renderer.s3.bucket -}}
    {{- fail "renderer.s3.bucket is required in advanced mode" -}}
  {{- end -}}
  {{- if not .Values.image.renderer.repository -}}
    {{- fail "image.renderer.repository is required in advanced mode" -}}
  {{- end -}}
{{- end -}}
{{- /* Round-trip through YAML to convert common.Values to plain maps */ -}}
{{- $v := .Values | toYaml | fromYaml -}}

{{- /* Inject container image refs. The `dras` container always exists; the
     `renderer` sidecar exists only in advanced mode (it's stripped below
     in standard mode). */ -}}
{{- $drasImg := .Values.image.dras -}}
{{- $drasTag := default .Chart.AppVersion $drasImg.tag -}}
{{- $_ := set $v.controllers.dras.containers.dras "image" (dict
      "repository" $drasImg.repository
      "tag" $drasTag
      "pullPolicy" $drasImg.pullPolicy) -}}

{{- if and (hasKey $v.controllers.dras "initContainers") (hasKey $v.controllers.dras.initContainers "renderer") -}}
  {{- $rendImg := .Values.image.renderer -}}
  {{- $rendTag := default .Chart.AppVersion $rendImg.tag -}}
  {{- $_ := set $v.controllers.dras.initContainers.renderer "image" (dict
        "repository" $rendImg.repository
        "tag" $rendTag
        "pullPolicy" $rendImg.pullPolicy) -}}
{{- end -}}

{{- /* Strip renderer pieces in standard mode. Includes any persistence
     entries' advancedMounts targeting `renderer` under the `dras`
     controller, since bjw-s common fails the render with "container not
     found" when an advancedMounts key references a stripped container. */ -}}
{{- if eq .Values.mode "standard" -}}
  {{- if hasKey $v.controllers.dras "initContainers" -}}
    {{- $_ := unset $v.controllers.dras.initContainers "renderer" -}}
    {{- /* If now empty, remove the initContainers map entirely so app-template
         doesn't render an empty `initContainers: {}` into the pod spec. */ -}}
    {{- if eq (len $v.controllers.dras.initContainers) 0 -}}
      {{- $_ := unset $v.controllers.dras "initContainers" -}}
    {{- end -}}
  {{- end -}}
  {{- $_ := unset $v.service "renderer" -}}
  {{- range $persistName, $persist := $v.persistence -}}
    {{- if and $persist.advancedMounts $persist.advancedMounts.dras -}}
      {{- $_ := unset $persist.advancedMounts.dras "renderer" -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{- /* Append Pushover envFrom on the dras container */ -}}
{{- $drasContainer := $v.controllers.dras.containers.dras -}}
{{- $envFrom := default (list) $drasContainer.envFrom -}}
{{- $envFrom = append $envFrom (dict "secretRef" (dict "name" .Values.dras.pushover.existingSecret)) -}}
{{- $_ := set $drasContainer "envFrom" $envFrom -}}

{{- /*
     RESERVED CONTAINER ENV NAMES — managed by this chart, do not set in values.

     dras container:
       STATION_IDS, INTERVAL — appended below.
       PUSHOVER_API_TOKEN, PUSHOVER_USER_KEY — sourced via envFrom secretRef above.
       RENDERER_URL — appended in advanced mode.

     renderer sidecar (advanced mode):
       S3_BUCKET, AWS_REGION — appended below.

     Last write wins (chart-managed entry is appended after operator entries,
     so it shadows them in Kubernetes' env merge).
*/ -}}
{{- /* Append STATION_IDS and INTERVAL envs on the dras container */ -}}
{{- $existing := $drasContainer.env -}}
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
{{- $_ := set $drasContainer "env" $envs -}}

{{- /* Advanced-mode-only wiring. Standard mode falls through to plain dras. */ -}}
{{- if eq .Values.mode "advanced" -}}

  {{- /* dras → renderer is intra-pod via loopback. Skips the Service hop
       and removes a DNS dependency from the hot path. */ -}}
  {{- $rendURL := printf "http://localhost:%v" .Values.renderer.service.port -}}
  {{- $drasEnv := $drasContainer.env -}}
  {{- $drasEnv = append $drasEnv (dict "name" "RENDERER_URL" "value" $rendURL) -}}
  {{- $_ := set $drasContainer "env" $drasEnv -}}

  {{- /* renderer sidecar: S3_BUCKET, AWS_REGION envs */ -}}
  {{- $rendContainer := $v.controllers.dras.initContainers.renderer -}}
  {{- /* Mirror the dras-side env normalization so an operator can supply
       `env:` as either a map ({FOO: bar}) or a list ([{name: FOO, value: bar}]).
       Without this, append errors with `Cannot push on type map`. */ -}}
  {{- $rendExisting := $rendContainer.env -}}
  {{- $rendEnv := list -}}
  {{- if kindIs "map" $rendExisting -}}
    {{- range $k, $val := $rendExisting -}}
      {{- $rendEnv = append $rendEnv (dict "name" $k "value" $val) -}}
    {{- end -}}
  {{- else if $rendExisting -}}
    {{- $rendEnv = $rendExisting -}}
  {{- end -}}
  {{- $rendEnv = append $rendEnv (dict "name" "S3_BUCKET" "value" .Values.renderer.s3.bucket) -}}
  {{- $rendEnv = append $rendEnv (dict "name" "AWS_REGION" "value" .Values.renderer.s3.region) -}}
  {{- $_ := set $rendContainer "env" $rendEnv -}}

{{- end -}}

{{ toYaml $v }}
{{- end -}}
