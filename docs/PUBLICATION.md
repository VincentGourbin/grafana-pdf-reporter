# Public publication checklist

This checklist is intended for the repository maintainer before making the
project public and publishing a release.

This covers the path to a Community-signed, Catalog-listed release, which
requires a prior review by the Grafana team (see "Signing and release"
below). An organization that wants to deploy internally without waiting for
that review should use the [private signing guide](PRIVATE-SIGNING.md)
instead; it does not require a Grafana review.

## Repository visibility

- [ ] Confirm the GitHub repository name and owner.
- [ ] Review the complete Git history for tokens, private URLs, customer data,
      and rendered PDFs.
- [ ] Remove or rewrite any private infrastructure references that are not
      needed by users.
- [ ] Set the repository description and topics (`grafana`, `pdf`, `reporting`,
      `observability`).
- [ ] Enable private vulnerability reporting and Dependabot where appropriate.
- [ ] Configure branch protection for `main` and require CI before merge.

## Release content

- [ ] Confirm `README.md`, [use cases](USE-CASES.md), and the
      [enterprise guide](DEPLOYMENT-ENTERPRISE.md) describe the current
      behavior.
- [ ] Confirm `LICENSE`, `SECURITY.md`, `CONTRIBUTING.md`, and `CHANGELOG.md`
      are present.
- [ ] Replace placeholder or unavailable contact details.
- [ ] Confirm screenshots contain no private dashboards or secrets.
- [ ] Confirm `src/plugin.json` links resolve after the repository is public.

## Build and validation

- [ ] Run `go test ./...` and `go vet ./...`.
- [ ] Run `govulncheck ./...`.
- [ ] Run `node --check src/module.amd.js`.
- [ ] Run `make all VERSION=x.y.z`.
- [ ] Validate the complete ZIP archive with Grafana's plugin validator.
- [ ] Test the release against the supported Grafana versions and the
      configured renderer.
- [ ] Verify Linux arm64/amd64, macOS arm64, and Windows amd64 artifacts.

## Signing and release

- [ ] Create or verify the Grafana developer/catalog account used to publish
      the plugin ID.
- [ ] Submit the plugin for Grafana review and obtain the assigned Community
      signature level before signing (public submission happens before
      signing, not after — see https://grafana.com/developers/plugin-tools/publish-a-plugin/publish-a-plugin).
- [ ] Sign the plugin using the assigned Grafana signing mode and root URLs.
- [ ] Confirm the signed archive does not include tokens or local build files.
- [ ] Create the annotated tag (`v0.2.0` for the first public release).
- [ ] Push the tag only after the final archive has passed validation.
- [ ] Publish release notes with migration instructions from 0.1.x.
- [ ] Announce the known Grafana 12.4.2 time-picker rendering limitation.

## After publication

- [ ] Re-run the validator after GitHub links become public.
- [ ] Verify the README screenshots and plugin links from a clean browser.
- [ ] Monitor issues and security reports.
- [ ] Keep any production migration and exposed-token rotation as a separate,
      manually approved operational change.
