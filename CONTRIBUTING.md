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

## Pull requests

Describe the user-visible behavior, Grafana versions tested, security impact,
and any compatibility or migration concern. Changes affecting rendering,
authentication, plugin settings, or deployment must update `README.md` and
`CHANGELOG.md`.

Keep commits focused. New dependencies require a reason, license review, and
vulnerability scan. Do not weaken TLS verification, authorization checks, or
resource limits to make a test pass.
