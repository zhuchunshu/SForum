# Volumes

Docker named volumes are declared in the Compose files. Production backups
should be written outside containers, under the host path configured by
`SFORUM_BACKUP_DIR` or `./backups` by default.

Runtime volumes store PostgreSQL, Redis, Meilisearch, uploaded attachments, and
installed extension packages. Public themes use package assets plus the Page
Registry; admin components load directly from immutable digest endpoints, so no
frontend release artifact volume is required.
