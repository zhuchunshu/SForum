# SForum SEO Reference

Independently installable Protocol V2 reference plugin for the SEO Registry.

## Surfaces proved

| Kind | Action | Behavior |
| --- | --- | --- |
| `title` | filter | Appends ` \| SEO Reference` |
| `meta` | add | Adds `description` name meta |
| `canonical` | filter | Ensures trailing slash (same-origin only) |
| `robots` | filter | Sets `noArchive` (tightens only) |
| `jsonld` | add | Adds `DiscussionForumPosting` node |
| `sitemap` | add | Adds daily sitemap entry for canonical URL |

Failure policy is `fallback` for every contribution. Titles `reference:fail`
and `reference:timeout` exercise Host recovery without core product edits.

## Layout

```text
sforum-seo-reference/
├── sforum.extension.json.tmpl
├── backend/
│   └── main.go
└── README.md
```

Exact package digests are filled at build/test time (`__BACKEND_DIGEST__`).

## Product gate

`apps/api/app/Support/Extensions/seo_reference_plugin_integration_test.go`
builds this package as a real subprocess, publishes all SEO declarations, and
asserts multi-kind success, provider failure fallback, and disable fallback.
