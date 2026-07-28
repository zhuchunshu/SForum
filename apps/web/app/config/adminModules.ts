export const ADMIN_DASHBOARD_PAGE_ID = '/'

export type AdminPermissionMode = 'all' | 'any'

export type AdminPageDefinition = {
  id: string
  labelKey: string
  icon: string
  componentName: string
  closable?: boolean
  badgeKey?: string
  requiredPermissions?: string[]
  permissionMode?: AdminPermissionMode
  /** 仅专业模式下出现在侧栏（默认关闭，面向日常运营隐藏扩展高级工具）。 */
  professionalMode?: boolean
  /** 仅运维管理开关打开时出现在侧栏（默认关闭）。 */
  operationsMode?: boolean
}

export type AdminNavigationPageEntry = {
  type: 'page'
  pageId: string
  badgeKey?: string
}

export type AdminNavigationFolderEntry = {
  type: 'folder'
  labelKey: string
  icon: string
  defaultOpen?: boolean
  children: readonly AdminNavigationEntry[]
}

export type AdminNavigationForumHomeEntry = {
  type: 'forum-home'
  labelKey: string
  icon: string
}

export type AdminNavigationEntry =
  | AdminNavigationPageEntry
  | AdminNavigationFolderEntry
  | AdminNavigationForumHomeEntry

export type AdminNavigationGroup = readonly AdminNavigationEntry[]

