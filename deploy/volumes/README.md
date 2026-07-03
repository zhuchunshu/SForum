# Volumes

Docker named volumes are declared in the Compose files. Production backups
should be written outside containers, under the host path configured by
`SFORUM_BACKUP_DIR` or `./backups` by default.
