/**
 * Runtime Page Registry validation.
 *
 * This file has TWO layers — do not confuse them:
 *
 * 1) Offline contract checks (always run, no servers, no Playwright):
 *    - Host wiring (SFPageOutlet, catch-all route, SSR loaderData only)
 *    - L2 not dynamically imported
 *    - API carries DataSchema on ResolvedPage / LoadForResolved
 *    - Fixture plugin + theme packages still declare expected pages
 *
 * 2) Optional live HTTP smoke (only when PAGE_REGISTRY_API is set):
 *    - Node fetch against a running API (and optional WEB base)
 *    - NOT a full browser E2E: no Nuxt hydration, no login UI navigation,
 *      disable→instant 404 in a real browser.
 *
 * Usage (live layer):
 *   PAGE_REGISTRY_API=http://127.0.0.1:18080 PAGE_REGISTRY_WEB=http://127.0.0.1:13000 \
 *     node tests/validate-page-registry-runtime.js
 *
 * Never kill the user's :3000 process; use isolated ports for live checks.
 */
import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(__dirname, '..')

const API = process.env.PAGE_REGISTRY_API || process.env.API_BASE || ''
const WEB = process.env.PAGE_REGISTRY_WEB || process.env.WEB_BASE || ''

function read(rel) {
  return fs.readFileSync(path.join(root, rel), 'utf8')
}

function mustExist(rel) {
  const full = path.join(root, rel)
  assert.ok(fs.existsSync(full), `missing ${rel}`)
  return full
}

function assertIncludes(haystack, needle, msg) {
  assert.ok(haystack.includes(needle), msg || `expected to find: ${needle}`)
}

function assertNotIncludes(haystack, needle, msg) {
  assert.ok(!haystack.includes(needle), msg || `must not contain: ${needle}`)
}

