# 2026-07-18 Route Stream Opaque Bytes Boundary

Status: Accepted (documentation freeze of shipped behavior)

## Context

P6 streamed transport uses Protocol V2 `RouteStreamFrame` with a single chunk
arm: `sforum.protocol.v2.DataChunk` (`sequence`, opaque `bytes data`, optional
`checksum`, `final`). Host Fiber bridges for SSE, WebSocket, multipart, and
generic stream pump raw bytes. Legacy Manifest V3 `requestSchema` and
`responseSchema` values may remain on non-HTTP routes as OpenAPI documentation
identity, but Host never applies JSON Schema validation to stream chunk payloads.

Three product options were recorded in the V3 progress ledger:

1. Opaque bytes only (status quo)
2. Mode-specific frame envelopes (`SseEvent`, `WebSocketMessage`, `MultipartPart`)
3. Single JSON document per chunk

## Decision

**Freeze option 1: opaque bytes** as the supported public boundary for route
streams in this platform version.

### Host guarantees

- Open/close metadata and mode preflight (status, required headers such as
  SSE `Content-Type`, WebSocket upgrade semantics)
- Chunk size limit, sequence, optional checksum, gRPC flow-control backpressure,
  one total stream lifetime budget, and an exact-runtime admission lease
- Fail-closed transport errors with Fail/StreamFailed before Cancel

### Host non-guarantees (plugin-owned)

- SSE event field grammar (`id` / `event` / `data`)
- WebSocket text vs binary application semantics. Host preserves inbound
  message boundaries but discards the opcode distinction and emits response
  chunks as binary messages.
- Multipart part headers and boundaries inside chunk bytes
- JSON Schema validation of chunk contents against Manifest `responseSchema`

### Manifest schemas on non-http modes

`requestSchema` and `responseSchema` are optional compatibility metadata on
non-HTTP routes. When present they may describe plugin-owned payloads in the
public OpenAPI fragment, but they are **not** Host chunk-validation contracts.
The internal JSON Route Schema Catalog excludes every non-HTTP binding.

Generated client metadata exposes `streamContract: sforum.route.opaque_bytes@1`
and `payloadValidation: plugin_owned`, including when the legacy schema ids are
omitted. HTTP routes keep their existing exact JSON Schema requirements and
Host validation.

### Deferred

- Mode-specific Protocol envelopes (option 2) and JSON document streams
  (option 3) remain additive future designs. They require a new decision,
  proto changes, Host validators, docs, and tests before any credit.

## Consequences

- Non-HTTP Schema is **honestly complete as an opaque boundary**, not as
  structured framing/validation.
- P6 streamed-transport credit still requires the joined behavior matrix and
  any remaining durable-incident join criteria from the progress ledger; this
  freeze alone does not raise P6 to 18/18.
- Plugin service streams (`ServiceStreamFrame` + `TypedDocument`) remain a
  separate, schema-validated surface and are unchanged.

## Evidence anchors

- `contracts/proto/sforum/plugin/v2/runtime.proto` (`RouteStreamFrame`)
- `contracts/proto/sforum/protocol/v2/common.proto` (`DataChunk`)
- Host stream validation in `protocol_v2_route_stream.go` (size/seq only)
- Stream e2e: `route_dispatcher_stream_integration_test.go`
