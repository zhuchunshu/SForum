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
const manifest = JSON.parse(read('extensions/fixtures/plugins/sforum-prebuilt-settings/sforum.extension.json'))
const component = read('apps/web/app/components/extensions/settings/SFTrustedSettingsComponent.vue')
const schemaRenderer = read('apps/web/app/components/extensions/settings/SFExtensionSettingsRenderer.vue')
const adminLayout = read('apps/web/app/layouts/admin.vue')
const extensionRoutes = read('apps/api/app/Http/Controllers/Extensions/routes.go')

assert(sdk.includes('AdminMicroFrontendBridgeV1'), 'Admin SDK must expose the prebuilt component bridge')
assert(!sdk.includes('AdminSlotProps') && !sdk.includes('useSForumAdminHost'), 'legacy Vue slot SDK must stay removed')
assert(!jobsPage.includes('contributionsFor('), 'Jobs page must not mount extension Vue slots')
assert(manifest.settings.ui.mode === 'component', 'prebuilt fixture must declare component settings mode')
assert(manifest.settings.ui.component.entry.endsWith('.mjs'), 'prebuilt fixture entry must be an authored module')
assert(component.includes('@vite-ignore') && component.includes('/frontend/assets/${digest.value}/'), 'component must load from immutable digest endpoints')
assert(component.includes("trustState === 'invalidated'") && component.includes('slot name="fallback"'), 'invalid trust and load failures must expose Schema fallback')
assert(schemaRenderer.includes('SFAdminFormFooter'), 'host Schema renderer must remain available')
assert(!adminLayout.includes('SFAdminReleaseNotice'), 'admin layout must not reference the removed Web Release notice')
assert(extensionRoutes.includes('/frontend/trust') && !extensionRoutes.includes('/web-releases'), 'only digest trust routes may remain')
assert(!exists('tests/fixtures/extensions/trusted-admin-plugin/sforum.extension.json'), 'legacy trusted Vue fixture must stay removed')

console.log('Buildless trusted admin component validation passed.')
