# Decision: MIT Project License

## Status

Accepted.

## Context

SForum identifies itself as a plugin-first open-source forum framework, but the
current repository tree had no root license. Historical releases through
`v2.7.7` used the MIT License. The project needs explicit terms for use,
modification, redistribution, commercial adoption, and contributions without
creating avoidable friction for operators or extension authors.

## Decision

Repository-authored SForum software is licensed under the MIT License with the
notice `Copyright (c) 2021-present Inkedus`.

Files and assets that carry their own license notices remain governed by those
notices. In particular, adding the root license does not replace the licenses
shipped with third-party fonts or other separately licensed material, and it
does not determine the license of independently distributed third-party
extensions.

## Consequences

- SForum may be used, modified, redistributed, sublicensed, and sold, including
  as part of proprietary products, when the MIT notice is preserved.
- The license contains no warranty and does not require modified or hosted
  versions to publish their source code.
- The choice continues the licensing model used by historical SForum releases
  and keeps the Core and plugin ecosystem permissive.
- A future license change would require a separate ownership and contributor
  review; this decision does not silently relicense third-party material.
