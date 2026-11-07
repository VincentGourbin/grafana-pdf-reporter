# Use cases

PDF Reporter is intended for teams that already use Grafana as the source of
truth and need a portable, branded snapshot of one or more dashboards.

## 1. Scheduled operations report

An operations team exports a daily or weekly report containing availability,
latency, error rate, and capacity dashboards.

Recommended setup:

- configure the plugin account described in the README;
- give the people who export reports access to the report dashboards;
- keep `minRole: Viewer` unless the Grafana deployment requires a stricter
  boundary;
- use the same `from` and `to` expressions as the operational runbook;
- use a neutral cover and a short retention period for generated PDFs.

The plugin generates the dashboard image through Grafana's native render
endpoint, so the report uses the same dashboard queries and time range as the
Grafana deployment.

## 2. Executive or customer-facing report

A platform or account team needs a readable PDF for a meeting, review, or
customer deliverable.

Recommended setup:

- configure the cover title, subtitle, accent, footer, and logo;
- select a light or dark theme deliberately for the target medium;
- use a background image with a calm, mostly empty center;
- keep customer or confidential information out of the cover background and
  branding fields;
- review the generated PDF before distribution.

The cover background is limited to 800 KiB. JPEG is recommended for textured
backgrounds because it reduces the size stored in Grafana configuration.

## 3. Incident review and post-mortem

An incident commander exports several dashboards for a precise incident
window, then attaches the PDF to a ticket or post-mortem.

Recommended setup:

- select the exact absolute or relative time range;
- export the dashboards as one bundle in a deliberate order;
- use the browser timezone so labels match the operator's working context;
- record the Grafana version and plugin version alongside the PDF.

The plugin does not alter dashboard data. It captures the rendered dashboard
and adds the selected cover page.

## 4. Multi-dashboard service review

A service owner wants one document covering application health, infrastructure
capacity, logs-derived signals, and SLOs.

The app supports selecting multiple dashboards and concatenates them into one
PDF. The person exporting must have access to every selected dashboard. A
bundle is limited to 25 dashboards and concurrent exports are limited to two
per app instance to bound resource consumption.

## 5. Enterprise Grafana behind a subpath or private PKI

The Grafana UI is served at `/grafana/`, or Grafana is reachable over HTTPS
with an internal enterprise CA.

Recommended setup:

- enable `GF_SERVER_SERVE_FROM_SUB_PATH=true` and use the same root URL in
  Grafana and the browser;
- set `grafanaURL` to the backend-reachable URL, including `/grafana` when the
  backend route is served under that subpath;
- provide the enterprise CA in `tlsCACert`;
- leave `tlsSkipVerify` set to `false`.

The subpath behavior is covered by the acceptance tests. `tlsSkipVerify` is a
controlled exception for a self-signed endpoint, not the recommended
enterprise configuration.

## 6. GitOps or reproducible deployment

An infrastructure team provisions the app instance together with Grafana and
keeps non-secret settings in version control.

Store `jsonData` in provisioning, inject the service-account token through the
deployment secret mechanism, and never commit `secureJsonData` values. The
example in the main [README](../README.md) is deliberately minimal; adapt the
secret substitution to the chosen Grafana deployment method.

## Not appropriate for

PDF Reporter is not a substitute for:

- per-panel or per-row authorization inside a dashboard;
- data masking, anonymization, or redaction;
- a long-term document-management or evidence-retention system;
- exporting dashboards that the caller cannot open in Grafana;
- rendering arbitrary websites or HTML supplied by untrusted users;
- high-volume batch rendering without capacity planning.

Access is checked per dashboard against the caller's own Grafana permissions.
The render itself runs as the plugin account, so data-source permissions that
differ between that account and the caller are not reflected in the PDF.
