# SForum Query Reference Plugin

Protocol V2 fixture for Host-owned Query Registry execution.

- Declares one public offset query and one login-gated query.
- Negotiates `query.runtime@1` and implements `InvokeQuery` / `FilterQueryResult`.
- Package Schema is Draft 2020-12 with `additionalProperties: false`.
- Filter rewrites `title` only; it never adds undeclared fields.

Built and exercised by
`apps/api/app/Support/Extensions/query_reference_plugin_integration_test.go`.
