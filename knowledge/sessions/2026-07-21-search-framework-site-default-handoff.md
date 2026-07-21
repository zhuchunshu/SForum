# 2026-07-21 Session Handoff — Search framework + site default

## Changed

- Core search framework: default protected site search (`sforum.search-site`),
  PostgreSQL `search_documents` + `PostgresSiteEngine`, Host short-circuit.
- Meilisearch remains optional (`extensions/optional/...`, gitignored tree);
  package built to `~/Downloads/sforum-search-meilisearch-20260721.zip`.
- Decision: `knowledge/decisions/2026-07-21-search-framework-site-default.md`
  supersedes “default no engine → 503”.

## Decisions

- D1 Host short-circuit for site search; D3 cannot uninstall builtin;
  D4 no dual-write v1; D5 external engine fail → unavailable (no auto failover).

## Next

- Admin UI polish for provider select/probe if not already wired via provider slots.
- Optional: dual-write site table always; zhparser for Chinese FTS later.

## Open Questions

- None blocking.
