#!/usr/bin/env bash
# End-to-end check of per-user dashboard access against a running Compose stack
# (see docs/TESTING.md). It exercises the real plugin through Grafana, signed in
# with a browser session, and exits non-zero on the first unexpected answer.
set -euo pipefail

BASE="${GRAFANA_URL:-http://localhost:3000}"
PLUGIN="$BASE/api/plugins/vincentgourbin-pdfreporter-app/resources"
ALLOWED="pdfrep-sample-landscape"
RESTRICTED="pdf-reporter-restricted-dash"
FOLDER="pdf-reporter-restricted"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

admin() { curl -sf -u admin:admin -H "Content-Type: application/json" "$@"; }

curl -sf -c "$WORK/viewer" -H "Content-Type: application/json" \
  -d '{"user":"viewer","password":"viewer-password"}' "$BASE/login" > /dev/null
curl -sf -c "$WORK/admin" -H "Content-Type: application/json" \
  -d '{"user":"admin","password":"admin"}' "$BASE/login" > /dev/null
VIEWER_ID=$(admin "$BASE/api/users/lookup?loginOrEmail=viewer" | jq -er .id)

# expect <label> <status> <curl args...>: a 200 must also be a real PDF.
expect() {
  local label=$1 want=$2 got magic
  shift 2
  got=$(curl -s -o "$WORK/out" -w '%{http_code}' "$@")
  magic=$(head -c 4 "$WORK/out")
  if [ "$got" != "$want" ] || { [ "$want" = 200 ] && [ "$magic" != "%PDF" ]; }; then
    echo "FAIL  $label: HTTP $got, want $want: $(head -c 200 "$WORK/out")"
    exit 1
  fi
  echo "ok    $label (HTTP $got)"
}

bundle() { echo "$PLUGIN/bundle?dashboards=$1"; }

expect "viewer exports a dashboard they can read" 200 -b "$WORK/viewer" "$(bundle "$ALLOWED")"
expect "viewer is refused a restricted dashboard" 403 -b "$WORK/viewer" "$(bundle "$RESTRICTED")"
expect "/generate applies the same check" 403 -b "$WORK/viewer" "$PLUGIN/generate?dashboard=$RESTRICTED"
expect "one restricted dashboard refuses the bundle" 403 -b "$WORK/viewer" "$(bundle "$ALLOWED,$RESTRICTED")"
expect "unknown dashboard looks like a forbidden one" 403 -b "$WORK/viewer" "$(bundle does-not-exist)"
expect "forged identity header changes nothing" 403 -b "$WORK/viewer" -H "X-Grafana-Id: forged" "$(bundle "$RESTRICTED")"
expect "admin exports the restricted dashboard" 200 -b "$WORK/admin" "$(bundle "$RESTRICTED")"

admin -X POST -d "{\"items\":[{\"userId\":$VIEWER_ID,\"permission\":1}]}" \
  "$BASE/api/folders/$FOLDER/permissions" > /dev/null
expect "granting the folder opens the dashboard" 200 -b "$WORK/viewer" "$(bundle "$RESTRICTED")"

admin -X POST -d "{\"title\":\"Nested\",\"uid\":\"$FOLDER-nested\",\"parentUid\":\"$FOLDER\"}" \
  "$BASE/api/folders" > /dev/null
admin -X POST -d "{\"dashboard\":{\"uid\":\"$FOLDER-nested-dash\",\"title\":\"Nested\",\"panels\":[]},\"folderUid\":\"$FOLDER-nested\"}" \
  "$BASE/api/dashboards/db" > /dev/null
expect "a parent grant reaches a nested dashboard" 200 -b "$WORK/viewer" "$(bundle "$FOLDER-nested-dash")"

admin -X POST -d '{"items":[]}' "$BASE/api/folders/$FOLDER/permissions" > /dev/null
expect "revoking the folder closes it again" 403 -b "$WORK/viewer" "$(bundle "$RESTRICTED")"
expect "revoking also closes the nested dashboard" 403 -b "$WORK/viewer" "$(bundle "$FOLDER-nested-dash")"

admin -X POST -d "{\"items\":[{\"userId\":$VIEWER_ID,\"permission\":1}]}" \
  "$BASE/api/dashboards/uid/$RESTRICTED/permissions" > /dev/null
expect "a grant on the dashboard alone opens it" 200 -b "$WORK/viewer" "$(bundle "$RESTRICTED")"
admin -X POST -d '{"items":[]}' "$BASE/api/dashboards/uid/$RESTRICTED/permissions" > /dev/null

SA_ID=$(admin -X POST -d '{"name":"e2e-viewer-sa","role":"Viewer"}' "$BASE/api/serviceaccounts" | jq -er .id)
SA_TOKEN=$(admin -X POST -d '{"name":"e2e"}' "$BASE/api/serviceaccounts/$SA_ID/tokens" | jq -er .key)
expect "a service account caller keeps its own limits" 403 -H "Authorization: Bearer $SA_TOKEN" "$(bundle "$RESTRICTED")"
expect "a service account caller exports what it can read" 200 -H "Authorization: Bearer $SA_TOKEN" "$(bundle "$ALLOWED")"

if [ "${ANONYMOUS:-}" = 1 ]; then
  expect "anonymous caller is refused" 403 "$(bundle "$ALLOWED")"
  expect "anonymous caller cannot forge an identity" 403 -H "X-Grafana-Id: forged" "$(bundle "$ALLOWED")"
fi

echo "All access checks passed."
