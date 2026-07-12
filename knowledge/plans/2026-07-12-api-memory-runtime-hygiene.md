# API Memory & Runtime Hygiene (P0–P2)

Status: **P0 done; P1/P2 cancelled (not needed for now)**  
Date: 2026-07-12  
Related: memory investigation session (dev `sforum-api` ~70–84 MB footprint;
SMTP plugin started twice under `EMBED_WORKER_IN_API=true`)

This plan is a **small, low-risk ops/runtime hygiene batch**. It is not a
product wave and does not replace extension density (E*) or framework
hardening (F*) work.

**Scope decision (2026-07-12):** Only **P0** (shared extension runtime under
embed) is in scope and landed. **P1** (tiered River worker defaults) and
**P2** (loopback pprof) are **cancelled / not planned**—keep the sections
below as historical design notes only; do not implement unless product
explicitly reopens them.

---

## Problem (measured)

| Observation | Detail |
|-------------|--------|
| API process | ~82–84 MB RSS / ~70 MB phys_footprint (idle + `/ready`) |
| Plugin children | 2× `sforum.smtp` backend plugin (~10 MB each) |
| Family total | ~103 MB |
| Binary (air) | ~57 MB, plain `go build` (no `-s -w`) |
| Root cause of double plugin | API `extensionRuntime.Reconcile` + embedded worker `newWorkerWithPool` builds a **second** runtime and reconciles again |

Idle ~70–100 MB for Go API + embed worker + go-plugin is a **normal baseline**,
not a leak. The structural waste is **duplicate plugin processes** under embed
mode; secondary fat is **default River MaxWorkers sum = 31**.

---

## Goals / Non-goals

### Goals

1. **P0 (done):** Under `EMBED_WORKER_IN_API=true`, each enabled backend plugin
   starts **at most once** in the process tree.
2. ~~**P1:** Development (and optionally small-deploy) defaults for queue
   workers are leaner…~~ — **cancelled (not needed for now)**
3. ~~**P2:** Optional, default-off pprof…~~ — **cancelled (not needed for now)**

### Non-goals

- In-process plugins (keep HashiCorp go-plugin isolation).
- Merging Redis session storage + business cache clients (optional follow-up).
- Plugin lazy-start on first hook (optional follow-up when many plugins).
- Changing Meilisearch/Postgres container memory.
- Treating ~70 MB idle RSS as a “must hit 20 MB” target.
- Broad refactors of Fiber, River, or extension protocol.

---

## Execution order

~~Recommended: P1 → P0 → P2.~~ **Superseded:** only P0 was implemented; P1/P2
cancelled.

---

## P0 — Share extension runtime when worker is embedded

### Intent

`bootstrap.NewAPI` already creates and reconciles `extensionRuntime`.
`newWorkerWithPool` currently always creates its own manager + Host API
gateway and reconciles again → **two plugin processes per extension**.

When the worker is embedded into the API process, the worker must **reuse**
the API’s extension runtime (and related Host API gateway if required for
mail/plugin jobs), not start a second stack.

Standalone `cmd/worker` / `NewWorker` must **keep** its own runtime (behavior
unchanged).

### Design constraints

1. **Ownership of Close**
   - API-owned runtime: closed only by API shutdown path.
   - Embedded worker must **not** call `extensionRuntime.Close` on the shared
     instance (or use a `ownsRuntime bool` / nil close hook).
   - Standalone worker still closes the runtime it created.
2. **Host API**
   - Worker needs Host API for plugin jobs / settings-backed plugin env.
   - Prefer injecting the API’s existing gateway/registrar into the shared
     path so plugin children still get one Host API registration story.
   - Avoid double `RegisterExtension` for the same extension id if that would
     mint conflicting credentials; share the same gateway.
3. **Reconcile**
   - Reconcile once after builtins sync (API path already does this).
   - Embedded worker must not re-Reconcile unless the shared manager is
     designed to be idempotent **and** does not spawn a second client
     (prefer skip entirely when runtime is injected).
4. **Mail / webhook workers**
   - `DeliverMailWorker.Sender` and any runtime-dependent handlers must use
     the **shared** manager so mail still hits the same plugin process.

### Likely code touch points

| Area | File(s) |
|------|---------|
| Embed assembly | `apps/api/bootstrap/app.go` (`shouldEmbedWorkerInAPI` block) |
| Worker construction | `apps/api/bootstrap/worker.go` (`newWorkerWithPool`) |
| Runtime interface | existing `extensionRuntime` in `bootstrap/app.go`; extend worker opts |
| Tests | `apps/api/bootstrap/app_test.go`, `worker_test.go` (or new focused test) |
| Docs (short) | `docs/development-and-deployment.md` note: embed shares plugin runtime |

### Implementation sketch (not prescriptive)

```text
type workerRuntimeDeps struct {
  ExtensionRuntime extensionsruntime.Manager-or-interface  // optional inject
  HostAPI          ...                                     // optional inject
  OwnsRuntime      bool                                    // close on Worker.Close?
}

// newWorkerWithPool(cfg, pool, logger, deps)
// if deps.ExtensionRuntime == nil → create + reconcile + OwnsRuntime=true (standalone)
// else → use injected, skip second Reconcile, OwnsRuntime=false (embed)
```

