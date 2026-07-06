# Forum Taxonomy And Tag Policy

Date: 2026-07-07

## Context

SForum is an open-source forum framework for different operators, not one
hard-coded deployment. New sites need safe defaults on first run, while mature
operators need to rename sections, change tag policy, hide public tag pages,
and reset configuration without editing code.

The first real forum taxonomy slice needed to support public navigation,
topic filtering, admin management, OpenAPI contracts, permission checks, and a
future search document shape without implementing full search indexing or
role-scoped category access yet.

## Decision

- Core owns two-level taxonomy: `category_groups` contain `categories`.
- Category visibility v1 is only `public` or `hidden`.
- Core owns reusable tags with `active`, `pending`, and `disabled` statuses.
- Tag creation policy is configurable through runtime settings:
  `controlled`, `review`, or `open`.
- Public tag pages are controlled by `forum.tags.public_pages` and can be
  disabled by operators.
- Recommended defaults remain `general`, controlled tags, public tag pages,
  and five tags per topic, but services resolve behavior from settings instead
  of hard-coding deployment assumptions.
- Admin taxonomy management is permission aware: categories require
  `category.manage`, tags require `tag.manage`, and shared forum settings allow
  either permission with owned-value checks.

## Consequences

- SForum now has stable core category and tag primitives for public pages,
  admin management, topic filters, and future search indexing.
- Role-scoped category permissions, tag merge history, taxonomy moderation
  queues, sitemap/search indexing, and abuse controls remain follow-up work.
- Plugins must use explicit events, settings, provider slots, or future SDK
  helpers. They must not override core forum routes, mutate core taxonomy table
  semantics directly, or bypass API policy checks.
- Future Meilisearch documents should include category group/category fields
  and active tag summaries, but full indexing and rebuild jobs are separate
  search work.
