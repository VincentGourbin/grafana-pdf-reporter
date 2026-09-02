# Public publication checklist

This checklist is intended for the repository maintainer before making the
project public and publishing a release.

This covers the path to a Community-signed, Catalog-listed release, which
requires a prior review by the Grafana team (see "Signing and release"
below). An organization that wants to deploy internally without waiting for
that review should use the [private signing guide](PRIVATE-SIGNING.md)
instead; it does not require a Grafana review.

## Repository visibility

- [x] Confirm the GitHub repository name and owner. (public, `VincentGourbin/grafana-pdf-reporter`)
- [x] Review the complete Git history for tokens, private URLs, customer data,
      and rendered PDFs. (scanned 2026-08-23: no secret-shaped strings or
      credential-shaped filenames in any commit, any branch)
- [ ] Remove or rewrite any private infrastructure references that are not
      needed by users.
- [ ] Set the repository description and topics (`grafana`, `pdf`, `reporting`,
      `observability`).
- [ ] Enable private vulnerability reporting and Dependabot where appropriate.
- [ ] Configure branch protection for `main` and require CI before merge.

## Release content

- [x] Confirm `README.md`, [use cases](USE-CASES.md), and the
      [enterprise guide](DEPLOYMENT-ENTERPRISE.md) describe the current
      behavior.
- [x] Confirm `LICENSE`, `SECURITY.md`, `CONTRIBUTING.md`, and `CHANGELOG.md`
      are present.
- [ ] Replace placeholder or unavailable contact details.
- [ ] Confirm screenshots contain no private dashboards or secrets.
- [x] Confirm `src/plugin.json` links resolve after the repository is public.

## Build and validation

- [x] Run `go test ./...` and `go vet ./...`. (no test files exist yet — `go
      test` passes vacuously; consider adding coverage for `strategy.go` and
      `render.go`'s query-building logic)
- [x] Run `govulncheck ./...`. (clean, run in CI on every push/tag)
- [ ] Run `npm run lint`, `npm run typecheck`, and `npm run build` for the
      standard frontend build.
- [ ] Run `npm ci`, `npm run typecheck`, `npm run lint`, and `npm run build`.
- [ ] Run `mage -v buildAll` and verify the four required backend artifacts.
- [ ] Validate the complete v0.2.2 ZIP archive with Grafana's plugin
      validator. Expected warnings: unsigned plugin and optional
      sponsorship-link recommendation.
- [x] Test the release against the supported Grafana versions and the
      configured renderer. (12.4.2 and 13.1.0, end-to-end PDF export
      verified visually on both)
- [x] Verify Linux arm64/amd64, macOS arm64, and Windows amd64 artifacts.
      (all 4 present in the v0.2.1 release zip, 0755 permissions)

## Signing and release

- [x] Create or verify the Grafana developer/catalog account used to publish
      the plugin ID. (account/org slug matches `vincentgourbin`, confirmed
      by the validator no longer warning about it)
- [x] Submit the plugin for Grafana review and obtain the assigned Community
      signature level before signing (public submission happens before
      signing, not after — see https://grafana.com/developers/plugin-tools/publish-a-plugin/publish-a-plugin).
      **Submitted 2026-08-23** (v0.2.1). Awaiting Grafana's automated +
      manual review; they assign a Community signature level once approved.
- [ ] Sign the plugin using the assigned Grafana signing mode and root URLs.
      (blocked on review outcome above — Community signing does not use
      `ROOT_URLS`/`make sign` the way private signing does; confirm the
      exact mechanics Grafana gives once the signature level is assigned)
- [ ] Confirm the signed archive does not include tokens or local build files.
- [x] Create the annotated tag (`v0.2.1`, moved once after an initial CI
      validator failure on relative README links — no release had been
      published under the failed tag, so moving it was safe).
- [x] Push the tag only after the final archive has passed validation.
- [ ] Publish release notes with migration instructions from 0.1.x.
- [ ] Announce the known Grafana 12.4.2 time-picker rendering limitation.

## After publication

- [ ] Re-run the validator after GitHub links become public.
- [ ] Verify the README screenshots and plugin links from a clean browser.
- [ ] Monitor issues and security reports.
- [ ] Keep any production migration and exposed-token rotation as a separate,
      manually approved operational change.
- [ ] Once Grafana responds to the submission: sign with the assigned level,
      republish, and update this checklist.
