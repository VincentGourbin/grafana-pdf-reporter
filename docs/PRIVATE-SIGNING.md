# Private signing for internal deployment

This guide is for an organization that wants to run PDF Reporter today,
without waiting for the plugin to be reviewed and listed on the Grafana
Catalog. A **private signature** is the standard Grafana mechanism for this:
it is bound to the specific root URLs where the organization's Grafana
instances are reachable, and any Grafana OSS instance can install it without
enabling unsigned plugin loading. This is not a workaround; it is Grafana's
documented path for internally-distributed plugins.

Community/Catalog signing is a separate process that requires a prior review
by the Grafana team; see the [publication checklist](PUBLICATION.md) if the
goal is a public Catalog listing instead of (or in addition to) internal use.

## 1. Create an access policy token (once per organization)

1. Sign in at [grafana.com](https://grafana.com) with the account that will
   own the signature.
2. Go to **My Account → Security → Access Policies**.
3. Create a policy with:
   - **Realm**: the organization's own realm (not `grafana.com`, which is
     reserved for Catalog-signed plugins);
   - **Scope**: `plugins:write`.
4. Generate a token from that policy. Copy it immediately; it is not shown
   again. Treat it like any other credential: store it in the organization's
   secret manager, not in a shell history file or a repository.

## 2. Build and sign a release

```bash
export GRAFANA_ACCESS_POLICY_TOKEN=glc_...   # from step 1

make all VERSION=x.y.z
make sign ROOT_URLS=https://grafana.example.com/,https://grafana.internal.example.com:3000/
```

`ROOT_URLS` must list **every** URL through which any user will reach
Grafana: the public FQDN, an internal LAN address, a reverse-proxy hostname,
a Grafana Cloud subpath, and so on. Grafana matches the request's `Host`
against this list at load time; an entry point that is missing from the list
makes the plugin silently refuse to load on that URL (Grafana logs
`signature-invalid`). Add every entry point up front — do not rely on
discovering a missing one in production.

`make sign` produces `dist/MANIFEST.txt`, which contains the SHA-256 of every
file in `dist/` plus the signature. Do not edit any file in `dist/` after
signing; that invalidates the manifest and Grafana will refuse to load the
plugin.

## 3. Deploy

```bash
cp -a dist /var/lib/grafana/plugins/vincentgourbin-pdfreporter-app
```

Restart Grafana. **Do not** set
`[plugins] allow_loading_unsigned_plugins` — a correctly signed `dist/`
does not need it, and leaving it set defeats the purpose of signing.

Verify in **Administration → Plugins → PDF Reporter** that the plugin loads
and shows a valid signature (not "unsigned"). If it does not load, check the
Grafana log for `signature-invalid` and confirm the URL used to reach
Grafana that day is in `ROOT_URLS`.

## 4. Re-sign on every release

The signature covers the exact file set and version in `dist/`. Any new
build — including a patch release, a configuration-only change to
`plugin.json`, or a rebuild with a different `VERSION` — must be re-signed
before deployment. There is no way to reuse a previous `MANIFEST.txt`
against different files. Treat "build → sign → deploy" as one atomic release
step; do not deploy an unsigned intermediate build.

## 5. Multiple environments

If staging and production are reachable through different root URLs and the
organization wants to test a build in staging before promoting the exact
same artifact to production, include both environments' URLs in the same
`ROOT_URLS` list and sign once. Signing separately per environment produces
different `dist/` archives (different `MANIFEST.txt`) and means staging did
not actually validate the artifact that reaches production.

## 6. Token hygiene

- Scope the access policy to `plugins:write` only; it does not need any
  other scope.
- Set an expiration on the token and rotate it before expiry, or rotate it
  immediately if it may have been exposed (for example, printed in a CI log).
- Store `GRAFANA_ACCESS_POLICY_TOKEN` as a CI/CD secret if signing is
  automated internally; the project's own `.github/workflows/ci.yml`
  intentionally does not run `make sign`, since the signing mode (private vs.
  eventual Catalog) is an organizational decision, not a build-pipeline
  default.
