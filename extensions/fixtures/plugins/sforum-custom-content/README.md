# SForum Custom Content Reference

Independently installable Protocol V2 reference plugin for Entity, Content,
Editor, and Navigation/Region registries.

## Surfaces proved

| Surface | Declaration |
| --- | --- |
| Entity | `entity.article` with soft-delete + import/export |
| Taxonomy | hierarchical `taxonomy.topic` bound to article |
| Field | indexed text `field.summary` |
| Content | vote / product-card / embed / workflow-form blocks |
| Editor | Tiptap node + command + toolbar (prebuilt L2) |
| Navigation | public nav item + sidebar region |
| Query | public `articles` list with Host Query Registry handler |

Exact package digests are filled at build/test time (`__BACKEND_DIGEST__`,
`__EDITOR_VOTE_DIGEST__`).

## Product gate

`apps/api/app/Support/Extensions/custom_content_reference_plugin_integration_test.go`
builds this package as a real subprocess, publishes the four registries, and
asserts resolve success plus disable/remove fallback without core product edits.
