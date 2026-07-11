const fs = require('fs')
const path = require('path')

const root = process.cwd()
const paths = {
  page: path.resolve(root, 'extensions/builtin/themes/sforum-default/layer/app/pages/index.vue'),
  navigation: path.resolve(root, 'extensions/builtin/themes/sforum-default/layer/app/components/SFHomeNavigation.vue'),
  topicRow: path.resolve(root, 'extensions/builtin/themes/sforum-default/layer/app/components/SFHomeTopicRow.vue'),
  layerConfig: path.resolve(root, 'extensions/builtin/themes/sforum-default/layer/nuxt.config.ts'),
  hostConfig: path.resolve(root, 'apps/web/nuxt.config.ts'),
  homeCss: path.resolve(root, 'extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-home.css'),
  themeCss: path.resolve(root, 'extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css'),
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
const layerConfig = read('layerConfig')
const hostConfig = read('hostConfig')
const homeCss = read('homeCss')
const themeCss = read('themeCss')
const seo = read('seo')
const locales = [JSON.parse(read('zh')), JSON.parse(read('en'))]

if (!hostConfig.includes('../../extensions/builtin/themes/sforum-default/layer') || !hostConfig.includes('extends')) {
  throw new Error('apps/web/nuxt.config.ts should extend the built-in default theme layer')
}

if (!layerConfig.includes('import.meta.url') || !layerConfig.includes('sforum-home.css')) {
  throw new Error('the default theme layer should register the layer-relative homepage stylesheet')
}

for (const token of ['<SFHomeNavigation', '<SFHomeTopicRow', 'parseForumHomeQuery', 'buildForumHomeQuery', 'forumHomeFeedKey', 'IntersectionObserver']) {
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

if (!topicRow.includes('topic.excerpt') || !topicRow.includes('topic.commentCount') || topicRow.includes('participants')) {
  throw new Error('SFHomeTopicRow must render only real topic summary metadata')
}

for (const token of ['grid-template-columns: 74px minmax(0, 1fr) 310px;', '.sforum-home__dock', '.sf-home-topic-row__heat', 'min-height: 40px;', 'overflow-wrap: anywhere;', 'prefers-reduced-motion: reduce']) {
  if (!homeCss.includes(token)) {
    throw new Error(`sforum-home.css is missing: ${token}`)
  }
}

for (const obsoleteColor of ['#0b1120', '#172033']) {
  if (homeCss.includes(obsoleteColor) || themeCss.includes(obsoleteColor)) {
    throw new Error(`homepage styles still use the old blue-black color: ${obsoleteColor}`)
  }
}

if (!seo.includes('/?q={search_term_string}') || seo.includes('/search?q={search_term_string}')) {
  throw new Error('schema.org SearchAction must point to the working homepage query')
}

for (const messages of locales) {
  for (const key of ['metaTitle', 'searchPlaceholder', 'notice', 'allTopics', 'categories', 'tags', 'clearFilters', 'searchResults']) {
    if (!messages.home[key]) {
      throw new Error(`homepage locale is missing home.${key}`)
    }
  }
  if (!messages.home.emptyState?.filteredDescription || !messages.home.feed?.loadMoreFailed || !messages.home.feed?.retryLoadMore) {
    throw new Error('homepage locale is missing constrained empty/retry copy')
  }
}

console.log('SForum hybrid homepage validation PASSED!')
