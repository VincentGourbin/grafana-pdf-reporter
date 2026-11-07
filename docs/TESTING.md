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

## Check per-user access

The environment creates a `viewer` user (password `viewer-password`) and a
**PDF Reporter Restricted** folder that this user cannot open. Sign in as
`viewer`: the app page does not list the restricted dashboard, and requesting it
directly is refused with HTTP 403:

```text
/api/plugins/vincentgourbin-pdfreporter-app/resources/bundle?dashboards=pdf-reporter-restricted-dash
```

To run the full scenario, covering grants, revocation, nested folders, bundles,
forged headers, and service-account callers:

```bash
scripts/e2e-access-control.sh
```

It needs `curl` and `jq`. It changes folder permissions and creates a service
account, so run it on a fresh environment. It checks access, not image content:
open a generated PDF to confirm the panels show data.

The stack runs on Grafana 13.1.0 with the managed plugin account. To test another
version, or an Org Admin token instead of the managed account:

```bash
GRAFANA_VERSION=12.4.2 docker compose up
PDF_REPORTER_FEATURES= docker compose up
```

## Clean up

```bash
docker compose down -v
```

## Notes

This local-only environment uses `admin`/`admin` and allows the unsigned
plugin. The account it uses is created at startup; no real token is stored in
the repository. For signed deployment, see
[PRIVATE-SIGNING.md](PRIVATE-SIGNING.md).
