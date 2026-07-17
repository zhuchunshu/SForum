# 2026-07-18 Route Stream Opaque Bytes Boundary

Status: Accepted (documentation freeze of shipped behavior)

## Context

P6 streamed transport uses Protocol V2 `RouteStreamFrame` with a single chunk
arm: `sforum.protocol.v2.DataChunk` (`sequence`, opaque `bytes data`, optional
`checksum`, `final`). Host Fiber bridges for SSE, WebSocket, multipart, and
generic stream pump raw bytes. Manifest V3 still requires `responseSchema` on
`add`/`replace` for all modes (catalog/OpenAPI identity), but Host never applies
JSON Schema validation to stream chunk payloads.

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
- Chunk size limit, sequence, optional checksum, backpressure via stream
  lifetime budget and admission lease
- Fail-closed transport errors with Fail/StreamFailed before Cancel

### Host non-guarantees (plugin-owned)

- SSE event field grammar (`id` / `event` / `data`)
- WebSocket text vs binary application semantics (Host bridge uses binary
  frames for chunk payloads today)
- Multipart part headers and boundaries inside chunk bytes
- JSON Schema validation of chunk contents against Manifest `responseSchema`

### Manifest `responseSchema` on non-http modes

Keep the existing authoring requirement for catalog and OpenAPI aggregation
identity. Document that the ref is **not** a Host chunk-validation contract.
Future optional-schema relaxations are a separate Manifest change and are not
part of this freeze.

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
