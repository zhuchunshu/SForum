# SForum AI Command Reference

CI fixture for the plugin-initiated AI completion path (`sforum.ai.complete@1`,
M1). One synchronous `observe` hook calls the Host command through the SDK
(`Host.Complete`) and returns the command's state, stable reason code, and text
as a hook result document, so the test can assert the Host's verdict instead of
guessing from a generic failure.

It exists so the end-to-end claim is tested against a real subprocess and a real
database, not a fake gateway:

- `apps/api/app/Support/Extensions/IntegrationTests/ai_command_reference_plugin_integration_test.go`
  builds `backend/`, renders `sforum.extension.json.tmpl` with exact digests, and
  drives the plugin on a disposable PostgreSQL cluster. It asserts:
  - a granted call reaches a mock provider, and `ai_executions` /
    `ai_usage_counters` attribute it to the calling plugin;
  - exhausting the extension daily quota returns `ai.quota_exceeded` with no
    outbound call and a denied execution trace scoped to the extension;
  - disabling the site gateway returns `ai.gateway_disabled` with no outbound
    call and a denied trace scoped to the site;
  - withholding the `ai.invoke` capability rejects the call at the command plan
    stage (`host.command_capability_denied`) before any transaction or outbound.
- The manifest declares `database.grants: ["host_commands"]` because every Host
  Command requires that authority grant; without it the call is refused with
  `host.command_authority_denied` before the command body runs.

The manifest is a template: `__BACKEND_DIGEST__` is replaced with the SHA-256 of
the freshly built backend binary before the package is loaded.
