# 2026-07-16 Trusted Plugin And Theme Platform V3 P6 Credit Correction

## Progress

- Exact weighted progress is **62.0303%**; display **62.0%**.
- P6 is **12/18**, P7 **14/22**, P8 **18/18**, and P9 **4/16**.
- P10-P13 and Program Definition of Done remain uncredited.

## Corrected Evidence

- The production application creates the Registry-backed dispatcher in
  `apps/api/bootstrap/app.go`, but `apps/api/app/Http/server.go` attaches its
  middleware only to the `/api/v1` Fiber group. Declarations for arbitrary
  public/admin root paths therefore cannot reach it in the real HTTP topology.
- Route schemas are compiled and enforced, but no route Manifest, Registry,
  Protocol V2, or OpenAPI field freezes the mutable request/response fields for
  filter contributions. The existing P6 matrix test explicitly excludes this
  behavior from its proof.
- Task-book rows 487-488 and 495 were reopened. Route Inspector credit remains
  accepted, so the conservative P6 count is 12 of 18 rather than 14 of 18.

## Decisions

- Completion accounting follows production reachability and explicit contract
  evidence, not disconnected unit fixtures.
- Repair arbitrary-path mounting without making Host-owned Safe Mode and CLI
  recovery overridable. Preserve CSRF, bearer, session, locale, and maintenance
  ordering for every route class.
- Mutable-field policy must be exact-artifact declaration-bound and enforced on
  the Host side after every filter result; SDK self-validation is not authority.

## Active Work

- P12 owns `Support/Extensions` runtime-set manager/protocol/full-set files and
  `Support/HostAPI/service_runtime_set*`.
- Query owns `Support/QueryRegistry` and `Support/HostAPI/query_registry_*`.
- Component, Navigation, Content, and Cache remain separate uncommitted families
  and must be reviewed, tested, and committed independently.
- External Grok/Codex CLI output is untrusted until the main agent checks exact
  file scope, diff, normal/race/vet results, and product contracts.

## Preserve

- Never stage `apps/api/app/Models/PageViewModels/source_test.go`.
- Never stage
  `extensions/builtin/plugins/sforum-content-policy/sforum.extension.json`.
- Do not push, tag, open a PR, create a branch/worktree, reset, clean, or rewrite
  history.

## Next

1. Land P12 and Query in independent, buildable commits.
2. Repair arbitrary route mounting with real `NewApp` public/admin/API tests.
3. Add Manifest/OpenAPI/Registry/Protocol mutable-field contracts and Host-side
   drift rejection.
4. Continue action, authority, SEO, and complete-matrix P6 exits.
