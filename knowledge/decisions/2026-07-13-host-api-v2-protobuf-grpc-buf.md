# Decision: Host API V2 Protobuf, gRPC, And Buf Toolchain

Date: 2026-07-13
Status: Accepted for V3 P3

## Context

Host API v1 combines two compatibility surfaces:

- the host starts plugin subprocesses with HashiCorp go-plugin protocol 1 and
  exposes one untyped `net/rpc` plugin interface;
- plugins call the host through a loopback JSON/HTTP gateway identified as
  `sforum.host/v1`.

V3 P3 requires typed, versioned, streaming contracts in both directions while
keeping existing v1 SMTP, storage, content-policy, and contract fixtures
operational. The repository currently uses Go 1.26.5 and already resolves
`github.com/hashicorp/go-plugin v1.8.0`, `google.golang.org/grpc v1.80.0`, and
`google.golang.org/protobuf v1.36.11`. No system `protoc`, Buf, or Protobuf Go
generator is assumed to be installed.

The maintenance check on 2026-07-13 found:

| Component | Checked release | License | Result |
| --- | --- | --- | --- |
| HashiCorp go-plugin | `v1.8.0` (latest) | MPL-2.0 | Keep; it provides subprocess lifecycle, gRPC, broker, streaming, and version negotiation. |
| grpc-go | `v1.80.0` in the repository; `v1.82.0` latest | Apache-2.0 | Keep the resolved version for the first contract slice; upgrade separately when needed. |
| protobuf-go | `v1.36.11` (latest) | BSD-3-Clause | Promote to a direct SDK/runtime dependency. |
| Buf CLI | `v1.71.0` (latest) | Apache-2.0 | Pin as a Go tool for lint, breaking checks, and generation. |
| protoc-gen-go-grpc | `v1.6.2` (latest) | Apache-2.0 | Pin as a local generator tool. |

## Options Considered

### Raw `protoc` plus shell scripts

The upstream compiler and Go plugins are mature, but this would add an
unversioned system binary prerequisite and require separate lint and breaking
change tooling. It is not selected.

### Buf with pinned local Go generators

Buf supplies deterministic module loading, lint, and breaking checks while the
official Go generators continue to own generated code. Pinning all three tools
in `apps/api/go.mod` avoids a hidden workstation dependency and avoids remote
generation services in CI. This is selected.

### Connect or an HTTP/JSON-only protocol

Connect is a mature Protobuf RPC stack, but the accepted architecture and
go-plugin broker use native gRPC. Adding another transport would not remove the
need for gRPC and would complicate streaming and process negotiation. It is not
selected for the subprocess protocol.

### Dynamic descriptors or handwritten wire types

These approaches avoid generated files but discard the compile-time API that
P3 is intended to provide. They are not selected.

## Decision

### Contract layout

Authoritative `.proto` sources live under `contracts/proto` in three versioned
packages:

- `sforum.protocol.v2`: envelopes, identity, authority, handshake, health,
  readiness, lifecycle, errors, progress, and tracing metadata;
- `sforum.plugin.v2`: host-to-plugin routes, hooks, files, jobs, lifecycle,
  provider calls, and plugin service discovery;
- `sforum.host.v2`: plugin-to-host queries, transactional commands, database,
  cache, jobs, schedules, services, secrets, files, HTTP, admin surfaces,
  identity, permissions, media, navigation, audit, and tracing.

Files remain split by responsibility even when they share a Protobuf package.
Generated Go code lives under `apps/api/sdk/plugin/v2/gen`; handwritten SDK
adapters and policy enforcement stay outside generated packages.

`buf.yaml` and `buf.gen.yaml` are committed beside the Protobuf module. CI uses
the pinned local `buf`, `protoc-gen-go`, and `protoc-gen-go-grpc` tools. Remote
Buf Schema Registry plugins are not required to build or test SForum.

### Protocol selection and compatibility

- go-plugin protocol version 1 remains `net/rpc` and continues to use the
  existing `sforum-plugin-v1` adapter without source or wire changes.
- protocol version 2 uses go-plugin gRPC and generated Protobuf services.
- `VersionedPlugins` performs the go-plugin handshake, but the installed
  Manifest selects the exact allowed protocol. A package declaring v2 does not
  silently fall back to v1, and a v1 package is never upgraded implicitly.
- Existing `Serve` remains the v1 SDK entry point. The v2 SDK exposes an
  explicit v2 server entry point; dual-protocol serving is a separate explicit
  compatibility helper.
- Rolling a migrated plugin back to v1 requires selecting the trusted v1
  artifact/Manifest. V1 is not removed before the P13 compatibility gates.

### Bidirectional Host API

The main gRPC connection exposes plugin-owned services. During the typed
handshake, the host allocates a go-plugin `GRPCBroker` endpoint for host-owned
services and sends its broker id to the plugin. The plugin dials that endpoint
only after the host validates its exact artifact, extension identity, declared
Host API range, and granted authority.

Every request carries a typed envelope containing actor, locale, trace id,
request id, deadline, extension id/version/digest, and granted authority. The
host derives extension identity and authority from the authenticated runtime;
wire values are disclosure context and cannot grant access.

### Safety defaults

- Unary and streaming calls enforce configured send/receive limits, bounded
  concurrency, deadlines, cancellation, and typed protocol mismatch errors.
- Transactional Host Commands require a versioned command id, idempotency key,
  dry-run/impact response, authoritative policy evaluation, atomic commit,
  audit record, and typed result. Arbitrary plugin callbacks never execute
  inside a host database transaction.
- Service discovery is host-brokered and resolves only active, trusted,
  version-compatible declarations from immutable registry snapshots.
- Generated code and generated documentation are drift-checked. Buf lint and
  breaking checks are part of the P3 gate.

## Consequences

- Go and non-Go plugins share the same language-neutral wire contract.
- Existing v1 plugins remain operational while migration is observable through
  explicit deprecation metrics.
- Tool downloads are larger, but versions and licenses are reviewable in the
  Go module rather than depending on workstation state.
- Host policy adapters still require handwritten domain implementations;
  generated RPC code does not itself authorize requests.
