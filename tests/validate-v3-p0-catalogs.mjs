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

const generated = spawnSync(process.execPath, ['scripts/v3-catalog/generate.mjs', '--check'], {
  cwd: root,
  encoding: 'utf8'
})
assert(generated.status === 0, generated.stderr || generated.stdout || 'V3 catalog drift check failed')

const routes = load('docs/extensions/v3/catalogs/routes.json')
const identities = load('docs/extensions/v3/catalog-identities.json')
assert(routes.length >= 130, `route inventory unexpectedly small: ${routes.length}`)
assert(identities.routes.length === routes.length, 'reviewed route identity map must cover every current route')
unique(routes, item => item.id, 'route inventory ids')
unique(routes, item => `${item.method} ${item.path}`, 'route inventory method/path pairs')
for (const route of routes) {
  assert(/^core\.route\.[a-z0-9_.]+$/.test(route.id), `invalid route id ${route.id}`)
  assert(/^sforum\.route\.[a-z0-9_.]+@1$/.test(route.contractVersion), `invalid route contract ${route.contractVersion}`)
  assert(route.access && route.policy, `${route.id} must record access and policy`)
  assert(existsSync(resolve(root, route.source.split(':')[0])), `${route.id} source is missing`)
}

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
