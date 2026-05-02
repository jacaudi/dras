#!/usr/bin/env bash
# Tests for scripts/sync-chart-version.sh
# Run: bash scripts/sync-chart-version_test.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SYNC="$SCRIPT_DIR/sync-chart-version.sh"

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

mkdir -p "$tmpdir/chart"
cat >"$tmpdir/chart/Chart.yaml" <<'EOF'
apiVersion: v2
name: dras
version: 0.0.0
appVersion: v0.0.0
description: test
EOF

failures=0
fail()  { echo "FAIL: $1"; failures=$((failures+1)); }
pass()  { echo "PASS: $1"; }

# Test 1: rewrites version (no v) and appVersion (with v)
cd "$tmpdir"
"$SYNC" 2.7.0
got_version=$(yq -r '.version' chart/Chart.yaml)
got_app=$(yq -r '.appVersion' chart/Chart.yaml)
if [ "$got_version" = "2.7.0" ] && [ "$got_app" = "v2.7.0" ]; then
  pass "rewrites version=2.7.0 and appVersion=v2.7.0"
else
  fail "expected version=2.7.0 appVersion=v2.7.0, got version=$got_version appVersion=$got_app"
fi

# Test 2: handles existing v-prefixed input gracefully
"$SYNC" v3.0.0
got_version=$(yq -r '.version' chart/Chart.yaml)
got_app=$(yq -r '.appVersion' chart/Chart.yaml)
if [ "$got_version" = "3.0.0" ] && [ "$got_app" = "v3.0.0" ]; then
  pass "strips leading v from version arg"
else
  fail "expected v stripping, got version=$got_version appVersion=$got_app"
fi

# Test 3: errors on missing argument
if "$SYNC" 2>/dev/null; then
  fail "expected non-zero exit on missing arg"
else
  pass "errors on missing arg"
fi

# Test 4: errors on missing Chart.yaml
mv chart chart.bak
if "$SYNC" 4.0.0 2>/dev/null; then
  fail "expected non-zero exit when chart/Chart.yaml is missing"
else
  pass "errors when Chart.yaml is missing"
fi
mv chart.bak chart

# Test 5: idempotent — running twice produces same result
"$SYNC" 5.0.0 >/dev/null
"$SYNC" 5.0.0 >/dev/null
got_version=$(yq -r '.version' chart/Chart.yaml)
got_app=$(yq -r '.appVersion' chart/Chart.yaml)
if [ "$got_version" = "5.0.0" ] && [ "$got_app" = "v5.0.0" ]; then
  pass "idempotent on repeat invocation"
else
  fail "idempotency check failed: version=$got_version appVersion=$got_app"
fi

if [ "$failures" -gt 0 ]; then
  echo "$failures test(s) failed"
  exit 1
fi
echo "All 5 tests passed."
