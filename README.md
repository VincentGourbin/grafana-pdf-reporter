# grafana-pdf-reporter

Grafana app plugin (Go + AMD JS) that exports dashboards to nicely-branded PDF
reports — single dashboard or multi-dashboard bundle, with a customizable
cover page (logo, accent color, background image).

## Features

- **Sidebar entry** (megamenu) → page with multi-select dashboard list +
  Grafana-native `TimeRangePicker`.
- **Cmd+K shortcut** to export the current dashboard directly.
- **Multi-dashboard bundle**: pick N dashboards → one concatenated PDF
  (cover + page per dashboard), each section uses A4 with the orientation
  that best fits the dashboard's natural aspect ratio (landscape / square /
  portrait, auto-detected via Grafana API `gridPos`).
- **Cover branding** via Plugin Settings: brand title, subtitle, footer
  texts, accent color, logo (PNG/JPEG ≤200 KB), full-page background image
  (PNG/JPEG ≤2 MB). A semi-transparent card overlays the background so it
  remains visible.
- **Image-gen prompt template** in Settings: copy-paste into DALL-E / FLUX /
  Imagen / Midjourney to generate a matching background. The prompt bakes in
  your accent color, theme, and reserved-zone constraints.
- **Live preview** of the cover page in the Settings view, matching the PDF
  output 1:1.
- **i18n FR/EN** auto-detected via `navigator.language`.
- **Theme** follows the user's current Grafana theme automatically.

## Architecture

```
   User Grafana (already SSO-authed)
            │
            ▼
   ┌─────────────────────────────────────────┐
   │  Grafana :3000                          │
   │   ├─ vincentgourbin-pdfreporter (this)  │  Go subprocess
   │   │    ├─ GET /resources/generate       │  1 dashboard
   │   │    │      ?dashboard=<uid>&from=&to=
   │   │    └─ GET /resources/bundle         │  N dashboards
   │   │           ?dashboards=uid1,uid2,...
   │   └─ image-renderer :8181  ──► Chromium ──► localhost:3000
   └─────────────────────────────────────────┘
```

Requires the official **`grafana-image-renderer`** running locally — the
plugin doesn't ship Chromium itself.

## Build

Everything is dockerized (Go 1.23 + Node 20) — no host deps required.

```bash
make all          # backend (arm64 + amd64 + darwin-arm64) + frontend → dist/
make backend      # backend only
make frontend     # frontend only
make clean        # nuke dist/
```

## Install

```bash
# 1. Copy dist/ to Grafana's plugin dir.
cp -a dist /var/lib/grafana/plugins/vincentgourbin-pdfreporter-app

# 2a. If unsigned (dev / personal): allow unsigned plugin loading.
#     In grafana.ini / env:
#       GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=vincentgourbin-pdfreporter-app
# 2b. If signed (private signature, see "Signing" below): nothing to do.

# 3. Restart Grafana.

# 4. As Grafana Admin, configure the plugin settings via UI:
#    Apps → PDF Reporter → Settings
#    - grafanaSAToken (secret): a Grafana Service Account token (Viewer role
#      is enough — used to fetch dashboard metadata).
#    - rendererAuthToken (secret): the X-Auth-Token of the local
#      image-renderer service.
#    - Optionally: cover branding fields (title, footer, accent, logo,
#      background, image-gen prompt template).
```

## Signing (private signature for production)

By default the plugin is unsigned — Grafana refuses to load it unless you
set `GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS`. The clean alternative is a
**private signature**: free, bound to a list of root URLs you control.

### Once: create a grafana.com access policy token

1. Sign in at https://grafana.com.
2. Org settings → **Access Policies** → **Create access policy** with
   the **`plugins:write`** scope on your organization realm.
3. Generate a token from that policy — copy it (`glc_...`), you won't see
   it again.

### Each build: sign with the token

```bash
export GRAFANA_ACCESS_POLICY_TOKEN=glc_...

# Sign for the URLs you'll actually access Grafana from:
make sign                                         # uses default ROOT_URLS
# or:
make sign ROOT_URLS=https://your-host/,https://lan-ip:3000/
```

The signature is **bound to the URLs** — accessing Grafana via any URL not
listed will silently refuse to load the plugin (Grafana logs
`signature-invalid`). Always list every entry point: external FQDN, LAN IP,
reverse-proxy URL, etc.

`make sign` produces `dist/MANIFEST.txt`. Redeploy `dist/` and you can drop
`GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS` from your Grafana config.

## Plugin Settings (Settings page)

| Field | Where stored | Description |
|---|---|---|
| `grafanaSAToken` | `secureJsonData` | Grafana SA token (Viewer) for `/api/dashboards/uid/<uid>` |
| `rendererAuthToken` | `secureJsonData` | X-Auth-Token sent to `image-renderer` |
| `grafanaURL` | `jsonData` | Default `https://127.0.0.1:3000` |
| `imageRendererURL` | `jsonData` | Default `http://127.0.0.1:8181` |
| `coverBrandTitle` | `jsonData` | Cover top-left brand line |
| `coverBrandSubtitle` | `jsonData` | Cover top-left subline |
| `coverFooterLeft` | `jsonData` | Cover footer left text |
| `coverFooterRight` | `jsonData` | Cover footer right text |
| `coverAccentHex` | `jsonData` | Hex color, e.g. `#10B981` |
| `coverLogoDataURL` | `jsonData` | PNG/JPEG dataURL ≤200 KB |
| `coverBackgroundDataURL` | `jsonData` | PNG/JPEG dataURL ≤2 MB |

## Development

```bash
# Local dev harness (no Grafana required) — stubs @grafana/data/ui/runtime
# and mocks fetch for /api/search and plugin settings.
python3 -m http.server 8765 --bind 127.0.0.1
open http://127.0.0.1:8765/dev/        # ?lang=fr or ?lang=en to switch
```

## License

Apache-2.0 (see `LICENSE`).