// ---------------------------------------------------------------------------
// Layer 1 — always-on offline contracts (honest: not browser E2E)
// ---------------------------------------------------------------------------
function validateOfflineContracts() {
  console.log('[page-registry] offline contracts…')

  // Host page outlet + dynamic add route
  const outlet = read('apps/web/app/components/SFPageOutlet.vue')
  assertIncludes(outlet, 'SFPageOutlet', 'SFPageOutlet component')
  assertIncludes(outlet, "new URLSearchParams({ id: props.page, path: route.path })", 'outlet must resolve by page id and current path')
  assertNotIncludes(outlet, 'CONSTRAINED_PAGES', 'protected pages must resolve through Registry')
  assertIncludes(outlet, '<slot />', 'protected templates must receive the Host page slot')
  assertIncludes(outlet, 'loaderData', 'outlet must type loaderData from SSR')

  const catchAll = read('apps/web/app/pages/[...sfRegistryPage].vue')
  assertIncludes(catchAll, '/pages/resolve-path?path=', 'catch-all must use resolve-path')
  assertIncludes(catchAll, 'loader-data', 'catch-all must pass SSR loader data into template')
  assertIncludes(catchAll, "navigateTo(localePath('/login'))", 'login access must navigate to login')
  // data-route prop binding is OK; forbid imperative client fetches of plugin data
  assertNotIncludes(catchAll, 'request(`/extensions', 'no client extension data fetch')
  assertNotIncludes(catchAll, 'fetch(data', 'no client fetch of plugin data')
  // Only host resolve-path request should appear
  const catchAllRequests = [...catchAll.matchAll(/request\s*<[^>]*>\s*\(\s*`([^`]+)`/g)].map((m) => m[1])
  for (const path of catchAllRequests) {
    assert.ok(path.startsWith('/pages/resolve-path'), `catch-all may only call resolve-path, got ${path}`)
  }

  const template = read('apps/web/app/components/SFThemeTemplate.vue')
  assertIncludes(template, 'loaderData', 'template consumes SSR loaderData')
  assertIncludes(template, '禁止客户端再请求插件 route', 'template must document no client plugin fetch')
  assertIncludes(template, "'identity.component.login_form': HostPageIsland", 'login replacement must preserve the Host form')
  assertIncludes(template, "'forum.component.topic_composer': HostPageIsland", 'topic replacement must preserve the Host composer')
  // L2 closed: no dynamic import() of remote/package widgets in this component
  assert.ok(!/\bimport\s*\(/.test(template), 'L2 dynamic import must stay closed in SFThemeTemplate')
  assertNotIncludes(template, 'SFExtensionWidget', 'L2 widget island must not mount')

  // Index home wrapped in outlet
  const index = read('apps/web/app/pages/index.vue')
  assertIncludes(index, 'SFPageOutlet', 'home uses SFPageOutlet')
  assertIncludes(index, 'forum.home', 'home page id')

  const errorPage = read('apps/web/app/error.vue')
  assertIncludes(errorPage, 'page="system.not_found"', '404 must resolve through the not-found catalog surface')
  assertIncludes(errorPage, 'v-if="isNotFound"', 'non-404 Host failures must not use the not-found provider')

  const componentsPage = read('apps/web/app/pages/components.vue')
  assertIncludes(componentsPage, 'page="dev.components"', 'dev component catalog must have a Page Registry outlet')

  // API: DataSchema must flow through ResolvedPage → LoadForResolved
  const catalog = read('apps/api/app/Support/Pages/catalog.go')
  assertIncludes(catalog, 'DataSchema', 'ResolvedPage must include DataSchema field')

  const registry = read('apps/api/app/Support/Pages/registry.go')
  assert.match(registry, /DataSchema:\s+c\.DataSchema/, 'Resolve(replace) must copy DataSchema from contribution')

  const gateway = read('apps/api/app/Support/Pages/loader_gateway.go')
  assert.match(gateway, /DataSchema:\s+resolved\.DataSchema/, 'LoadForResolved must forward DataSchema')

  // Controller enforces access before loader (ordering signal)
  const controller = read('apps/api/app/Http/Controllers/Pages/controller.go')
  assertIncludes(controller, 'enforcePageAccess', 'controller must enforce access')
  assertIncludes(controller, 'LoadForResolved', 'controller must call loader gateway on resolve')
  assertIncludes(controller, 'LoadForContribution', 'controller must call loader on resolve-path')

  // Extension lifecycle owns Page Registry updates directly.
  const lifecycle = read('apps/api/app/Models/Extensions/service.go')
  assertIncludes(lifecycle, 'RegisterPluginPackage', 'plugin enable must register its Page Registry package')
  assertIncludes(lifecycle, 'ClearExtension', 'plugin disable must clear its Page Registry entries')

  // Fixture plugin still proves add + replace + login surfaces
  const fixtureTheme = JSON.parse(read('extensions/fixtures/plugins/page-registry-demo/theme.json'))
  const pages = fixtureTheme.pages || []
  assert.ok(pages.some((p) => p.action === 'add' && p.path === '/demo-docs/:slug'), 'fixture add /demo-docs/:slug')
  assert.ok(pages.some((p) => p.action === 'add' && p.access === 'login'), 'fixture login page')
  assert.ok(pages.some((p) => p.action === 'replace' && p.target === 'forum.home'), 'fixture home replace')

  // Signal Garden / default theme packages exist for activate-without-rebuild story
  mustExist('tests/fixtures/themes/sforum-signal-garden/theme.json')
  mustExist('extensions/builtin/themes/sforum-default/theme.json')

  // Migration for contract_version present
  mustExist('apps/api/database/migrations/202607130003_page_provider_contract_version.sql')
  const mig = read('apps/api/database/migrations/202607130003_page_provider_contract_version.sql')
  assertIncludes(mig, 'contract_version', 'migration adds contract_version')

  console.log('[page-registry] offline contracts passed (not browser E2E)')
}

// ---------------------------------------------------------------------------
// Layer 2 — optional live HTTP smoke (Node fetch; still not Playwright E2E)
// ---------------------------------------------------------------------------
async function getJSON(url, opts = {}) {
  const res = await fetch(url, opts)
  const text = await res.text()
  let body
  try {
    body = JSON.parse(text)
  } catch {
    body = text
  }
  return { status: res.status, body, headers: res.headers }
}

async function validateLiveSmoke() {
  console.log('[page-registry] live HTTP smoke API=', API, 'WEB=', WEB || '(none)')
  console.log('[page-registry] NOTE: this is Node fetch smoke, not Playwright browser E2E')

  {
    const r = await getJSON(`${API}/api/v1/pages/resolve?id=forum.home`)
    assert.equal(r.status, 200, 'resolve core status')
    assert.equal(r.body?.data?.provider, 'core', 'default provider is core unless approved replace')
    console.log('✓ resolve core home')
  }

  {
    const r = await getJSON(`${API}/api/v1/pages/resolve-path?path=/no-such-page-xyz`)
    assert.equal(r.status, 404)
    console.log('✓ resolve-path 404')
  }

  {
    const r = await getJSON(`${API}/api/v1/pages/catalog`)
    assert.equal(r.status, 200)
    assert.ok(Array.isArray(r.body?.data) || Array.isArray(r.body))
    console.log('✓ public catalog')
  }

  {
    const r = await getJSON(`${API}/api/v1/site/active-theme/skin`)
    assert.equal(r.status, 200)
    console.log('✓ active theme skin')
  }

  {
    const r = await getJSON(`${API}/api/v1/pages/resolve-path?path=/demo-docs/hello`)
    if (r.status === 200) {
      assert.equal(r.body?.data?.action, 'add')
      console.log('✓ demo-docs/hello add page live')
    } else {
      console.log('· demo-docs not registered (plugin not enabled in this env)')
    }
  }

  {
    // login-gated path should 401 when anonymous if fixture members page is enabled
    const r = await getJSON(`${API}/api/v1/pages/resolve-path?path=/demo-members`)
    if (r.status === 401) {
      console.log('✓ demo-members anonymous 401')
    } else if (r.status === 404) {
      console.log('· demo-members not registered')
    } else {
      console.log(`· demo-members status ${r.status} (plugin state dependent)`)
    }
  }

  if (WEB) {
    const res = await fetch(WEB + '/')
    const html = await res.text()
    assert.ok(res.status === 200 || res.status === 302, 'web home status')
    // Smoke only: HTML must not advertise open L2 dynamic widget import entrypoints
    assert.ok(!html.includes('import("/extensions/') && !html.includes("import('/extensions/"), 'no extension dynamic import in HTML smoke')
    console.log('✓ web home HTML smoke (status + no extension dynamic import)')
  }

  console.log('[page-registry] live HTTP smoke passed (still not full browser E2E)')
}

async function main() {
  validateOfflineContracts()

  if (!API) {
    console.log('[page-registry] live HTTP smoke SKIPPED (set PAGE_REGISTRY_API for isolated-port smoke)')
    console.log('  Full browser E2E (Nuxt render, login navigation, disable→404) is not covered by this script.')
    return
  }
  await validateLiveSmoke()
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
