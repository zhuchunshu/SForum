# SForum AI Tool Reference

CI fixture for the M2 `ai.tool` contribution path. It declares exactly one AI
tool (`sforum.ai-tool-reference.order_lookup`) and answers it through the
reserved `sforum.ai.tool` ProviderCall slot. It requests no other capability and
does not call back into the Host.

It exists so the end-to-end claim can be tested against a real subprocess
instead of a fake invoker:

- `apps/api/app/Support/Extensions/IntegrationTests/ai_tool_reference_plugin_integration_test.go`
  builds `backend/`, renders `sforum.extension.json.tmpl` with exact digests, and
  drives `AIToolRuntimeInvoker` against the running plugin.
- It also proves the two Host-side refusals that must not depend on plugin
  goodwill: a declaration that is not frozen in the runtime manifest, and a
  per-call deadline that drifted from the declared `timeoutMs`.

The manifest is a template: `__BACKEND_DIGEST__` is replaced with the SHA-256 of
the freshly built backend binary before the package is loaded.

Two failure shapes are deliberate:

- `{"orderNumber": ""}` returns an *explainable* tool failure
  (`AIToolResult{IsError: true}`) — the model may retry with better arguments.
- `{"fail": true}` returns a Go error. The Host replaces it with a generic line,
  so the fixture's internal text never reaches model-visible content.
