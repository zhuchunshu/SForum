const fs = require('fs')
const path = require('path')

const root = path.resolve(__dirname, '..')
const nuxtConfig = fs.readFileSync(path.join(root, 'apps/web/nuxt.config.ts'), 'utf8')
const runtimeScript = fs.readFileSync(path.join(root, 'apps/web/scripts/runtime.mjs'), 'utf8')
const webReleaseContract = fs.readFileSync(path.join(root, 'apps/web/scripts/web-release-contract.mjs'), 'utf8')
const devRuntimeScript = fs.readFileSync(path.join(root, 'apps/web/scripts/dev-theme-runtime.mjs'), 'utf8')
const devPlainScript = fs.readFileSync(path.join(root, 'apps/web/scripts/dev-plain.mjs'), 'utf8')
const devLifecycleScript = fs.readFileSync(path.join(root, 'apps/web/scripts/dev-theme-lifecycle.mjs'), 'utf8')
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
assertIncludes(nuxtConfig, 'SFORUM_DEFAULT_THEME_LAYER', 'isolated web releases must inject the default theme fallback layer')
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
assertIncludes(webReleaseContract, "mode === 'default'", 'runtime script must handle default mode in current.json')
assertIncludes(webReleaseContract, 'path.isAbsolute', 'runtime script must resolve relative server paths')
assertIncludes(runtimeScript, 'fallbackServer', 'runtime script must provide a fallback server helper')
assertIncludes(runtimeScript, 'replaceTarget', 'production runtime must keep blue-green target replacement')
assertIncludes(webPackage.scripts.build, 'nuxt build', 'web build script must run Nuxt build')
if (webPackage.scripts.build.includes('NUXT_BUILD_DIR=.nuxt-build')) {
  throw new Error('web build script must not override the theme worker NUXT_BUILD_DIR environment')
}
assertIncludes(webPackage.scripts.dev, 'dev-theme-runtime.mjs', 'web dev must use the theme-aware supervisor')
assertIncludes(webPackage.scripts['dev:plain'], 'scripts/dev-plain.mjs', 'web dev:plain must use the release-ack wrapper')
assertIncludes(webPackage.scripts['dev:nuxt'], 'nuxt dev --host 0.0.0.0 --dotenv ../../.env', 'web dev:nuxt must keep raw Nuxt dev')
assertIncludes(devPlainScript, 'startPlainReleaseAck', 'web dev:plain must acknowledge Web Releases')
assertIncludes(devPlainScript, "['x', 'nuxt', 'dev'", 'web dev:plain must still launch Nuxt dev')
assertIncludes(webPackage.scripts['theme:runtime'], 'scripts/runtime.mjs', 'theme:runtime must run production theme supervisor')
assertIncludes(devRuntimeScript, 'SFORUM_THEME_LAYER', 'dev supervisor must inject SFORUM_THEME_LAYER')
assertIncludes(devRuntimeScript, 'current.json', 'dev supervisor must read current.json')
assertIncludes(devRuntimeScript, 'fs.watch', 'dev supervisor must watch release changes')
assertIncludes(devRuntimeScript, 'dev:nuxt', 'dev supervisor must spawn the bare dev:nuxt process')
assertIncludes(devRuntimeScript, 'SIGTERM', 'dev supervisor must handle SIGTERM to clean up children')
assertIncludes(devRuntimeScript, 'createDevThemeLifecycle', 'dev supervisor must use the serial lifecycle coordinator')
assertIncludes(devRuntimeScript, 'stopProcessGroup', 'dev supervisor must stop the old process group before replacement')
if (devRuntimeScript.includes('replaceTarget')) {
  throw new Error('dev supervisor must not start parallel Nuxt candidates through replaceTarget')
}
assertIncludes(webReleaseContract, "mode === 'default'", 'dev lifecycle must represent default mode explicitly')
assertIncludes(devLifecycleScript, 'restartRequested', 'dev lifecycle must coalesce current.json changes')
assertIncludes(devLifecycleScript, "signalGroup(pid, 'SIGKILL')", 'dev lifecycle must bound process-group shutdown')

console.log('Theme runtime validation passed.')
