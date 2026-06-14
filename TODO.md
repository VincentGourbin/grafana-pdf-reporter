# TODO

Items left from the initial development sprint. Plugin is **functional and
private-signed** ; the work below is about making it publishable on the public
[Grafana Catalog](https://grafana.com/grafana/plugins/).

## Going public (community-signed plugin)

### 1. Repo housekeeping

- [ ] Flip repo visibility to **public** (Settings → Change visibility).
- [ ] Add **LICENSE** (Apache-2.0 — aligned with most Grafana plugins).
- [ ] Write **CHANGELOG.md** following [Keep a Changelog](https://keepachangelog.com/).
- [ ] Tag a real semver release: `git tag v0.1.0 && git push --tags`.

### 2. README to Grafana standards

- [ ] Hero **GIF or short MP4** (≤10s) showing the export flow — required by reviewers.
- [ ] 3-5 PNG screenshots in `src/img/screenshots/` (sidebar entry, settings page, cover preview, sample PDF page).
- [ ] Configuration section that documents each Settings field, defaults, and impact.
- [ ] Compatibility matrix (Grafana versions / Go versions tested).
- [ ] Troubleshooting section (the OPERATIONS.md content from `optimorin-jetson` can be lifted).

### 3. plugin.json fully filled

- [ ] `info.author.name`, `info.author.url`.
- [ ] `info.links`: homepage, docs, repo, bug tracker.
- [ ] `info.screenshots`: array of `{ name, path }` matching `src/img/screenshots/`.
- [ ] `info.version`: actual semver, not `dev`.
- [ ] `info.updated`: ISO date.

### 4. GitHub Actions CI

- [ ] Workflow: on push tag → `make all` → produce signed `dist/` → zip artifact `vincentgourbin-pdfreporter-app-vX.Y.Z.zip`.
- [ ] Run `npx @grafana/plugin-validator dist/` and fail on any error.
- [ ] Bonus: E2E with `@grafana/plugin-e2e` (Playwright) against an ephemeral Grafana container.
- [ ] Secret `GRAFANA_ACCESS_POLICY_TOKEN` stored in Actions secrets for the sign step.

### 5. Submission

- [ ] Open https://grafana.com → Plugins → **Submit a plugin**.
- [ ] Upload the signed zip + provide the public repo URL.
- [ ] Expect 2–4 weeks for human review with 3–5 round trips on:
      security, perf, UI/UX, code quality, Grafana plugin guideline conformance.

### 6. Post-approval

- [ ] Plugin lands in **Administration → Plugins → All** for every Grafana install.
- [ ] Subsequent releases go through a lighter mini-review.

## Lighter alternative — open-source without certification

If the official Catalog isn't worth the cycles right now, just doing:

- [ ] Repo public + LICENSE + CHANGELOG + tag + minimal README

is enough for any user to clone, build, and self-sign. The plugin stays
out of the Catalog but is usable by anyone who wants it. Most plugins start
this way.

## Beyond publication (nice-to-haves)

- [ ] Persist user **presets** (saved selections of dashboards + period) so a
      report definition can be re-played in one click — backend storage in
      Plugin `jsonData` keyed by user ID, or a small KV via the plugin
      storage API.
- [ ] Schedule recurring exports (daily/weekly) and deliver via email or
      Telegram — would integrate naturally with the existing Grafana
      alerting contact points.
- [ ] PDF cover with **dashboard-level variables** rendered as a table
      (the values picked at export time).
- [ ] Image-gen integration: a single click to call a configured DALL-E /
      Replicate / Imagen endpoint with the existing prompt template, then
      auto-fill the background. Requires an API key field in
      `secureJsonData`.

## Open ideas worth exploring later

- Bundling beyond 25 dashboards (current hard cap) — chunked rendering with
  a progress stream over SSE.
- Per-dashboard time range overrides inside a single bundle (currently the
  whole bundle shares one period).
- Light-theme tuning of the cover (the current accent palette was designed
  for dark first).
