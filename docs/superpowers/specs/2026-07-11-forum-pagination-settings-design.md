# Forum Pagination Settings Design

## Goal

Make public forum pagination use safe, operator-configurable defaults. Topic
lists and comment lists each default to 20 items, and operators can change the
two values independently from the existing forum settings page.

This setting applies only to public forum content. Admin tables, account
sessions, attachments, extension logs, and internal batch processing keep
their existing page sizes.

## Settings

Store the following runtime options in the existing `web_options` table:

- `forum.pagination.topics_per_page`: default `20`
- `forum.pagination.comments_per_page`: default `20`

Both values accept integers from 1 through 100. Missing values resolve to 20,
so existing installations require no data backfill. Saving or restoring forum
settings persists the resolved values through the established forum settings
manager.

The existing one-click forum settings reset restores both values to 20 along
with the other recommended forum defaults. It does not alter secrets or
unrelated settings.

## Server-Authoritative Resolution

The API owns default resolution. When `perPage` is absent or less than or equal
to zero:

- public topic lists use `forum.pagination.topics_per_page`;
- Meilisearch topic results use `forum.pagination.topics_per_page`;
- public comment lists use `forum.pagination.comments_per_page`.

An explicit positive `perPage` remains a caller override and is clamped to the
existing maximum of 100. The existing topic/comment deep-page limit of 200 is
unchanged.

The forum settings resolver supplies typed pagination values to the forum and
search services. This keeps the default consistent for the built-in web theme,
plugins, and third-party API clients without making them read public options
and duplicate fallback logic.

## Frontend Behavior

The built-in default theme stops sending a hard-coded `perPage: 20` for:

- the homepage feed, including search and infinite loading;
- category topic pages;
- tag topic pages;
- topic comment pages.

Initial and subsequent requests rely on the API default. Pagination and
infinite-scroll calculations use the `perPage` returned in each API response.
This preserves SSR-first rendering and the current homepage infinite-scroll
experience while making its batch size configurable.

## Admin Experience

Add a "Pagination display" section to the existing forum settings page with
two numeric inputs:

- Topics per load
- Comments per load

Each input shows 20 as the recommended value and enforces the 1-100 range.
Field validation remains next to the input and does not auto-dismiss. A
successful save or restore uses the existing theme-aware success Toast with a
10-second duration; operation errors remain visible until dismissed.

## Authorization

Reading forum settings continues to use the existing forum-settings access
rules. Updating either pagination value requires the existing
`settings.manage` permission, with the API remaining authoritative.

The forum settings update path keeps field-level ownership checks. An operator
with only `category.manage` or `tag.manage` may continue updating the fields
owned by those permissions but cannot change pagination values. No new
permission key is introduced.

## API Contract

Extend the forum settings response and update request schemas with the two
typed pagination fields. Document their default (`20`), minimum (`1`), maximum
(`100`), and `settings.manage` write requirement. Topic, search, and comment
list contracts continue to expose their effective `perPage` response value.

## Validation And Failure Behavior

The backend rejects stored or submitted pagination values outside 1-100 as
invalid forum settings. Missing options are not errors and resolve to the
recommended defaults. A settings lookup failure remains an API failure rather
than silently changing page size.

Frontend normalization protects the form from malformed legacy responses by
falling back to 20, but the backend is the source of truth for persisted
validation and request behavior.

## Test Strategy

Backend tests cover:

- missing options resolving both values to 20;
- accepting values from 1 through 100 and rejecting out-of-range values;
- reset restoring both values to 20;
- allowed and denied pagination updates for `settings.manage`;
- topic, search, and comment defaults using the correct configured value;
- explicit `perPage` overriding the configured default and retaining the
  maximum clamp;
- the existing page-200 clamp remaining intact.

Frontend tests cover:

- forum settings normalization, payload generation, and reset defaults;
- admin form range validation and permission-aware editing;
- homepage infinite-scroll termination using the response `perPage`;
- category, tag, and comment page counts using the response `perPage`;
- requests no longer forcing a hard-coded page size.

Verification includes focused Go and frontend tests, OpenAPI reference
validation, Nuxt typecheck, and the relevant repository validation scripts.

## Documentation

Update `knowledge/modules/forum.md` and `knowledge/modules/options.md` after
implementation. Add a decision record describing why API-side default
resolution was chosen over frontend-only option consumption, and add a session
handoff when implementation ends.
