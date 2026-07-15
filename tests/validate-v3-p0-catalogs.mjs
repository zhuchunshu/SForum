import { existsSync, readFileSync } from 'node:fs'
import { spawnSync } from 'node:child_process'
import { resolve } from 'node:path'

const root = process.cwd()

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

function load(path) {
  return JSON.parse(readFileSync(resolve(root, path), 'utf8'))
}

function unique(items, key, label) {
  const seen = new Set()
  for (const item of items) {
    const value = key(item)
    assert(value, `${label} contains an empty identity`)
    assert(!seen.has(value), `${label} contains duplicate ${value}`)
    seen.add(value)
  }
}

function normalizeRoutePath(...parts) {
  const value = parts.join('/').replaceAll(/\/+/g, '/')
  return value === '/' ? value : value.replace(/\/$/, '')
}

function validateRouteSource(route) {
  const separator = route.source.lastIndexOf(':')
  assert(separator > 0, `${route.id} source has no line number`)
  const source = route.source.slice(0, separator)
  const lineNumber = Number(route.source.slice(separator + 1))
  const lines = readFileSync(resolve(root, source), 'utf8').split('\n')
  assert(Number.isSafeInteger(lineNumber) && lineNumber > 0 && lineNumber <= lines.length, `${route.id} source line is invalid`)

  const routers = new Map([['api', '']])
  for (let index = 0; index < lineNumber - 1; index++) {
    const group = lines[index].match(/(\w+)\s*:=\s*(\w+)\.Group\("([^"]*)"\)/)
    if (group && routers.has(group[2])) routers.set(group[1], normalizeRoutePath(routers.get(group[2]), group[3]))
  }
  const registration = lines[lineNumber - 1].match(/(\w+)\.(Get|Post|Put|Patch|Delete|All)\("([^"]*)"/)
  assert(registration, `${route.id} source line is not a route registration`)
  const method = registration[2] === 'All' ? '*' : registration[2].toUpperCase()
  assert(method === route.method, `${route.id} source method ${method} does not match ${route.method}`)
  assert(routers.has(registration[1]), `${route.id} source router ${registration[1]} has no discoverable group`)
  const path = normalizeRoutePath('/api/v1', routers.get(registration[1]), registration[3])
  assert(path === route.path, `${route.id} source path ${path} does not match ${route.path}`)
}

const generated = spawnSync(process.execPath, ['scripts/v3-catalog/generate.mjs', '--check'], {
  cwd: root,
  encoding: 'utf8'
})
assert(generated.status === 0, generated.stderr || generated.stdout || 'V3 catalog drift check failed')

const routes = load('docs/extensions/v3/catalogs/routes.json')
const identities = load('docs/extensions/v3/catalog-identities.json')
assert(routes.length === 223, `route inventory must contain exactly 223 reviewed routes: ${routes.length}`)
assert(identities.routes.length === routes.length, 'reviewed route identity map must cover every current route')
unique(routes, item => item.id, 'route inventory ids')
unique(routes, item => `${item.method} ${item.path}`, 'route inventory method/path pairs')
for (const route of routes) {
  assert(/^core\.route\.[a-z0-9_.]+$/.test(route.id), `invalid route id ${route.id}`)
  assert(/^sforum\.route\.[a-z0-9_.]+@1$/.test(route.contractVersion), `invalid route contract ${route.contractVersion}`)
  assert(route.access && route.policy, `${route.id} must record access and policy`)
  assert(route.guard && ['public', 'login', 'guest', 'super_admin', 'permission_any', 'contextual'].includes(route.guard.kind), `${route.id} has no typed reviewed guard`)
  assert(Array.isArray(route.guard.permissions), `${route.id} guard permissions must be explicit`)
  if (route.guard.kind === 'permission_any') assert(route.guard.permissions.length > 0 && !route.guard.evaluatorId, `${route.id} permission guard is incomplete`)
  if (route.guard.kind === 'contextual') assert(/^core\.guard\.[a-z0-9_.]+$/.test(route.guard.evaluatorId), `${route.id} contextual evaluator is missing`)
  if (!['permission_any', 'contextual'].includes(route.guard.kind)) assert(route.guard.permissions.length === 0 && !route.guard.evaluatorId, `${route.id} metadata guard has contextual fields`)
  validateRouteSource(route)
}

