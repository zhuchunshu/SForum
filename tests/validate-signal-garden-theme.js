const fs = require('fs')
const path = require('path')

const root = path.resolve(__dirname, '..')
// extensions/dev is gitignored; CI uses the tracked runtime fixture mirror.
const themeRootCandidates = [
  path.join(root, 'extensions/dev/themes/sforum-signal-garden'),
  path.join(root, 'tests/fixtures/themes/sforum-signal-garden'),
]
const themeRoot = themeRootCandidates.find((candidate) => fs.existsSync(candidate))
  || themeRootCandidates[0]
const manifestPath = path.join(themeRoot, 'sforum.extension.json')
const themeJsonPath = path.join(themeRoot, 'theme.json')

function assert(condition, message) {
  if (!condition) {
    throw new Error(message)
  }
}

function read(file) {
  return fs.readFileSync(path.join(themeRoot, file), 'utf8')
}

assert(fs.existsSync(manifestPath), 'Signal Garden manifest must exist')
assert(fs.existsSync(themeJsonPath), 'Signal Garden theme.json must exist')

const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'))
assert(manifest.id === 'sforum.signal-garden', 'manifest id must be sforum.signal-garden')
assert(manifest.type === 'theme', 'manifest type must be theme')
// Runtime L0/L1：manifest 不再保留 Nuxt Layer 前端字段。
assert(!Object.hasOwn(manifest, 'frontend'), 'runtime Signal Garden must not declare frontend')
assert(!manifest.backend, 'theme manifest must not declare backend runtime')
assert(!manifest.routes, 'theme manifest must not declare routes')
assert(!manifest.hooks, 'theme manifest must not declare hooks')
assert(!manifest.events, 'theme manifest must not declare events')
assert(!manifest.jobs, 'theme manifest must not declare jobs')
assert(!manifest.providers, 'theme manifest must not declare providers')
assert(!manifest.migrations, 'theme manifest must not declare migrations')
assert(!manifest.permissions, 'theme manifest must not declare permissions')

const themeJson = JSON.parse(fs.readFileSync(themeJsonPath, 'utf8'))
assert(Array.isArray(themeJson.pages) && themeJson.pages.length > 0, 'theme.json must declare pages')
const home = themeJson.pages.find((p) => p.target === 'forum.home' || p.id?.includes('home'))
assert(home, 'theme.json must declare a forum.home replace contribution')
assert(home.action === 'replace', 'home contribution must be replace')
assert(home.template, 'home contribution must point to a template')
assert(fs.existsSync(path.join(themeRoot, home.template)), `template ${home.template} must exist`)

assert(themeJson.skin?.css?.length, 'theme.json must declare skin.css')
for (const rel of themeJson.skin.css) {
  assert(fs.existsSync(path.join(themeRoot, rel)), `skin css ${rel} must exist`)
}
if (themeJson.skin.tokens) {
  assert(fs.existsSync(path.join(themeRoot, themeJson.skin.tokens)), 'skin tokens file must exist')
}

const css = read(themeJson.skin.css[0])
// 至少保留 Signal Garden 视觉身份 token 或品牌类名之一。
const hasTokens = ['--sg-leaf', '--sg-sun', '--sg-coral', 'signal-garden', 'sg-hero'].some((token) => css.includes(token))
assert(hasTokens, 'theme CSS must retain Signal Garden visual markers')

const homeTemplate = read(home.template)
assert(homeTemplate.includes('sf-home-page') || homeTemplate.includes('forum.home') || homeTemplate.includes('signal-garden'), 'home template must reference host island or garden markers')

console.log('Signal Garden runtime theme validation passed.')
