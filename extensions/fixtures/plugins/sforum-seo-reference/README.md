# SForum SEO Reference

This Protocol V2 fixture filters one controlled SEO title through the shared
`ProviderCall` RPC. It receives no actor, session, raw request, HTML, or
arbitrary head access. The Host still validates mutation scope, robots policy,
canonical/hreflang origin, sitemap policy, and the final typed document.

Build from the repository checkout, replace the manifest digest placeholders
with the resulting SHA-256, then package/upload it through the normal inert
installer. Production activation never compiles this source and still requires
the exact-artifact executable trust flow.

```bash
cd extensions/fixtures/plugins/sforum-seo-reference/backend
CGO_ENABLED=0 go build -mod=mod -trimpath -buildvcs=false -o plugin .
```

The integration test also exercises provider failure and disable fallback so a
broken or stopped plugin never removes the Core SEO baseline.
