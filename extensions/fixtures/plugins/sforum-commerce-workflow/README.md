# SForum Commerce Workflow Reference

Independently installable Protocol V2 reference plugin for a trusted
commerce/workflow vertical.

## Surfaces proved

| Surface | Declaration |
| --- | --- |
| Routes | add + alias + redirect + SSE stream |
| Database | own_schema with retain/export retention |
| Hooks | filter evaluate + async observe |
| Jobs | exponential settle job |
| Cache | permission-scoped order namespace |
| Services | versioned order lookup service |
| Components | prebuilt L2 order card |
| OpenAPI | namespaced fragment |

## Extender

`sforum-commerce-workflow-ext` depends on this package and extends its hooks
and services without editing core product code.

## Product gate

`apps/api/app/Support/Extensions/commerce_workflow_reference_plugin_integration_test.go`
builds both packages as real subprocesses and asserts happy path plus
disable/failure fallback.
