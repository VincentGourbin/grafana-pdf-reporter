# Changelog

All notable changes to this project are documented here.

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
