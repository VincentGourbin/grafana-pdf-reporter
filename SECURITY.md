# Security policy

## Supported versions

Security fixes are developed against the latest published release. Deployments
should stay on a supported Grafana OSS version and keep the image renderer and
Grafana dependencies patched according to the organization's vulnerability
management process.

## Reporting a vulnerability

Please do not open a public issue for an unpatched security vulnerability.
Use GitHub's private vulnerability reporting feature for this repository when
it is enabled. If it is not available, contact the repository maintainer
through the private contact channel associated with the GitHub account
`VincentGourbin` and include:

- a short description and impact;
- affected plugin and Grafana versions;
- clear reproduction steps or a minimal proof of concept;
- any suggested mitigation;
- whether the issue is already publicly known.

Do not include real Grafana tokens, customer data, dashboard exports, or
private hostnames in the report. Redact logs and configuration before sharing.

The maintainer will acknowledge receipt when practicable, investigate the
report, coordinate a fix, and publish a release note once disclosure is safe.

## Deployment security expectations

PDF Reporter is not a data-loss-prevention or anonymization product. Use a
least-privileged service account, encrypted Grafana secure settings, strict
TLS verification, signed plugin packages, and an appropriate retention policy
for generated PDFs.
