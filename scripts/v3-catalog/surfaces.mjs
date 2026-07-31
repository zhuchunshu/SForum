export const matrixFamilies = [
  'routes',
  'hooks',
  'queries',
  'adminComponents',
  'publicComponents',
  'identityPermissions',
  'media',
  'navigationRegions',
  'cacheInvalidation',
  'jobs',
  'lifecycle'
]

const planned = phase => ({ state: 'planned', phase })
const open = (contract, phase) => ({ state: 'open', contract, phase })
const closed = reason => ({ state: 'closed', reason })

export const extensionSurfaceMatrix = {
  system: {
    routes: open('health/ready and future Route Registry defaults', 'P6'),
    hooks: planned('P7'),
    queries: planned('P7'),
    adminComponents: open('admin.overview.* descriptor surfaces', 'P7'),
    publicComponents: closed('Pre-plugin health and recovery output is Host-owned and non-overridable.'),
    identityPermissions: closed('Safe mode and CLI recovery use host-operator authority outside web RBAC.'),
    media: closed('System health owns no media objects.'),
    navigationRegions: planned('P9'),
    cacheInvalidation: planned('P11'),
    jobs: open('worker heartbeat and audit cleanup schedules', 'P7'),
    lifecycle: closed('Pre-plugin boot, safe mode, and CLI recovery cannot be replaced by extensions.')
  },
  systemUpdates: {
    routes: open('admin release status and forced-check routes', 'P6'),
    hooks: closed('Release checks expose status only and do not publish product lifecycle events.'),
    queries: closed('Release metadata is fetched from the configured GitHub-compatible source, not the Query Registry.'),
    adminComponents: open('Core-owned update status and mirror-source settings tab', 'P7'),
    publicComponents: closed('Release status and source configuration are operator-only.'),
    identityPermissions: open('admin.access for status; settings.site.manage for outbound checks and source configuration', 'P7'),
    media: closed('Release checks own no media lifecycle.'),
    navigationRegions: closed('The update signal is fixed admin shell chrome, not an extension navigation region.'),
    cacheInvalidation: open('bounded in-process cache keyed by current build and release source', 'P11'),
    jobs: closed('Checks are request-driven and use no durable background job.'),
    lifecycle: closed('Core update discovery does not install, execute, or replace artifacts.')
  },
  adminOverview: {
    routes: open('admin overview aggregate route', 'P6'),
    hooks: planned('P7'),
    queries: planned('P7'),
    adminComponents: open('admin.dashboard.widgets descriptors', 'P7'),
    publicComponents: closed('Administrative runtime telemetry is not public presentation.'),
    identityPermissions: open('admin.access', 'P7'),
    media: closed('The overview aggregates metadata and owns no media lifecycle.'),
    navigationRegions: open('admin dashboard navigation', 'P9'),
    cacheInvalidation: planned('P11'),
    jobs: open('worker heartbeat and queue-lag aggregates', 'P7'),
    lifecycle: planned('P4')
  },
  extensions: {
    routes: open('namespaced proxy and Page Registry administration', 'P6'),
    hooks: open('extension.enabled/disabled and lifecycle audit', 'P7'),
    queries: planned('P7'),
    adminComponents: open('settings schema and trusted admin component fallback', 'P7'),
    publicComponents: open('Page Registry L0/L1', 'P9'),
    identityPermissions: open('extension.view/plugin.manage/theme.manage', 'P7'),
    media: planned('P10'),
    navigationRegions: open('manifest admin navigation', 'P9'),
    cacheInvalidation: planned('P11'),
    jobs: open('extension.plugin_job and event delivery', 'P7'),
    lifecycle: open('Protocol V2 enable/disable/upgrade/uninstall lifecycle', 'P4')
  },
  identity: {
    // OAuth callback + browser session issue/renew/destroy remain Host-owned and
    // closed to Route Registry replacement; providers only return assertions.
    // Closed deliberately (security/integrity): GET /auth/providers/{id}/callback,
    // AuthSession issue/renew/destroy, PKCE verifier, callback state store, and
    // subject HMAC — plugins may only return bounded assertions.
    routes: open('auth, external providers, users, roles, tokens, sessions; OAuth callback + browser session issue/renew/destroy closed to Route Registry replacement (Host-owned state/PKCE/session integrity)', 'P6'),
    hooks: open('user.before_register and user.registered', 'P7'),
    queries: planned('P7'),
    adminComponents: open('Login Methods host page + SFExtensionSettingsRenderer for auth plugins', 'P7'),
    publicComponents: open('Host islands: login/register provider buttons, callback feedback, account-security linked accounts', 'P9'),
    identityPermissions: open('RBAC, overrides, sessions, risk policy, identity.provider.manage', 'P7'),
    media: open('avatar attachment integration', 'P10'),
    navigationRegions: planned('P9'),
    cacheInvalidation: planned('P11'),
    jobs: open('identity.cleanup_sessions', 'P7'),
    lifecycle: open('external auth activation CAS; disable/uninstall/Safe Mode/ForceDrain/artifact drift remove effective availability; links retained inert; no auto-activation on SyncBuiltins', 'P7')
  },
  forum: {
    routes: open('forum read/write and taxonomy routes', 'P6'),
    hooks: open('topic/comment/category/tag action and filter catalog', 'P7'),
    queries: planned('P7'),
    adminComponents: planned('P7'),
    publicComponents: open('descriptor contributions and Page Registry pages', 'P9'),
    identityPermissions: open('topic/post/category/tag permission policies', 'P7'),
    media: open('content attachment references', 'P10'),
    navigationRegions: open('forum.nav.items descriptors', 'P9'),
    cacheInvalidation: open('forum generation cache', 'P11'),
    jobs: open('search indexing and idle-topic schedule', 'P7'),
    lifecycle: planned('P4')
  },
  profile: {
    routes: open('public and self profile/avatar routes', 'P6'),
    hooks: planned('P7'),
    queries: planned('P7'),
    adminComponents: planned('P7'),
    publicComponents: open('profile tabs descriptor and page contract', 'P9'),
    identityPermissions: open('self ownership plus attachment.upload', 'P7'),
    media: open('avatar source and attachment references', 'P10'),
    navigationRegions: planned('P9'),
    cacheInvalidation: planned('P11'),
    jobs: closed('Current profile writes have no independent durable work.'),
    lifecycle: planned('P4')
  },
  attachments: {
    routes: open('upload/read/admin/settings routes', 'P6'),
    hooks: open('attachment.before_upload/uploaded', 'P7'),
    queries: planned('P7'),
    adminComponents: planned('P7'),
    publicComponents: planned('P9'),
    identityPermissions: open('attachment.upload/manage/settings.manage', 'P7'),
    media: open('storage provider slot and authoritative read policy', 'P10'),
    navigationRegions: planned('P9'),
    cacheInvalidation: planned('P11'),
    jobs: open('attachments.cleanup_orphans', 'P7'),
    lifecycle: planned('P4')
  },
  database: {
    routes: open('read-only database inspector and CSV export', 'P6'),
    hooks: planned('P7'),
    queries: open('catalog metadata and bounded table reads', 'P7'),
    adminComponents: planned('P7'),
    publicComponents: closed('Database inspection is an admin-only security boundary.'),
    identityPermissions: open('database.manage', 'P7'),
    media: closed('Database inspection owns no media transformation or delivery.'),
    navigationRegions: open('admin database navigation', 'P9'),
    cacheInvalidation: closed('Inspector reads authoritative PostgreSQL state and does not cache rows.'),
    jobs: closed('Current read-only inspection and streaming CSV require no durable job.'),
    lifecycle: planned('P4')
  },
  entityMeta: {
    routes: open('definition and entity-value routes', 'P6'),
    hooks: open('entity_meta.updated', 'P7'),
    queries: planned('P7'),
    adminComponents: planned('P7'),
    publicComponents: planned('P9'),
    identityPermissions: open('entity_meta.manage plus entity ownership policy', 'P7'),
    media: closed('Current EAV values do not own attachment/media references.'),
    navigationRegions: open('admin entity-meta navigation', 'P9'),
    cacheInvalidation: planned('P11'),
    jobs: closed('Current metadata writes are synchronous and have no durable projection.'),
    lifecycle: planned('P4')
  },
  options: {
    routes: open('public/admin options and feature flags', 'P6'),
    hooks: planned('P7'),
    queries: planned('P7'),
    adminComponents: open('host settings forms and reset flows', 'P7'),
    publicComponents: closed('Options provide data; presentation belongs to pages/themes.'),
    identityPermissions: open('option-owner permission dispatch', 'P7'),
    media: open('brand/SEO attachment references', 'P10'),
    navigationRegions: open('site chrome structured records', 'P9'),
    cacheInvalidation: open('short-TTL cache with write invalidation', 'P11'),
    jobs: closed('Option writes are synchronous and currently require no durable work.'),
    lifecycle: planned('P4')
  },
  siteChrome: {
    routes: open('public/admin navigation, announcements, and friend links', 'P6'),
    hooks: planned('P7'),
    queries: planned('P7'),
    adminComponents: open('personalization site-chrome forms and lists', 'P7'),
    publicComponents: open('navbar, announcement, footer/link consumers', 'P9'),
    identityPermissions: open('settings.site.manage', 'P7'),
    media: open('brand attachment references are coordinated through Options/Attachments', 'P10'),
    navigationRegions: open('current structured navigation; target Navigation/Region Registry', 'P9'),
    cacheInvalidation: planned('P11'),
    jobs: closed('Current chrome updates are synchronous and need no durable job.'),
    lifecycle: planned('P4')
  },
  jobs: {
    routes: open('queue and schedule workbench', 'P6'),
    hooks: planned('P7'),
    queries: planned('P7'),
    adminComponents: open('jobs table/action/detail surfaces', 'P7'),
    publicComponents: closed('Queue operations are not a public presentation surface.'),
    identityPermissions: open('jobs.view/jobs.manage', 'P7'),
    media: closed('Queue runtime owns no media policy.'),
    navigationRegions: planned('P9'),
    cacheInvalidation: planned('P11'),
    jobs: open('River kinds, queues, retries, schedules', 'P7'),
    lifecycle: open('worker/runtime drain ownership', 'P4')
  },
  moderation: {
    routes: open('reports, policy, workbench, decisions', 'P6'),
    hooks: planned('P7'),
    queries: planned('P7'),
    adminComponents: planned('P7'),
    publicComponents: planned('P9'),
    identityPermissions: open('moderation.manage/review/view_ip', 'P7'),
    media: open('publication-aware attachment reads', 'P10'),
    navigationRegions: planned('P9'),
    cacheInvalidation: open('publication decisions invalidate forum/search reads', 'P11'),
    jobs: open('notification and search projections', 'P7'),
    lifecycle: planned('P4')
  },
  mail: {
    routes: open('mail provider/policy/delivery/test routes', 'P6'),
    hooks: planned('P7'),
    queries: planned('P7'),
    adminComponents: planned('P7'),
    publicComponents: closed('Mail transport has no public presentation; user notices belong to Notifications.'),
    identityPermissions: open('settings.mail.manage and compatibility parent', 'P7'),
    media: closed('Mail transport owns no upload/media processing.'),
    navigationRegions: planned('P9'),
    cacheInvalidation: planned('P11'),
    jobs: open('mail.deliver', 'P7'),
    lifecycle: open('provider disable fallback', 'P4')
  },
  notifications: {
    routes: open('recipient inbox/read/test routes', 'P6'),
    hooks: planned('P7'),
    queries: planned('P7'),
    adminComponents: planned('P7'),
    publicComponents: open('notification inbox and unread navigation', 'P9'),
    identityPermissions: open('current recipient ownership and settings policy', 'P7'),
    media: closed('Inbox payloads do not own upload/media processing.'),
    navigationRegions: open('navbar unread entry', 'P9'),
    cacheInvalidation: planned('P11'),
    jobs: open('notification fanout and mail projection', 'P7'),
    lifecycle: planned('P4')
  },
  search: {
    routes: open('public search and admin reindex routes', 'P6'),
    hooks: planned('P7'),
    queries: planned('P7'),
    adminComponents: planned('P7'),
    publicComponents: planned('P9'),
    identityPermissions: open('search.manage and publication visibility policy', 'P7'),
    media: closed('Search indexes media metadata only through reviewed document schemas.'),
    navigationRegions: planned('P9'),
    cacheInvalidation: open('index enqueue/delete and rebuild invalidation', 'P11'),
    jobs: open('search.index_topic/search.delete_topic/reindex', 'P7'),
    lifecycle: planned('P4')
  },
  seo: {
    routes: open('sitemap entries and admin SEO asset routes', 'P6'),
    hooks: planned('P7'),
    queries: planned('P7'),
    adminComponents: planned('P7'),
    publicComponents: open('metadata and structured-data consumers', 'P9'),
    identityPermissions: open('seo.manage plus public visibility policy', 'P7'),
    media: open('SEO image attachment integration', 'P10'),
    navigationRegions: planned('P9'),
    cacheInvalidation: open('SEO/theme/content revision invalidation', 'P11'),
    jobs: closed('Current sitemap/metadata resolution is synchronous; search projection is owned by Search.'),
    lifecycle: planned('P4')
  },
  webhooks: {
    routes: open('admin endpoint/delivery catalog and inbound source route', 'P6'),
    hooks: open('observe-event bridge to outbound deliveries', 'P7'),
    queries: planned('P7'),
    adminComponents: planned('P7'),
    publicComponents: closed('Webhook configuration and deliveries are operator-only.'),
    identityPermissions: open('settings.manage/settings.site.manage', 'P7'),
    media: closed('Webhook payloads cannot act as an upload/media pipeline.'),
    navigationRegions: open('admin webhook navigation', 'P9'),
    cacheInvalidation: closed('Endpoint/delivery state is read from authoritative PostgreSQL.'),
    jobs: open('webhook.deliver', 'P7'),
    lifecycle: planned('P4')
  },
  localization: {
    routes: closed('Locale negotiation is middleware; language-pack routes are not implemented yet.'),
    hooks: planned('P7'),
    queries: planned('P7'),
    adminComponents: planned('P7'),
    publicComponents: planned('P9'),
    identityPermissions: planned('P7'),
    media: closed('Localization owns package catalogs, not user media.'),
    navigationRegions: planned('P9'),
    cacheInvalidation: planned('P11'),
    jobs: planned('P7'),
    lifecycle: planned('P4')
  }
}

