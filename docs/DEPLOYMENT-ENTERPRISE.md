# Enterprise deployment guide

This guide describes a conservative deployment of PDF Reporter in Grafana
OSS. It is a deployment baseline, not a certification or a claim of
compliance with a specific framework.

## Before installation

Confirm:

1. Grafana is version 12.4.2 or later; 12.4.2 and 13.1.0 are tested.
2. Native Grafana image rendering works independently of the plugin.
3. The renderer is configured through Grafana `[rendering]` settings.
4. The plugin has an account that can read caller permissions, as described
   below.
5. The plugin backend can reach `grafanaURL` over the intended network path.
6. The deployment has a plan for plugin signature verification and upgrades.

## Recommended plugin account

The plugin checks each caller's dashboard access, which requires
`users.permissions:read`. Enable the `externalServiceAccounts` feature toggle and
set `managed_service_accounts_enabled = true` under `[auth]`, so Grafana
provisions an account limited to the permissions the plugin declares. This
feature is in public preview and supports a single organization.

Otherwise, create a dedicated service account with the Org Admin role. On
Grafana OSS, no lower role holds `users.permissions:read`. This token is an
administrator credential: do not share it with other tools, and rotate it
through the secret-management process used by the organization.

The token belongs in Grafana's encrypted `secureJsonData` and must not appear
in Git, provisioning repositories, logs, screenshots, or issue reports.

## TLS baseline

Use HTTPS for a remote Grafana endpoint and keep:

```yaml
tlsSkipVerify: false
```

For an internal PKI, set `tlsCACert` to the required PEM CA certificate. Use
`tlsSkipVerify: true` only as a time-bounded exception for a controlled
self-signed endpoint, with an explicit risk acceptance and compensating
network controls.

## Authorization baseline

Set `minRole` according to the least-privilege policy for the app page:

```yaml
minRole: Viewer
```

Use `Editor` or `Admin` only when the deployment genuinely requires it. The
minimum role protects access to the export operation. On top of it, the plugin
confirms that the caller can read every requested dashboard, using the signed
identity Grafana forwards, and refuses the whole export otherwise.

## Resource controls

The plugin bounds resource use with:

- at most two complete PDF exports per app instance;
- a 30-second queue wait before returning HTTP 429;
- a maximum of 25 dashboards per bundle;
- a maximum cover background size of 800 KiB;
- configurable viewport, scale, and render timeout values.

Set operational limits outside the plugin as well: reverse-proxy request
timeouts, Grafana/plugin process memory limits, log retention, and PDF storage
retention.

## Installation and verification

1. Install the signed plugin archive into the Grafana plugin directory or
   through the Grafana Catalog when available. See the
   [private signing guide](PRIVATE-SIGNING.md) to produce a signed archive
   bound to this organization's Grafana root URLs.
2. Restart Grafana and verify the plugin signature is accepted.
3. Configure the plugin account and TLS settings.
4. Open the app as a Viewer and export a dashboard that the Viewer can read.
5. Confirm the generated PDF contains the expected time range and timezone.
6. Confirm an account below `minRole` receives HTTP 403.
7. Confirm a dashboard the Viewer cannot open returns HTTP 403.
8. Record plugin version, Grafana version, renderer version, and configuration
   in the deployment record.

## Monitoring and incident response

Monitor Grafana/plugin logs for failed exports, HTTP 403/429/502 responses,
render latency, and repeated authentication failures. Do not enable verbose
request-header logging in production because headers can contain credentials.

If the service-account token may have been exposed:

1. revoke it immediately;
2. issue a replacement with the same least-privilege scope;
3. update the app instance through the secret-management process;
4. review Grafana and reverse-proxy logs for use of the old token;
5. record the incident and retain only the minimum required evidence.

## Upgrade and rollback

Test the new plugin against the target Grafana version, validate the archive
with Grafana's plugin validator, and retain the previous signed archive for
rollback. Review the changelog for breaking changes, especially rendering,
TLS, and service-account configuration changes.
