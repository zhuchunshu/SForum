import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'

const root = process.cwd()

function file(path: string) {
  return resolve(root, path)
}

function read(path: string) {
  return readFileSync(file(path), 'utf8')
}

function assert(condition: unknown, message: string): asserts condition {
  if (!condition) {
    throw new Error(message)
  }
}

const adminRouteModule = file('apps/web/app/utils/adminRoutePrefix.ts')
assert(existsSync(adminRouteModule), 'Missing admin route prefix helper')

const routes = await import(pathToFileURL(adminRouteModule).href)

assert(routes.DEFAULT_ADMIN_ROUTE_PREFIX === '/control-panel', 'Default admin prefix must be /control-panel')
assert(routes.normalizeAdminRoutePrefix(undefined) === '/control-panel', 'Undefined prefix should use the default')
assert(routes.normalizeAdminRoutePrefix('') === '/control-panel', 'Blank prefix should use the default')
assert(routes.normalizeAdminRoutePrefix('staff') === '/staff', 'Prefix should gain a leading slash')
assert(routes.normalizeAdminRoutePrefix('/staff/') === '/staff', 'Prefix should lose trailing slashes')
assert(routes.joinAdminRoutePath('/staff', 'roles') === '/staff/roles', 'Child paths should join cleanly')
assert(routes.joinAdminRoutePath('/staff', '/') === '/staff', 'Root child path should not add a trailing slash')
assert(routes.normalizeAdminChildPath('roles') === '/roles', 'Admin child paths should normalize to leading slash')
assert(routes.resolveAdminRouteChildPath('/control-panel', '/control-panel') === '/', 'Admin root route should resolve to dashboard id')
assert(routes.resolveAdminRouteChildPath('/control-panel', '/en-US/control-panel/roles') === '/roles', 'Localized admin route should resolve to child id')
assert(routes.resolveAdminRouteChildPath('/control-panel', '/forum') === null, 'Non-admin routes should not resolve to admin child ids')

assert(existsSync(file('apps/web/app/composables/useAdminRoutes.ts')), 'Missing runtime admin route composable')
assert(existsSync(file('apps/web/app/layouts/admin.vue')), 'Missing admin dashboard layout')
assert(existsSync(file('apps/web/app/config/adminModules.ts')), 'Missing low-code admin module registry')
assert(existsSync(file('apps/web/app/composables/useAdminPage.ts')), 'Missing admin page registration composable')

const adminModulesModule = await import(pathToFileURL(file('apps/web/app/config/adminModules.ts')).href)
const adminPageDefinitions = adminModulesModule.adminPageDefinitions as Array<{
  id: string
  labelKey: string
  icon: string
  componentName: string
  requiredPermissions?: string[]
  permissionMode?: string
}>
const isAdminNavigationEntryActive = adminModulesModule.isAdminNavigationEntryActive as (entry: unknown, pageId: string) => boolean
const shouldOpenAdminNavigationEntry = adminModulesModule.shouldOpenAdminNavigationEntry as (entry: unknown, pageId: string) => boolean
const isExtensionAdminPageId = adminModulesModule.isExtensionAdminPageId as (pageId: string) => boolean
const adminPageIds = adminPageDefinitions.map(page => page.id)
for (const requiredPageId of [
  '/',
  '/users',
  '/roles',
  '/permissions',
  '/settings',
  '/personalization',
  '/seo',
  '/database',
  '/attachments',
  '/forum/categories',
  '/forum/tags',
  '/forum/settings',
  '/extensions',
  '/extensions/plugins',
  '/extensions/themes',
  '/extensions/settings',
  '/extensions/events'
]) {
  assert(adminPageIds.includes(requiredPageId), `Admin module registry should define ${requiredPageId}`)
}
assert(adminPageDefinitions.every(page => page.icon.startsWith('i-lucide-')), 'Admin page registry should use lucide icons')
assert(adminPageDefinitions.find(page => page.id === '/permissions')?.permissionMode === 'any', 'Permission matrix should allow role.manage or user.manage')
for (const extensionPageId of ['/extensions', '/extensions/plugins', '/extensions/themes', '/extensions/settings', '/extensions/events']) {
  assert(adminPageDefinitions.find(page => page.id === extensionPageId)?.requiredPermissions?.includes('extension.manage'), `${extensionPageId} should require extension.manage`)
}

const adminRoutesComposable = read('apps/web/app/composables/useAdminRoutes.ts')
assert(adminRoutesComposable.includes('useI18n'), 'Admin routes should read the active locale directly')
assert(adminRoutesComposable.includes('defaultLocale'), 'Admin routes should compare against the default locale')
assert(!adminRoutesComposable.includes('useLocalePath'), 'Admin routes should not rely on stale i18n path resources')

