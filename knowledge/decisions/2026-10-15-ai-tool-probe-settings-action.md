# Decision: `ai_tool_probe` Settings Action

## Status

Accepted (2026-10-15). Implemented in this change.

## Context

A plugin's Schema settings page could already offer a "test" button through the
`provider_probe` settings action: the Host starts a short-lived Protocol V2
runtime seeded with the operator's draft values as `SFORUM_SETTING_*` env vars,
then calls the provider probe for a declared provider slot
(`apps/api/app/Support/Extensions/settings_actions.go`).

A plugin whose only executable surface is an AI tool (`manifest.aiTools[]`) had
no way to offer that same "test my configuration" button: it has no provider
slot, so `provider_probe` is unavailable by construction. The plugin author's
only options were to declare a fake provider slot (misstating what the plugin
does) or to ship no probe at all.

The transport already existed. `sforum.ai.tool`/`invoke` on the versioned
`ProviderCall` family (`apps/api/app/Support/Extensions/protocol_v2_ai_tool.go`,
`ProtocolStarter.InvokeVersionedAITool`) carries exact-declaration binding,
contract version, a deadline that must equal the declared `timeoutMs`, and an
error envelope. So the missing piece was only a settings-action *kind* that
routes to it.

## Decisions

### 0. The probe starter inherits the active runtime's trust source

Found by running the action against a real uploaded plugin (not by a unit test),
fixed in the same change.

`ProbeSettingsAction` built its short-lived starter as
`NewProtocolStarter(ProtocolStarterConfig{Settings: …})` — with **no trust
source**. `protocolV2ClientConfig` only falls back to a synthetic identity when
the extension is builtin:

```go
if s.trust != nil { … } else if extension.Source == extensions.SourceBuiltin {
	trustIdentity = RuntimeTrustIdentity{TrustGrantID: "builtin", …}
} else { err = ErrTrustGrantNotFound }
```

So every *uploaded* plugin's settings probe failed with
`resolve protocol v2 trust identity: extensions: executable trust grant not found`.
This was a pre-existing gap on the `provider_probe` path too — the only plugins
shipping a settings action in-tree are builtins (`sforum.smtp`,
`sforum.storage-fs`), so nothing had ever exercised it.

The probe starter now inherits the trust source from the manager's active
starter (`settingsActionProbeStarter`), because:

- the probe executes the *same* executable artifact as the live runtime, so it
  must clear the same bar — giving the probe a weaker requirement would open a
  "execute plugin code without a trust grant" channel;
- the operator-facing failure was a bare `extension.settings_action_unavailable`
  with no cause, which is exactly the wrong signal for a feature whose purpose is
  diagnosis.

Written as a free function, not a method: `RuntimeInvoker` is at its 28-method
cap.

### 1. `aiTools[]` is the declaration surface; no new `probes[]` family

The new kind reuses the plugin's existing `aiTools[]` declarations instead of
introducing a `probes[]` manifest family.

Reasons:

- A probe of an AI tool *is* a call to that tool. A second declaration family
  would let a probe name a handler that no `aiTools[]` entry binds, so "what the
  operator tested" and "what the model will call" could diverge — exactly the
  drift the frozen-declaration transport exists to prevent.
- `aiTools[]` already carries everything a probe needs (id, contract version,
  handler, name, `timeoutMs`) and is already validated, normalized, JSON-schema
  constrained, and frozen per runtime artifact. A new family would duplicate
  that validation surface.
- The operator-visible question is "does my configured tool work?", not "does
  my probe handler work?". Binding the action to `tool: <aiTools[].id>` makes
  that the only thing the manifest can express.

Consequence: an `ai_tool_probe` action requires the named tool to be declared in
the same manifest, namespaced under the extension id (`IsOwnedID`), and requires
at least one draft field. Anything else is `ErrInvalidManifest`.

### 2. `provider_probe` keeps requiring a provider slot

An action still requires `type: plugin` and a non-empty `backend.entry`; only
the "which declaration must exist" clause became kind-aware. `provider_probe`
keeps requiring `len(manifest.Providers) > 0` because the probe it runs
(`ProviderProbe`, or `StorageProbe` for `attachment.storage.provider`) is
addressed by provider slot. Relaxing that check to "any action is fine" would
have let a `provider_probe` action validate on a plugin that the Host can never
probe, turning a manifest error into a runtime "unavailable" dead end.

### 3. The `SettingsActionRuntime` signature was generalised, not extended

`RuntimeInvoker` is at its non-growth receiver-method cap (28) and both
`app/Support/Extensions` and `app/Models/Extensions` flat directories are at
their file caps, so a second `ProbeAIToolSettingsAction` method was not
available. Instead the interface's single method now takes a request object:

```go
type SettingsActionProbeRequest struct {
	ProviderSlot string
	AIToolID     string
	Values       map[string]string
}

type SettingsActionRuntime interface {
	ProbeSettingsAction(ctx context.Context, extension Extension, request SettingsActionProbeRequest) (SettingsActionProbeResult, error)
}
```

