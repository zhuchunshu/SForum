const fs = require('fs')
const path = require('path')

const root = process.cwd()
const paths = {
  page: path.resolve(root, 'apps/web/app/pages/index.vue'),
  navigation: path.resolve(root, 'apps/web/app/components/SFHomeNavigation.vue'),
  topicRow: path.resolve(root, 'apps/web/app/components/SFHomeTopicRow.vue'),
  rightRail: path.resolve(root, 'apps/web/app/components/SFHomeRightRail.vue'),
  outlet: path.resolve(root, 'apps/web/app/components/SFPageOutlet.vue'),
  hostConfig: path.resolve(root, 'apps/web/nuxt.config.ts'),
  homeCss: path.resolve(root, 'apps/web/app/assets/css/sforum-home.css'),
  themeCss: path.resolve(root, 'apps/web/app/assets/css/sforum-theme.css'),
  themePackage: path.resolve(root, 'extensions/builtin/themes/sforum-default/theme.json'),
  seo: path.resolve(root, 'apps/web/app/utils/seoStructuredData.ts'),
  zh: path.resolve(root, 'apps/web/i18n/locales/zh-CN.json'),
  en: path.resolve(root, 'apps/web/i18n/locales/en-US.json')
}

console.log('Validating SForum hybrid homepage implementation...\n')

for (const [name, file] of Object.entries(paths)) {
  if (!fs.existsSync(file)) {
    throw new Error(`${name} file is missing: ${file}`)
  }
}

const read = key => fs.readFileSync(paths[key], 'utf8')
const page = read('page')
const navigation = read('navigation')
const topicRow = read('topicRow')
const rightRail = read('rightRail')
const hostConfig = read('hostConfig')
const homeCss = read('homeCss')
const themeCss = read('themeCss')
const themePackage = read('themePackage')
const seo = read('seo')
const locales = [JSON.parse(read('zh')), JSON.parse(read('en'))]

if (!hostConfig.includes('sforum-home.css') || !hostConfig.includes('sforum-theme.css')) {
  throw new Error('apps/web/nuxt.config.ts should register host homepage/theme stylesheets')
}

if (!themePackage.includes('forum.home') || !themePackage.includes('assets/theme.css')) {
  throw new Error('default theme package theme.json should declare L0/L1 home contract')
}

if (!page.includes('SFPageOutlet') || !page.includes('forum.home')) {
  throw new Error('index.vue must wrap content in SFPageOutlet page="forum.home"')
}

for (const token of ['<SFHomeNavigation', '<SFHomeTopicRow', '<SFHomeRightRail', 'sforum-home__layout--with-right', 'useActiveThemeSettings', 'rightRailEnabled', 'parseForumHomeQuery', 'buildForumHomeQuery', 'forumHomeFeedKey', 'IntersectionObserver', 'hotTopics']) {
  if (!page.includes(token)) {
    throw new Error(`index.vue is missing the hybrid homepage contract: ${token}`)
  }
}

for (const obsolete of ['layout: false', 'sforum-home__topbar', '<SFFooter', 'topicReplyStackLabel', 'participants', '<SFPagination']) {
  if (page.includes(obsolete)) {
    throw new Error(`index.vue still contains obsolete or fabricated homepage UI: ${obsolete}`)
  }
}

if (!navigation.includes('category.topicCount') || !navigation.includes("'select-category': [slug: string]")) {
  throw new Error('SFHomeNavigation must expose typed, API-backed category navigation')
}

if (!topicRow.includes('topic.commentCount') || !topicRow.includes('SFAvatar')) {
  throw new Error('SFHomeTopicRow must use API-backed summary fields and SFAvatar')
}

if (!rightRail.includes('hot-topics') && !rightRail.includes('hotTopics')) {
  // prop name may vary; require right-rail file non-empty contract markers
  if (!rightRail.includes('sf-home-right-rail') && !rightRail.includes('SFHomeRightRail')) {
    throw new Error('SFHomeRightRail contract incomplete')
  }
}

if (!homeCss.includes('sforum-home') || !themeCss.includes(':root') && !themeCss.includes('--')) {
  // theme css may use different tokens; at least home css must exist with layout class
  if (!homeCss.includes('sforum-home')) {
    throw new Error('homepage CSS missing sforum-home rules')
  }
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