const nuxtConfig = read('apps/web/nuxt.config.ts')
assert(nuxtConfig.includes('NUXT_PUBLIC_ADMIN_ROUTE_PREFIX'), 'Nuxt config should expose the admin route prefix')
assert(nuxtConfig.includes('pages:extend'), 'Nuxt config should rewrite admin page routes')
assert(nuxtConfig.includes('rewriteAdminPageRoutes'), 'Nuxt config should use a focused admin route rewrite helper')

const adminExtensionsHelper = read('apps/web/app/utils/adminExtensions.ts')
for (const requiredThemeState of ["'activate'", "'queued'", "'building'", "'activating'", "'failed'"]) {
  assert(adminExtensionsHelper.includes(requiredThemeState), `Theme action state should include ${requiredThemeState}`)
}
assert(adminExtensionsHelper.includes('themeRelease?.status'), 'Theme action state should inspect latest theme release status')
assert(!adminExtensionsHelper.includes("'verifyOnly'"), 'Theme action state should not keep the old verify-only runtime placeholder')

const envExample = read('.env.example')
const productionEnvExample = read('.env.production.example')
assert(envExample.includes('NUXT_PUBLIC_ADMIN_ROUTE_PREFIX=/control-panel'), '.env.example should document the admin route prefix')
assert(productionEnvExample.includes('NUXT_PUBLIC_ADMIN_ROUTE_PREFIX=/control-panel'), '.env.production.example should document the admin route prefix')

for (const authPage of [
  'extensions/builtin/themes/sforum-default/layer/app/pages/login.vue',
  'extensions/builtin/themes/sforum-default/layer/app/pages/register.vue'
]) {
  const content = read(authPage)
  assert(content.includes('useAdminRoutes') || content.includes('useAuthReturnNavigation'), `${authPage} should use the admin route or centralized auth return helper`)
  assert(!content.includes("? '/admin'"), `${authPage} should not hard-code the legacy /admin prefix`)
}
assert(!existsSync(file('apps/web/app/pages/login.vue')), 'Login page should be owned by the default theme layer')
assert(!existsSync(file('apps/web/app/pages/register.vue')), 'Register page should be owned by the default theme layer')

const adminPagePathsById: Record<string, string> = {
  '/': 'apps/web/app/pages/admin/index.vue',
  '/roles': 'apps/web/app/pages/admin/roles.vue',
  '/users': 'apps/web/app/pages/admin/users.vue',
  '/permissions': 'apps/web/app/pages/admin/permissions.vue',
  '/settings': 'apps/web/app/pages/admin/settings/index.vue',
  '/settings/mail': 'apps/web/app/pages/admin/settings/mail.vue',
  '/settings/avatar': 'apps/web/app/pages/admin/settings/avatar.vue',
  '/personalization': 'apps/web/app/pages/admin/personalization.vue',
  '/moderation': 'apps/web/app/pages/admin/moderation.vue',
  '/seo': 'apps/web/app/pages/admin/seo.vue',
  '/database': 'apps/web/app/pages/admin/database.vue',
  '/attachments': 'apps/web/app/pages/admin/attachments.vue',
  '/forum/categories': 'apps/web/app/pages/admin/forum/categories.vue',
  '/forum/tags': 'apps/web/app/pages/admin/forum/tags.vue',
  '/forum/settings': 'apps/web/app/pages/admin/forum/settings.vue',
  '/extensions': 'apps/web/app/pages/admin/extensions/index.vue',
  '/extensions/plugins': 'apps/web/app/pages/admin/extensions/plugins.vue',
  '/extensions/themes': 'apps/web/app/pages/admin/extensions/themes.vue',
  '/extensions/settings': 'apps/web/app/pages/admin/extensions/settings.vue',
  '/extensions/events': 'apps/web/app/pages/admin/extensions/events.vue',
  '/extensions/contributions': 'apps/web/app/pages/admin/extensions/contributions.vue',
  '/extensions/releases': 'apps/web/app/pages/admin/extensions/releases.vue',
  '/jobs': 'apps/web/app/pages/admin/jobs.vue',
  '/search': 'apps/web/app/pages/admin/search.vue'
}

