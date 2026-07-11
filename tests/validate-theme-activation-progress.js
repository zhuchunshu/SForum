const fs = require('fs')
const path = require('path')

const root = path.resolve(__dirname, '..')
const themesPagePath = path.join(root, 'apps/web/app/pages/admin/extensions/themes.vue')
const overviewPagePath = path.join(root, 'apps/web/app/pages/admin/extensions/index.vue')
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
assert(themesPage.includes('extensionLocalizedDisplay'), 'themes list must resolve localized theme display copy')
assert(themesPage.includes('display.description'), 'themes list must show localized theme description')
assert(themesPage.includes('display.name'), 'themes list must show localized theme name')
assert(themesPage.includes('locale.value'), 'themes list must recompute when the UI locale changes')

const pluginsPagePath = path.join(root, 'apps/web/app/pages/admin/extensions/plugins.vue')
const pluginsPage = fs.readFileSync(pluginsPagePath, 'utf8')
assert(pluginsPage.includes('extensionLocalizedDisplay'), 'plugins list must resolve localized plugin display copy')
assert(pluginsPage.includes('display.description'), 'plugins list must show localized plugin description')
assert(pluginsPage.includes('display.name'), 'plugins list must show localized plugin name')
assert(pluginsPage.includes('locale.value'), 'plugins list must recompute when the UI locale changes')

const overviewPage = fs.readFileSync(overviewPagePath, 'utf8')
assert(overviewPage.includes('themeActivationProgress'), 'extensions overview page must use theme activation progress helper')
assert(overviewPage.includes('hasThemeActivationInProgress'), 'extensions overview page must detect in-progress activations')
assert(overviewPage.includes('setInterval'), 'extensions overview page must poll while activation is in progress')
assert(overviewPage.includes('clearInterval'), 'extensions overview page must stop polling when no activation is in progress')
assert(overviewPage.includes('UProgress'), 'extensions overview page must render a progress bar for theme activation')
assert(overviewPage.includes('extensionLocalizedDisplay'), 'extensions overview list must resolve localized extension display copy')
assert(overviewPage.includes('display.description'), 'extensions overview list must show localized extension description')
assert(overviewPage.includes('display.name'), 'extensions overview list must show localized extension name')
assert(overviewPage.includes('locale.value'), 'extensions overview list must recompute when the UI locale changes')

for (const localePath of [zhLocalePath, enLocalePath]) {
  const locale = fs.readFileSync(localePath, 'utf8')
  for (const key of ['themeProgress', 'viewBuildLog', 'hideBuildLog', 'emptyBuildLog']) {
    assert(locale.includes(`"${key}"`), `${path.basename(localePath)} must define ${key}`)
  }
}

console.log('Theme activation progress validation passed.')
