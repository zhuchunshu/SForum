# Volumes

Docker named volumes are declared in the Compose files. Production backups
should be written outside containers, under the host path configured by
`SFORUM_BACKUP_DIR` or `./backups` by default.

`theme_releases` is retained as the physical volume name for upgrade
compatibility. API, worker, and web mount it at
`/var/lib/sforum/theme-releases`; the canonical environment variable is
`WEB_RELEASE_ROOT` (`SFORUM_WEB_RELEASE_ROOT` in the web supervisor).

The volume contains immutable release artifacts plus durable coordination
signals: `current.json` for desired state, `active.json` for the proxy target,
`failures/{releaseId}.json` for rejected candidates, and per-release artifact,
registry, and development-input directories. API and worker also mount the
separate immutable `extension_packages` volume; the web container does not.