export const currentBackendSurfaces = {
  cache: [
    ['forum.category_groups', 'generation + 60s'],
    ['forum.categories', 'generation + 60s'],
    ['forum.tags', 'generation + 60s'],
    ['forum.topic', 'topic id + 30s'],
    ['forum.topic_list', 'generation + 15s'],
    ['options.runtime', 'short TTL + write invalidation']
  ],
  content: [
    ['forum.topic', 'topic metadata + shared post content'],
    ['forum.comment', 'tree reply + shared post content'],
    ['forum.post', 'raw, sanitized HTML, plain text, editor/render versions'],
    ['forum.post_revision', 'source snapshot'],
    ['forum.category', 'two-level taxonomy category'],
    ['forum.tag', 'moderated topic taxonomy'],
    ['entity_meta.user', 'host-owned EAV fields'],
    ['entity_meta.topic', 'host-owned EAV fields']
  ],
  data: [
    ['identity.user', 'users, credentials, roles, permissions, sessions'],
    ['profile.user', 'public profile and avatar reference'],
    ['attachments.attachment', 'metadata, references, provider object'],
    ['moderation.case', 'reports, settings, immutable decisions'],
    ['notifications.notification', 'recipient-owned inbox'],
    ['mail.delivery', 'durable delivery state'],
    ['extensions.extension', 'package, version, settings, trust, events'],
    ['jobs.river_job', 'River-owned durable work'],
    ['options.web_option', 'typed runtime key/value'],
    ['site.chrome', 'navigation, announcements, friend links'],
    ['webhooks.endpoint', 'endpoint and delivery state'],
    ['search.document', 'rebuildable Meilisearch projection']
  ]
}
