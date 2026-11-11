# Contributing

## Development setup

The standard frontend tooling uses Node 22 or newer. For local development,
run:

```bash
npm install
npm run dev
docker compose up
```

Before opening a pull request, run:

```bash
npm run lint
npm run typecheck
npm run build
git diff --check
```

Use the pinned/current Go toolchain described in `go.mod`. Do not commit
generated secrets, Grafana tokens, private dashboards, or rendered PDFs.

## Building

The build produces Linux arm64/amd64, Darwin arm64, and Windows amd64 backend
binaries:

```bash
npm run build
mage -v buildAll
make backend
```

Run backend checks with `go test ./...`, `go vet ./...`, and
`govulncheck ./...` using Go 1.26.11 or newer. Tag releases with semver (`v0.2.0`,
for example); CI builds and validates tagged distributions. The
[public-release checklist](docs/PUBLICATION.md) covers repository review,
signing, validation, release tagging, and post-publication verification.

## Installing a local build

```bash
make all
cp -a dist /var/lib/grafana/plugins/vincentgourbin-pdfreporter-app
```

For local development only, allow the unsigned plugin:

```ini
[plugins]
allow_loading_unsigned_plugins = vincentgourbin-pdfreporter-app
```

Restart Grafana after installing the plugin. Production deployments should use a
catalog/community signature or a private signature; do not enable unsigned
loading broadly. See the [private signing guide](docs/PRIVATE-SIGNING.md).

## Pull requests

Describe the user-visible behavior, Grafana versions tested, security impact,
and any compatibility or migration concern. Changes affecting rendering,
authentication, plugin settings, or deployment must update `README.md` and
`CHANGELOG.md`.

Keep commits focused. New dependencies require a reason, license review, and
vulnerability scan. Do not weaken TLS verification, authorization checks, or
resource limits to make a test pass.
