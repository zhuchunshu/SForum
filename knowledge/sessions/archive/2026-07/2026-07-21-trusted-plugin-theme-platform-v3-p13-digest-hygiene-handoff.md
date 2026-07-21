# 2026-07-21 Session Handoff — V3 P13 residual hygiene

## Status

- P0–P12 complete. P13 implementable work still closed at **~99.7%**.
- This session closed non-LTS residual hygiene only.

## Commits

- `d9e9a1aa1` chore(smtp): refresh backend package digest after rebuild
- `9ef32ae89` chore(scripts): refresh digests for all built-in plugins
- `a3284bcba` test(cli): expect Manifest V3 contract for builtin smtp validate

## Verified

- `extension test` for smtp / storage-fs / content-policy: PASS
- `go test ./cmd/sforum/`: PASS
- `go test ./app/Support/APILTS/`: PASS
- `extension api-lts`: CanRemoveWindow false for protocol.v1 and request-time-loader

## Still open (policy)

1. request-time loader residual deletion
2. Protocol V1 path deletion
3. Compatibility path removal

Do **not** claim V3 100% until RemoveAfter + zero-shim + checklist 1–7.

## Exact next

Wait for LTS window. Optional non-credit evidence polish only (mobile viewport
matrix, warm SSR re-sample). No further task-book implementation rows.
