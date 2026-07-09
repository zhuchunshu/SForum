# Unicode Tag Slugs

Date: 2026-07-10

## Context

Chinese-language operators expect to create and use tags such as `中文标签`
from the topic composer. The initial implementation treated `tagSlugs` like
ASCII URL slugs and rejected non-ASCII characters in both frontend and backend
validation.

## Decision

Tag slugs accept Unicode letters/numbers plus hyphens. Public tag URLs continue
to use `encodeURIComponent` on the frontend and normal URL decoding in the
backend query path. Category slugs, default category settings, and topic slugs
keep their existing ASCII-oriented rules.

## Consequences

- Chinese tags can be created through `review` or `open` tag creation modes and
  can filter public pages through encoded `/tags/:tagSlug` URLs.
- Existing ASCII tags remain valid and continue to normalize to lowercase.
- No new transliteration dependency is needed, and tag display names remain the
  tag value for composer-created tags until a separate tag-name input is added.
