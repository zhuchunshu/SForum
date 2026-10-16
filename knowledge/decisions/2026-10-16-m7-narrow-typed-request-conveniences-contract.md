# M7: Narrow Typed Request Conveniences — Contract Decision

- Date: 2026-10-16
- Status: accepted
- Milestone: M7 of
  `knowledge/plans/2026-08-22-extension-authoring-speed-and-controlled-power.md`
- Supersedes nothing. Extends the frozen contracts from
  `2026-10-16-extension-authoring-contracts-freeze.md` without changing them.

## Decision

**Admit no new request/response fields.** The candidate list in M7 is resolved
by inventory: every capability that has a real consumer is already admitted, and
every remaining candidate has **no consumer in the repository**, so it is
rejected with the evidence below. No protobuf message is added, which is exactly
what M7 task 4 permits ("add typed Protobuf messages … only after the ADR is
updated" — the ADR now records that no message is warranted).

## Admitted set (already implemented — evidence, not intent)

| Field / capability | Source owner | Sensitivity | Normalization & limits | Spoofing boundary | Grants / authority |
| --- | --- | --- | --- | --- | --- |
| `RouteRequest.headers` (declared request header projection) | Host request authority projection | mixed; credentials are **not** included unless raw | canonical name preserved, duplicate values preserved in order; `httpguts` name/value validation; whole document bounded by the 4 MiB transport cap | `x-sforum-*` always dropped; hop-by-hop (`Connection`-nominated, `te`, `trailer`, `transfer-encoding`, `upgrade`, `keep-alive`, `proxy-*`) dropped; `host`, `content-length`, `x-csrf-token` dropped | filtered mode for `core.guard.*` and `kind: custom`; raw mode **only** for a package-declared `kind: raw_request` guard (updated 2026-10-16 by decision R1: `core.guard.raw_request` no longer grants raw authority and is refused at validation) |
| `RequestContext.locale` | Host request locale | low | Host-resolved locale tag | client cannot forge a different locale than the Host resolved | any route/hook/query invocation |
| `RequestContext.trace` | Host trace context | low | opaque trace ids | plugins can only echo it | same |
| `RequestContext.deadline` | Host deadline | low | absolute timestamp | plugins cannot extend it | same |
| `RequestContext.idempotency_key` | Host idempotency owner | medium (correlation, not a credential) | opaque key; one-use semantics stay Host-owned | plugin cannot mint or substitute it | same |
| `RequestContext.actor` | Host identity | medium | Host-validated projection; role/permission keys only | **deliberately carries no user agent** (see rejections) | authenticated invocations only |
| `RouteResponse.headers` (Host-validated response headers) | Host response validation | mixed | validated before commit; streaming responses have the header removed | plugin cannot bypass Host validation | per route action/stage |
| `RouteRequest.query_parameter_values` (lossless) | Host | low | ordered by key, per-key value order preserved; legacy field keeps the first value | — | same |

Consumers proving these are real: `extensions/fixtures/plugins/sforum-custom-content/backend/routes.go:317`,
`extensions/fixtures/plugins/sforum-commerce-workflow/backend/routes.go:233`,
`extensions/dev/plugins/sforum.marketplace/backend/routes.go:407` (trace),
`sforum-membership-reference` and `sforum-admin-surface-reference` (actor/locale/idempotency).

## Rejected candidates (with evidence)

| Candidate | Verdict | Evidence |
| --- | --- | --- |
| Host-resolved client IP / proxy classification | **rejected for now** | no `client_ip` / `ClientIP` / `clientIp` reference exists anywhere under `extensions/**` or in the SDK. The Host owns proxy trust for its own middleware (`config.TrustProxy` / `TrustedProxies`, `config.go:100-106`), so a plugin-side field would need its own classification contract first. **Admission condition:** a named consumer plus a Host-derived, `TrustProxy`-aware field that never forwards the client's own claim. |
| User agent / device summary | **rejected** | the only references are *negative assertions* that the actor projection must not carry it (`sforum-membership-reference/backend/main.go:185`, `sforum-auth-github/backend/plugin.go:200`). A summary field would contradict a contract that is already tested in the safe direction. |
| Response cookie operations for plugin-owned names | **rejected (stays closed)** | no cookie message exists in `runtime.proto`; the request contract states credential material is outside the projection (`runtime.proto:361`). A plugin-owned cookie namespace would need a Host-validated cookie jar and its own ownership rules first. |
| Redirect hints | **rejected (already owned by the Host)** | plugins never emit HTTP redirects. The GitHub identity provider returns a typed `{"status":"redirect","redirectUrl":…}` document and the **Host** performs the browser redirect (`sforum-auth-github/backend/plugin.go:150-155`). A route-level redirect hint would create a second owner. |
| Cache hints | **rejected (already owned by the Host)** | cache semantics live in `app/Support/CacheRegistry` with Host-owned invalidation and publication; a plugin hint would bypass that owner. |
| Rate-limit hints | **rejected (already owned by the Host)** | route execution policies are Host-frozen (`host.ip_write@1`, `host.cors.same_origin@1`, … in the reviewed route policy publication); plugins cannot lower or raise them. |

## Stays closed (no change, listed so it is not reopened by accident)

Raw `Cookie`, unrestricted `Authorization` (only reachable through the explicitly
declared `raw_request` authority, which is a separate high-risk grant),
connection handles, request body streams (the body is a typed document),
Fiber types/locals, and arbitrary response callbacks. Evidence:
`app/Support/Extensions/protocol_v2_request_authority.go:139-160`
(credential + hop-by-hop + `x-sforum-*` blocking), `runtime.proto:49-70`
(typed body, no stream handle on `RouteRequest`), and the exit-gate check that
no SDK file or proto mentions Fiber.

## Tests added by this decision

`apps/api/app/Support/Extensions/protocol_v2_request_authority_test.go`:

- `TestProtocolV2RequestAuthorityRejectsCRLFAndInvalidHeaderNames` — CRLF in a
  value, CRLF/space/colon in a name, and NUL in a value are rejected at the
  projection (`ErrProtocolV2RequestAuthorityInvalid`) and **never reach the
  plugin** (call counter asserted zero).
- `TestProtocolV2RequestAuthorityDoesNotVouchForClientIdentityHeaders` — pins the
  spoofing boundary for proxy/client identity headers: `x-forwarded-for`,
  `x-real-ip`, `forwarded` and `user-agent` are forwarded **as the client's own
  claim**, and the Host adds no `x-sforum-*`-style vouched identity header.
- `TestProtocolV2RequestAuthorityBoundsOversizedDocumentsAtTheRPCBoundary` —
  oversized request documents fail at the RPC boundary with the production
  message limits applied, and never reach the plugin.

Already covered before this decision (kept as the existing contract):
hop-by-hop and `x-sforum-*` stripping plus duplicate-value preservation
(`TestProtocolV2RequestAuthorityStripsHostMetadataBeforeRPC`), exact frozen
route/guard binding for raw authority
(`TestProtocolV2RawAuthorityRequiresExactFrozenRoute` / `…ExactFrozenGuard`), and
stream parity (`TestProtocolV2RawAuthorityIsPreservedForRouteStreams`).

**Oversized documents** are bounded by the transport, not by a projection-level
cap: `DefaultMaxMessageBytes = 4 << 20`
(`apps/api/sdk/plugin/v2/transport.go:17`) is applied as
`MaxCallRecvMsgSize`/`MaxCallSendMsgSize` on the production dial options
(`app/Support/Extensions/protocol_v2_client.go:183-186`) and as
`MaxRecvMsgSize`/`MaxSendMsgSize` on the SDK server
(`apps/api/sdk/plugin/v2/transport.go:89-90`).
`TestProtocolV2RequestAuthorityBoundsOversizedDocumentsAtTheRPCBoundary` now
assembles the **same limits in-process** (server `MaxRecvMsgSize`, client call
options) and asserts that a request document over the cap fails at the RPC
boundary and never reaches the plugin. This is in-process evidence with
production limits, not subprocess evidence — stated as such.

## Residual (explicit, not claimed as done)

- No field-level count/size cap exists inside the projection; if a future
  admission adds one, this ADR must record the number and its failure mode.
- Client-IP/proxy classification stays rejected until a named consumer exists;
  when it is admitted, the spoofing boundary must be Host-derived (never the
  client's own `x-forwarded-for` claim) and this ADR must be revised first.
