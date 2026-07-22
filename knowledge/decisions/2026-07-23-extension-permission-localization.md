# Extension-Owned Permission Localization

## Status

Accepted and implemented.

## Context

Plugin permissions appear in Host-owned role and permission screens. Core's
static `admin.permissionCatalog` covers built-in permissions, but cannot own
copy for arbitrary extension keys. Falling back directly to a plugin key left
operators with labels such as `sforum.admin-surface-reference.manage`.

## Decision

- `permissionDefinitions.label` and `description` use the existing
  `LocalizedText` contract: a legacy/default string or a locale-to-text map.
- Translation values remain inside the exact extension artifact and are bound
  into Identity Registry publication evidence.
- Host persistence stores the default copy and locale maps as presentation
  metadata. This does not grant permissions or change catalog ownership.
- Permission APIs resolve extension copy using the negotiated request locale.
- Host permission catalogs expose only extension permissions whose latest
  Identity Registry declaration is active. Tombstones retain ownership and
  audit history but are not assignable product capabilities.
- Admin UI precedence is built-in Core i18n, API-provided extension copy, then
  the stable permission key.
- Core locale files must not add keys for individual extensions.

## Consequences

Existing string-only manifests remain valid. Extension upgrades can revise
their own permission copy after the existing catalog-owner check, while an
untracked package still cannot claim a Host permission key. Locale changes are
artifact changes and therefore remain covered by normal trust and lifecycle
review.
