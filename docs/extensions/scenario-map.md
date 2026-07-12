# Extension scenario map

Quick “I want to…” index for plugin authors. The fuller narrative lives in the
[authoring guide](./authoring-guide.md) (SMTP mail + filesystem storage +
content policy workflow references + scenario table).

| I want to… | Use |
| --- | --- |
| Run code after a topic is created | observe `topic.created` (+ optional Host API job) |
| Change or reject a new topic | filter `topic.before_create` |
| Change or reject a topic edit | filter `topic.before_update` |
| Change or reject a new reply | filter `comment.before_create` |
| Reject registration | validate `user.before_register` |
| Reject an upload by metadata | validate `attachment.before_upload` |
| Add a topic detail button | contribution `forum.topic.actions` |
| Add topic badges / sidebar / list pills | `forum.topic.badges` / `sidebar` / `list.badges` |
| Add public nav entries | contribution `forum.nav.items` |
| Swap outbound mail transport | provider `mail.provider` → `sforum.smtp` |
| Swap attachment storage | provider `attachment.storage.provider` → `sforum.storage-fs` |
| Swap full-text search | provider `search.provider` (Wave E7) |
| Store per-topic structured data | entity meta (F4.4 / E3) |
| Own HTTP API under the host proxy | manifest `routes` + backend `RouteTarget` |
| Call host from the plugin process | Host API + declared `capabilities` |
| End-to-end **workflow** sample | enable `sforum.content-policy` |
| End-to-end **mail provider** sample | enable `sforum.smtp` + select in Mail settings |
| End-to-end **storage provider** sample | enable `sforum.storage-fs` + select in Attachment settings |

Generated host catalogs (events, contribution points, capabilities, provider
slots, schedules): [catalogs/](./catalogs/README.md).
