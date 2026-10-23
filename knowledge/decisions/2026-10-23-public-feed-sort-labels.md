# Decision: Public Feed Sort Labels And No Silent `latest` Fallback

## Status

Accepted

## Date

2026-10-23

## Context

The homepage feed defaults to `active` (last activity), matching
`decisions/2026-10-03-public-list-default-sort-active.md`. The public control
still labelled the two time-based options as 最新 (`active`) and 最新帖子
(`latest`). The pair reads as two spellings of the same thing, so an explicit
`?sort=latest` URL looked like "the default" and the site default was hard to
explain. A `/?sort=latest` request reproduces that state exactly; the no-param
homepage highlights 最新 (`active`) and `GET /topics` without `sort` matches
`sort=active`.

Separately, `Service.ListTopics` resolved an empty or invalid sort to `latest`
when the site option was empty or unrecognized. That contradicted the
recommended default and could put the unindexed creation-order sort on the
public feed's hot path.

## Decision

- Public and admin labels use one vocabulary: 最近活跃 (`active`) / 最新发表
  (`latest`) / 热门 (`hot`). i18n keys are unchanged; only values change.
- The admin "default list sort" select lists `active` first and marks it
  `（推荐）`, with help text explaining that new replies bump a topic and that
  `active` matches the public activity index.
- `active` stays the recommended default. Existing installations keep their
  stored value; operators change it in admin or via "restore recommended".
  Code does not rewrite a stored operator choice.
- Sort resolution is: explicit valid value > site default > `active`. Empty or
  invalid values never fall back to `latest` on their own; `latest` only wins
  when an operator explicitly stored it as the site default.
- URL semantics are unchanged: an empty `sort` means "site default"; an
  explicit `?sort=...` stays shareable and the control highlights it truthfully.

## Consequences

- The control no longer contains two near-synonymous "latest" labels. The word
  最新 survives only in 最新发表 (creation order).
- No API, OpenAPI, cursor, or schema change. The only behavior change is for
  invalid or empty sort resolution, which now follows the site default.
- `latest` remains supported, but the public feed has no
  `(is_pinned, created_at, id)` index. If an installation makes `latest` the
  default, add that index instead of reordering in the client.
