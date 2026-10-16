# Runbook: Verify A Theme Switch Updates The Public Shell Without Rebuilding Nuxt

- Date: 2026-10-16
- Applies to: Route A (theme-authored shell) layer 6, M2 task 7, M10 row 1
- Companion check: `node tests/registry-page-shell-runtime-evidence.mjs`

This runbook is the reproducible form of the manual theme-switch verification that
was performed once by hand. It proves the claim that switching the active theme
changes the public shell **without** rebuilding the Nuxt app, and that the number
of visible chrome owners does not change.

## Preconditions

- Dev stack up: web on `:3000` (operator-owned), API on `:8080` (air-managed).
- An installed probe extension with a `site` and a `bare` public page. The probe
  `acme.iconprobe` provides `/probe-site`, `/probe-default` (no layout declared)
  and `/probe-bare`.
- A theme that declares a shell template (`theme.json.shell`) and a theme that does
  not. `sforum.default-theme` declares one; `extensions/fixtures/themes/plugin-shell-demo`
  is the fixture used by Go tests.

## Steps

1. Record the baseline. The script prints one line per page plus the active theme
   identity, and fails on any mismatch:

   ```bash
   node tests/registry-page-shell-runtime-evidence.mjs
   ```

   Expected with a shell-declaring active theme: `/probe-site` and
   `/probe-default` report `shell=theme`, `navbarOwners=1`, `footers=1`;
   `/probe-bare` reports `shell=none`, `navbarOwners=0`, `footers=0`; the active
   theme line shows the provider and the digest that both identity endpoints
   report.

2. Switch the active theme to one that declares **no** shell (admin flow; the
   endpoint is `POST /api/v1/admin/extensions/:id/activate` with the exact-artifact
   trust confirmation the admin UI performs). If the theme source was edited,
   rebuild and re-stage first — source edits alone do not update the active
   artifact:

   ```bash
   ./scripts/build-builtin-plugins.sh   # rsyncs themes too; only plugins are compiled
   # then restart the API so SyncBuiltins stages the exact digest
   ```

3. Re-run the script. Expected difference: the `site` pages now report
   `shell=host` instead of `shell=theme`, the **chrome counts stay identical**
   (`navbarOwners=1`, `footers=1`), `/probe-bare` stays `none/0/0`, and the active
   theme identity line shows the other provider and digest. Nuxt is not rebuilt
   between steps 1 and 3: the shell decision is data on the resolve response, not a
   build-time artifact.

4. Switch back to the original theme and re-run the script; it must return to the
   step-1 output. If it does not, the switch left state behind — that is a finding,
   not a flake.

## What each signal proves

| Signal | Meaning |
| --- | --- |
| `data-registry-shell-source` = `theme` / `host` / `none` | Which branch of `SFRegistryPageShell` rendered: theme shell, host fallback, or explicit bare. |
| `data-registry-shell` = `site` / `bare` | The contribution's layout, independent of which shell source rendered it. |
| `navbarOwners` and `footers` | Exactly one chrome owner per render path: the desktop and mobile nav are one navbar component's responsive pair (exactly one `<nav>` each), and there is exactly one `sf-footer`. Bare has none. |
| `selectedProvider` / `selectedPackageDigest` | The extension artifact that owns the page. |
| `shellProvider` / `shellPackageDigest` vs `/api/v1/site/active-theme/skin` | The two identity endpoints must agree on the active theme artifact; the script asserts agreement. |

## Known limits of this runbook

- It is a **narrow-viewport-free** check: it reads SSR HTML and the resolve API. It
  cannot prove visual layout, so it does not replace the mobile-width browser QA.
  The browser QA at 390x844 is still carrier-blocked (see the M10 matrix); the
  desktop pass and a real browser pass at the carrier's 500px width were both done.
- Step 2 mutates shared dev state (the active theme). Always run step 4. On
  2026-10-16 the manual run of this sequence showed `source: theme → host` with
  unchanged chrome counts and was reverted successfully.
