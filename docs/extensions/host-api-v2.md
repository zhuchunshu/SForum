# Host API v2 and non-Go runtimes

Host API v2 is SForum's typed backend plugin protocol. The wire schema is
language-neutral Protobuf, but process startup is also part of the contract: a
plugin is not compatible merely because it implements the generated gRPC
services. It must preserve the HashiCorp go-plugin handshake, AutoMTLS, broker,
exact-artifact identity, runtime-token, authority, and resource-limit rules in
this document.

The Go SDK is the first supported SDK. A non-Go implementation should use a
small Go launcher based on that SDK unless it also has a tested, pinned
implementation of the go-plugin v1.8.0 core protocol. SForum does not currently
publish a stable non-Go launcher or promise compatibility with an independently
reimplemented go-plugin control plane.

## Authoritative contracts

The source schemas are split into three versioned packages:

| Package | Direction and ownership |
| --- | --- |
| `sforum.protocol.v2` | Shared handshake, identity, request/response context, limits, health, readiness, errors, and progress |
| `sforum.plugin.v2` | Host-to-plugin runtime, route, hook, file, job, lifecycle, provider, and service calls |
| `sforum.host.v2` | Plugin-to-host query/command, database, cache, job, schedule, discovery, secret, file, HTTP, admin, identity, permission, media, navigation, audit, and tracing calls |

The `.proto` files under `contracts/proto/sforum` are authoritative. Generated
Go files under `apps/api/sdk/plugin/v2/gen` are build products, not a second
schema source. Handwritten policy and SDK behavior live outside `gen`.

Generate and verify the Go SDK from the repository root:

```bash
./scripts/proto.sh generate
./scripts/proto.sh check
```

The script builds the pinned Buf and official Go generators from
`tools/proto/go.mod`; no system `protoc` or remote generation service is needed.
`check` runs Buf lint, regenerates from a clean output directory, and fails when
the checked-in Go SDK differs.

For another language, pin the official Protobuf and gRPC generators in that
language's own build, generate all files below `contracts/proto`, and preserve
the package and fully qualified service names exactly. Do not translate field
names into a handwritten JSON/HTTP protocol. The generated messages still need
the process and trust integration described below.

## Manifest selection and no downgrade

A v2 backend declares the exact pair:

```json
{
  "backend": {
    "entry": "backend/plugin",
    "rpc": "hashicorp-go-plugin",
    "protocolVersion": 2,
    "hostApiVersion": "sforum.host@2"
  }
}
```

Executable packages must select the current contracts explicitly:

| Identifier | Meaning |
| --- | --- |
| `sforum.host@2` | Manifest V3 `backend.hostApiVersion` |
| `sforum.host/v2` | Wire value in `HandshakeRequest.host_api_version` and generated SDK constants |

The host maps the Manifest identifier to the canonical wire
`host_api_version`; a plugin handshake must select `sforum.host/v2`. Only gRPC
Protocol V2 is accepted. Installation or startup fails on a missing or invalid
Manifest V3 declaration, any `protocolVersion` other than `2`, an unsupported
Host API contract, or an incompatible handshake response. There is no runtime
downgrade or legacy transport fallback.

## Go runtime

Embed the default v2 server, implement only the generated methods the plugin
owns, and use the explicit v2 entry point:

```go
package main

import (
  pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2"
)

type server struct {
  *pluginv2.Server
}

func main() {
  pluginv2.Serve(&server{Server: pluginv2.NewServer()})
}
```

`pluginv2.Server` validates the typed handshake, binds the exact runtime, and
provides health/readiness defaults. After a successful handshake,
`Server.Host()` exposes generated Host API clients. Build every plugin-to-host
request with `Host.RequestContext(parent)`; it copies actor, locale, and trace
context where present while replacing runtime-owned identity and authority and
clamping the deadline.

## Process handshake and AutoMTLS

The host starts only the executable declared and digest-bound by the Manifest.
HashiCorp go-plugin then supplies the process control plane:

