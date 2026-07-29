# Decision: Attachment Storage as a Plugin Provider Slot

## Status

Accepted; multi-instance selection, Host SecretStore ownership, and the removal
of FTP/SFTP built-ins are superseded by
`2026-07-30-multi-instance-s3-storage.md`.

## Context

Attachment object storage today is **L1–partial L2** on the provider maturity
ladder (`knowledge/plans/2026-07-12-extension-surface-density.md`):

- Host slot name: `attachment.storage.provider` (`Support/Storage.ProviderSlot`)
- Concrete drivers live **in core** under `Support/Storage` (`local`,
  `aliyun_oss`, `tencent_cos`, `ftp`, `sftp`)
- Operators select via `web_options.attachment.provider`; admin supports
  select / restore defaults / probe
- F3.5 chose “drivers stay in core; slot name reserved” — good for zero-config,
  insufficient for third-party S3/MinIO/R2 without a core PR

Mail already proves the end-to-end pattern (**L4–L6**):

- Slot `mail.provider`
- Plugin declares `providers: [{ slot, label, … }]`
- Host routes work through go-plugin RPC (`SendMail`)
- Secrets live in `extension_settings`; admin select / settings / test / restore
- Reference: protected builtin `sforum.smtp`

Product north star: storage reaches the same operator loop. E5
(`sforum.content-policy`) is the **workflow** reference; storage is the next
**service** vertical after mail.

Related:

- `knowledge/decisions/2026-07-05-attachment-storage-providers.md` (core
  `Adapter` + first driver set — **still valid** for in-tree drivers)
- `knowledge/decisions/2026-07-07-mail-provider-contract.md` (mail shape)
- `knowledge/modules/attachments.md`, `knowledge/modules/mail.md`
- F3.5 note in `plans/archive/2026-07/2026-07-12-framework-hardening-waves.md` — **superseded
  for future drivers** by this decision (core may keep existing drivers)

## Decision

### 1. Slot and maturity target

- Slot id remains **`attachment.storage.provider`** (stable; do not rename).
- Target maturity: **L4–L6** — plugin registration, host routes Put/Open/Delete
  (and Probe) through plugin RPC when selected, plugin-owned settings/secrets,
  admin test connection, reference plugin + authoring docs.
- Core keeps at least **`local`** as permanent zero-config L1 fallback and
  one-click restore target.

### 2. Host interface boundary (what business code sees)

Domain code (`Models/Attachments`) continues to depend only on
**`storage.Adapter`** (Put / Open / Delete / Stat / Exists / PublicURL /
SignedURL / Probe). It must **not** import go-plugin or know extension process
details.

When the selected provider is a plugin:

1. Host **resolver** maps selection → either `NewAdapter` (core driver) or a
   **plugin-backed `Adapter`** implementation owned by host support code.
2. That adapter translates `Adapter` calls into plugin RPC (E6.2).
3. Attachments service keeps using `adapterFactory` / runtime settings as today.

Sketch (E6.1 implements; types may live under `Support/Storage`):

```text
attachment.provider (web_options)
        │
        ▼
  ResolveSelection ──► core driver id ──► storage.NewAdapter(config)
                 └──► plugin extension id ──► PluginStorageAdapter
                                              (RPC via Extensions runtime)
```

### 3. Selection model (single option key)

Keep **`attachment.provider`** as the **single** operator selection key
(attachment admin already uses it). Do **not** introduce a second parallel
selection table for v1 (mail’s `mail_provider_selection` stays mail-specific).

| Value form | Meaning |
| --- | --- |
| Core driver id | Existing: `local`, `aliyun_oss`, `tencent_cos`, `ftp`, `sftp` (blank → `local`) |
| `plugin:<extensionId>` | Enabled plugin that declared `providers[].slot = attachment.storage.provider` |

Rules:

- Prefix **`plugin:`** is mandatory for plugin selection so extension ids
  never collide with core driver ids.
- Selecting a plugin requires: plugin **enabled**, declares the slot, backend
  runtime available (same spirit as mail).
- **Restore recommended defaults** → `local` + existing safe upload knobs
  (size, allow-lists, etc.); does not wipe other plugins’ `extension_settings`.
- Disabling the selected plugin: clear selection back to **`local`** (or refuse
  disable until operator switches — prefer **auto-fallback to local** + audit
  log, matching “no orphan RPC” goal).
- Candidate list (admin): core drivers from `DriverCatalog()` **plus** enabled
  plugins declaring the slot (label, extensionId, health).

Pure helpers for encoding live in `Support/Storage` as of E6.0
(`PluginSelectionPrefix`, `FormatPluginSelection`, `ParseSelection`); wiring
into options validation is **E6.1**.

### 4. What stays in core vs plugins

| In core (v1+) | In plugins |
| --- | --- |
| `local` driver (required fallback) | New vendor backends (S3/MinIO/R2, …) |
| Existing OSS/COS/FTP/SFTP drivers **until** optional E6.5 migration | Third-party / operator-specific storage |
| Upload policy (size, MIME, extension), object key templates, metadata DB | Transport credentials & vendor SDK |
| Attachment ACL / visibility / soft-delete / orphan cleanup | Optional PublicURL/SignURL string generation |
| Host-owned content API (`GET .../content`) | — |

**E6.5 (later):** optionally move OSS/COS/FTP/SFTP to builtin plugins with
compat aliases so old `attachment.provider=aliyun_oss` still works. Not required
for E6 exit.

### 5. Plugin RPC contract (sketch for E6.2)

Extend go-plugin `PluginProtocol` with storage methods (names indicative):

