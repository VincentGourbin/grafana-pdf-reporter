#!/bin/sh
# Prepare the local Compose stack: enable the plugin, give it an account, and
# create a Viewer with a dashboard they cannot open.
set -eu

GRAFANA_URL="http://grafana:3000"
ADMIN_USER="admin"
ADMIN_PASS="admin"
PLUGIN_ID="vincentgourbin-pdfreporter-app"
FOLDER_UID="pdf-reporter-restricted"

api() {
  curl -sf -u "$ADMIN_USER:$ADMIN_PASS" -H "Content-Type: application/json" "$@"
}

echo "[bootstrap] reading current plugin jsonData..."
JSON_DATA=$(api "$GRAFANA_URL/api/plugins/$PLUGIN_ID/settings" | jq -c '.jsonData // {}')

SECURE_JSON_DATA="{}"
case ",${PDF_REPORTER_FEATURES:-}," in
  *,externalServiceAccounts,*) MANAGED=1 ;;
  *) MANAGED= ;;
esac
if [ -z "$MANAGED" ]; then
  # Without externalServiceAccounts nothing provisions the plugin's account, and
  # only an Org Admin holds users.permissions:read.
  echo "[bootstrap] creating an Org Admin service account..."
  SA_ID=$(api -X POST -d '{"name":"pdf-reporter-reviewer","role":"Admin"}' \
    "$GRAFANA_URL/api/serviceaccounts" | jq -er '.id')
  SA_TOKEN=$(api -X POST -d '{"name":"reviewer-token"}' \
    "$GRAFANA_URL/api/serviceaccounts/$SA_ID/tokens" | jq -er '.key')
  SECURE_JSON_DATA="{\"grafanaSAToken\":\"$SA_TOKEN\"}"
fi

echo "[bootstrap] enabling plugin..."
api -X POST \
  -d "{\"enabled\":true,\"pinned\":true,\"jsonData\":$JSON_DATA,\"secureJsonData\":$SECURE_JSON_DATA}" \
  "$GRAFANA_URL/api/plugins/$PLUGIN_ID/settings" > /dev/null

echo "[bootstrap] creating the viewer user and a restricted folder..."
# Tolerate a restart that kept the data: these may already exist.
api -X POST -d '{"name":"Viewer","login":"viewer","email":"viewer@example.com","password":"viewer-password"}' \
  "$GRAFANA_URL/api/admin/users" > /dev/null || echo "[bootstrap] viewer already exists"
api -X POST -d "{\"title\":\"PDF Reporter Restricted\",\"uid\":\"$FOLDER_UID\"}" \
  "$GRAFANA_URL/api/folders" > /dev/null || echo "[bootstrap] folder already exists"
api -X POST \
  -d "{\"dashboard\":{\"uid\":\"pdf-reporter-restricted-dash\",\"title\":\"Restricted dashboard\",\"panels\":[]},\"folderUid\":\"$FOLDER_UID\",\"overwrite\":true}" \
  "$GRAFANA_URL/api/dashboards/db" > /dev/null
api -X POST -d '{"items":[]}' "$GRAFANA_URL/api/folders/$FOLDER_UID/permissions" > /dev/null

echo "[bootstrap] done. Open http://localhost:3000 (admin/admin)."