The two paths are mutually exclusive by construction: `provider_probe` fills
`ProviderSlot`, `ai_tool_probe` fills `AIToolID`. The Host decides which one
before starting the runtime, and an unknown kind is refused with
`ErrSettingsActionUnavailable` without touching the plugin. Method count is
unchanged on `RuntimeInvoker`, `Manager`, and `SettingsService`.

### 4. `Details`/`Suggestions` are unused on this path

The AI tool result contract carries only `content` and `isError`
(`VersionedAIToolResponse`). So `ai_tool_probe` maps `isError == false` to
`{OK: true, Reason: "extension.settings_action_ai_tool_ok", Message: content}`
and `isError == true` to the `..._ai_tool_failed` reason, and never populates
`Details`/`Suggestions`. Everything the operator needs to read — the diagnostic,
the failing endpoint, the fix — belongs in the tool's `content` text, which the
Host bounds and returns as `message`. The `provider_probe` path is unchanged and
still forwards details/suggestions.

### 5. The probe runs even while the plugin's own runtime switch is off

Like `provider_probe`, the AI tool probe starts a fresh isolated runtime that
cannot register routes, events, jobs, schedules, or providers, and it is not
gated on the plugin being enabled. Operators configure and test before enabling;
requiring the plugin to be live first would force them to turn on an untested
configuration to test it. The runtime is stopped with the same 2-second timeout
on the way out.

### 6. Draft values are both env and arguments

The probe runtime is seeded with the draft values as `SFORUM_SETTING_*` (the
plugin's own configuration view) and the same map is JSON-encoded into the tool
call's `arguments`. The tool therefore receives one set of facts in the two
places it can read them, rather than a settings view that disagrees with what
the model would pass. The deadline passed to the transport is exactly
`timeoutMs`; the transport rejects any drift.

## Verification

- `cd apps/api && go build ./...`
- `go vet ./app/Support/ExtensionManifest/... ./app/Models/Extensions/... ./app/Support/Extensions/...`
- `go test ./app/Support/ExtensionManifest/... ./app/Models/Extensions/... ./app/Support/Extensions/... ./bootstrap/...`
- `node tests/validate-architecture-boundaries.mjs` (flat-directory and
  receiver-method caps unchanged; no new production file)
- `gofmt -l app/Support/ExtensionManifest app/Models/Extensions app/Support/Extensions`
- New tests: manifest accept/deny matrix for the kind (declared tool without
  providers accepted; empty/foreign/undeclared tool, empty fields, unknown kind
  denied; `provider_probe` still requires a provider slot), and runtime-path
  tests asserting the action forwards `AIToolID` plus draft values, maps the
  tool result to success/failure, leaves details/suggestions empty, and returns
  `ErrSettingsActionUnavailable` for a missing declared tool without starting a
  probe. A unit case also pins that a transport failure (probe runtime cannot
  start) becomes the host-owned unavailable result rather than a fabricated
  "the tool said it failed".
- Real subprocess: `IntegrationTests/settings_action_ai_tool_probe_integration_test.go`
  drives `Manager.ProbeSettingsAction` against the `sforum-ai-tool-reference`
  fixture (`Source: uploaded`, i.e. the branch that §0 fixed) and asserts the
  success mapping, the explainable-failure mapping, refusal of an undeclared tool
  id, and that a string-typed draft value reaching a boolean parameter stays an
  explainable failure instead of an internal error.
- End to end in a running site against an uploaded plugin
  (`sforum.fetch-url`, external package, out of this repo): the settings page
  renders the action with `available: true`; executing it returns
  `extension.settings_action_ai_tool_ok` with a real fetch of
  `https://example.com/` through the operator's draft proxy; an empty draft
  allowlist and a non-whitelisted draft target each return
  `extension.settings_action_ai_tool_failed` with the plugin's own explanation
  plus the effective draft configuration.

## Open Questions

- Should an AI tool probe be able to declare a *sample* argument set distinct
  from the settings fields (for tools whose interesting input is not
  configuration)? Today the draft settings values are the only arguments, which
  is right for "test my configuration" but not for tools that need a real
  payload. Note the consequence observed in the integration test: because draft
  values are strings, a tool parameter typed `boolean` can never be exercised
  through the probe.
- A probe failure at the transport layer surfaces as
  `extension.settings_action_unavailable`, and the wrapped cause is not written
  anywhere the operator can see (§0 was only found by reproducing the transport
  path in a test). Surfacing a bounded cause in the audit record would turn that
  class of failure from "unavailable" into an actionable message.
- The `provider_probe` path returns structured `Details`/`Suggestions`; whether
  AI tools should get an equivalent structured envelope (instead of putting
  everything in `content`) is a question for the tool-result contract, not for
  this action kind.
