# Changelog

All notable changes to this project are documented here.

## [0.3.0] - 2026-11-11

### Security

- Exports now check that the caller can read every requested dashboard, using
  the signed identity token Grafana forwards to the plugin backend. Previously,
  any caller at or above `minRole` could export any dashboard by UID. A refused
  dashboard fails the whole export with HTTP 403, before anything is rendered.
- Callers without a signed identity, such as anonymous users, can no longer
  export.
- Service-account callers are checked against their own permissions too.

### Changed

- **Breaking:** the plugin account must now hold `users.permissions:read`. Enable
  the `externalServiceAccounts` feature toggle with
  `managed_service_accounts_enabled`, or use an Org Admin token as
  `grafanaSAToken`. Exports return HTTP 502 until this is done.
- **Breaking:** the minimum supported Grafana version is now 12.4.2, the oldest
  tested version.
- Backend requests use the SDK HTTP client, so Grafana's configured response
  limit applies. Rendered images are also capped at 256 MiB.
- Rendered images above 64 million pixels are refused before they are decoded.
- An unparseable `tlsCACert` now fails the export instead of being ignored.
- `renderTimeoutSec` values above 75 seconds now take effect; the HTTP client
  previously stopped at 90 seconds.
- Manual build and installation steps moved from the README to `CONTRIBUTING.md`.
- Updated Go to 1.26.11, the Grafana plugin SDK to 0.298.0, and the dependencies
  reported by `govulncheck` and `osv-scanner`.

## [0.2.3] - 2026-09-02

### Fixed

- Lowered the `go.mod` language directive to 1.26.5 (the SDK minimum) so the
  Grafana plugin validator's bundled `govulncheck` (Go 1.26.6) can scan the
  module. No functional change.

## [0.2.2] - 2026-09-02

### Changed

- Migrated the frontend and backend builds to Grafana's standard
  `@grafana/create-plugin` and Mage tooling.
- Frontend source maps are now included in the plugin archive for validator
  and source inspection.

### Fixed

- Enforced TLS 1.2 as the minimum TLS version for backend requests.
- Removed direct environment-variable configuration for memory limits;
  `GOMEMLIMIT` remains honored by the Go runtime and `memLimitMiB` is now
  configurable in plugin `jsonData`.

## [0.2.1] - 2026-08-23

### Fixed

- Dashboard rendering now auto-fits the actual content height
  (`height=-1&fullPageImage=true`, the same parameters as Grafana's native
  "Export as image") instead of a fixed viewport height per layout strategy.
  Any dashboard whose content exceeded the previous fixed height (1080px
  landscape / 1200px square / 2400px portrait) was silently cropped in the
  exported PDF, with no error. Found while building the reviewer sample
  dashboards: an 8-panel "square" dashboard was truncated to 4 panels.
- The `viewportWidth` setting (`jsonData`) is now actually applied as a
  render-width override; it was decoded but never used. The corresponding
  `viewportHeight` setting is removed — it was equally unused, and height is
  now always automatic.

## [0.2.0] - 2026-08-22

### Added

- Native Grafana `/render/d/<uid>` rendering with service-account
  authentication.
- Browser timezone forwarding and Grafana subpath support.
- Minimum Grafana role guard (`Viewer`, `Editor`, or `Admin`).
- Strict TLS configuration with optional enterprise CA and proxy support.
- Two-export concurrency limit, server-side image-size guard, Windows build,
  CI, vulnerability scanning, and Grafana plugin validation.
- Maintained `github.com/go-pdf/fpdf` PDF implementation.
- Go 1.26.7 toolchain and refreshed Grafana/OTel/gRPC dependencies.

### Breaking changes

- `imageRendererURL` and `rendererAuthToken` settings are no longer used.
- Grafana image rendering must be configured through Grafana's native
  `[rendering]` settings.
- TLS verification is strict by default. Existing self-signed deployments
  must explicitly set `tlsSkipVerify: true` or provide `tlsCACert`.
- The default Grafana URL is now `http://localhost:3000`.

### Security

- Grafana does not forward caller cookies or authorization headers to app
  resource handlers on 12.4.2 or 13.1.0. The plugin therefore applies a
  configurable minimum-role guard; dashboard exports use the configured
  service account and must be scoped to the minimum required permissions.

## [0.1.0] - Unreleased

- Initial private-signed implementation.