const frontendTrustPolicies = new Map([
  ['core.route.extensions.frontend_confirmation', 'active super_admin challenge issuance'],
  ['core.route.extensions.grant_frontend_trust', 'active super_admin plus actor-bound exact frontend artifact confirmation'],
  ['core.route.extensions.revoke_frontend_trust', 'active super_admin trust revocation']
])
for (const [id, policy] of frontendTrustPolicies) {
  const route = routes.find(item => item.id === id)
  assert(route?.policy === policy, `${id} frontend trust policy drifted`)
  assert(route.guard.kind === 'super_admin', `${id} must retain its typed super_admin guard`)
}
const exactFrontendConfirmationRoutes = routes.filter(route => route.policy.includes('exact frontend artifact confirmation'))
assert(exactFrontendConfirmationRoutes.length === 1 && exactFrontendConfirmationRoutes[0].id === 'core.route.extensions.grant_frontend_trust', 'only frontend trust grant may claim exact artifact confirmation')

const ui = load('docs/extensions/v3/catalogs/ui-surfaces.json')
assert(ui.length >= 110, `UI inventory unexpectedly small: ${ui.length}`)
assert(identities.ui.length === ui.length, 'reviewed UI identity map must cover every current UI surface')
unique(ui, item => item.id, 'UI surface ids')
for (const item of ui) {
  assert(item.contractVersion.endsWith('@1'), `${item.id} has no v1 contract`)
  assert(existsSync(resolve(root, item.source)), `${item.id} source is missing`)
}

const backend = load('docs/extensions/v3/catalogs/backend-surfaces.json')
for (const key of ['events', 'contributionPoints', 'providerSlots', 'schedules', 'jobKinds', 'cache', 'content', 'data']) {
  assert(Array.isArray(backend[key]) && backend[key].length > 0, `backend inventory ${key} is empty`)
}

const matrix = load('docs/extensions/v3/extension-surface-matrix.json')
const families = ['routes', 'hooks', 'queries', 'adminComponents', 'publicComponents', 'identityPermissions', 'media', 'navigationRegions', 'cacheInvalidation', 'jobs', 'lifecycle']
const requiredModules = ['system', 'adminOverview', 'extensions', 'identity', 'forum', 'profile', 'attachments', 'database', 'entityMeta', 'options', 'siteChrome', 'jobs', 'moderation', 'mail', 'notifications', 'search', 'seo', 'webhooks', 'localization']
for (const module of requiredModules) assert(matrix[module], `Extension Surface Matrix is missing core module ${module}`)
assert(Object.keys(matrix).length === requiredModules.length, 'Extension Surface Matrix contains an unreviewed aggregate or unknown module')
for (const [module, values] of Object.entries(matrix)) {
  for (const family of families) {
    const value = values[family]
    assert(value, `${module} is missing matrix family ${family}`)
    assert(['open', 'planned', 'closed'].includes(value.state), `${module}.${family} has invalid state`)
    if (value.state === 'closed') assert(value.reason, `${module}.${family} is closed without a reason`)
    else assert(value.phase && value.phase.startsWith('P'), `${module}.${family} is open/planned without a phase`)
  }
}

const traceability = readFileSync(resolve(root, 'knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3-traceability.md'), 'utf8')
const traceRows = traceability.split('\n').filter(line => /^\| \d+ \| (theme|plugin) \|/.test(line))
assert(traceRows.length === 99, `traceability must contain exactly 99 rows, found ${traceRows.length}`)
assert(traceRows.filter(line => line.includes('| theme |')).length === 27, 'traceability must contain 27 theme rows')
assert(traceRows.filter(line => line.includes('| plugin |')).length === 72, 'traceability must contain 72 plugin rows')

for (const path of [
  'docs/extensions/v3/README.md',
  'docs/extensions/v3/governance.md',
  'docs/extensions/v3/performance-baseline.md'
]) {
  assert(existsSync(resolve(root, path)), `missing reviewed V3 governance artifact ${path}`)
}

console.log(`V3 P0 catalogs validated: ${routes.length} routes, ${ui.length} UI surfaces, 99 traceability rows.`)
