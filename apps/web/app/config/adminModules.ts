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
    requiredPermissions: ['user.manage']
  },
  {
    id: '/personalization',
    labelKey: 'admin.nav.personalization',
    icon: 'i-lucide-palette',
    componentName: 'AdminPersonalization',
    requiredPermissions: ['settings.manage']
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
    requiredPermissions: ['role.manage', 'user.manage'],
    permissionMode: 'any'
  },
  {
    id: '/settings',
    labelKey: 'admin.nav.settings',
    icon: 'i-lucide-settings-2',
    componentName: 'AdminSettings',
    requiredPermissions: ['settings.manage']
  },
  {
    id: '/seo',
    labelKey: 'admin.nav.seo',
    icon: 'i-lucide-search',
    componentName: 'AdminSeo',
    requiredPermissions: ['seo.manage']
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
    id: '/extensions',
    labelKey: 'admin.nav.extensionOverview',
    icon: 'i-lucide-layout-dashboard',
    componentName: 'AdminExtensions',
    requiredPermissions: ['extension.manage']
  },
  {
    id: '/extensions/plugins',
    labelKey: 'admin.nav.extensionPlugins',
    icon: 'i-lucide-plug',
    componentName: 'AdminExtensionPlugins',
    requiredPermissions: ['extension.manage']
  },
  {
    id: '/extensions/themes',
    labelKey: 'admin.nav.extensionThemes',
    icon: 'i-lucide-palette',
    componentName: 'AdminExtensionThemes',
    requiredPermissions: ['extension.manage']
  },
  {
    id: '/extensions/settings',
    labelKey: 'admin.nav.extensionSettings',
    icon: 'i-lucide-sliders-horizontal',
    componentName: 'AdminExtensionSettings',
    requiredPermissions: ['extension.manage']
  },
  {
    id: '/extensions/events',
    labelKey: 'admin.nav.extensionEvents',
    icon: 'i-lucide-scroll-text',
    componentName: 'AdminExtensionEvents',
    requiredPermissions: ['extension.manage']
  }
] as const satisfies readonly AdminPageDefinition[]

export const adminSidebarNavigation = [
  [
    { type: 'page', pageId: ADMIN_DASHBOARD_PAGE_ID },
    { type: 'page', pageId: '/personalization' },
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
      labelKey: 'admin.nav.system',
      icon: 'i-lucide-settings-2',
      children: [
        { type: 'page', pageId: '/settings' },
        { type: 'page', pageId: '/seo' }
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
        { type: 'page', pageId: '/extensions/settings' },
        { type: 'page', pageId: '/extensions/events' }
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
