const fs = require('fs')
const path = require('path')

const root = path.resolve(__dirname, '..')
const nuxtConfig = fs.readFileSync(path.join(root, 'apps/web/nuxt.config.ts'), 'utf8')
const runtimePlain = fs.readFileSync(path.join(root, 'apps/web/scripts/runtime-plain.mjs'), 'utf8')
const webPackage = JSON.parse(fs.readFileSync(path.join(root, 'apps/web/package.json'), 'utf8'))
const pageCatalog = fs.readFileSync(path.join(root, 'docs/extensions/page-catalog.md'), 'utf8')
const defaultThemeJson = fs.readFileSync(path.join(root, 'extensions/builtin/themes/sforum-default/theme.json'), 'utf8')
const outlet = fs.readFileSync(path.join(root, 'apps/web/app/components/SFPageOutlet.vue'), 'utf8')
const home = fs.readFileSync(path.join(root, 'apps/web/app/pages/index.vue'), 'utf8')

function assertIncludes(source, pattern, message) {
  if (!source.includes(pattern)) {
    throw new Error(message)
  }
}

function assertNotIncludes(source, pattern, message) {
  if (source.includes(pattern)) {
    throw new Error(message)
  }
}

// Host no longer extends public theme Nuxt layers.
assertNotIncludes(nuxtConfig, 'extends: themeLayers', 'nuxt.config must not extend public theme layers')
assertNotIncludes(nuxtConfig, 'SFORUM_THEME_LAYER', 'nuxt.config must not select public themes via SFORUM_THEME_LAYER')
assertIncludes(nuxtConfig, 'sforum-home.css', 'host must register migrated public CSS')
assertIncludes(nuxtConfig, 'sforum-theme.css', 'host must register migrated public theme CSS')

// Production runtime is plain Nitro (no current.json theme switch).
assertIncludes(runtimePlain, 'Nitro', 'runtime-plain must start Nitro directly')
assertNotIncludes(runtimePlain, 'current.json', 'runtime-plain must not watch theme current.json')
assertIncludes(webPackage.scripts['theme:runtime'], 'runtime-plain.mjs', 'theme:runtime must use plain Nitro runtime')
assertIncludes(webPackage.scripts.dev, 'nuxt dev', 'web dev must run Nuxt directly without theme layer supervisor')
assertIncludes(webPackage.scripts.build, 'nuxt build', 'web build script must run Nuxt build')

// Page registry contracts
assertIncludes(pageCatalog, 'forum.home', 'page catalog must document forum.home')
assertIncludes(defaultThemeJson, 'forum.home', 'default theme package must declare home replace')
assertIncludes(outlet, 'page:', 'SFPageOutlet must declare page prop')
assertIncludes(home, 'SFPageOutlet', 'home page must use SFPageOutlet')
assertIncludes(home, 'forum.home', 'home page must declare page id forum.home')

// Admin Web Release path remains available via optional compose script.
assertIncludes(webPackage.scripts['dev:compose'] || '', 'dev-theme-runtime.mjs', 'dev:compose may still compose admin registry for trusted plugins')

console.log('Theme runtime validation passed (Page Registry / no layer activation).')
