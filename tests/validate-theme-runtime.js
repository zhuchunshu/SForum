const fs = require('fs')
const path = require('path')

const root = path.resolve(__dirname, '..')
const nuxtConfig = fs.readFileSync(path.join(root, 'apps/web/nuxt.config.ts'), 'utf8')
const runtimeScript = fs.readFileSync(path.join(root, 'apps/web/scripts/runtime.mjs'), 'utf8')

function assertIncludes(source, pattern, message) {
  if (!source.includes(pattern)) {
    throw new Error(message)
  }
}

assertIncludes(nuxtConfig, 'SFORUM_THEME_LAYER', 'nuxt.config.ts must read SFORUM_THEME_LAYER')
assertIncludes(nuxtConfig, 'SFORUM_NITRO_OUTPUT_DIR', 'nuxt.config.ts must read SFORUM_NITRO_OUTPUT_DIR')
assertIncludes(nuxtConfig, 'themeLayers', 'nuxt.config.ts must build a themeLayers array')
assertIncludes(runtimeScript, 'SFORUM_THEME_RELEASE_ROOT', 'runtime script must read SFORUM_THEME_RELEASE_ROOT')
assertIncludes(runtimeScript, 'current.json', 'runtime script must watch current.json')
assertIncludes(runtimeScript, 'spawn', 'runtime script must spawn the selected Nitro server')
assertIncludes(runtimeScript, 'fs.watch', 'runtime script must watch release changes')

console.log('Theme runtime validation passed.')