export const adminPageDefinitions = [
  {
    id: ADMIN_DASHBOARD_PAGE_ID,
    labelKey: 'admin.nav.dashboard',
    icon: 'i-lucide-layout-dashboard',
    componentName: 'AdminIndex',
    closable: false
  },
  {
    id: '/users',
    labelKey: 'admin.nav.userManagement',
    icon: 'i-lucide-contact',
    componentName: 'AdminUsers',
    requiredPermissions: ['user.view', 'user.manage'],
    permissionMode: 'any'
  },
  {
    id: '/personalization',
    labelKey: 'admin.nav.personalization',
    icon: 'i-lucide-palette',
    componentName: 'AdminPersonalization',
    // 外观与前台壳合并到同一页；任一权限即可进入，页内再按 tab 过滤。
    requiredPermissions: ['settings.appearance.manage', 'settings.site.manage'],
    permissionMode: 'any'
  },
  {
    // 保留页面定义供旧链接 / 重定向；不挂侧栏。
    id: '/site-chrome',
    labelKey: 'admin.nav.siteChrome',
    icon: 'i-lucide-panel-top',
    componentName: 'AdminSiteChromeRedirect',
    requiredPermissions: ['settings.site.manage', 'settings.appearance.manage'],
    permissionMode: 'any'
  },
  {
    id: '/roles',
    labelKey: 'admin.nav.userGroups',
    icon: 'i-lucide-shield-check',
    componentName: 'AdminRoles',
    badgeKey: 'admin.nav.rolesBadge',
    requiredPermissions: ['role.manage']
  },
  {
    id: '/permissions',
    labelKey: 'admin.nav.permissionManagement',
    icon: 'i-lucide-shield-check',
    componentName: 'AdminPermissions',
    requiredPermissions: ['role.manage', 'user.view', 'user.manage'],
    permissionMode: 'any'
  },
  {
    id: '/settings',
    labelKey: 'admin.nav.settings',
    icon: 'i-lucide-settings-2',
    componentName: 'AdminSettings',
    requiredPermissions: ['settings.site.manage']
  },
  {
    id: '/settings/mail',
    labelKey: 'admin.nav.mailSettings',
    icon: 'i-lucide-mail',
    componentName: 'AdminMailSettings',
    requiredPermissions: ['settings.mail.manage']
  },
  {
    id: '/settings/notifications',
    labelKey: 'admin.nav.notificationSettings',
    icon: 'i-lucide-bell-ring',
    componentName: 'AdminNotificationSettings',
    requiredPermissions: ['settings.notifications.manage']
  },
  {
    id: '/settings/login-methods',
    labelKey: 'admin.nav.loginMethods',
    icon: 'i-lucide-log-in',
    componentName: 'AdminLoginMethods',
    requiredPermissions: ['identity.provider.manage']
  },
  {
    id: '/settings/avatar',
    labelKey: 'admin.nav.avatarSettings',
    icon: 'i-lucide-user-round-cog',
    componentName: 'AdminAvatarSettings',
    requiredPermissions: ['settings.avatar.manage']
  },
  {
    id: '/settings/features',
    labelKey: 'admin.nav.featureFlags',
    icon: 'i-lucide-toggle-left',
    componentName: 'AdminFeatureFlags',
    requiredPermissions: ['settings.site.manage']
  },
  {
    id: '/entity-meta',
    labelKey: 'admin.nav.entityMeta',
    icon: 'i-lucide-tags',
    componentName: 'AdminEntityMeta',
    requiredPermissions: ['entity_meta.manage', 'settings.manage'],
    permissionMode: 'any'
  },
  {
    id: '/moderation',
    labelKey: 'admin.nav.moderation',
    icon: 'i-lucide-shield-alert',
    componentName: 'AdminModeration',
    requiredPermissions: ['moderation.manage']
  },
  {
    id: '/seo',
    labelKey: 'admin.nav.seo',
    icon: 'i-lucide-search',
    componentName: 'AdminSeo',
    requiredPermissions: ['seo.manage']
  },
  {
    id: '/database',
    labelKey: 'admin.nav.database',
    icon: 'i-lucide-database',
    componentName: 'AdminDatabase',
    requiredPermissions: ['database.manage'],
    operationsMode: true
  },
  {
    id: '/attachments',
    labelKey: 'admin.nav.attachments',
    icon: 'i-lucide-paperclip',
    componentName: 'AdminAttachments',
    requiredPermissions: ['attachment.settings.manage', 'attachment.manage'],
    permissionMode: 'any'
  },
  {
    id: '/forum/categories',
    labelKey: 'admin.nav.forumCategories',
    icon: 'i-lucide-folder-tree',
    componentName: 'AdminForumCategories',
    requiredPermissions: ['category.manage']
  },
  {
    id: '/forum/tags',
    labelKey: 'admin.nav.forumTags',
    icon: 'i-lucide-tags',
    componentName: 'AdminForumTags',
    requiredPermissions: ['tag.manage']
  },
  {
    id: '/forum/settings',
    labelKey: 'admin.nav.forumSettings',
    icon: 'i-lucide-sliders-horizontal',
    componentName: 'AdminForumSettings',
    requiredPermissions: ['category.manage', 'tag.manage', 'forum.settings.manage', 'search.manage'],
    permissionMode: 'any'
  },
  {
    id: '/forum/content',
    labelKey: 'admin.nav.forumContent',
    icon: 'i-lucide-files',
    componentName: 'AdminForumContent',
    requiredPermissions: ['topic.edit_any', 'topic.revision.view_any', 'post.edit_any', 'post.revision.view_any'],
    permissionMode: 'any'
  },
  {
    id: '/extensions',
    labelKey: 'admin.nav.extensionOverview',
    icon: 'i-lucide-layout-dashboard',
    componentName: 'AdminExtensions',
    requiredPermissions: ['extension.view', 'extension.plugin.manage', 'extension.theme.manage'],
    permissionMode: 'any'
  },
  {
    id: '/extensions/plugins',
    labelKey: 'admin.nav.extensionPlugins',
    icon: 'i-lucide-plug',
    componentName: 'AdminExtensionPlugins',
    requiredPermissions: ['extension.plugin.manage', 'extension.view'],
    permissionMode: 'any'
  },
  {
    id: '/extensions/themes',
    labelKey: 'admin.nav.extensionThemes',
    icon: 'i-lucide-palette',
    componentName: 'AdminExtensionThemes',
    requiredPermissions: ['extension.theme.manage', 'extension.view'],
    permissionMode: 'any'
  },
  {
    // 旧商城入口：保留定义供重定向，不挂侧栏
    id: '/extensions/store',
    labelKey: 'admin.nav.extensionStore',
    icon: 'i-lucide-store',
    componentName: 'AdminExtensionStoreRedirect',
    requiredPermissions: ['extension.plugin.manage', 'extension.theme.manage'],
    permissionMode: 'any'
  },
  {
    id: '/extensions/store/themes',
    labelKey: 'admin.nav.extensionStoreThemes',
    icon: 'i-lucide-palette',
    componentName: 'AdminExtensionStoreThemes',
    requiredPermissions: ['extension.theme.manage', 'extension.view'],
    permissionMode: 'any'
  },
  {
    id: '/extensions/store/plugins',
    labelKey: 'admin.nav.extensionStorePlugins',
    icon: 'i-lucide-plug',
    componentName: 'AdminExtensionStorePlugins',
    requiredPermissions: ['extension.plugin.manage', 'extension.view'],
    permissionMode: 'any'
  },
  {
    id: '/extensions/settings',
    labelKey: 'admin.nav.extensionSettings',
    icon: 'i-lucide-sliders-horizontal',
    componentName: 'AdminExtensionSettings',
    requiredPermissions: ['extension.plugin.manage'],
    professionalMode: true
  },
  {
    id: '/extensions/events',
    labelKey: 'admin.nav.extensionEvents',
    icon: 'i-lucide-scroll-text',
    componentName: 'AdminExtensionEvents',
    requiredPermissions: ['extension.view'],
    professionalMode: true
  },
  {
    id: '/extensions/contributions',
    labelKey: 'admin.nav.extensionContributions',
    icon: 'i-lucide-blocks',
    componentName: 'AdminExtensionContributions',
    requiredPermissions: ['extension.view'],
    professionalMode: true
  },
  {
    id: '/extensions/pages',
    labelKey: 'admin.nav.extensionPages',
    icon: 'i-lucide-layout-template',
    componentName: 'AdminExtensionPages',
    requiredPermissions: ['extension.view', 'extension.theme.manage'],
    permissionMode: 'any',
    professionalMode: true
  },
  {
    id: '/extensions/route-providers',
    labelKey: 'admin.nav.extensionRouteProviders',
    icon: 'i-lucide-route',
    componentName: 'AdminExtensionRouteProviders',
    requiredPermissions: ['extension.view'],
    professionalMode: true
  },
  {
    id: '/extensions/route-inspector',
    labelKey: 'admin.nav.extensionRouteInspector',
    icon: 'i-lucide-scan-search',
    componentName: 'AdminExtensionRouteInspector',
    requiredPermissions: ['extension.view'],
    professionalMode: true
  },
  {
    id: '/extensions/cache-inspector',
    labelKey: 'admin.nav.extensionCacheInspector',
    icon: 'i-lucide-database-zap',
    componentName: 'AdminExtensionCacheInspector',
    requiredPermissions: ['extension.view'],
    professionalMode: true
  },
  {
    id: '/extensions/asset-inspector',
    labelKey: 'admin.nav.extensionAssetInspector',
    icon: 'i-lucide-package',
    componentName: 'AdminExtensionAssetInspector',
    requiredPermissions: ['extension.view'],
    professionalMode: true
  },
  {
    id: '/extensions/template-inspector',
    labelKey: 'admin.nav.extensionTemplateInspector',
    icon: 'i-lucide-layout-template',
    componentName: 'AdminExtensionTemplateInspector',
    requiredPermissions: ['extension.view'],
    professionalMode: true
  },
  {
    id: '/extensions/component-inspector',
    labelKey: 'admin.nav.extensionComponentInspector',
    icon: 'i-lucide-boxes',
    componentName: 'AdminExtensionComponentInspector',
    requiredPermissions: ['extension.view'],
    professionalMode: true
  },
  {
    id: '/extensions/navigation-inspector',
    labelKey: 'admin.nav.extensionNavigationInspector',
    icon: 'i-lucide-map',
    componentName: 'AdminExtensionNavigationInspector',
    requiredPermissions: ['extension.view'],
    professionalMode: true
  },
  {
    id: '/extensions/registry-catalogs',
    labelKey: 'admin.nav.extensionRegistryCatalogs',
    icon: 'i-lucide-library',
    componentName: 'AdminExtensionRegistryCatalogs',
    requiredPermissions: ['extension.view'],
    professionalMode: true
  },
  {
    id: '/extensions/provider-slots',
    labelKey: 'admin.nav.extensionProviderSlots',
    icon: 'i-lucide-waypoints',
    componentName: 'AdminExtensionProviderSlots',
    requiredPermissions: ['extension.view'],
    professionalMode: true
  },
  {
    id: '/search',
    labelKey: 'admin.nav.search',
    icon: 'i-lucide-search',
    componentName: 'AdminSearch',
    requiredPermissions: ['search.manage']
  },
  {
    id: '/jobs',
    labelKey: 'admin.nav.jobs',
    icon: 'i-lucide-list-checks',
    componentName: 'AdminJobs',
    requiredPermissions: ['jobs.view'],
    operationsMode: true
  },
  {
    id: '/schedules',
    labelKey: 'admin.nav.schedules',
    icon: 'i-lucide-calendar-clock',
    componentName: 'AdminSchedules',
    requiredPermissions: ['jobs.view'],
    operationsMode: true
  },
  {
    id: '/webhooks',
    labelKey: 'admin.nav.webhooks',
    icon: 'i-lucide-webhook',
    componentName: 'AdminWebhooks',
    requiredPermissions: ['settings.manage', 'settings.site.manage'],
    permissionMode: 'any',
    operationsMode: true
  }
] as const satisfies readonly AdminPageDefinition[]

