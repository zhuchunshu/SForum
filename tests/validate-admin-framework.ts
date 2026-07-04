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

assert(existsSync(file('apps/web/app/composables/useAdminRoutes.ts')), 'Missing runtime admin route composable')
assert(existsSync(file('apps/web/app/layouts/admin.vue')), 'Missing admin dashboard layout')

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

for (const adminPage of [
  'apps/web/app/pages/admin/index.vue',
  'apps/web/app/pages/admin/roles.vue',
  'apps/web/app/pages/admin/users.vue',
  'apps/web/app/pages/admin/permissions.vue'
]) {
  const content = read(adminPage)
  assert(content.includes("layout: 'admin'"), `${adminPage} should use the admin layout`)
  assert(content.includes('UDashboardToolbar'), `${adminPage} should render inside the Nuxt UI dashboard shell`)
}

const adminLayout = read('apps/web/app/layouts/admin.vue')
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

console.log('Admin framework validation passed.')
