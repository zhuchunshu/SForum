const fs = require('fs')
const path = require('path')

const root = path.resolve(__dirname, '..')
const nuxtConfig = fs.readFileSync(path.join(root, 'apps/web/nuxt.config.ts'), 'utf8')
const webPackage = JSON.parse(fs.readFileSync(path.join(root, 'apps/web/package.json'), 'utf8'))
const webDockerfile = fs.readFileSync(path.join(root, 'apps/web/Dockerfile'), 'utf8')
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

// Production and development both run Nuxt directly; no release supervisor remains.
assertIncludes(webPackage.scripts.start, '.output/server/index.mjs', 'web start must run the Nuxt output directly')
assertIncludes(webDockerfile, 'CMD ["bun", ".output/server/index.mjs"]', 'web image must run the Nuxt output directly')
assertNotIncludes(webDockerfile, 'theme-releases', 'web image must not contain a theme release volume')
assertIncludes(webPackage.scripts.dev, 'nuxt dev', 'web dev must run Nuxt directly without theme layer supervisor')
assertIncludes(webPackage.scripts.build, 'nuxt build', 'web build script must run Nuxt build')

// Page registry contracts
assertIncludes(pageCatalog, 'forum.home', 'page catalog must document forum.home')
assertIncludes(defaultThemeJson, 'forum.home', 'default theme package must declare home replace')
assertIncludes(outlet, 'page:', 'SFPageOutlet must declare page prop')
assertIncludes(home, 'SFPageOutlet', 'home page must use SFPageOutlet')
assertIncludes(home, 'forum.home', 'home page must declare page id forum.home')

assertNotIncludes(JSON.stringify(webPackage.scripts), 'dev:compose', 'legacy dev compose script must stay removed')

console.log('Theme runtime validation passed (Page Registry / no layer activation).')
