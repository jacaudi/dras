#!/usr/bin/env bash
# Synchronize chart/Chart.yaml version and appVersion with the supplied release version.
#
# - chart/Chart.yaml `version` is the SemVer string with NO leading 'v' (Helm requires this).
# - chart/Chart.yaml `appVersion` carries a 'v' prefix to match container image tags.
#
# Called by semantic-release's @semantic-release/exec prepareCmd at release time.
#
# Usage: scripts/sync-chart-version.sh <version>
#
#   <version> may be supplied with or without a leading 'v'. semantic-release passes
#   the un-prefixed form via ${nextRelease.version}; both forms are accepted for
#   manual invocation.
set -euo pipefail

if [[ $# -ne 1 || -z "${1:-}" ]]; then
  echo "Usage: $0 <version>" >&2
  exit 2
fi

raw="$1"
version="${raw#v}"
app_version="v${version}"

chart_file="chart/Chart.yaml"
if [[ ! -f "$chart_file" ]]; then
  echo "Error: $chart_file not found (cwd: $(pwd))" >&2
  exit 1
fi

yq -i ".version = \"${version}\" | .appVersion = \"${app_version}\"" "$chart_file"

echo "Updated $chart_file: version=${version}, appVersion=${app_version}"
