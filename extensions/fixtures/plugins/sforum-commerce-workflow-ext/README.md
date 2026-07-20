# SForum Commerce Workflow Extender

Optional Protocol V2 extender for `sforum.commerce-workflow`.

## Surfaces proved

| Surface | Behavior |
| --- | --- |
| Dependency | required `sforum.commerce-workflow ^1.0.0` |
| Hooks | filter targeting commerce order.evaluate |
| Services | audit service added by the extender package |

This package must remain independently installable after the base commerce
plugin is trusted; it never requires core product edits.
