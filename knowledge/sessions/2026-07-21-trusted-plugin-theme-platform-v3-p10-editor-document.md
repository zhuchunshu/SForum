# 2026-07-21 Session Handoff — P10 Editor Document Pipeline (8/15)

**Branch:** `main` only.
**Overall:** display **79.0%**. **P10:** **8/15**.

## Commit

- `2cc754041` feat(editor): add Host editor document pipeline and storage triple

## Closed

- Storage version `sforum.editor-document@1`
- Accept pipeline: parse/validate/normalize/render/sanitize + markdown/plain/search/excerpt
- Ordered stages including embed/SEO extension points
- Tests: XSS corpus, URL strip, unsupported fallback, disabled plugin, migration

## Next

1. Entity Type / Taxonomy / Field Schema contracts
2. Plugin-extend-plugin content/entity versioned dependencies
3. Attachment read/write policy + rich-content XSS product binding
4. Reference blocks + media plugin proofs

## Dirty WIP — DO NOT STAGE

route-inspector web, content-policy, PageViewModels, go.mod, host-api-v2,
websocket revoke test, ADR noise.

## Verification

```bash
cd apps/api && go test ./app/Support/EditorDocument/ -count=1
```
