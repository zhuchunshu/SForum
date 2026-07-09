# Extension Platform v2

SForum should give site operators a WordPress-level extension experience while
giving developers a modern, controlled, maintainable platform.

The goal is not to recreate WordPress' PHP include execution model. SForum is a
Go API plus Nuxt SSR application, so extensions must run through explicit host
contracts: manifests, subprocess RPC, provider slots, events, controlled
routes, extension settings, admin pages, build checks, and rollback paths.

## Product Promise

### Site Operator View

- Upload a ZIP package.
- Let SForum inspect the manifest before anything runs.
- Review requested permissions, routes, provider slots, admin pages,
  migrations, and risk notes.
- Enable a plugin or activate a theme only after checks pass.
- Configure safe defaults and restore recommended defaults in one click.
- See logs, health status, event delivery attempts, and failure reasons.
- Disable or roll back when an extension breaks.

### Developer View

- Use the SForum SDK and CLI to create plugins and themes.
- Declare permissions, admin pages, routes, events, provider slots, settings,
  migrations, and compatibility in `sforum.extension.json`.
- Package and publish extensions without relying on core monkey-patching.
- Debug locally with a runtime that mirrors production boundaries.

### Core View

- Core is the host framework, not a dumping ground for optional verticals.
- Plugins cannot override arbitrary core routes, patch services, trust raw
  cookies, or bypass API policy checks.
- Core exposes stable contracts. Extensions declare intent. SForum renders or
  executes only explicitly registered capabilities.

## Current State

Plugin foundations are already present:

- ZIP upload.
- `sforum.extension.json` manifest validation.
- Plugin enable and disable lifecycle.
- HashiCorp go-plugin subprocess runtime.
- Health/preflight checks.
- Declared plugin route proxying.
- Event and hook foundation.
- Plugin settings pages.
- Event delivery record foundation.

Theme foundations are intentionally narrower:

- Theme packages can be uploaded, installed, and verified.
- Only the protected built-in `sforum.default-theme` is applied.
- Uploaded themes cannot yet be activated.
- Nuxt still statically extends the default theme layer.

Extension points are still early:

- Current events cover lifecycle, registration, `topic.before_create`,
  topic/comment creation, and attachment upload.
- Provider slots exist as a skeleton and need real business flows.

## Target Product Loop

Extension Platform v2 is complete only when the full loop works:

- Install-time manifest review is readable before activation.
- Enable-time review shows permissions, routes, providers, admin pages,
  migrations, and risks.
- Plugin startup performs runtime checks and exposes actionable failure
  reasons.
- Runtime logs, event deliveries, and extension errors are visible in admin.
- Disabling a plugin stops its process and removes its runtime routes.
- Failed plugin enable rolls back to disabled.
- Failed theme activation rolls back to the previous active theme.
- Settings have safe defaults and a one-click reset path.

## Admin Management Rules

Plugins and themes may both register admin pages, admin routes, and sidebar
menu entries, but sidebar exposure is opt-in.

- Do not add extension pages to the admin sidebar unless the manifest explicitly
  marks a page with `menu: true`.
- The `Manage` action must support an extension-defined destination.
- `Manage` must always resolve to an in-admin route, never a direct external
  URL.
- A generated system detail page may exist, but it must not be forced as the
  only management entry.
- Extension page content must support more than fixed `about` and `settings`
  pages.
- When a plugin is disabled, runtime capabilities are off, but base management
  pages remain available for inspection, configuration, and re-enabling.
- When a theme is inactive, it may expose settings and management pages, but it
  must not take over public UI or inject sidebar entries by default.

## Manifest Direction

The existing top-level `adminPages` shape can be migrated or compatibility
mapped into an `admin` object:

```json
{
  "admin": {
    "entry": "/settings",
    "pages": [
      {
        "path": "/settings",
        "label": "设置",
        "view": "settings",
        "menu": false
      },
      {
        "path": "/dashboard",
        "label": "控制台",
        "view": "content",
        "menu": true,
        "icon": "i-lucide-layout-dashboard",
        "order": 100
      }
    ]
  }
}
```

`admin.entry` is the destination for the extension list `Manage` action.
`admin.pages[]` declares in-admin pages mounted under a host-owned namespace
such as `/control-panel/extensions/{id}/pages/settings`.

`Manage` resolution should use this order:

1. Use `admin.entry` when present.
2. Otherwise use a declared `/settings` page when present.
3. Otherwise use the first declared admin page.
4. Otherwise use the generated system detail page.

Sidebar resolution should use this order:

- Only pages with `menu: true` appear in the sidebar.
- `menu` defaults to `false`.
- Disabled plugins do not contribute runtime sidebar items.
- Inactive themes do not contribute public UI or automatic sidebar items.

