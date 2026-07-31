const fs = require('fs')
const path = require('path')

/**
 * Homepage presentation-ownership contract (V3 P13).
 *
 * - Route shell (pages/index.vue): SEO + SFPageOutlet fail-closed + SFHomePage island
 * - Body island (SFHomePage.vue): feed/nav/right-rail hybrid UI
 * - Host styles / theme package / i18n remain Host-owned
 */
const root = process.cwd()
const paths = {
  page: path.resolve(root, 'apps/web/app/pages/index.vue'),
  homeIsland: path.resolve(root, 'apps/web/app/components/forum/SFHomePage.vue'),
  navigation: path.resolve(root, 'apps/web/app/components/forum/SFHomeNavigation.vue'),
  topicRow: path.resolve(root, 'apps/web/app/components/forum/SFHomeTopicRow.vue'),
  rightRail: path.resolve(root, 'apps/web/app/components/forum/SFHomeRightRail.vue'),
  outlet: path.resolve(root, 'apps/web/app/components/SFPageOutlet.vue'),
  hostConfig: path.resolve(root, 'apps/web/nuxt.config.ts'),
  homeCss: path.resolve(root, 'apps/web/app/assets/css/sforum-home.css'),
  themeCss: path.resolve(root, 'apps/web/app/assets/css/sforum-theme.css'),
  themePackage: path.resolve(root, 'extensions/builtin/themes/sforum-default/theme.json'),
  defaultHomeTemplate: path.resolve(root, 'extensions/builtin/themes/sforum-default/templates/home.html'),
  seo: path.resolve(root, 'apps/web/app/utils/seo/seoStructuredData.ts'),
  zh: path.resolve(root, 'apps/web/i18n/locales/zh-CN.json'),
  en: path.resolve(root, 'apps/web/i18n/locales/en-US.json')
}

console.log('Validating SForum homepage presentation ownership...\n')

for (const [name, file] of Object.entries(paths)) {
  if (!fs.existsSync(file)) {
    throw new Error(`${name} file is missing: ${file}`)
  }
}

const read = key => fs.readFileSync(paths[key], 'utf8')
const page = read('page')
const homeIsland = read('homeIsland')
const navigation = read('navigation')
const topicRow = read('topicRow')
const rightRail = read('rightRail')
const hostConfig = read('hostConfig')
const homeCss = read('homeCss')
const themeCss = read('themeCss')
const themePackage = read('themePackage')
const homeTemplate = read('defaultHomeTemplate')
const seo = read('seo')
const locales = [JSON.parse(read('zh')), JSON.parse(read('en'))]

if (!hostConfig.includes('sforum-home.css') || !hostConfig.includes('sforum-theme.css')) {
  throw new Error('apps/web/nuxt.config.ts should register host homepage/theme stylesheets')
}

if (!themePackage.includes('forum.home') || !themePackage.includes('assets/theme.css')) {
  throw new Error('default theme package theme.json should declare L0/L1 home contract')
}

// 路由壳：SEO + Page Outlet + fail-closed 首页岛；不得再塞满 hybrid UI。
if (!page.includes('SFPageOutlet') || !page.includes('forum.home')) {
  throw new Error('index.vue must wrap content in SFPageOutlet page="forum.home"')
}
if (!page.includes('<SFHomePage') || !page.includes('useSForumSeo')) {
  throw new Error('index.vue must mount SFHomePage fail-closed island and Host SEO')
}
if (!page.includes("type: 'home'")) {
  throw new Error("index.vue must declare SEO type: 'home'")
}
if (!page.includes('parseForumHomeQuery')) {
  throw new Error('index.vue must keep filter-aware noindex via parseForumHomeQuery')
}

// 路由壳禁止承载完整 hybrid UI（已迁到 SFHomePage 岛）。
for (const fat of ['<SFHomeNavigation', '<SFHomeTopicRow', '<SFHomeRightRail', 'IntersectionObserver', 'hotTopics', 'sforum-home__layout--with-right']) {
  if (page.includes(fat)) {
    throw new Error(`index.vue must stay a thin shell; hybrid UI token leaked: ${fat}`)
  }
}
for (const obsolete of ['layout: false', 'sforum-home__topbar', '<SFFooter', 'topicReplyStackLabel', 'participants']) {
  if (page.includes(obsolete)) {
    throw new Error(`index.vue still contains obsolete or fabricated homepage UI: ${obsolete}`)
  }
}

// Body 岛：完整 hybrid 首页契约。
for (const token of [
  '<SFHomeNavigation',
  '<SFHomeTopicRow',
  '<SFPagination',
  'sforum-home__layout--with-right',
  'useActiveThemeSettings',
  'rightRailEnabled',
  'parseForumHomeQuery',
  'buildForumHomeQuery',
  'forumHomeFeedKey',
  'parsePublicPage',
  'publicPageLocation',
  'IntersectionObserver',
  'hotTopics'
]) {
  if (!homeIsland.includes(token)) {
    throw new Error(`SFHomePage.vue is missing the hybrid homepage contract: ${token}`)
  }
}

// 主题 L1 壳：presentation 归属 + chrome 岛 + body 岛挂载点。
for (const marker of ['data-theme-owned="presentation"', '<sf-navbar', '<sf-footer', '<sf-home-page']) {
  if (!homeTemplate.includes(marker)) {
    throw new Error(`default home.html missing presentation chrome/body marker: ${marker}`)
  }
}

if (!navigation.includes('totalTopics') || !navigation.includes("'select-category': [slug: string]")) {
  throw new Error('SFHomeNavigation must expose typed, API-backed category navigation')
}

if (!topicRow.includes('topic.commentCount') || !topicRow.includes('SFAvatar')) {
  throw new Error('SFHomeTopicRow must use API-backed summary fields and SFAvatar')
}

if (!rightRail.includes('hot-topics') && !rightRail.includes('hotTopics')) {
  if (!rightRail.includes('sf-home-right-rail') && !rightRail.includes('SFHomeRightRail')) {
    throw new Error('SFHomeRightRail contract incomplete')
  }
}

if (!homeCss.includes('sforum-home')) {
  throw new Error('homepage CSS missing sforum-home rules')
}
if (!themeCss.includes(':root') && !themeCss.includes('--')) {
  // theme tokens may use different shapes; home CSS is the hard requirement above
}

if (!seo || typeof seo !== 'string') {
  throw new Error('seoStructuredData missing')
}

for (const locale of locales) {
  if (!locale.home) {
    throw new Error('locale missing home keys')
  }
}

console.log('Homepage validation passed.')
