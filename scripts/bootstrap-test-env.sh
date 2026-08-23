#!/bin/sh
# Provision a disposable service-account token for the local Compose stack.
set -eu

GRAFANA_URL="http://grafana:3000"
ADMIN_USER="admin"
ADMIN_PASS="admin"
PLUGIN_ID="vincentgourbin-pdfreporter-app"

echo "[bootstrap] creating service account..."
SA_ID=$(curl -sf -u "$ADMIN_USER:$ADMIN_PASS" -X POST \
  -H "Content-Type: application/json" \
  -d '{"name":"pdf-reporter-reviewer","role":"Viewer"}' \
  "$GRAFANA_URL/api/serviceaccounts" | jq -er '.id')

echo "[bootstrap] creating token..."
SA_TOKEN=$(curl -sf -u "$ADMIN_USER:$ADMIN_PASS" -X POST \
  -H "Content-Type: application/json" \
  -d '{"name":"reviewer-token"}' \
  "$GRAFANA_URL/api/serviceaccounts/$SA_ID/tokens" | jq -er '.key')

echo "[bootstrap] reading current plugin jsonData..."
JSON_DATA=$(curl -sf -u "$ADMIN_USER:$ADMIN_PASS" \
  "$GRAFANA_URL/api/plugins/$PLUGIN_ID/settings" | jq -c '.jsonData // {}')

echo "[bootstrap] enabling plugin and injecting service-account token..."
curl -sf -u "$ADMIN_USER:$ADMIN_PASS" -X POST \
  -H "Content-Type: application/json" \
  -d "{\"enabled\":true,\"pinned\":true,\"jsonData\":$JSON_DATA,\"secureJsonData\":{\"grafanaSAToken\":\"$SA_TOKEN\"}}" \
  "$GRAFANA_URL/api/plugins/$PLUGIN_ID/settings" > /dev/null

echo "[bootstrap] done. Open http://localhost:3000 (admin/admin)."
