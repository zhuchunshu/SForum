const fs = require('fs')
const path = require('path')

const root = path.resolve(__dirname, '..')
const themesPagePath = path.join(root, 'apps/web/app/pages/admin/extensions/themes.vue')
const zhLocalePath = path.join(root, 'apps/web/i18n/locales/zh-CN.json')
const enLocalePath = path.join(root, 'apps/web/i18n/locales/en-US.json')

function assert(condition, message) {
  if (!condition) {
    throw new Error(message)
  }
}

const themesPage = fs.readFileSync(themesPagePath, 'utf8')
assert(themesPage.includes('themeActivationProgress'), 'themes page must use theme activation progress helper')
assert(themesPage.includes('hasThemeActivationInProgress'), 'themes page must detect in-progress activations')
assert(themesPage.includes('setInterval'), 'themes page must poll while activation is in progress')
assert(themesPage.includes('clearInterval'), 'themes page must stop polling when no activation is in progress')
assert(themesPage.includes('UProgress'), 'themes page must render a progress bar for theme activation')
assert(themesPage.includes('buildLog'), 'themes page must expose theme release build logs')

for (const localePath of [zhLocalePath, enLocalePath]) {
  const locale = fs.readFileSync(localePath, 'utf8')
  for (const key of ['themeProgress', 'viewBuildLog', 'hideBuildLog', 'emptyBuildLog']) {
    assert(locale.includes(`"${key}"`), `${path.basename(localePath)} must define ${key}`)
  }
}

console.log('Theme activation progress validation passed.')
