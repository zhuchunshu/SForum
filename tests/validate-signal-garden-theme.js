const fs = require('fs')
const path = require('path')

const root = path.resolve(__dirname, '..')
const themeRoot = path.join(root, 'extensions/dev/themes/sforum-signal-garden')
const manifestPath = path.join(themeRoot, 'sforum.extension.json')
const layerRoot = path.join(themeRoot, 'layer')

function assert(condition, message) {
  if (!condition) {
    throw new Error(message)
  }
}

function read(file) {
  return fs.readFileSync(path.join(themeRoot, file), 'utf8')
}

assert(fs.existsSync(manifestPath), 'Signal Garden manifest must exist')
assert(fs.existsSync(layerRoot), 'Signal Garden Nuxt layer directory must exist')

const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'))
assert(manifest.id === 'sforum.signal-garden', 'manifest id must be sforum.signal-garden')
assert(manifest.type === 'theme', 'manifest type must be theme')
assert(manifest.frontend?.layer === 'layer', 'manifest frontend.layer must point to layer')
assert(!manifest.backend, 'theme manifest must not declare backend runtime')
assert(!manifest.routes, 'theme manifest must not declare routes')
assert(!manifest.hooks, 'theme manifest must not declare hooks')
assert(!manifest.events, 'theme manifest must not declare events')
assert(!manifest.jobs, 'theme manifest must not declare jobs')
assert(!manifest.providers, 'theme manifest must not declare providers')
assert(!manifest.migrations, 'theme manifest must not declare migrations')
assert(!manifest.permissions, 'theme manifest must not declare permissions')

const nuxtConfig = read('layer/nuxt.config.ts')
assert(nuxtConfig.includes('fileURLToPath'), 'theme layer CSS must resolve from the layer path')
assert(nuxtConfig.includes('signal-garden.css'), 'theme layer must register signal-garden.css')

const css = read('layer/app/assets/css/signal-garden.css')
for (const token of ['--sg-leaf', '--sg-sun', '--sg-coral', '--sg-ink', '--sg-paper']) {
  assert(css.includes(token), `theme CSS must define ${token}`)
}
for (const selector of ['.signal-garden-shell', '.sg-navbar', '.sg-feed-row', '.sg-community-panel', '.dark']) {
  assert(css.includes(selector), `theme CSS must include ${selector}`)
}

for (const file of [
  'layer/app/layouts/default.vue',
  'layer/app/layouts/auth.vue',
  'layer/app/components/SignalGardenNavbar.vue',
  'layer/app/components/SignalGardenFooter.vue',
  'layer/app/pages/index.vue',
  'layer/app/pages/login.vue',
  'layer/app/pages/register.vue'
]) {
  assert(fs.existsSync(path.join(themeRoot, file)), `${file} must exist`)
}

const homepage = read('layer/app/pages/index.vue')
for (const marker of ['signal-garden-home', 'sg-feed-row', 'sg-community-panel', 'useSForumSeo']) {
  assert(homepage.includes(marker), `homepage must include ${marker}`)
}

console.log('Signal Garden theme validation passed.')
