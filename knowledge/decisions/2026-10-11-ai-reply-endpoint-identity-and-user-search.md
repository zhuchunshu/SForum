# Decision: The Reply Bot Sees Author Handles, And Members Are Searchable

## Status

Accepted (2026-10-11). Implemented in the same change: prompt version
`forum-reply@4`, a sixth built-in tool (`user-search`), and a public user
search read path in `app/Models/Profile`.

## Context

The read-only tool set from
`decisions/2026-10-03-ai-read-only-chat-tools.md` assumed the bot could look a
member up. It could not, and the reason was in the context, not in the tools:

- `aireply.ReplyContext` carried **display names only** (`TopicAuthorName`,
  `ParentAuthorName`, `TriggerAuthor`, `ThreadComment.AuthorName`). Every one of
  those values was `COALESCE(NULLIF(display_name, ''), username)`, so the stable
  handle was thrown away at read time.
- `user-profile-read` requires the **username**. With only a nickname the model
  could name the person it was replying to but had no value to pass, so the tool
  was effectively unreachable for the users actually in the conversation.
- There was no way to go the other direction either. `forum-search` searches
  topics, so "who on this site writes about X" or "find the member called 张三"
  had no path at all: the model either guessed a handle (fabrication) or refused.

The one existing user lookup, `GET /api/v1/composer/references?kind=user`,
is the authenticated composer picker. It is not reusable here: it takes an
`identity.Actor`, returns avatar rows for the picker, and treats an empty query
as `%%` — that is, "list every active user", which is a directory export nobody
should hand to a language model.

## Decision

1. **Identity travels with every quoted utterance.** `ReplyContext` and
   `ThreadComment` now carry `AuthorRef{Username, DisplayName}` instead of a
   single name string. The prompt renders `AuthorRef.Label()`:

   - both present → `显示名（@用户名）`
   - only a handle → `@用户名`
   - only a display name → the display name
   - identical display name and handle → `@用户名` once (many accounts set the
     display name to the username; `alice（@alice）` is pure noise)
   - both missing → a caller-supplied fallback (`某位成员` / `这条发言`), never an
     empty label, because a line without a subject makes the model attribute the
     text to the previous author.

   The reader returns the **raw** display name (`COALESCE(display_name, '')`)
   instead of substituting the username, so the fallback rules live in exactly
   one place (`Label`) rather than being split between SQL and Go.

2. **`user-search` is the missing half of the pair.** A sixth built-in tool
   resolves a nickname to a handle and a profile link; `user-profile-read` then
   reads the public profile. It is declared in `aireply.ReplyToolAllowlist`
   next to `user-profile-read`, so the reply purpose can use it and an
   injected prompt still cannot reach a tool the purpose never advertised.

3. **The search read path is public, bounded, and cannot enumerate.** New
   `profile.Service.SearchPublicUsers` + `Store.SearchPublicUsers`:

   - rejects an empty query with `ErrProfileInvalid` — the empty query never
     reaches SQL, so `%%` cannot become a directory listing;
   - escapes `%`, `_` and `\` in the query (LIKE metacharacters are literals);
   - matches `username_lower` and `lower(display_name)` case-insensitively,
     active accounts only, exact match first, then display-name order;
   - returns `UserSearchHit{UserID, Username, DisplayName}` — a type with no
     email, status, role or history field, so this path cannot leak them even by
     accident, unlike `UserProfileSummary` which carries the email;
   - caps results (default 8, hard max 20) and the query length (80 runes).

4. **A searchable handle is not new information.** Handles are already public
   through `@mention` syntax, `/u/{username}` profiles and the authenticated
   mention picker. What changes is that the bot can now resolve a name a member
   already typed in a public topic, instead of inventing one.

## Alternatives rejected

- **Put the numeric user id in the prompt and let tools resolve by id.** The
  handle is already unique and is what mentions, profile URLs and the existing
  tool schema use. An id column in the prompt buys nothing and costs tokens on
  every quoted comment.
- **Reuse `forum.ReferenceSelector` for the tool.** It is gated on an actor, it
  shapes rows for the picker (avatars, `secondaryLabel`), and it matches `%%`
  when the query is empty. Making it public would mean changing the semantics of
  an existing authenticated surface to fit a new caller.
- **Have `forum-search` search users too.** A tool that returns two unrelated
  result shapes (topics and members) makes the model's result parsing and the
  operator's expectations worse; the two questions have different answers.
- **Skip the tool and only put handles in the prompt.** That fixes "look up the
  person I am replying to" but leaves "find the member named 张三" unsolvable,
  which is the second half of the reported gap.

## Consequences

- The prompt grows by one author label per quoted utterance (a handful of tokens
  per reply). `ReplyPromptVersion` is `forum-reply@4`; existing execution traces
  keep their old version, so the wording change stays attributable.
- An operator who customized `reply.systemPrompt` does not get the new
  "关于成员" guidance (their text replaces the default), but the identity labels
  and the tool are contract-level and apply regardless. The console badge says
  "已自定义" for exactly that reason, and restoring the default is one click.
- A model can now ask for members by keyword. The result is capped per call, the
  query must be non-empty, and non-active accounts (banned/disabled) are never
  returned; a determined user could still probe the directory letter by letter.
  That residual surface is the same as the logged-in mention picker, and the
  reply purpose is only reachable when an operator has enabled the bot and left
  `gates.toolCallsPerReply` above zero.
- `user-search` is registered by the assembly only when the profile service is
  wired, matching the existing rule that a missing dependency means the tool
  does not exist rather than the tool existing and always failing.
