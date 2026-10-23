# sforum.storage-s3

Protected built-in **S3-compatible attachment storage provider** for the
attachment storage slot.

## Ownership boundary

| Concern | Owner |
| --- | --- |
| Instance identity, SecretStore references, historical routing, probe, writer selection | **Host** |
| AWS S3 / MinIO / Cloudflare R2 transport and vendor behavior | **This plugin** |
| File upload/download/display pipelines, policies, orphan cleanup | **Host** |

## Features

- One enabled plugin process can host multiple independently configured
  instances (AWS S3, MinIO, Cloudflare R2, or compatible);
- the Host stores the selected instance on each attachment, so switching the
  current writer does not move or orphan existing objects;
- **multi-instance roots cannot be selected as writers**; only enabled
  instances are eligible;
- provider behavior stays in the plugin: FTP/SFTP transports are not part of
  this package or Core.

## Configuration

Open **Admin → Attachment Configuration → Basic Configuration** to select the
storage provider and manage instances. New built-in storage providers require
explicit enable before use. Probe data follows the request locale while keeping
stable machine-readable reasons and raw stored diagnostics.

## Cloudflare R2 quick configuration

Create a storage instance in **Admin → Attachment Configuration → Basic
Configuration** with the S3-compatible provider:

- **Endpoint**: `https://<account_id>.r2.cloudflarestorage.com`;
- **Bucket**: the R2 bucket name;
- **Region**: `auto`;
- **Access Key ID / Secret Access Key**: an R2 API token with Object Read &
  Write for the bucket;
- **Addressing style**: keep **Auto** — R2 endpoints resolve to Path-style
  automatically (R2 does not support Virtual-host addressing), and request
  checksums are sent only when the API requires them so R2 accepts uploads;
- **Public base URL**: optional; use an R2 custom domain or the `r2.dev`
  public URL. Leave empty to keep public attachments on the Host proxy.

Use **Test draft configuration** before saving: the probe writes, reads, and
deletes a temporary object, and surfaces the raw vendor error when the
credentials or permissions are wrong.

## Local filesystem alternative

`extensions/builtin/plugins/sforum-storage-fs` provides the default local
filesystem storage; the operator can switch the writer between providers
without moving existing objects.