API embed path:

```text
embeddedWorker, err = newWorkerWithPool(cfg, pool, logger, workerRuntimeDeps{
  ExtensionRuntime: extensionRuntime,
  HostAPI:          hostAPIGateway, // or registrar already bound
  OwnsRuntime:      false,
})
```

### Acceptance criteria

- [x] With `EMBED_WORKER_IN_API=true` and one enabled backend plugin (e.g.
      `sforum.smtp`), process tree shows **one** plugin child, not two.
- [x] With embed **false**, standalone worker still starts plugins and can
      deliver mail / run extension-related jobs.
- [x] Shutdown: embed mode does not double-close / panic; API close still
      stops plugins after worker stop (order: stop River → then close runtime).
- [x] Unit/integration test: inject fake starter or count `Start` calls;
      embed path asserts single start per extension id.
- [ ] Manual: `ps` / `pgrep -fl backend/plugin` before/after on dev API.

### Risk notes

- Medium-low: Close ordering and Host API sharing are the only sharp edges.
- Do **not** change plugin protocol or go-plugin handshake.
- Keep `NewWorker` public signature stable if possible; add optional deps via
  unexported helper + thin wrappers rather than breaking call sites.

### Out of scope for P0

- Lazy plugin start.
- Deduplicating runtimes across **multiple machines** (N/A).

---

## P1 — Environment-tier defaults for River queue workers

> **Cancelled (2026-07-12):** Not needed for now. Left as design notes only.
> Do not implement unless explicitly reopened. Production defaults stay as
> today; operators can still set `JOB_QUEUE_*_WORKERS` if needed.

### Intent

Current defaults (all envs if unset):

| Queue | Default MaxWorkers |
|-------|-------------------:|
| critical | 4 |
| default | 8 |
| search | 6 |
| mail | 4 |
| notifications | 6 |
| maintenance | 2 |
| theme | 1 |
| **sum** | **31** |

For development / single-node small deploys this is fat. Env vars already
override (`JOB_QUEUE_*_WORKERS`); code should apply **tiered defaults** when
env is unset.

### Proposed defaults

| Queue | `development` default | `production` (and other non-dev) default |
|-------|----------------------:|-----------------------------------------:|
| critical | 1 | 4 |
| default | 2 | 8 |
| search | 1 | 6 |
| mail | 1 | 4 |
| notifications | 1 | 6 |
| maintenance | 1 | 2 |
| theme | 1 | 1 |
| **sum** | **8** | **31** |

Rules:

1. Explicit env always wins (`envPositiveInt` / existing loaders unchanged in
   precedence).
2. Tier is derived from `APP_ENV` (same as `EmbedWorkerInAPI` convention:
   only `development` is the lean tier unless we later add `staging`).
3. Optional later: `JOB_QUEUE_PROFILE=small|default` — **not required** in
   this batch; avoid new knobs unless needed.

### Likely code touch points

| Area | File(s) |
|------|---------|
| Config load | `apps/api/config/config.go` |
| Jobs FromAppConfig | `apps/api/app/Support/Jobs/config.go` (only if defaults move here) |
| Tests | `config/config_test.go`, `app/Support/Jobs/config_test.go` |
| Docs | `docs/development-and-deployment.md` (table of defaults by env) |

Prefer applying tier defaults in **`config.Load`** so both API insert-only
client and worker see the same numbers via `FromAppConfig`.

### Acceptance criteria

- [ ] `APP_ENV=development` without `JOB_QUEUE_*` → lean sum (8) as above.
- [ ] `APP_ENV=production` without overrides → current prod-scale defaults (31).
- [ ] Setting any `JOB_QUEUE_*_WORKERS` overrides that queue regardless of env.
- [ ] Existing tests updated; new cases for tier + override.
- [ ] Docs list both default tables.

### Risk notes

- Low: worst case is slower queue drain → raise env. No data model change.
- Do not lower production defaults in this PR without an explicit product
  decision; keep prod numbers as today.

---

## P2 — Optional restricted pprof

> **Cancelled (2026-07-12):** Not needed for now. Left as design notes only.
> Do not implement unless explicitly reopened. Diagnosis remains `ps` /
> footprint / existing logs.

### Intent

Today diagnosis is `ps` / `footprint` only. Add **opt-in** Go pprof so heap
and goroutines can be profiled when investigating memory.

### Security / safety rules (non-negotiable)

1. **Default off** (`PPROF_ENABLED` false / unset).
2. Bind **loopback only** by default (`PPROF_ADDR=127.0.0.1:6060` or similar).
3. Do **not** mount pprof on the public Fiber app routes without auth.
4. Prefer a **separate** `http.Server` on loopback over `/debug/pprof` on `:8081`.
5. Production: document “leave disabled”; if ever enabled, still loopback +
   network policy / SSH tunnel.
6. No open proxy to pprof from Nuxt or admin UI in this batch.

