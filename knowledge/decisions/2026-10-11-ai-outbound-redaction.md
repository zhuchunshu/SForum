# Decision: Outbound AI Redaction Is Implemented In The Gateway

## Status

Accepted (2026-10-11). Implemented in the same change.

## Context

`RedactionSettings` shipped with the AI gateway, has defaults with all three
categories enabled, and the AI console renders a toggle plus per-category
switches with the copy "内容脱敏 — 提交给供应商之前移除可识别信息。不会影响站内
存储的内容。" Nothing read those settings: no code outside the settings type and
the admin controller mentioned them. The M1 decision record listed the missing
implementation as a real exposure created by plugin-supplied content.

A setting that promises redaction and silently does nothing is worse than no
setting: the operator has recorded a decision that the system does not honour,
and the reassurance travels further than the gap does.

## Decision

### 1. The seam is `Gateway.Execute`, before the cache key

Every content-bearing outbound path goes through `Gateway.Execute`: the reply
bot (`Models/AIReply`), the plugin command (`HostAPI`), the diagnostics assistant
(`Models/AI`), and each step of the Host-owned tool loop (`Orchestrator`). The
model-list probe builds its own wire request but carries no prompt.

Redaction therefore happens in exactly one place, immediately after the provider
profile is resolved and **before** `CacheKey` is computed. Doing it after the
cache lookup would produce two cache entries for the same redacted body and would
keep the raw text in the cache key input. A test asserts that two requests which
differ only in the personal data they contain share one outbound call and one
cache entry.

`ApplyRedaction` returns a copy: the caller keeps its original request (it may
still echo it in the UI or write it to its own records), and only the copy that
leaves the Host is redacted.

### 2. Placeholders, not deletion

Matches become `[redacted-email]`, `[redacted-phone]`, `[redacted-ip]`. Deleting
them would splice neighbouring sentences together and make the model misread the
prompt; an explicit marker tells the model that something was removed without
telling it what.

### 3. Conservative matching, because a false positive is worse than a miss

- Email requires a domain dot and a 2+ letter TLD, so community `@mentions` are
  untouched.
- IPv4 requires four octets with per-octet range checks, so `1.2.3`, `10.0.0`
  and version strings are untouched.
- IPv6 uses a "three or more colon-separated hex groups" heuristic (including the
  compressed form) rather than the full RFC grammar: the full regex is not
  reviewable, and this shape essentially never occurs in forum prose.
- Phone matching accepts only three unambiguous shapes: a `+`-prefixed
  international number, a parenthesised area code, and a mainland-China mobile
  number. Bare digit runs are deliberately left alone — order numbers, revision
  ids, timestamps, and counters are bare digits, and mangling them corrupts the
  prompt in ways that are far harder to diagnose than a missed phone number.

A test pins the "must not change" list, and `Disabled`, "armed but empty", and
idempotence are all asserted.

### 4. Tool-call arguments are redacted as JSON, not as text

`Messages[].ToolCalls[].Arguments` is a JSON document the Host will send back to
the provider on the next step. Text-substituting inside it could break quoting or
structure and hand the model invalid JSON, so the arguments are parsed, their
string values are redacted recursively, and only then re-marshalled. If parsing
fails the value was already invalid, so the text path is used instead — and when
nothing matched, the original bytes are returned unchanged, so an
already-formatted JSON body is never reshuffled.

### 5. Attachments and streaming are out of scope by fact, not by policy

The AI path has no streaming, and the confirmed product scope excludes
multimodal input, so there is no binary channel to redact today. If either
arrives, it needs its own decision: text redaction cannot inspect an image.

## Consequences

- Operators who left the shipped default (enabled, all three categories) now get
  what the console has been claiming. Prompts may differ from what a caller
  passed in; the Host API v2 doc now says so explicitly, so plugin authors do not
  rely on byte-for-byte round-tripping.
- Redaction is not a secret filter: it covers recognisable personal data, not
  credentials. That is stated for plugin authors rather than implied.
- Redaction output is not itself logged anywhere new: execution traces still
  carry only sizes, reasons, and summaries, so this change does not create a new
  copy of user content in the database.
- Redacting before the cache key means a redacted prompt can be served from a
  cache entry produced by a different original — which is correct, because the
  provider would have seen the same bytes.