1. The process must accept the magic cookie
   `SFORUM_PLUGIN=sforum-plugin-v1` and select application protocol version `2`
   from `PLUGIN_PROTOCOL_VERSIONS`.
2. Protocol v2 is gRPC-only. The first protocol line on stdout follows the
   go-plugin core format and reports core protocol `1`, application protocol
   `2`, listener network/address, transport `grpc`, and the AutoMTLS server
   certificate. Plugin logs belong on stderr; output before this line breaks
   startup.
3. The host enables go-plugin `AutoMTLS`. The plugin consumes the one-process
   client certificate in `PLUGIN_CLIENT_CERT`, creates a one-process server
   certificate, requires and verifies the client certificate, and returns the
   server leaf certificate in the protocol line. TLS 1.2 is the minimum. Plain
   gRPC is not a valid fallback.
4. The main gRPC server must expose the go-plugin controller, stdio, and
   bidirectional `plugin.GRPCBroker` control services as well as
   `sforum.plugin.v2.PluginRuntimeService`.
5. SForum sends a typed `HandshakeRequest`. The plugin must select
   `sforum.plugin` major `2`, minor `0`, accept `sforum.host/v2`, and reject an
   incomplete identity, token shorter than 32 bytes, or missing limits.

The process handshake and typed `Handshake` RPC are separate and both are
required. The magic cookie prevents accidental execution as an ordinary CLI;
it is not authentication. AutoMTLS authenticates this process connection, and
the typed runtime binding supplies the exact trust identity.

## Host broker and runtime token

The host allocates a runtime-scoped go-plugin `GRPCBroker` service before the
typed handshake and sends its nonzero `host_broker_id` in `HandshakeRequest`.
The plugin uses that id through the existing `plugin.GRPCBroker/StartStream`
control stream to dial the host-owned gRPC services over the same AutoMTLS trust
relationship. `host_broker_id` is an opaque per-process locator, not a network
port, credential, or reusable service id.

The handshake also carries a random `runtime_token`. Store it only in process
memory. Every unary call and stream opened toward the Host API must send the
raw token as the single binary gRPC metadata value:

```text
x-sforum-runtime-token-bin
```

Do not hex/base64 encode that metadata value, persist it, place it in a message
payload, command line, environment variable, URL, or log, or forward it to a
different process. The `-bin` metadata key carries the token bytes using the
language's gRPC binary-metadata API. The host compares a SHA-256 digest of the
received value in constant time and rejects missing, duplicate, or stale
tokens. A process restart receives a new token, broker id, runtime epoch, and
instance id; old channels must not reconnect.

## Exact request context

Every v2 request and each streaming open frame carries
`sforum.protocol.v2.RequestContext`. Its typed `ExtensionIdentity`,
`AuthorityGrant`, and `Actor` messages must remain generated wire values:

| Field | Rule |
| --- | --- |
| `request_id` | Required and unique within the runtime instance |
| `trace` | Propagate the host-provided trace; create a new child span rather than replacing correlation |
| `actor` | Propagate the authenticated actor supplied by the host; an autonomous plugin call has no user actor |
| `locale` | Propagate the call locale; use `und` only when none exists |
| `deadline` | Required, valid, and in the future; also set the native gRPC deadline |
| `extension` | Exact handshake `extension_id`, version, artifact digest, trust grant id, runtime epoch, and instance id |
| `granted_authority` | Exact ordered handshake disclosure; receiving or echoing it never creates authority |
| `idempotency_key` | Required where the invoked command contract requires idempotency |

For plugin-to-host calls, SForum compares `extension` and
`granted_authority` to its runtime binding before invoking a service. Do not
construct, reorder, add, or drop authority entries. Host service adapters still
perform the authoritative permission, resource, declaration, and policy check;
wire actor and authority fields are disclosure context, not bearer grants.

