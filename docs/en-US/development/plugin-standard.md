# Plugin repository standard

A SForum plugin ships in one of two carriers: an in-repository package
(`extensions/builtin`, `extensions/optional`) or an independent repository used
as an external extension source root. Both share the same package contract; only
the repository-level files differ. This page answers two questions: **what a
plugin must contain**, and **where each piece lives**.

For the package contract itself (Manifest V3 fields, trust, Host API) see the
[plugin authoring guide](../../extensions/authoring-guide.md); for brand asset
sizes and validation see [brand assets](../../extensions/brand-assets.md).

## 1. Independent repository skeleton

```text
sforum-<name>/
├── README.md                  Operator-facing: problem, install, configuration
├── AGENTS.md                  Collaborator/AI entry: layout, dev loop, gates, host constraints
├── NOTES.md                   Design tradeoffs and measured findings
├── CHANGELOG.md               Release-by-release changes
├── LICENSE
├── plugins/<id>/              The package; exactly one copy per repository
│   ├── sforum.extension.json
│   ├── manifest/{settings.json,langs/}
│   ├── backend/               Required when the plugin has a backend
│   ├── brand/                 Required, see section 4
│   └── schemas/ frontend/ templates/ openapi/   As needed
├── scripts/prepare-package.sh Build + refresh digests + every gate
└── .github/workflows/{ci,release}.yml
```

| File | Required | Purpose |
| --- | --- | --- |
| `README.md` | Yes | Operator view. Social card first, then what problem it solves, install steps, tunable boundaries |
| `AGENTS.md` | Yes | First stop for collaborators and AI. Must state the extension id, package path, dev loop, verification standard, host constraints |
| `NOTES.md` | Recommended | Tradeoffs and measured numbers (timeouts, truncation, rate limits). Host constraints belong here, not only in a chat log |
| `CHANGELOG.md` | Yes | Every release; breaking changes called out explicitly |
| `LICENSE` | Yes | Full license text |
| `plugins/<id>/` | Yes | Package directory; one extension id may appear in one source only |
| `scripts/prepare-package.sh` | Yes | One command: build → digest → validate → brand check → test |
| `.github/workflows/` | Yes | CI and release; both call `go run ./cmd/sforum` from a SForum checkout |

## 2. Package skeleton

| Path | Required | Notes |
| --- | --- | --- |
| `sforum.extension.json` | Yes | Explicit `manifestVersion: 3`; `packageFiles` digests match the files |
| `manifest/settings.json` + `manifest/langs/` | When settings exist | Referenced through `includes`, never inlined into the main manifest |
| `backend/` | When a backend exists | Static build (`CGO_ENABLED=0`); entry matches the `executable` package file |
| `brand/icon.svg`, `brand/icon.png`, `brand/social-preview.png` | Yes | See section 4 |
| `schemas/` | When request/response contracts exist | Same source as `openapi` declarations |
| `frontend/` | When prebuilt frontend exists | Build output only; `extension build` owns the build |
| `templates/` | Themes and Page Registry pages | Path must agree across `theme.json`, `templates[]`, and `packageFiles[]` |
| `README.md` | Yes | In-repo packages need one too: GitHub opens it on its own |

## 3. Things that must exist and must stay paired

- Manifest declarations and `packageFiles` digests: any file change requires
  `extension digest --write`, or the package is refused at enable time.
- Permission keys and recommended role mappings: plugins declare, the host grants.
- Recommended defaults plus one-click reset for every operator-facing setting.
- Brand assets and the README first screen: see section 4.

## 4. Brand assets

Every shipped package carries `brand/`:

| File | Size | Budget |
| --- | --- | --- |
| `brand/icon.svg` | `viewBox="0 0 512 512"`, self-contained | 40 KiB |
| `brand/icon.png` | 512×512 PNG | 120 KiB |
| `brand/social-preview.png` | 1280×640 PNG (Chinese primary card) | 600 KiB |

Generate and validate:

```bash
# Flags at a glance
node scripts/brand/generate.mjs --help
# Preview (writes nothing into the repo) plus contact sheets
node scripts/brand/generate.mjs --draft /tmp/sforum-brand --sheet /tmp/sforum-brand
# Regenerate one package (matched by package-directory suffix)
node scripts/brand/generate.mjs --only sforum-smtp
# Write
node scripts/brand/generate.mjs
# Validate (independent repositories call the host CLI the same way)
cd <SForum>/apps/api && go run ./cmd/sforum extension brand check <package-path>
```

`brand/` stays out of `packageFiles` (branding is not a runtime artifact), but it
is part of the package tree and therefore changes the artifact digest. That is a
normal artifact change; do not bump a version just to silence the drift. Full
rules, template parameters, and the GitHub manual upload step live in
[brand assets](../../extensions/brand-assets.md).

## 5. Gates

| Where | What runs |
| --- | --- |
| Plugin repository, local | `scripts/prepare-package.sh`: build → `extension digest --write` → `extension validate` → `extension brand check` → `extension test` |
| Plugin repository, CI | The same chain against a SForum checkout pinned by `SFORUM_REF`; the release workflow adds `extension package --exclude-source` |
| SForum repository | `go test ./...` (includes the brand asset contract), `node tests/validate-docs.mjs`, `node tests/validate-architecture-boundaries.mjs`, `./scripts/test.sh --fast` |

The order is deliberate: `brand check` runs after `validate` and before `test`,
so a missing image fails at the contract stage instead of at release. The host
CLI is the only entry point — do not re-implement a private checker inside a
plugin repository.

## 6. Records and knowledge base

- **Inside SForum**: `knowledge/` is project memory. Changes to trust, digests,
  permissions, data surfaces, or public UI contracts require a record under
  `knowledge/decisions/`; feature-area changes update `knowledge/modules/`; each
  workstream ends with a hot handoff under `knowledge/sessions/`, pointed at by
  `knowledge/index.md`.
- **Inside a plugin repository**: do not clone SForum's knowledge base layout.
  Host constraints and measured findings go to `NOTES.md`, collaborator rules to
  `AGENTS.md`, releases to `CHANGELOG.md`. Cross-repository architecture
  decisions stay in SForum's decision records; link to them.

## 7. New plugin checklist

1. Scaffold with `make:plugin`, fix the extension id (`sforum.*`) and type.
2. Write `sforum.extension.json`: `manifestVersion: 3`, plus capabilities,
   settings, events, routes, and permissions as needed.
3. Ship recommended defaults and a reset path with every setting.
4. Add an entry to `scripts/brand/spec.json`, generate and validate the three
   `brand/` files.
5. Put the social card on the README first screen, then install, configuration,
   and tunable boundaries.
6. Write the dev loop and verification standard into `AGENTS.md`; record
   tradeoffs and measurements in `NOTES.md`.
7. Run `prepare-package.sh`, then install → trust → enable in the admin.
8. When a contract surface changes, add the decision record in SForum.
