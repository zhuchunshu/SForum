# SForum Media Optimize Reference

Independently installable Protocol V2 reference plugin for the Media Pipeline
Registry.

## Surfaces proved

| Surface | Behavior |
| --- | --- |
| MIME policy | Declares PNG/JPEG/WebP policy bound to Host permission |
| Transforms | Thumbnail + preview WebP variants (background execution) |
| Jobs | `sforum.media-optimize.variants` background optimize job |
| Fallback | Manifest maps to `fallback_original`; disable keeps source immutable |

Exact package digests are filled at build/test time (`__BACKEND_DIGEST__`).

## Product gate

`apps/api/app/Support/Extensions/media_optimize_reference_plugin_integration_test.go`
builds this package as a real subprocess, publishes the media graph, and
asserts plan + disable/original fallback without core product edits.
