# SForum Events Command Reference

CI fixture for `sforum.events.emit@1` (M1.5). One synchronous `observe` hook calls
the Host through the SDK's `EmitEvent` helper and returns the command's verdict
(state, stable reason code, published name, delivery counters) as a hook result
document.

It exists to prove, against a real subprocess and the real Host event bus, the one
rule the SDK cannot enforce on its own:

- a plugin may publish a name inside its own namespace (the declared
  `sforum.events-command-reference.custom-requested` event is the declaration side
  of that rule);
- the same call fails with `events.namespace_forbidden` when the name belongs to
  another plugin or to Core, even though the plugin is the one making the call and
  the SDK saw nothing wrong with the request.

Shape validation that both sides share (`events.kind_invalid`,
`events.payload_invalid`) is covered by unit tests on the Host command and on the
SDK helper instead of being duplicated here.

`apps/api/app/Support/Extensions/IntegrationTests/events_command_reference_plugin_integration_test.go`
builds `backend/`, renders `sforum.extension.json.tmpl` with exact digests, and
drives the plugin on a disposable PostgreSQL cluster.

The manifest is a template: `__BACKEND_DIGEST__` is replaced with the SHA-256 of
the freshly built backend binary before the package is loaded.