for (const page of adminPageDefinitions) {
  const adminPage = adminPagePathsById[page.id]
  assert(adminPage, `Admin module registry points to an unknown page id ${page.id}`)
  const content = read(adminPage)
  assert(content.includes("layout: 'admin'"), `${adminPage} should use the admin layout`)
  assert(content.includes('UDashboardToolbar'), `${adminPage} should render inside the Nuxt UI dashboard shell`)
  assert(content.includes('useAdminPage'), `${adminPage} should use low-code admin page registration`)
  assert(content.includes(`useAdminPage('${page.id}')`), `${adminPage} should register itself by page id only`)
  assert(content.includes(`name: '${page.componentName}'`), `${adminPage} should keep component name aligned with registry`)
  assert(!content.includes('useAdminTabs'), `${adminPage} should not wire tabs manually`)
  assert(!content.includes('openTab('), `${adminPage} should not hard-code tab metadata`)
}

const adminLayout = read('apps/web/app/layouts/admin.vue')
const adminModules = read('apps/web/app/config/adminModules.ts')
const adminFooterPath = 'apps/web/app/components/SFAdminFooter.vue'
assert(existsSync(file(adminFooterPath)), 'Admin shell should have a dedicated global footer component')
const adminFooter = read(adminFooterPath)
for (const requiredComponent of [
  'UDashboardGroup',
  'UDashboardSidebar',
  'UDashboardPanel',
  'UNavigationMenu',
  'UIcon'
]) {
  assert(adminLayout.includes(requiredComponent), `Admin layout should use ${requiredComponent}`)
}

assert(adminLayout.includes('i-lucide-'), 'Admin layout should use Nuxt Icon lucide icons')
assert(adminLayout.includes('adminSidebarNavigation'), 'Admin layout should build sidebar navigation from the module registry')
assert(adminLayout.includes('canAccessAdminPage'), 'Admin layout should hide registry pages by frontend-visible permissions')
assert(adminLayout.includes('<SFAdminFooter />'), 'Admin layout should render the dedicated global footer')
assert(!adminLayout.includes('<SFFooter'), 'Admin layout should not reuse the public site footer')
assert(adminFooter.includes('data-testid="sforum-admin-footer"'), 'Admin footer should expose a stable test id')
assert(adminFooter.includes('data-testid="sforum-admin-footer-left"'), 'Admin footer should expose a stable left content area')
assert(adminFooter.includes('data-testid="sforum-admin-footer-right"'), 'Admin footer should expose a stable right content area')
assert(adminFooter.includes('admin.shell.footerCopyright'), 'Admin footer should use admin shell i18n copy')
assert(adminFooter.includes('admin.shell.footerProductSummary'), 'Admin footer should render official product summary copy')
assert(adminFooter.includes('justify-between'), 'Admin footer content should be split across left and right sides')
assert(adminFooter.includes('new Date().getFullYear()'), 'Admin footer should render the current copyright year')
assert(!adminLayout.includes("label: '系统配置'"), 'Admin layout should not hard-code Chinese sidebar labels')
assert(!adminLayout.includes('navigationItems = computed(() => ['), 'Admin layout should not hard-code sidebar menu arrays')
assert(adminModules.includes('admin.nav.personalization'), 'Admin modules should expose the personalization top-level menu')
assert(adminModules.includes('i-lucide-palette'), 'Personalization menu should use the palette icon')
assert(adminModules.includes('admin.nav.system'), 'Admin modules should expose the system navigation folder via translation key')
assert(adminModules.includes('admin.nav.extensions'), 'Admin modules should expose the extension manager menu')
assert(adminModules.includes('i-lucide-blocks'), 'Extension manager menu should use the blocks icon')
assert(adminModules.includes('admin.nav.extensionOverview'), 'Admin modules should expose extension overview submenu')
assert(adminModules.includes('admin.nav.extensionPlugins'), 'Admin modules should expose plugin manager submenu')
assert(adminModules.includes('admin.nav.extensionThemes'), 'Admin modules should expose theme manager submenu')
assert(adminModules.includes('admin.nav.extensionSettings'), 'Admin modules should expose extension settings submenu')
assert(adminModules.includes('admin.nav.extensionEvents'), 'Admin modules should expose extension events submenu')

