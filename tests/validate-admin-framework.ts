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
const adminPageIds = adminPageDefinitions.map(page => page.id)
for (const requiredPageId of ['/', '/users', '/roles', '/permissions', '/settings', '/personalization', '/seo', '/attachments']) {
  assert(adminPageIds.includes(requiredPageId), `Admin module registry should define ${requiredPageId}`)
}
assert(adminPageDefinitions.every(page => page.icon.startsWith('i-lucide-')), 'Admin page registry should use lucide icons')
assert(adminPageDefinitions.find(page => page.id === '/permissions')?.permissionMode === 'any', 'Permission matrix should allow role.manage or user.manage')

const adminRoutesComposable = read('apps/web/app/composables/useAdminRoutes.ts')
assert(adminRoutesComposable.includes('useI18n'), 'Admin routes should read the active locale directly')
assert(adminRoutesComposable.includes('defaultLocale'), 'Admin routes should compare against the default locale')
assert(!adminRoutesComposable.includes('useLocalePath'), 'Admin routes should not rely on stale i18n path resources')

const nuxtConfig = read('apps/web/nuxt.config.ts')
assert(nuxtConfig.includes('NUXT_PUBLIC_ADMIN_ROUTE_PREFIX'), 'Nuxt config should expose the admin route prefix')
assert(nuxtConfig.includes('pages:extend'), 'Nuxt config should rewrite admin page routes')
assert(nuxtConfig.includes('rewriteAdminPageRoutes'), 'Nuxt config should use a focused admin route rewrite helper')

const envExample = read('.env.example')
const productionEnvExample = read('.env.production.example')
assert(envExample.includes('NUXT_PUBLIC_ADMIN_ROUTE_PREFIX=/control-panel'), '.env.example should document the admin route prefix')
assert(productionEnvExample.includes('NUXT_PUBLIC_ADMIN_ROUTE_PREFIX=/control-panel'), '.env.production.example should document the admin route prefix')

for (const authPage of ['apps/web/app/pages/login.vue', 'apps/web/app/pages/register.vue']) {
  const content = read(authPage)
  assert(content.includes('useAdminRoutes'), `${authPage} should use the admin route helper`)
  assert(!content.includes("? '/admin'"), `${authPage} should not hard-code the legacy /admin prefix`)
}

const adminPagePathsById: Record<string, string> = {
  '/': 'apps/web/app/pages/admin/index.vue',
  '/roles': 'apps/web/app/pages/admin/roles.vue',
  '/users': 'apps/web/app/pages/admin/users.vue',
  '/permissions': 'apps/web/app/pages/admin/permissions.vue',
  '/settings': 'apps/web/app/pages/admin/settings/index.vue',
  '/personalization': 'apps/web/app/pages/admin/personalization.vue',
  '/seo': 'apps/web/app/pages/admin/seo.vue',
  '/attachments': 'apps/web/app/pages/admin/attachments.vue'
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
assert(!adminLayout.includes("label: '系统配置'"), 'Admin layout should not hard-code Chinese sidebar labels')
assert(!adminLayout.includes('navigationItems = computed(() => ['), 'Admin layout should not hard-code sidebar menu arrays')
assert(adminModules.includes('admin.nav.personalization'), 'Admin modules should expose the personalization top-level menu')
assert(adminModules.includes('i-lucide-palette'), 'Personalization menu should use the palette icon')
assert(adminModules.includes('admin.nav.system'), 'Admin modules should expose the system navigation folder via translation key')

console.log('Admin framework validation passed.')