For host-to-plugin calls, bind the first accepted handshake identity in memory
and reject later health, readiness, and business requests carrying a different
runtime identity. A repeated handshake is valid only when token, identity, and
`host_broker_id` all match the existing binding.

## Route invocation contract

`InvokeRoute` executes one Host-selected stage of an immutable Route Registry
plan. The plugin must treat `route_action`, `invocation_stage`, schema refs, and
mutable-field lists as frozen inputs rather than selecting or widening them.
The accepted stage/action and response shapes are:

| `invocation_stage` | Route actions | Valid plugin result |
| --- | --- | --- |
| `ROUTE_INVOCATION_STAGE_HANDLER` | `add`, `replace` | A terminal status, headers, and typed body, or a validated `stream_follows` preflight. Patches, `prior_response`, and mutable-field lists are forbidden. |
| `ROUTE_INVOCATION_STAGE_REQUEST` | `global_middleware`, `before`, `filter`, `wrap` | `request_patch` only. Terminal status/headers/body, `stream_follows`, `prior_response`, and `response_patch` are forbidden. |
| `ROUTE_INVOCATION_STAGE_RESPONSE` | `filter`, `wrap`, `after` | `response_patch` only, against the Host-supplied `prior_response`. Terminal status/headers/body, `stream_follows`, and `request_patch` are forbidden. |

`mutable_request_fields` and `mutable_response_fields` are the exact ordered
RFC 6901 allowlists frozen with the route declaration. Every patch path must
match an entry byte-for-byte in the corresponding list; changing or reordering
the transmitted allowlist, patching an undeclared path, or returning the wrong
patch direction fails the invocation closed. The Host validates and applies
the ordered RFC 6902 `add`, `replace`, and `remove` subset authoritatively.
`Location` and `Link` are Host-owned response-mutation fields and cannot appear
in a response mutation allowlist. An `add` or `replace` terminal handler may
return `Location` as part of its complete response; declarative redirects and
response modifiers cannot author it. `Link` is stricter: plugin terminal and
streaming responses have the complete header removed. Canonical, preload,
pagination, and other link relations must use a versioned Host surface; route
alias, redirect, rewrite, and SEO policy remain the only canonical authorities.

For patch values, `RoutePatchOperation.value_json` is the authoritative wire
representation. `add` and `replace` require the bytes of one complete valid
JSON value, while `remove` requires `value_json` to be empty. Do not set the
deprecated `google.protobuf.Value value` field: the Host rejects any legacy
`value`, even when `value_json` is also present, because the legacy field can
round JSON integers through IEEE-754 doubles.

Queries have two compatible wire representations. Field 17
`query_parameter_values` is lossless: the Host sorts entries by key and
preserves every key's value order, including empty-string values. Legacy field
8 `query_parameters` remains populated with each key's first value. A
legacy-only query is emitted in both forms; when both forms are supplied, their
first values must match, and a key with no values is invalid.

An unsafe buffered HTTP operation that declares `required.24h@1` may compose
with request modifiers. The Host records every request stage, including an
empty patch, as an encrypted, plan-bound transcript. A replay evaluates the
current guard and request schema, reapplies only the previously Host-validated
patches under the current allowlists, and returns the stored response without
invoking any modifier or handler plugin again. The order of repeated query
values is part of the request fingerprint.

Credential-bearing request fields (`Cookie`, `Authorization`, proxy
authorization, API/auth/CSRF tokens, and `Idempotency-Key`) cannot be mutable in
a required-replay chain. Lifecycle publication and the runtime Dispatcher both
reject that combination before plugin execution. Mutation transcripts use a
dedicated HKDF-derived AES-256-GCM key rooted in `APP_OPTION_ENC_KEY`; plaintext
patch values are never stored in the replay record. Rotating that key makes
existing encrypted mutation transcripts fail closed until their 24-hour TTL
expires or the records are removed. The Host never re-executes plugins to
recover them. Response-only V1/V2 replay records contain no mutation transcript
and remain readable across this key rotation.

