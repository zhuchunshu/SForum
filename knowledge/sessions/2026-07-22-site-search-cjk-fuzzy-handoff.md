# 2026-07-22 Site Search CJK And Fuzzy Handoff

## Changed

- Default PostgreSQL site search now combines weighted `simple` FTS, Unicode
  Han unigram/bigram `cjk_tsv` for title/excerpt/body, and indexed `pg_trgm`
  title/excerpt fuzzy matching.
- Ranking is relevance-first with exact-title and strict-title similarity
  boosts, then pinned/activity/topic-ID tie-breakers.
- Migration `202607220050` adds generated `fuzzy_text` and its GIN trigram
  index. The migrator installs `pg_trgm` in Host-owned
  `sforum_host_extensions`, preserving the Core ownership boundary.
- Docker development DB migrated and 78 existing search documents reindexed;
  every document containing Han text has a non-empty CJK vector.
- OpenAPI and search frontend comments now describe the selected provider and
  default Chinese/fuzzy behavior accurately.

## Decisions

- `decisions/2026-07-22-default-site-search-cjk-fuzzy.md` records the survey and
  choice of built-in n-grams plus official PostgreSQL `pg_trgm` over a required
  native tokenizer or external service.
- Keep trigram matching on title/excerpt only to bound index size; CJK body
  matching uses the compact tsvector path.
- Keep PostgreSQL's default word-similarity threshold and improve precision in
  ranking instead of globally lowering the candidate threshold.

## Verification

- Real PostgreSQL search integration: Chinese single-character, infix, body,
  typo tolerance, ACL/delete, and relevance ordering passed.
- Fresh database migration and repeat migration passed; `pg_trgm` and its schema
  are owned by the migration login, not the constrained Core owner.
- GIN query plans cover FTS, CJK, literal contains, and trigram operators.
- OpenAPI refs, Nuxt typecheck, homepage/search unit tests, focused Bootstrap,
  migrator, and Forum controller tests passed.
- Browser: `/?q=分享一断` ranked all `分享一段...` results before `分享一份...`;
  opening the first result reached the correct topic with no console errors.

## Next

- The full repository gate is blocked outside search by missing reviewed stable
  identity mapping for `apps/web/app/components/SFAdminThemeActivateDialog.vue`
  in the V3 catalog generator. Resolve in its owning workstream, then rerun
  `./scripts/test.sh`.

## Open Questions

- None for site search. Operators needing dictionary-aware segmentation,
  synonyms, or broader typo tolerance can select the optional Meilisearch
  provider and reindex.