export const adminSidebarNavigation = [
  [
    { type: 'page', pageId: ADMIN_DASHBOARD_PAGE_ID },
    {
      type: 'folder',
      labelKey: 'admin.nav.userPermission',
      icon: 'i-lucide-user-cog',
      children: [
        { type: 'page', pageId: '/users' },
        { type: 'page', pageId: '/roles' },
        { type: 'page', pageId: '/permissions' }
      ]
    },
    {
      type: 'folder',
      labelKey: 'admin.nav.forum',
      icon: 'i-lucide-message-square-text',
      children: [
        { type: 'page', pageId: '/moderation' },
        { type: 'page', pageId: '/forum/categories' },
        { type: 'page', pageId: '/forum/tags' },
        { type: 'page', pageId: '/forum/settings' },
        { type: 'page', pageId: '/forum/content' }
      ]
    },
    {
      type: 'folder',
      labelKey: 'admin.nav.system',
      icon: 'i-lucide-settings-2',
      children: [
        { type: 'page', pageId: '/settings' },
        { type: 'page', pageId: '/settings/mail' },
        { type: 'page', pageId: '/settings/notifications' },
        { type: 'page', pageId: '/settings/login-methods' },
        { type: 'page', pageId: '/settings/avatar' },
        { type: 'page', pageId: '/settings/features' },
        { type: 'page', pageId: '/entity-meta' },
        { type: 'page', pageId: '/personalization' },
        { type: 'page', pageId: '/seo' },
        { type: 'page', pageId: '/search' }
      ]
    },
    {
      type: 'folder',
      labelKey: 'admin.nav.extensions',
      icon: 'i-lucide-blocks',
      children: [
        { type: 'page', pageId: '/extensions' },
        { type: 'page', pageId: '/extensions/plugins' },
        { type: 'page', pageId: '/extensions/themes' },
        { type: 'page', pageId: '/extensions/pages' },
        { type: 'page', pageId: '/extensions/route-providers' },
        { type: 'page', pageId: '/extensions/route-inspector' },
        { type: 'page', pageId: '/extensions/cache-inspector' },
        { type: 'page', pageId: '/extensions/asset-inspector' },
        { type: 'page', pageId: '/extensions/template-inspector' },
        { type: 'page', pageId: '/extensions/component-inspector' },
        { type: 'page', pageId: '/extensions/navigation-inspector' },
        { type: 'page', pageId: '/extensions/registry-catalogs' },
        { type: 'page', pageId: '/extensions/provider-slots' },
        { type: 'page', pageId: '/extensions/settings' },
        { type: 'page', pageId: '/extensions/events' },
        { type: 'page', pageId: '/extensions/contributions' }
      ]
    },
    {
      // 应用商城独立一级菜单：主题 / 插件货架，与本地扩展管理分离
      type: 'folder',
      labelKey: 'admin.nav.extensionStore',
      icon: 'i-lucide-store',
      children: [
        { type: 'page', pageId: '/extensions/store/themes' },
        { type: 'page', pageId: '/extensions/store/plugins' }
      ]
    },
    {
      // 运维工具不属于配置项，放在扩展管理之后
      type: 'folder',
      labelKey: 'admin.nav.operations',
      icon: 'i-lucide-server-cog',
      children: [
        { type: 'page', pageId: '/database' },
        { type: 'page', pageId: '/jobs' },
        { type: 'page', pageId: '/schedules' },
        { type: 'page', pageId: '/webhooks' }
      ]
    },
    { type: 'page', pageId: '/attachments' }
  ],
  [
    {
      type: 'forum-home',
      labelKey: 'admin.nav.forumHome',
      icon: 'i-lucide-house'
    }
  ]
] as const satisfies readonly AdminNavigationGroup[]

