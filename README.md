# grafana-pdf-reporter

Grafana app plugin: export dashboards to PDF with a custom cover page.

## Architecture

```
┌─ Grafana ────────────────────────────────────────┐
│                                                  │
│   [Bouton "Export PDF" dans la nav du dashboard] │
│           │                                      │
│           ▼ frontend (React, src/module.ts)      │
│   navigate to /api/plugins/<id>/resources/       │
│                 generate?dashboard=<uid>         │
│           │                                      │
│           ▼ backend (Go, pkg/plugin/)            │
│   1. fetchDashboardTitle (Grafana /api/dashboards│
│      /uid/<uid>)                                 │
│   2. renderDashboardPNG (call image-renderer)    │
│   3. cropToContent (Pillow-equivalent in Go)     │
│   4. buildReportPDF (gofpdf : cover + dashboard) │
│           │                                      │
│           ▼ return                               │
│   binary PDF download                            │
└──────────────────────────────────────────────────┘
```

Le plugin assume que **`grafana-image-renderer`** est installé (officiel Grafana, container ou plugin).

## Setup

### Backend Go

```bash
go mod download
make backend  # cross-compile linux/arm64, linux/amd64, darwin/arm64
```

### Frontend

```bash
npm install
npm run build  # produit dist/module.js
```

### Build complet

```bash
make all       # backend + frontend → dist/
```

## Installation sur Grafana

1. Copier `dist/` dans `<grafana-data>/plugins/vincentgourbin-pdfreporter-app/`
2. Activer les plugins non-signés (dev) :
   ```
   GF_PLUGINS_ALLOW_LOADING_UNSIGNED=vincentgourbin-pdfreporter-app
   ```
3. Configurer les env vars du plugin (Service Account Grafana, etc.) :
   ```
   GF_PDFREPORTER_SA_TOKEN=glsa_...
   GF_PDFREPORTER_IMAGE_RENDERER_URL=http://127.0.0.1:8181
   GF_PDFREPORTER_RENDERER_AUTH_TOKEN=...
   ```
4. Restart Grafana
5. Activer le plugin : Configuration > Plugins > PDF Reporter > Enable

## Config

| Env var | Default | Description |
|---|---|---|
| `GF_PDFREPORTER_GRAFANA_URL` | `https://127.0.0.1:3000` | URL interne Grafana |
| `GF_PDFREPORTER_SA_TOKEN` | _required_ | Service Account token (rôle Viewer) |
| `GF_PDFREPORTER_IMAGE_RENDERER_URL` | `http://127.0.0.1:8181` | URL image-renderer |
| `GF_PDFREPORTER_RENDERER_AUTH_TOKEN` | _required_ | X-Auth-Token image-renderer |
| `GF_PDFREPORTER_VIEWPORT_WIDTH` | `1280` | Largeur viewport CSS |
| `GF_PDFREPORTER_VIEWPORT_HEIGHT` | `3000` | Hauteur viewport (≫ dashboard → crop) |
| `GF_PDFREPORTER_RENDER_TIMEOUT_SEC` | `60` | Timeout render |

## Status

🚧 **MVP en cours**. Stub backend + frontend en place, build pipeline à finaliser
(webpack pour le frontend). Pour l'instant, prototype Python en parallèle dans
`optimorin-jetson/jetson/pdf-renderer/`.
