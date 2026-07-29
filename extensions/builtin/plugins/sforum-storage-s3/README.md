# sforum.storage-s3

Protected built-in S3-compatible attachment storage provider. One enabled
plugin process can host multiple independently configured AWS S3, MinIO,
Cloudflare R2, or compatible instances. The Host stores the selected instance
on each attachment, so switching the current writer does not move or orphan
existing objects.