export function normalizeAdminPageId(id?: string | null) {
  const value = `${id || ''}`.trim()

  if (!value || value === '/') {
    return ADMIN_DASHBOARD_PAGE_ID
  }

  return `/${value.replace(/^\/+|\/+$/g, '')}`
}

export function isExtensionAdminPageId(id?: string | null) {
  const pageId = normalizeAdminPageId(id)
  return /^\/extensions\/[^/]+\/pages(?:\/|$)/.test(pageId)
}

export function findAdminPageDefinition(id: string): AdminPageDefinition | undefined {
  const pageId = normalizeAdminPageId(id)
  return adminPageDefinitions.find(page => page.id === pageId)
}

export function requireAdminPageDefinition(id: string): AdminPageDefinition {
  const page = findAdminPageDefinition(id)

  if (!page) {
    throw new Error(`Unknown admin page: ${id}`)
  }

  return page
}

export function isAdminNavigationEntryActive(entry: AdminNavigationEntry, pageId?: string | null): boolean {
  const activePageId = normalizeAdminPageId(pageId)

  if (entry.type === 'page') {
    return normalizeAdminPageId(entry.pageId) === activePageId
  }

  if (entry.type === 'folder') {
    if (entry.labelKey === 'admin.nav.extensions' && isExtensionAdminPageId(activePageId)) {
      return true
    }

    return entry.children.some(child => isAdminNavigationEntryActive(child, activePageId))
  }

  return false
}