| RPC | Role |
| --- | --- |
| `StoragePut` (chunked) | Write object by key |
| `StorageOpen` / `StorageGetChunk` | Read object bytes |
| `StorageDelete` | Delete by key |
| `StorageStat` / `StorageExists` | Optional but recommended |
| `StoragePublicURL` / `StorageSignedURL` | Optional URL helpers |
| `StorageProbe` | Admin “test connection” |

**Streaming / size (critical difference from mail):**

- Mail payloads are small strings; storage objects can be tens of MB.
- **v1 uses chunked RPC** (recommended default chunk **1 MiB**), not a single
  base64 blob of the whole file.
- Host enforces **max object size** = configured `attachment.max_file_size_mb`
  (and avatar/SEO-specific limits on those paths) **before** and while
  streaming; plugin must reject oversize if host misbehaves.
- Host may tee the upload stream for SHA-256 (existing behavior) while
  forwarding chunks — avoid loading the entire file twice into memory when
  practical.
- **Timeouts:** longer than hook defaults; start from provider
  `timeoutMs` (manifest) with a host ceiling (e.g. 60s–120s for put). Reuse
  F2.3 circuit breaker / concurrency on the extension runtime.
- Capability: plugins that call remote APIs need **`net.outbound`**; always
  **`settings.own`** for credentials; **`host.api`** only if needed.

**Failure policy:**

- **Upload / Put:** fail **closed** when selected plugin is missing, degraded,
  circuit-open, or RPC errors — clear operator-facing error
  (`attachment.storage_unavailable` / dedicated reason codes).
- **Read / Open:** **no** automatic multi-backend fallback in v1. If the object
  was stored on a previous backend, Open fails until the operator migrates
  data or re-selects that backend.
- **Multi-backend migration tooling:** out of scope for E6 (E9 trigger).
- Browser **direct upload / presigned browser PUT:** still deferred
  (2026-07-05 decision unchanged). Server-mediated upload remains the only
  supported path.

### 6. URL and security authority

- **ACL and visibility** are host-owned. Plugins never authorize downloads.
- Preferred client path remains host APIs (`GET /attachments/:id/content` and
  metadata) after permission checks.
- Plugin `PublicURL` / `SignedURL` may return strings for public or
  time-limited access; host still decides **whether** to expose them based on
  attachment visibility and actor permissions.
- Object **keys** are host-generated (path template); plugins must not invent
  alternate key namespaces for the same attachment row.
- Secrets for plugin backends live in **`extension_settings`** (masked, blank
  keeps existing). Core driver secrets stay in existing masked `web_options`
  fields. Do not scatter plugin credentials into attachment options.

### 7. Admin / operator loop (copy mail; implement E6.3)

1. Install/enable storage plugin (capability review).
2. Attachment settings: select core driver **or** `plugin:<id>` from one list.
3. Configure plugin settings (deep-link / existing extension settings chrome).
4. **Test connection** → host `Probe` → plugin `StorageProbe`.
5. **Restore recommended defaults** → local + safe upload knobs.
6. Disable/swap → no orphan RPC; fallback to local when selection invalid.

### 8. Library survey (reference plugin only)

| Option | Verdict |
| --- | --- |
| Add AWS SDK / MinIO SDK to **core** | **Reject** for E6 — keeps core free of new vendor SDKs |
| S3-compatible client **inside reference plugin** (AWS SDK v2 or MinIO Go) | **Accept** for product-shaped reference (MinIO-friendly) |
| Thin HTTP PUT/GET fixture plugin | Accept as **CI/dev** alternative if cloud creds unavailable |
| Go CDK `blob` in core | Deferred; not required for slot design |

**Reference plugin recommendation (E6.4):**

- Primary: **S3-compatible** builtin or dev plugin (e.g. `sforum.s3` or
  `sforum.storage-s3`), MinIO-tested.
- Optional fixture: in-memory or temp-dir plugin for contract tests without
  network.

Core continues to use existing OSS/COS libraries only for **in-tree** drivers.

### 9. SDK / catalog consequences

- Catalog text for `attachment.storage.provider` becomes **plugin-implementable**
  (not “core only”).
- Public SDK (`sdk/plugin`): Noop defaults for storage RPCs (mirror `SendMail`);
  helpers after protocol lands (E6.2).
- Authoring guide: “implement a storage provider” section in E6.4.

### 10. Non-goals (E6.0–E6.4)

- Multi-backend live migration / dual-write
- Browser presigned upload credentials
- Forcing OSS/COS/FTP out of core on day one
- Plugin SQL migration executor
- Search slot (E7) — same ladder later, separate decision

## Consequences

- F3.5 “plugin RPC not yet” is **closed for the target architecture**; code
  remains L1 until E6.1+.
- Third parties can plan S3/MinIO plugins against a stable selection encoding
  and Adapter-facing host boundary without waiting for core PRs per vendor.
- Operators keep a single attachment settings mental model; mail and storage
  share the same enable → select → configure → test → restore loop.
- E6.1 can implement resolver + candidates without reopening stream/ACL
  debates; E6.2 implements chunked RPC per this note.

## Implementation order

| Slice | Deliverable |
| --- | --- |
| **E6.0** (this doc) | Decision + selection helpers + catalog wording |
| E6.1 | Resolver, candidates API, restore/fallback wiring |
| E6.2 | PluginProtocol storage RPCs + SDK Noop/helpers |
| E6.3 | Admin UI select/test/settings deep-link |
| E6.4 | Reference S3-compatible plugin + docs |
| E6.5 | Optional core-driver extraction / compat aliases |