## Admin Page Capability Layers

### Layer 1: Host-Generated Pages

The host renders safe built-in pages such as about, settings, permission
summary, installation notes, and generated configuration forms.

### Layer 2: Extension Content Pages

The manifest may declare Markdown or other safe content that the host renders.
This is appropriate for instructions, simple dashboards, and basic status
pages.

### Layer 3: Extension-Owned Frontend Pages

Future support may allow packaged frontend assets, iframes, web components, or
remote components. This layer must be gated by CSP, permissions, route
registration, resource validation, and version compatibility checks. It should
not be the first implementation path.

## SDK Direction

The SDK may eventually offer a developer-friendly wrapper:

```ts
registerAdminMenu({
  extensionId: 'demo.plugin',
  path: '/dashboard',
  label: 'Demo 控制台',
  icon: 'i-lucide-plug',
  order: 100
})
```

The source of truth should still be manifest declarations. SDK calls can
generate or validate manifest entries, but core should build menus from the
manifest.

## Typed Contributions

Typed contributions borrow the useful ergonomics from old SForum's `Itf`
pattern without copying the global PHP hook model.

Plugins may declare ordered `contributions[]` in `sforum.extension.json`.
Core owns the known contribution-point catalog, validates each payload, and
resolves effective runtime contributions from enabled plugins only. Consumers
remain explicit: the owning module decides how to interpret a payload and what
is safe to render.

The first contribution point is `forum.topic.actions`:

```json
{
  "routes": [
    {"path": "/topic-actions/bookmark", "methods": ["POST"], "access": "login"}
  ],
  "contributions": [
    {
      "point": "forum.topic.actions",
      "id": "demo.bookmark",
      "order": 200,
      "label": {"zh-CN": "收藏", "en-US": "Bookmark"},
      "icon": "i-lucide-bookmark",
      "payload": {
        "type": "extensionRoute",
        "method": "POST",
        "path": "/topic-actions/bookmark",
        "confirm": true
      }
    }
  ]
}
```

Contribution descriptors cannot contain raw HTML, frontend component paths,
callbacks, core API paths, external URLs, or route overrides. Topic action
execution still goes through `/api/v1/extensions/{extensionId}/*`, so plugin
route access checks remain authoritative.

## Roadmap

### Phase 1: Make Plugins Truly Usable

Use a real mail provider plugin as the first vertical slice.

Goals:

- Upload a plugin package.
- Show what the manifest intends to do before enabling it.
- Show requested permissions and risk notes.
- Start the backend subprocess.
- Run health checks.
- Proxy declared plugin routes.
- Persist and reset plugin settings.
- Show logs, event deliveries, and failure reasons.
- Stop the process and remove runtime routes on disable.
- Roll back automatically when enable fails.

The mail provider plugin should exercise provider selection, secrets,
settings, no-op fallback, error visibility, admin UX, SDK documentation, and
scaffolding.

### Phase 2: Make Provider Slots First-Class

Prioritize these slots:

- `mail.provider`
- `notification.channel`
- `payment.provider`
- `search.provider`
- `attachment.storage.provider`
- `editor.sanitizer.provider`
- `auth.risk.provider`

Core owns provider-neutral lifecycle and shared state. Vendor behavior stays in
plugins. Examples: SMTP mail, Stripe payments, Alipay payments, Feishu
notifications, Meilisearch search, and alternative storage providers.

### Phase 3: Theme Activation Pipeline

Theme activation must be a build-and-switch workflow:

1. Upload theme.
2. Validate manifest.
3. Build Nuxt in a temporary location.
4. Run health checks.
5. Generate a preview address.
6. Let an administrator confirm activation.
7. Atomically switch the active frontend artifact.
8. Roll back to the previous theme on failure.

The operator experience should feel like activating a WordPress theme; the
implementation should behave like a deployment pipeline.

### Phase 4: Lifecycle Completion

Add:

- Upgrade.
- Rollback.
- Uninstall.
- Plugin database migrations.
- Data retention and cleanup policy.
- Plugin dependency checks.
- SForum version compatibility checks.
- Signature and trusted source metadata.
- Marketplace metadata.

### Phase 5: Developer Experience

Provide:

- `sforum make:plugin`
- `sforum make:theme`
- Plugin SDK.
- Local debugging mode.
- Packaging command.
- Manifest documentation.
- Permission documentation.
- Event documentation.
- Provider documentation.
- Example plugins.

Example plugin priority:

- SMTP mail provider.
- Webhook notification channel.
- Payment sandbox provider.
- Custom topic review plugin.

## Guiding Line

Operator experience like WordPress; runtime mechanics unlike WordPress. Core
provides capabilities, developers declare intent, and SForum renders or runs
only what is explicitly registered.
