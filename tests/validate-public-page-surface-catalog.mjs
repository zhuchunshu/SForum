#!/usr/bin/env node
// 公开页运行时面目录的离线契约校验。
//
// 目录本身由 scripts/v3-catalog/generate.mjs 生成（--check 保证新鲜度），这里钉住
// 三件生成器不会自己拒绝的事：
//   1) 词表固定：正文来源恰好是 template|component|schema，layout 恰好是 site|bare；
//   2) 页面集合与 Go 的 Pages.coreCatalog 完全一致（既不多也不少）；
//   3) 目录内部自洽：每个页面的区域都真实存在，受审查正文岛在宿主岛清单里。
import { readFileSync } from 'node:fs'
import { spawnSync } from 'node:child_process'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const read = path => readFileSync(resolve(root, path), 'utf8')

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

const catalog = JSON.parse(read('docs/extensions/v3/catalogs/public-page-surfaces.json'))
const markdown = read('docs/extensions/v3/catalogs/public-page-surfaces.md')

assert(
  JSON.stringify(catalog.views) === JSON.stringify(['component', 'schema', 'template']),
  `public page views must be exactly component|schema|template, got ${JSON.stringify(catalog.views)}`
)
assert(
  JSON.stringify(catalog.layouts) === JSON.stringify(['bare', 'site']),
  `public page layouts must be exactly bare|site, got ${JSON.stringify(catalog.layouts)}`
)

const accessValues = new Set(['public', 'login', 'guest', 'moderation', 'permission'])
const regionIds = new Set(catalog.regions.map(region => region.id))
assert(regionIds.size > 0, 'catalog must declare at least one region')

const pageIds = new Set()
for (const page of catalog.pages) {
  assert(page.id && page.contractVersion, `page row is incomplete: ${JSON.stringify(page)}`)
  // Virtual 页面（错误边界等）没有路径；其余页面必须有路径，否则作者无法声明它。
  if (page.virtual) {
    assert(!page.pathPattern, `virtual page ${page.id} must not declare a path`)
  } else {
    assert(page.pathPattern, `page ${page.id} is missing its path pattern`)
  }
  assert(accessValues.has(page.access), `page ${page.id} declares unknown access ${page.access}`)
  assert(!pageIds.has(page.id), `page ${page.id} is listed twice`)
  pageIds.add(page.id)
  assert(
    JSON.stringify(page.views) === JSON.stringify(catalog.views),
    `page ${page.id} does not advertise the fixed view vocabulary`
  )
  assert(
    JSON.stringify(page.layouts) === JSON.stringify(catalog.layouts),
    `page ${page.id} does not advertise the fixed layout vocabulary`
  )
  for (const region of page.regions) {
    assert(regionIds.has(region), `page ${page.id} references unknown region ${region}`)
  }
}

// 受审查的正文出口与公共 chrome 岛必须在宿主岛清单里：主题靠它们挂载正文与顶栏/页脚。
for (const island of ['sf-plugin-page-body', 'sf-navbar', 'sf-footer']) {
  assert(catalog.hostIslands.includes(island), `host island ${island} is missing from the catalog`)
}

// 与 Go 唯一手写来源逐一对齐：既不能漏页，也不能凭空多页。
const goSource = read('apps/api/app/Support/Pages/catalog.go')
const start = goSource.indexOf('var coreCatalog = []PageDefinition{')
assert(start >= 0, 'cannot locate Pages.coreCatalog')
const body = goSource.slice(start, goSource.indexOf('\n}\n', start) + 1)
const goPageIds = new Set([...body.matchAll(/ID:\s*"([^"]+)"/g)].map(match => match[1]))
assert(goPageIds.size > 0, 'Pages.coreCatalog parsed empty')
for (const id of goPageIds) {
  assert(pageIds.has(id), `core page ${id} is missing from the public page surface catalog`)
}
for (const id of pageIds) {
  assert(goPageIds.has(id), `public page surface catalog lists unknown page ${id}`)
}

// Markdown 视图必须覆盖全部词表，否则读者会以为只有一种正文来源。
for (const value of [...catalog.views, ...catalog.layouts]) {
  assert(markdown.includes(`\`${value}\``), `markdown surface doc does not mention ${value}`)
}

// 生成器自检：目录与来源一致（避免手工编辑绕过生成）。
const check = spawnSync('node', ['scripts/v3-catalog/generate.mjs', '--check'], { cwd: root, encoding: 'utf8' })
assert(check.status === 0, `generated catalogs are stale:\n${check.stdout}${check.stderr}`)

console.log(
  `Public page surface catalog OK: ${pageIds.size} pages, ${regionIds.size} regions, ` +
  `${catalog.views.length} views, ${catalog.layouts.length} layouts, ${catalog.hostIslands.length} host islands.`
)
