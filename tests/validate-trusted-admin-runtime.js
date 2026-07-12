const fs = require('fs')
const path = require('path')

const root = path.resolve(__dirname, '..')
const read = relative => fs.readFileSync(path.join(root, relative), 'utf8')
const exists = relative => fs.existsSync(path.join(root, relative))

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

const sdk = read('apps/web/packages/admin-sdk/src/index.ts')
const jobsPage = read('apps/web/app/pages/admin/jobs.vue')
const manifest = JSON.parse(read('tests/fixtures/extensions/trusted-admin-plugin/sforum.extension.json'))
const zh = JSON.parse(read('tests/fixtures/extensions/trusted-admin-plugin/frontend/admin/locales/zh-CN.json'))
const en = JSON.parse(read('tests/fixtures/extensions/trusted-admin-plugin/frontend/admin/locales/en-US.json'))
const runtimeTypes = read('apps/web/app/runtime/admin-extensions/types.ts')
const nuxtConfig = read('apps/web/nuxt.config.ts')
const releaseRoute = read('apps/web/server/routes/__sforum/admin-release.get.ts')
const plainRuntime = read('apps/web/scripts/runtime-plain.mjs')
const releaseAck = read('apps/web/scripts/dev-plain-release-ack.mjs')
const releaseContract = read('apps/web/scripts/web-release-contract.mjs')
const slotCatalog = read('apps/web/app/runtime/admin-extensions/catalog.ts')
const extensionRoutes = read('apps/api/app/Http/Controllers/Extensions/routes.go')

const slots = ['admin.jobs.table.columns', 'admin.jobs.row.actions', 'admin.jobs.detail.sections']
for (const slot of slots) {
  assert(sdk.includes(`'${slot}'`), `Admin SDK is missing ${slot}`)
  assert(jobsPage.includes(`contributionsFor('${slot}')`), `Jobs page does not render ${slot}`)
  assert(manifest.contributions.some(item => item.point === slot), `Fixture does not consume ${slot}`)
  assert(slotCatalog.includes(`'${slot}'`), `Frontend slot catalog is missing ${slot}`)
}
assert(sdk.includes('AdminSlotProps'), 'Admin SDK must expose typed slot props')
assert(runtimeTypes.includes('AdminComponentRegistry'), 'Runtime types must expose the component registry')
assert(nuxtConfig.includes('registry.client.ts'), 'Trusted component registry must remain client-only')
assert(releaseRoute.includes('no-store'), 'Admin release endpoint must disable caching')
// 公开主题不再经 production theme supervisor 切换 Nitro；可信管理端发布仍用 contract + ack。
assert(plainRuntime.includes('Nitro') && !plainRuntime.includes('current.json'), 'Production runtime must start Nitro without theme current.json')
assert(releaseAck.includes('writeActiveAcknowledgement') && releaseContract.includes('active.json') && releaseContract.includes('failures'), 'Web release ack/contract must acknowledge active and failed releases')
assert(extensionRoutes.includes('/frontend/trust') && extensionRoutes.includes('/web-releases'), 'Trusted runtime API routes are missing')
assert(Object.keys(zh).length > 0 && Object.keys(en).length > 0, 'Fixture must provide both locale catalogs')
assert(exists('apps/web/app/runtime/admin-extensions/empty-metadata.ts'), 'SSR fallback metadata module is missing')
assert(exists('apps/web/app/runtime/admin-extensions/empty-registry.client.ts'), 'Client fallback registry module is missing')

console.log('Trusted admin runtime validation passed.')
