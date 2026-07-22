# Forum day-to-day

[← Usage](./README.md)

## Content model

| Term | Meaning |
| --- | --- |
| **Topic** | User-facing discussion thread |
| **Comment** | Tree reply under a topic |
| **Post** | Shared content row (body/revisions); not a public product word |
| **Category / Tag** | Taxonomy; tags support Unicode slugs |

## Operator checklist

1. Maintain category trees and tags (icons/colors as needed).  
2. Configure forum policy: lengths, nesting, edit windows, cooldowns, guest read.  
3. Guest-read off means unauthenticated users cannot read protected lists/details.  
4. Moderator actions: lock, pin, hide/restore (permission-gated).  
5. Moderation workflows: reports and optional pre-publication review.  

Limits are server-authoritative and should match composer UX.

## Content editing and revision history

- Author edits still follow the site's edit window. Staff with `topic.edit_any`
  or `post.edit_any` can handle the matching content in the control panel. Each
  effective save creates a new accepted version.
- **Forum management → Content management** browses topics and comments in all
  lifecycle states. Revision timelines are staff-only, requiring
  `topic.revision.view_any` or `post.revision.view_any`; list responses never
  expose raw source.
- Restore requires the corresponding history permission plus `*_edit_any` and
  a reason. It appends a new version, never rewrites an old one, and does not
  change moderation or lifecycle state.
- Concurrent editors receive a conflict for a stale version token. Reload the
  latest content or inspect history before editing again; V1 has no force save.
- Only `super_admin` can permanently redact a non-current version, with a
  reason and typed `REDACT` confirmation. A redacted version keeps only its
  header and cannot be previewed or restored.

## Member actions

- Browse home, categories, tags  
- Create/edit topics and comments within policy windows  
- Profile and account settings (avatar strategy, sessions)  
- Report content  

## Engagement loop (product track)

View-count increment, reactions, and bookmarks may still be incomplete depending on version—see [Roadmap](../roadmap.md).

## Next

- [Search](./search.md)  
- [Admin](./admin.md)  
