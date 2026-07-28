# 2026-07-09 Avatar Strategy System

## Status

Accepted.

## Context

SForum needs avatars that work out of the box for self-hosted forums, while
still letting operators choose Gravatar, a static default image, and user
uploads. Upload behavior needs image-specific limits without changing the
general attachment rules, and the default should not depend on an external
service.

## Decision

- Treat avatars as profile behavior backed by the attachment system. Uploaded
  avatars are public attachments and take priority over fallback providers.
- Store avatar runtime settings in `web_options` under `avatar.*`, guarded by
  `settings.manage`.
- Default fallback provider is `initials`, so a new install has local avatars
  without outbound network dependency.
- Keep Gravatar as an optional provider with configurable base URL and
  `sha256` as the recommended hash algorithm. `md5` remains supported only for
  legacy mirror compatibility.
- Support a `static` fallback provider only when an operator supplies
  `avatar.default_static_url`.
- Reuse the attachment storage pipeline for avatar bytes, but enforce
  avatar-specific upload size, dimension, GIF, and compression settings.
- Use the official pure-Go `golang.org/x/image/draw` implementation for
  JPEG/PNG center-crop and resize, and `github.com/rwcarlsen/goexif` only for
  the bounded EXIF Orientation read. Do not use `bimg` as the default
  dependency because it requires libvips and a C toolchain in deployment
  environments.
- Allow GIF avatars only when `avatar.allow_gif` is enabled. Animated GIFs are
  preserved and not compressed; size and dimension limits still apply.

## Consequences

- Operators get beginner-friendly local initials by default and can opt into
  external avatar sources later.
- Profile responses expose a stable `avatar` view, so themes do not need to
  know whether an avatar came from upload, Gravatar, static URL, or initials.
- Referenced avatar attachments increase/decrease attachment reference counts,
  preventing orphan cleanup from deleting currently used avatars.
- External Gravatar URLs are rendered as normal images on the frontend, avoiding
  build-time image-domain restrictions for custom avatar mirrors.

## Security Amendment (2026-07-29)

`github.com/disintegration/imaging v1.6.2` was removed from Core and the media
optimization fixture because its unpatched TIFF path can panic on crafted
paletted images (`GHSA-q7pp-wcgr-pffx`). SForum does not expose TIFF as a
supported avatar or optimization format. The replacement decoder explicitly
accepts only JPEG and PNG before transform, preserves all eight EXIF orientation
values, bounds optional EXIF scanning to 1 MiB, and does not register a TIFF
decoder in production binaries.
