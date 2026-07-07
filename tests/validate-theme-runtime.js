const fs = require('fs')
const path = require('path')

const root = path.resolve(__dirname, '..')
const nuxtConfig = fs.readFileSync(path.join(root, 'apps/web/nuxt.config.ts'), 'utf8')
const runtimeScript = fs.readFileSync(path.join(root, 'apps/web/scripts/runtime.mjs'), 'utf8')
const webPackage = JSON.parse(fs.readFileSync(path.join(root, 'apps/web/package.json'), 'utf8'))

function assertIncludes(source, pattern, message) {
  if (!source.includes(pattern)) {
    throw new Error(message)
  }
}

function assertMatches(source, pattern, message) {
  if (!pattern.test(source)) {
    throw new Error(message)
  }
}

assertIncludes(nuxtConfig, 'SFORUM_THEME_LAYER', 'nuxt.config.ts must read SFORUM_THEME_LAYER')
assertIncludes(nuxtConfig, 'SFORUM_NITRO_OUTPUT_DIR', 'nuxt.config.ts must read SFORUM_NITRO_OUTPUT_DIR')
assertIncludes(nuxtConfig, 'themeLayers', 'nuxt.config.ts must build a themeLayers array')
assertIncludes(nuxtConfig, 'defaultThemeLayer', 'nuxt.config.ts must keep the built-in default theme layer')
assertMatches(
  nuxtConfig,
  /uploadedThemeLayer\s*\?\s*\[\s*uploadedThemeLayer\s*,\s*defaultThemeLayer\s*\]\s*:\s*\[\s*defaultThemeLayer\s*\]/,
  'uploaded themes must be prepended before the default theme fallback layer'
)
assertIncludes(
  nuxtConfig,
  '默认主题必须始终作为最后的 fallback layer',
  'nuxt.config.ts must document the default theme fallback contract'
)
assertIncludes(runtimeScript, 'SFORUM_THEME_RELEASE_ROOT', 'runtime script must read SFORUM_THEME_RELEASE_ROOT')
assertIncludes(runtimeScript, 'current.json', 'runtime script must watch current.json')
assertIncludes(runtimeScript, 'spawn', 'runtime script must spawn the selected Nitro server')
assertIncludes(runtimeScript, 'fs.watch', 'runtime script must watch release changes')
assertIncludes(webPackage.scripts.build, 'nuxt build', 'web build script must run Nuxt build')
if (webPackage.scripts.build.includes('NUXT_BUILD_DIR=.nuxt-build')) {
  throw new Error('web build script must not override the theme worker NUXT_BUILD_DIR environment')
}

console.log('Theme runtime validation passed.')
