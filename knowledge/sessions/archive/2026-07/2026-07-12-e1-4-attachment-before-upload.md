# 2026-07-12 Session Handoff — E1.4 attachment.before_upload

## Changed

- Catalog: `attachment.before_upload` (`validate`, fail_closed, 2000ms)
  - Payload: `actorUserId`, `contentType`, `sizeBytes`, `filename`
  - Patch allowlist: empty (reject-only)
- Attachments `storePreparedUpload`: invoke after host inspect/MIME/size,
  before storage `Put` and metadata create
- Shared by `Upload`, `UploadAvatar`, `UploadSEOImage`
- Attachments controller: `RejectedError` → HTTP 422 with stable `reason`
- Tests: allow path emits before+uploaded; reject skips storage/DB; host
  invalid type skips plugin; no raw body in payload
- Docs: regenerated catalogs; authoring guide scenario row
- Plan: E1.4 done; E1 exit criteria (≥4 sync hooks) satisfied

## Decisions

- Gate lives in `storePreparedUpload` so all upload entry points share one
  policy point without duplicating hooks
- Uses sniffed `contentType` from host inspect (not client-claimed MIME)
- No raw file bytes in RPC (metadata stage only)

## Next

1. Optional **E1.5** comment/topic delete-or-hide observe audit
2. **E2** public contribution points
3. Product fork: **E6** storage provider plugin slot (north star)

## Open Questions

- Whether avatar/SEO uploads need a purpose field in the payload later
  (`purpose: avatar|seo|general`) for finer plugin policy