export function shouldOpenAdminNavigationEntry(entry: AdminNavigationEntry, pageId?: string | null): boolean {
  return entry.type === 'folder' && (entry.defaultOpen === true || isAdminNavigationEntryActive(entry, pageId))
}

export function canAccessAdminPage(page: AdminPageDefinition, can: (permission: string) => boolean) {
  const permissions = page.requiredPermissions ?? []

  if (permissions.length === 0) {
    return true
  }

  if (page.permissionMode === 'any') {
    return permissions.some(permission => can(permission))
  }

  return permissions.every(permission => can(permission))
}

/** 侧栏可见性偏好（系统高级设置 Modal 开关，默认均关闭）。 */
export type AdminNavVisibility = {
  professionalMode: boolean
  operationsMode: boolean
}

/** 侧栏是否应展示该页面：权限 + 高级设置偏好。 */
export function shouldShowAdminPageInNav(
  page: AdminPageDefinition,
  can: (permission: string) => boolean,
  visibility: AdminNavVisibility | boolean
) {
  // 兼容旧调用：第三个参数曾是 professionalMode 布尔值。
  const flags: AdminNavVisibility = typeof visibility === 'boolean'
    ? { professionalMode: visibility, operationsMode: true }
    : visibility

  if (!canAccessAdminPage(page, can)) {
    return false
  }
  if (page.professionalMode && !flags.professionalMode) {
    return false
  }
  if (page.operationsMode && !flags.operationsMode) {
    return false
  }
  return true
}
