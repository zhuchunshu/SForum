# 2026-07-22 Site Search CJK, Fuzzy And Index Repair Handoff

## Changed

- Default PostgreSQL site search now combines weighted `simple` FTS, Unicode
  Han unigram/bigram `cjk_tsv` for title/excerpt/body, and indexed `pg_trgm`
  title/excerpt fuzzy matching.
- Ranking is relevance-first with exact-title and strict-title similarity
  boosts, then pinned/activity/topic-ID tie-breakers.
- Migration `202607220050` adds generated `fuzzy_text` and its GIN trigram
  index. The migrator installs `pg_trgm` in Host-owned
  `sforum_host_extensions`, preserving the Core ownership boundary.
- Follow-up migration `202607220051` adds and backfills weighted
  `metadata_tsv` for author, category, tags, and topic slug. Chinese metadata is
  also written to `cjk_tsv` by the indexer.
- Search index/delete jobs now deduplicate only in active River states, and
  worker engine failures return retryable errors. Full reindex deliberately
  bypasses uniqueness so historical completed rows cannot suppress repair.
- Docker development DB completed a real 57-topic rebuild. All 57 public
  topics now have search documents; the previous seven missing topics (51, 52,
  53, 54, 56, 59, 60) were repaired.
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
- Treat full rebuild as an explicit repair operation: it must enqueue every
  public topic, even when an old unique River job exists for the same ID.

## Verification

- Real PostgreSQL search integration: Chinese single-character, infix, body,
  typo tolerance, author/category/tag metadata, ACL/delete, and relevance
  ordering passed.
- Real River integration proves a completed incremental job permits a new job
  while a simultaneous active job still deduplicates; unit coverage proves
  engine-unavailable work returns an error for retry.
- Fresh database migration and repeat migration passed; `pg_trgm` and its schema
  are owned by the migration login, not the constrained Core owner.
- GIN query plans cover FTS, CJK, metadata, literal contains, and trigram
  operators. `小明` metadata lookup uses
  `search_documents_metadata_tsv_idx`.
- OpenAPI refs, Nuxt typecheck, homepage/search unit tests, focused Bootstrap,
  migrator, and Forum controller tests passed.
- API: `小明` returns topic 60; previously missing topics 53/54 are searchable;
  typo query `分享一断` returns repaired topic 59 first.
- Browser: typing `小明` in the home search box and pressing Enter renders
  `大家好，我是小明`; no console errors.
- Full `go test ./...` passed after rerunning the process-list test outside the
  restricted sandbox (the initial failure was only `/bin/ps` permission).

## Next

- API development server is running at `http://127.0.0.1:8081` with the
  embedded worker. The user-owned Nuxt server remains at port 3000.

## Open Questions

- None for site search. Operators needing dictionary-aware segmentation,
  synonyms, or broader typo tolerance can select the optional Meilisearch
  provider and reindex.
