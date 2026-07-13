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

These identifiers belong to different compatibility layers and must not be
substituted blindly:

| Identifier | Meaning |
| --- | --- |
| `sforum.host@2` | Canonical Manifest V3 `backend.hostApiVersion` for new packages |
| `sforum.host-api@2` | Legacy P2 Manifest identifier, accepted for already published fixtures and trust documents |
| `sforum.host/v2` | Canonical wire value in `HandshakeRequest.host_api_version` and generated SDK constants; accepted as a Manifest input alias only for early v2 compatibility |

The host maps the canonical Manifest identifier and compatibility aliases to
the canonical wire `host_api_version`; a plugin handshake must select
`sforum.host/v2`. The host permits only gRPC for this declaration. It fails
startup on a missing or unsupported Manifest Host API contract, a different
wire Host API version, a net/rpc subprocess, or an incompatible protocol
response. It never retries the same artifact as v1. New packages should emit
only `sforum.host@2` in the Manifest.

Protocol v1 remains the compatibility path for existing packages: omitted or
`protocolVersion: 1` selects the existing net/rpc runtime and `sforum.host/v1`.
Rolling a migrated plugin back means selecting its previously trusted v1
artifact and Manifest. It is not a fallback after any v2 lifecycle or migration
work starts, and it can require a new exact-artifact trust confirmation.

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

## Limits, deadlines, and cancellation

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
