# Testing this plugin (reviewer / evaluation environment)

This launches a disposable Grafana + renderer + PDF Reporter environment with
three sample dashboards. They demonstrate automatic PDF page orientation:
landscape, square viewport, and portrait.

## Prerequisites

Docker and Docker Compose.

## Run

```bash
make all
docker compose up
```

Wait for the `bootstrap` service to print
`[bootstrap] done. Open http://localhost:3000 (admin/admin).`.

## Try it

1. Open http://localhost:3000 and sign in with `admin` / `admin`.
2. Open the **PDF Reporter Samples** folder to view the three dashboards.
3. Open **PDF Reporter** from the left menu.
4. Select one, two, or all sample dashboards and click **Export PDF**.
5. Inspect the download: each selected dashboard gets a cover and rendered
   page. The Landscape, Square, and Portrait names describe the automatic
   render strategy being exercised. The render height always auto-fits the
   dashboard's actual content, so nothing is cropped regardless of panel
   count; Square uses a 1600px-wide viewport while retaining landscape A4,
   Portrait uses a 1280px-wide viewport with portrait A4.

## Clean up

```bash
docker compose down -v
```

## Notes

This local-only environment uses `admin`/`admin` and allows the unsigned
plugin. It creates a disposable Grafana service-account token at startup; no
real token is stored in the repository. For signed deployment, see
[PRIVATE-SIGNING.md](PRIVATE-SIGNING.md).
