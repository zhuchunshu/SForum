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
    requiredPermissions: ['settings.appearance.manage']
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
    id: '/settings/avatar',
    labelKey: 'admin.nav.avatarSettings',
    icon: 'i-lucide-user-round-cog',
    componentName: 'AdminAvatarSettings',
    requiredPermissions: ['settings.avatar.manage']
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
    requiredPermissions: ['database.manage']
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
    requiredPermissions: ['category.manage', 'tag.manage', 'forum.settings.manage'],
    permissionMode: 'any'
  },
  {
    id: '/extensions',
    labelKey: 'admin.nav.extensionOverview',
    icon: 'i-lucide-layout-dashboard',
    componentName: 'AdminExtensions',
    requiredPermissions: ['extension.view', 'extension.plugin.manage', 'extension.theme.manage', 'extension.release.manage'],
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
    // 应用商城：框架占位页，安装/检索能力后续再接
    id: '/extensions/store',
    labelKey: 'admin.nav.extensionStore',
    icon: 'i-lucide-store',
    componentName: 'AdminExtensionStore',
    requiredPermissions: ['extension.plugin.manage']
  },
  {
    id: '/extensions/settings',
    labelKey: 'admin.nav.extensionSettings',
    icon: 'i-lucide-sliders-horizontal',
    componentName: 'AdminExtensionSettings',
    requiredPermissions: ['extension.plugin.manage']
  },
  {
    id: '/extensions/events',
    labelKey: 'admin.nav.extensionEvents',
    icon: 'i-lucide-scroll-text',
    componentName: 'AdminExtensionEvents',
    requiredPermissions: ['extension.view']
  },
  {
    id: '/extensions/contributions',
    labelKey: 'admin.nav.extensionContributions',
    icon: 'i-lucide-blocks',
    componentName: 'AdminExtensionContributions',
    requiredPermissions: ['extension.view']
  },
  {
    id: '/extensions/releases',
    labelKey: 'admin.nav.extensionReleases',
    icon: 'i-lucide-rocket',
    componentName: 'AdminExtensionReleases',
    requiredPermissions: ['extension.release.manage']
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
    requiredPermissions: ['jobs.view']
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
        { type: 'page', pageId: '/forum/settings' }
      ]
    },
    {
      type: 'folder',
      labelKey: 'admin.nav.system',
      icon: 'i-lucide-settings-2',
      children: [
        { type: 'page', pageId: '/settings' },
        { type: 'page', pageId: '/settings/mail' },
        { type: 'page', pageId: '/settings/avatar' },
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
        { type: 'page', pageId: '/extensions/store' },
        { type: 'page', pageId: '/extensions/settings' },
        { type: 'page', pageId: '/extensions/events' },
        { type: 'page', pageId: '/extensions/contributions' },
        { type: 'page', pageId: '/extensions/releases' }
      ]
    },
    {
      // 运维工具不属于配置项，放在扩展管理之后
      type: 'folder',
      labelKey: 'admin.nav.operations',
      icon: 'i-lucide-server-cog',
      children: [
        { type: 'page', pageId: '/database' },
        { type: 'page', pageId: '/jobs' }
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