const adminSidebarNavigation = adminModulesModule.adminSidebarNavigation as Array<Array<unknown>>
const firstSidebarGroup = adminSidebarNavigation[0] as Array<{
  type: string
  labelKey?: string
  children?: Array<{ type: string, pageId?: string }>
}>
const forumFolder = firstSidebarGroup.find(entry => entry.type === 'folder' && entry.labelKey === 'admin.nav.forum')
assert(forumFolder, 'Admin sidebar should keep a forum folder')
assert(
  !firstSidebarGroup.some(entry => entry.type === 'page' && entry.pageId === '/moderation'),
  'Moderation management should live under the forum folder, not the top-level sidebar'
)
assert(forumFolder.children?.some(entry => entry.pageId === '/moderation'), 'Forum folder should contain the moderation page')
assert(
  forumFolder.children?.map(entry => entry.pageId).join(',') === '/moderation,/forum/categories,/forum/tags,/forum/settings',
  'Forum folder should keep the approved submenu order'
)
const systemFolder = firstSidebarGroup.find(entry => entry.type === 'folder' && entry.labelKey === 'admin.nav.system')
assert(systemFolder, 'Admin sidebar should keep a system folder')
assert(
  !firstSidebarGroup.some(entry => entry.type === 'page' && entry.pageId === '/personalization'),
  'Personalization should live under the system folder, not the top-level sidebar'
)
assert(systemFolder.children?.some(entry => entry.pageId === '/personalization'), 'System folder should contain the personalization page')
assert(
  systemFolder.children?.map(entry => entry.pageId).join(',') === '/settings,/settings/avatar,/personalization,/seo,/database,/search,/jobs',
  'System folder should keep the approved settings submenu order'
)
assert(!systemFolder.children?.some(entry => entry.pageId === '/extensions'), 'System folder should not contain the extension overview page')
const extensionFolder = firstSidebarGroup.find(entry => entry.type === 'folder' && entry.labelKey === 'admin.nav.extensions')
assert(extensionFolder, 'Admin sidebar should expose extensions as an independent folder')
assert(extensionFolder.children?.map(entry => entry.pageId).join(',') === '/extensions,/extensions/plugins,/extensions/themes,/extensions/settings,/extensions/events,/extensions/contributions,/extensions/releases', 'Extension folder should keep the approved submenu order')
const extensionEventsPage = read('apps/web/app/pages/admin/extensions/events.vue')
assert(extensionEventsPage.includes('data-testid="admin-extension-events-page"'), 'Extension event log page should expose a stable page wrapper for layout checks')
assert(extensionEventsPage.includes('data-testid="admin-extension-events-page" class="min-w-0 shrink-0"'), 'Extension event log page wrapper should not shrink inside the admin flex scroll container')
assert(typeof isAdminNavigationEntryActive === 'function', 'Admin navigation should expose an active-state helper')
assert(typeof shouldOpenAdminNavigationEntry === 'function', 'Admin navigation should expose an initial-open helper')
assert(typeof isExtensionAdminPageId === 'function', 'Admin navigation should identify dynamic extension admin pages')
assert(isExtensionAdminPageId('/extensions/sforum.default-theme/pages/about'), 'Dynamic extension admin page ids should be recognized')
assert(!isExtensionAdminPageId('/extensions/themes'), 'Static extension manager pages should not be treated as dynamic extension admin pages')
assert(isAdminNavigationEntryActive(extensionFolder, '/extensions/themes'), 'Extension folder should be active when one of its child pages is active')
assert(isAdminNavigationEntryActive(extensionFolder, '/extensions/sforum.default-theme/pages/about'), 'Extension folder should be active for dynamic extension admin pages')
assert(!isAdminNavigationEntryActive(systemFolder, '/extensions/themes'), 'System folder should not be active for extension child pages')
assert(shouldOpenAdminNavigationEntry(extensionFolder, '/extensions/themes'), 'Active sidebar folders should open initially')
assert(shouldOpenAdminNavigationEntry(extensionFolder, '/extensions/sforum.default-theme/pages/about'), 'Extension folder should open for dynamic extension admin pages')
assert(!shouldOpenAdminNavigationEntry(systemFolder, '/extensions/themes'), 'Inactive sidebar folders should stay collapsed by default')
assert(firstSidebarGroup
  .filter(entry => entry.type === 'folder')
  .every(entry => entry.defaultOpen !== true), 'Top-level admin sidebar folders should not all be configured as default-open')
assert(adminLayout.includes('active: isActive'), 'Admin layout should pass active state to top-level folder items')
assert(adminLayout.includes('shouldOpenAdminNavigationEntry(entry, currentAdminPageId)'), 'Admin layout should open only the active/default folder initially')
assert(adminLayout.includes('adminTabs.activateTab(tabId)'), 'Admin layout should activate existing custom tabs from the current route')
assert(adminLayout.includes('openExtensionRoutePlaceholderTab(tabId)'), 'Admin layout should open route-backed placeholder tabs for dynamic extension pages')
assert(adminLayout.includes('class="min-h-0 flex-1 overflow-y-auto pr-1"'), 'Admin sidebar navigation should scroll independently when there are many menu items')

console.log('Admin framework validation passed.')
