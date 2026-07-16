# 2026-07-16 Trusted Plugin And Theme Platform V3 Runtime Ownership Checkpoint

## Status

- Weighted progress is **62.9%** (`62.8586%` exact). P6 is 13/18, P7 is
  14/22, P8 is 18/18, P9 is 4/16, and P12 has its first credited row.
- The long-running goal remains active on `main`. P0-P13, all 99 authoritative
  rows, 14 final boundaries, reference packages, and final gates are not complete.

## Changed

- `04b159441` adds concurrent-safe exact Theme genesis publication.
- `873e48248` adds independent Theme heartbeat, revision catch-up, exact ack
  validation, lease-loss cancellation, and process-local fail-closed admission.
- `d46fd3597` gives the API process bounded Theme ownership, merges plugin/Theme
  terminal failures, restores the protected default before reporting, and drains
  HTTP before dependency shutdown.

## Verification

- Extensions/bootstrap/cmd focused normal and race passed.
- Extensions/bootstrap/cmd vet and `go build ./...` passed.
- Theme genesis real PostgreSQL normal and race passed, including 32 concurrent
  producers and a concurrent activation.
- Theme watcher real PostgreSQL convergence passed three times; focused real
  PostgreSQL race also passed.
- Stubborn notification shutdown, heartbeat error/deadline, in-flight apply
  cancellation, Begin/Complete/Fail lease loss, initial N/N+1 catch-up, invalid
  genesis, in-flight activation fallback, and permanent admission closure pass.

## Decisions

- P12 receives only task-row 1 credit. The already checked broken-system-extension
  recovery test remains P1 evidence and is not double-counted.
- Any Theme heartbeat uncertainty is terminal. Continuing without durable lease
  ownership could publish local state the boot is no longer authorized to attest.
- Terminal fallback and normal Safe Mode restoration use the same lock order as
  activation/apply. A failed process cannot reactivate a non-default theme.
- Grok/Codex CLI repository delegation remains blocked by the managed private-repo
  disclosure policy; do not retry or route private diffs around it.

## Next

1. Commit Identity migration 033 independently, then durable root/leaf publication
   and production bootstrap after PostgreSQL normal/race review.
2. Repair Cache SDK hard wait, token redaction/cleanup, revision propagation, and
   convenience helpers; do not credit the P11 Cache row before all operations and
   failure evidence close.
3. Continue SEO provider transport and Host final policy, then SSR/sitemap and
   JavaScript-disabled failure evidence.
4. Resume P6 only after the user confirms the five recommended route boundaries.

## Preserve

- Never stage `apps/api/app/Models/PageViewModels/source_test.go`.
- Never stage
  `extensions/builtin/plugins/sforum-content-policy/sforum.extension.json`.
- Identity, Cache, SEO, generated docs, and the two user-owned files have separate
  ownership. The primary agent alone stages and commits reviewed file sets.
