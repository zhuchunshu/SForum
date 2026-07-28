# Decision: Operator-Owned Public Navigation With Plugin Contributions

## Status

Accepted

## Date

2026-07-27

## Context

SForum already has three related but incomplete navigation paths:

- `site_nav_items` stores operator-managed rows and seeds Home, Categories,
  Tags, and Search.
- `SFNavbar` consumes those rows and appends `forum.nav.items` contributions.
- the V3 Navigation/Region Registry models versioned menu, header, footer, and
  sidebar contributions with exact-artifact ownership.

The public sidebar still owns a separate hard-coded list plus a dynamic category
list. The admin navigation tab supports create, enable, disable, and delete,
but sorting is a numeric `position` field and there is no placement model,
recommended-default reset, automatic history, or portable backup.

Putting menu content inside one theme's settings would couple site information
architecture to presentation. It would also make theme switching, plugin
lifecycle, backup, and Core emergency fallback unreliable.

## Decision

### Core Owns Navigation Content And Placement

Core owns the operator navigation document, stable location catalog, defaults,
validation, permissions, revisions, snapshots, import/export, and public
resolution.

The first stable public locations are:

- `public.topbar.primary`
- `public.sidebar.primary`
- `public.mobile.primary`
- `public.footer.primary`

One item may have placements in several locations. Each placement has its own
enabled state and order. Moving an item changes a placement; it does not rewrite
the underlying route or extension declaration.

The admin surface remains under **System -> Personalization -> Navigation**.
It may become a dedicated page when the editor outgrows the current tab, but it
does not move into an extension theme's settings.

### Themes Own Presentation

Themes decide how a supported location is rendered: geometry, density,
responsive behavior, overflow, icon treatment, and visual hierarchy. They do
not own, persist, or mutate the operator navigation document.

Theme settings may control presentation choices such as compact mode, sidebar
visibility, icon visibility, or category-block presentation. Switching themes
must not delete or rewrite navigation configuration. An unsupported location is
retained and shown to the operator as not rendered by the active theme.

Core emergency and Page Registry fallback surfaces must use the same resolved
navigation document and geometry contract. They must not introduce a separate
hard-coded navigation authority.

### Navigation Sources Stay Distinct

Resolved navigation composes three sources:

1. Core definitions such as Home, Categories, Tags, Search, and the dynamic
   Categories block.
2. Operator-created links.
3. Enabled plugin contributions.

Core definitions use stable keys and a code-owned recommended placement
catalog. Recommended defaults are not migration-only seed data. The stored
placement document, rather than a read-time resolver overlay, is the effective
Core placement authority: missing placements stay absent until initialization
or an explicit restore-defaults command materializes them.

The recommended V1 locations are Home/Categories/Tags in the topbar,
Home/Categories/Tags plus the dynamic Categories block in the sidebar, and
Home/Categories/Tags in the mobile menu. Footer navigation starts empty;
copyright and friend links retain their existing owners, and operators may add
footer navigation explicitly.

Operator-created links use portable stable keys that do not depend on database
sequence ids.

Plugin contributions remain owned by the active exact artifact. They are not
copied into the operator item table. Operator preferences reference the stable
`extension id + contribution id` identity and may override placement, order,
visibility, label, or icon (including explicitly suppressing it) within Host
policy. Runtime resolution still binds
the contribution to the active artifact.

The existing `forum.nav.items` path is compatibility input. The implementation
must audit its production wiring and project it through the canonical V3
Navigation/Region authority rather than creating another plugin registry.
Compatibility removal requires the normal API LTS evidence.

### Plugin Injection Is Bounded

Safe additive contributions may use `add`, `before`, and `after` behavior.
Behavior-changing `replace`, `hide`, `filter`, and `wrap` remains subject to
the existing exact-artifact trust, conflict selection, inspection, audit, and
Safe Mode rules.

Plugins cannot:

- write operator navigation tables directly;
- inject raw HTML, arbitrary JavaScript, or arbitrary DOM mutations;
- create public links into `/admin` or raw `/api` authority;
- remove Host-owned session, notification, locale, appearance, recovery, or
  other safety-critical controls through ordinary menu contribution;
- make navigation visibility substitute for route or API authorization.

Plugin disable, uninstall, trust revoke, artifact mismatch, or Safe Mode makes
the contribution unavailable. Operator placement preferences may remain inert
so a later compatible contribution can recover its position. Missing plugin
references are visible in admin and import previews; they never become broken
public links.

### Dynamic Content Is A Bounded Block

The public category list is represented as a Core dynamic navigation block,
not copied into ordinary link rows. Category names, visibility, icons, counts,
and ordering remain owned by the Forum taxonomy module.

V1 preserves the current category-list behavior. Additional block settings such
as subset selection or count limits require a later explicit contract rather
than arbitrary query or template code.

### Revisions, Defaults, And Backup Are First-Class

All accepted navigation mutations are transactional, revisioned, audited, and
bump the public-surface revision used by SSR/cache variation.

The new editor uses a batch apply command with compare-and-swap revision
checking. Reordering is persisted as one transaction rather than a stream of
single-row writes.

Every accepted apply, import, default restore, or snapshot restore stores the
previous navigation document as an automatic snapshot. V1 retains the newest
20 automatic snapshots. Restoring a snapshot first snapshots the current
document.

Recommended-default restore supports one location or all locations, previews
the change, and never silently removes plugin-owned declarations. It restores
operator placements and overrides to the Core recommendation.

Portable export uses a versioned document:

`sforum.site-navigation-backup@1`

It contains stable source references, labels, links, placements, ordering, and
enabled state, but no database sequence ids, secrets, raw executable material,
or artifact bytes. Import is validate-and-preview first, then an explicit
compare-and-swap apply in either replace or merge mode.

Navigation export is a portable configuration backup. It does not replace the
operator's PostgreSQL and secret backup process.

### Permission And Safety

V1 reuses `settings.site.manage` for navigation administration. Public reads
remain safe and expose only resolved visible items. A separate
`navigation.manage` permission is deferred until there is a demonstrated role
delegation need.

Internal links, external `http(s)` links, extension routes, and Core dynamic
blocks are typed separately. The API validates schemes, reserved paths, size,
count, nesting, and import limits. External new-tab links render with safe
opener isolation.

Visibility may be public, anonymous, authenticated, or permission-filtered.
This controls presentation only. The target API or route remains the
authoritative authorization boundary.

## Consequences

### Positive

- Operators keep one navigation configuration across theme switches.
- Topbar, sidebar, mobile, footer, Core fallback, and plugin items share one
  resolution authority.
- Plugins gain useful injection without direct database or DOM authority.
- Drag sorting, keyboard/button sorting, defaults, history, and backup have a
  stable backend contract.
- Disabled plugins and unsupported themes degrade visibly without broken
  public links or lost configuration.

### Costs

- The current flat `site_nav_items` table and immediate CRUD API need a
  compatibility migration.
- Public resolve now varies by location, locale, actor visibility, navigation
  revision, active extension snapshot, and active theme capabilities.
- Built-in themes, Core fallback, admin UI, OpenAPI, extension catalogs, cache
  invalidation, and runtime activation tests must evolve together.

## Implementation

Execute:

`knowledge/plans/2026-07-27-configurable-public-navigation-platform.md`

Do not weaken these boundaries silently during implementation. Amend this
decision first if M0 production-wiring evidence requires a different canonical
registry or compatibility strategy.