The request `path` is the immutable normalized transport path selected by the
execution plan. A Host-authorized `/params/*` mutation changes only subsequent
`path_parameters`; it never rewrites `path` or triggers route selection again.
Plugin handlers may therefore receive the original `path` with a different,
Host-proven parameter value. Core, alias, redirect, and rewrite terminals reject
parameter mutation, and the private Host provenance bit is never sent on wire.

Streaming composition is not available. `stream_follows` is valid only for an
exact handler-stage plugin route and cannot accompany a buffered body. The Host
rejects composed streaming chains and rejects terminal or streaming fields from
request/response mutation stages, so `before`, `filter`, `wrap`, `after`, and
global middleware cannot be composed around a stream.

## Limits, deadlines, and cancellation

### Notification emission command

`notifications.emit@1` is an actorless Host Command for one exact plugin
artifact's declared notification types. The Go SDK exposes
`NotificationEmitRequest` and `EmitNotification`; no protobuf service addition
is needed because the command uses `HostCommandService` and typed documents.

The Host binds the call to the authenticated artifact, grant, epoch, instance,
locale, deadline, and trace. It then validates namespace ownership, active type
and payload versions, the exact schema file/digest and value, a maximum of 50
explicit active recipients, the declared target, a 16 KiB payload, idempotency,
effective recipient policy, and 60 committed requests per rolling minute.
Plugin actor/session evidence is rejected before dispatch. Bulk broadcast and
raw Core notification table access are absent. Audits contain type identity,
sizes/counts, and stable reasons only; payload values and recipient ids are not
logged.

Current host-owned defaults are part of the compatibility test surface:

| Limit | Current value |
| --- | --- |
| Maximum gRPC send or receive message | 4 MiB (`4194304` bytes) |
| Active calls per gRPC server | 16 across unary calls and streams |
| Default RPC deadline | 5 seconds |
| Handshake/health deadline | 5 seconds |

The host advertises limits in `HandshakeRequest.limits`; a non-Go runtime must
apply the lower of its own reviewed limit and the advertised host limit. Apply
limits to the main plugin server, the Host broker client, and every stream
message, not only unary calls. Large files and HTTP bodies use bounded
`DataChunk` streams rather than one oversized message.

Honor a caller's shorter native gRPC deadline. If no deadline is present, add
the advertised default; never extend a deadline beyond the host default.
Propagate cancellation to work, broker calls, and stream producers, release the
concurrency slot when a call ends, and stop sending after cancellation. Map
expected failures to gRPC status plus the stable `ErrorDetail` contract where
the response defines it.

## Non-Go implementation checklist

A non-Go plugin is compatible only when its integration test proves all of the
following against a real SForum host subprocess:

- generated code comes from the committed v2 Protobuf sources and preserves
  fully qualified package/service names and streaming shapes;
- the go-plugin protocol line, magic cookie, exact version `2`, gRPC transport,
  AutoMTLS certificate exchange, control services, and broker stream are
  compatible with the host's pinned go-plugin core;
- the typed handshake rejects incomplete identity, incompatible version,
  missing limits, and stale repeat handshakes;
- `host_broker_id` is dialed through the broker control stream and every Host
  call carries exactly one raw `x-sforum-runtime-token-bin` value;
- request contexts preserve actor/locale/trace while exact identity, ordered
  authority, request id, and deadline pass host validation;
- 4 MiB message bounds, 16-call concurrency, timeout, cancellation, health,
  readiness, crash, restart, and stale-token behavior match the Go SDK;
- v2 mismatch fails closed and never starts or reconnects through v1.

The recommended first non-Go architecture is a thin Go launcher built against
the generated Go SDK. It owns go-plugin, AutoMTLS, broker, token, and context
enforcement and talks to language-specific business code through a private,
bounded local interface. That launcher and every executable it starts must be
declared and digest-bound in the package. A direct implementation in another
language is acceptable only after its go-plugin control-plane compatibility
suite is pinned and maintained; using ordinary gRPC alone is insufficient.