### Likely code touch points

| Area | File(s) |
|------|---------|
| Config | `apps/api/config/config.go` (`PPROF_ENABLED`, `PPROF_ADDR`) |
| Start/stop | `apps/api/cmd/api/main.go` and/or `bootstrap` close hook |
| Optional worker | same flags on `cmd/worker` if useful (nice-to-have) |
| Docs | `docs/development-and-deployment.md` short “Profiling” section |
| Tests | config parse tests; optional “server starts when enabled” with httptest |

### Acceptance criteria

- [ ] Default: no pprof listener.
- [ ] `PPROF_ENABLED=true` + default addr → `http://127.0.0.1:6060/debug/pprof/`
      works locally; heap profile downloadable.
- [ ] Shutdown closes pprof server cleanly with API.
- [ ] Docs: how to enable, how to `go tool pprof`, warning not for public net.
- [ ] No OpenAPI public contract for pprof (internal only).

### Risk notes

- Low if default-off + loopback.
- Reject any design that exposes pprof on `0.0.0.0` without an extra explicit
  dangerous flag (and even then, prefer not in v1).

---

## Task checklist (implementation)

Use this as the session todo list. Check off in this file when done.

### PR-A / Task group P1 — Queue worker tier defaults

> **Cancelled — do not implement.**

- [ ] ~~**T1.1** Decide final default tables…~~
- [ ] ~~**T1.2** Implement tiered defaults in `config.Load`…~~
- [ ] ~~**T1.3** Update `config_test.go` + `Jobs/config_test.go`…~~
- [ ] ~~**T1.4** Update `docs/development-and-deployment.md`…~~
- [ ] ~~**T1.5** `go test` for `./config/...` `./app/Support/Jobs/...`.~~

### PR-B / Task group P0 — Shared runtime under embed

- [x] **T0.1** Map current Close / Reconcile / Host API ownership on paper
      (API vs `newWorkerWithPool` vs standalone `NewWorker`).
- [x] **T0.2** Add inject path for extension runtime (+ Host API as needed)
      into worker construction; standalone path unchanged.
- [x] **T0.3** Wire `NewAPI` embed branch to inject API runtime; set
      `OwnsRuntime=false`.
- [x] **T0.4** Ensure mail/plugin job handlers use shared runtime.
- [x] **T0.5** Fix shutdown order: stop embedded River → API closes runtime.
- [x] **T0.6** Tests: single Start per extension when embedded; standalone
      still owns runtime.
- [ ] **T0.7** Manual verify on dev: one `backend/plugin` process for SMTP.
- [x] **T0.8** Short doc note under embed worker section.
- [x] **T0.9** `go test ./bootstrap/...` (+ any extension runtime tests).

### PR-C / Task group P2 — Restricted pprof

> **Cancelled — do not implement.**

- [ ] ~~**T2.1** Config flags `PPROF_ENABLED`, `PPROF_ADDR`…~~
- [ ] ~~**T2.2** Start optional pprof server…~~
- [ ] ~~**T2.3** Config tests; optional smoke test.~~
- [ ] ~~**T2.4** Docs: profiling section + security warnings.~~
- [ ] ~~**T2.5** Manual: enable, capture heap…~~

### Close-out (any PR)

- [x] **T9.1** Session handoff under `knowledge/sessions/`.
- [x] **T9.2** Pointer in `knowledge/index.md` Latest Handoff (when landed).
- [ ] **T9.3** Optional decision note only if Host API sharing choice is
      non-obvious (`knowledge/decisions/`).

---

## Verification matrix

| Check | P0 | P1 | P2 |
|-------|:--:|:--:|:--:|
| `go test` affected packages | ✓ | ✓ | ✓ |
| Process tree: 1 plugin under embed | ✓ | | |
| Standalone worker still works | ✓ | | |
| Dev defaults sum ≈ 8 without env | | ✓ | |
| Prod defaults unchanged without env | | ✓ | |
| Env override wins | | ✓ | |
| pprof off by default | | | ✓ |
| pprof loopback when on | | | ✓ |

Manual memory re-check after P0 (optional but recommended):

```bash
# after API up with embed + smtp enabled
pgrep -fl 'tmp/sforum-api|backend/plugin'
# expect: 1 sforum-api + 1 plugin (not 2 plugins)
```

---

## Follow-ups (explicitly deferred)

1. Share / merge Redis clients (session Fiber storage vs business cache).
2. Lazy plugin start on first invoke.
3. Production Dockerfile `-ldflags="-s -w"` (build hygiene, not runtime).
4. Admin UI memory metrics (needs pprof or runtime metrics first).
5. Lower production worker defaults for “small VPS” profile (product decision).

---

## Success definition

Batch is **done** when:

1. Embed mode no longer doubles plugin processes (**P0** — landed).
2. Knowledge base has handoff; P0 task boxes are checked.
3. ~~P1 / P2~~ — **cancelled**; not part of success criteria.

No requirement to hit a specific RSS number; expect roughly **−10–20 MB**
family RSS on current dev (one fewer plugin process).
