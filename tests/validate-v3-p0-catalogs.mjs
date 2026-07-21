import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { spawnSync } from 'node:child_process'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'

const root = process.cwd()

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

function load(path) {
  return JSON.parse(readFileSync(resolve(root, path), 'utf8'))
}

function runGenerator(...arguments_) {
  return spawnSync(process.execPath, ['scripts/v3-catalog/generate.mjs', ...arguments_], {
    cwd: root,
    encoding: 'utf8'
  })
}

function runGit(cwd, ...arguments_) {
  const result = spawnSync('git', arguments_, { cwd, encoding: 'utf8' })
  assert(result.status === 0, result.stderr || result.stdout || `git ${arguments_.join(' ')} failed`)
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

const generated = runGenerator('--check')
assert(generated.status === 0, generated.stderr || generated.stdout || 'V3 catalog drift check failed')

const routes = load('docs/extensions/v3/catalogs/routes.json')
const identities = load('docs/extensions/v3/catalog-identities.json')
const retiredIdentities = load('docs/extensions/v3/catalog-retired-identities.json')

function assertIdentityMutationRejected(name, mutate, expected) {
  const directory = mkdtempSync(join(tmpdir(), 'sforum-v3-identities-'))
  try {
    const changed = structuredClone(identities)
    const changedRetired = structuredClone(retiredIdentities)
    mutate(changed, changedRetired)
    const identitiesPath = join(directory, 'catalog-identities.json')
    const retiredPath = join(directory, 'catalog-retired-identities.json')
    writeFileSync(identitiesPath, `${JSON.stringify(changed, null, 2)}\n`)
    writeFileSync(retiredPath, `${JSON.stringify(changedRetired, null, 2)}\n`)
    const result = runGenerator('--check', `--identities=${identitiesPath}`, `--retired-identities=${retiredPath}`)
    const output = `${result.stderr || ''}${result.stdout || ''}`
    assert(result.status !== 0, `${name} mutation unexpectedly passed generation`)
    assert(output.includes(expected), `${name} mutation failed for the wrong reason:\n${output}`)
  } finally {
    rmSync(directory, { recursive: true, force: true })
  }
}

function assertIdentityMutationAccepted(name, mutate) {
  const directory = mkdtempSync(join(tmpdir(), 'sforum-v3-identities-'))
  try {
    const changed = structuredClone(identities)
    const changedRetired = structuredClone(retiredIdentities)
    mutate(changed, changedRetired)
    const identitiesPath = join(directory, 'catalog-identities.json')
    const retiredPath = join(directory, 'catalog-retired-identities.json')
    const retiredCatalogPath = join(directory, 'ui-retired-identities.json')
    writeFileSync(identitiesPath, `${JSON.stringify(changed, null, 2)}\n`)
    writeFileSync(retiredPath, `${JSON.stringify(changedRetired, null, 2)}\n`)
    writeFileSync(retiredCatalogPath, `${JSON.stringify(changedRetired, null, 2)}\n`)
    const result = runGenerator('--check', `--identities=${identitiesPath}`, `--retired-identities=${retiredPath}`, `--retired-catalog=${retiredCatalogPath}`)
    assert(result.status === 0, `${name} mutation failed:\n${result.stderr || result.stdout || ''}`)
  } finally {
    rmSync(directory, { recursive: true, force: true })
  }
}

const retiredUIIdentity = {
  id: 'core.component.retired.legacy_banner',
  contractVersion: 'sforum.component.retired.legacy_banner@1',
  kind: 'component',
  owners: ['public'],
  state: 'retired',
  source: 'apps/web/app/components/LegacyBanner.vue'
}
const retiredUITombstone = {
  id: retiredUIIdentity.id,
  contractVersion: retiredUIIdentity.contractVersion
}

function assertRetiredIdentityDeletionDetected() {
  const directory = mkdtempSync(join(tmpdir(), 'sforum-v3-retired-deletion-'))
  try {
    const retiredDirectory = join(directory, 'docs/extensions/v3')
    const catalogDirectory = join(retiredDirectory, 'catalogs')
    const retiredPath = join(retiredDirectory, 'catalog-retired-identities.json')
    const retiredCatalogPath = join(catalogDirectory, 'ui-retired-identities.json')
    const withTombstone = { schemaVersion: 1, ui: [retiredUITombstone] }
    mkdirSync(catalogDirectory, { recursive: true })
    runGit(directory, 'init', '--quiet')
    runGit(directory, 'config', 'user.name', 'SForum Catalog Test')
    runGit(directory, 'config', 'user.email', 'catalog-test@sforum.invalid')
    writeFileSync(retiredPath, `${JSON.stringify(withTombstone, null, 2)}\n`)
    writeFileSync(retiredCatalogPath, `${JSON.stringify(withTombstone, null, 2)}\n`)
    runGit(directory, 'add', '.')
    runGit(directory, 'commit', '--quiet', '-m', 'record retired component')
    writeFileSync(retiredPath, `${JSON.stringify(retiredIdentities, null, 2)}\n`)
    writeFileSync(retiredCatalogPath, `${JSON.stringify(retiredIdentities, null, 2)}\n`)
    runGit(directory, 'add', '.')
    runGit(directory, 'commit', '--quiet', '-m', 'delete and regenerate retired catalogs')

    const result = runGenerator('--check', `--retired-identities=${retiredPath}`, `--retired-catalog=${retiredCatalogPath}`, `--retired-history-root=${directory}`)
    const output = `${result.stderr || ''}${result.stdout || ''}`
    assert(result.status !== 0 && output.includes('append-only retired UI tombstone'), `retired identity deletion plus regeneration passed:\n${output}`)
  } finally {
    rmSync(directory, { recursive: true, force: true })
  }
}

assert(identities.schemaVersion === 2, 'reviewed identity map must use explicit UI ownership schema v2')
assert(retiredIdentities.schemaVersion === 1 && Array.isArray(retiredIdentities.ui), 'reviewed retired identity ledger has an invalid schema')
assertIdentityMutationAccepted('retired UI exclusion', (changed, changedRetired) => {
  changed.ui.push(retiredUIIdentity)
  changedRetired.ui.push(retiredUITombstone)
})
assertRetiredIdentityDeletionDetected()
assertIdentityMutationRejected('UI id collision', changed => {
  changed.ui[1].id = changed.ui[0].id
}, 'duplicate core.component')
assertIdentityMutationRejected('retired UI id reuse', (changed, changedRetired) => {
  changedRetired.ui.push(retiredUITombstone)
  changed.ui[0].id = retiredUIIdentity.id
}, 'reuses a retired id')
assertIdentityMutationRejected('retired UI contract reuse', (changed, changedRetired) => {
  changedRetired.ui.push(retiredUITombstone)
  changed.ui[0].contractVersion = retiredUIIdentity.contractVersion
}, 'reuses retired contract')
assertIdentityMutationRejected('UI owner drift', changed => {
  const item = changed.ui.find(candidate => candidate.id === 'core.component.admin.sfadmin_footer')
  item.owners = ['public']
}, 'ownership drift')
assertIdentityMutationRejected('UI source drift', changed => {
  changed.ui[0].source = 'apps/web/app/components/Moved.vue'
}, 'new or moved UI surface')

assert(routes.length === 249, `route inventory must contain exactly 249 reviewed routes: ${routes.length}`)
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
const retiredUI = load('docs/extensions/v3/catalogs/ui-retired-identities.json')
// 127 baseline + 22 Host body islands / public chrome from V3 presentation ownership.
assert(ui.length === 149, `UI inventory must contain exactly 149 reviewed surfaces: ${ui.length}`)
assert(JSON.stringify(retiredUI) === JSON.stringify(retiredIdentities), 'generated retired UI reservation catalog drifted from its reviewed ledger')
const activeUIIdentities = identities.ui.filter(item => item.state === 'active')
assert(activeUIIdentities.length === ui.length, 'active reviewed UI identity map must cover every current UI surface')
unique(ui, item => item.id, 'UI surface ids')
unique(ui, item => item.contractVersion, 'UI surface contracts')
unique(ui, item => item.source, 'UI surface sources')
unique(identities.ui, item => item.id, 'reviewed UI ids')
unique(identities.ui, item => item.contractVersion, 'reviewed UI contracts')
unique(retiredIdentities.ui, item => item.id, 'retired UI tombstone ids')
unique(retiredIdentities.ui, item => item.contractVersion, 'retired UI tombstone contracts')
unique(activeUIIdentities, item => item.source, 'active reviewed UI sources')
for (const item of identities.ui) assert(['active', 'retired'].includes(item.state), `${item.id} has invalid identity state`)
const reviewedUI = new Map(activeUIIdentities.map(item => [item.source, item]))
const ownershipCounts = { public: 0, admin: 0, shared: 0 }
for (const item of ui) {
  assert(/^core\.component\.[a-z0-9_.-]+$/.test(item.id), `invalid component id ${item.id}`)
  assert(/^sforum\.component\.[a-z0-9_.-]+@1$/.test(item.contractVersion), `${item.id} has no v1 contract`)
  assert(['page', 'component'].includes(item.kind), `${item.id} has invalid kind ${item.kind}`)
  assert(Array.isArray(item.owners) && item.owners.length > 0, `${item.id} has no explicit owner`)
  assert(item.owners.every(owner => ['public', 'admin'].includes(owner)), `${item.id} has an invalid owner`)
  assert(JSON.stringify(item.owners) === JSON.stringify(['public', 'admin'].filter(owner => item.owners.includes(owner))), `${item.id} owners are not canonical`)
  if (item.kind === 'page') assert(item.owners.length === 1 && item.route, `${item.id} page ownership/route is incomplete`)
  else assert(!item.route, `${item.id} component unexpectedly owns a page route`)
  assert(existsSync(resolve(root, item.source)), `${item.id} source is missing`)
  const reviewed = reviewedUI.get(item.source)
  assert(reviewed && reviewed.id === item.id && reviewed.contractVersion === item.contractVersion, `${item.id} drifted from its reviewed identity`)
  assert(reviewed.kind === item.kind && JSON.stringify(reviewed.owners) === JSON.stringify(item.owners), `${item.id} kind/ownership drifted from review`)
  if (item.owners.includes('public')) ownershipCounts.public++
  if (item.owners.includes('admin')) ownershipCounts.admin++
  if (item.owners.length === 2) ownershipCounts.shared++
}
assert(ownershipCounts.public > 0 && ownershipCounts.admin > 0 && ownershipCounts.shared > 0, 'public/admin/shared ownership classes must all be represented')

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
