import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import {
  PAGE_REGION_PAGES,
  collectRegionWidgetRefs,
  parsePageRegionsPayload,
  safePageRegionHref
} from '../../app/composables/pages/usePageRegions'
import { PAGE_REGION_IDS } from '../../app/config/pageRegionCatalog.gen'

function source(relative: string) {
  return readFileSync(new URL(relative, import.meta.url), 'utf8')
}

const validPayload = {
  schemaVersion: 'sforum.page-regions@1',
  page: 'forum.home',
  regions: [
    {
      id: 'content_after',
      kind: 'content',
      items: [
        {
          extensionId: 'demo.plugin',
          contributionId: 'demo.link',
          label: { 'en-US': 'Browse tags' },
          icon: 'i-lucide-tags',
          kind: 'link',
          href: '/tags',
          order: 10
        },
        {
          extensionId: 'demo.plugin',
          contributionId: 'demo.action',
          label: { 'en-US': 'Ping' },
          kind: 'action',
          method: 'POST',
          path: '/region/ping',
          order: 20
        },
        {
          extensionId: 'demo.plugin',
          contributionId: 'demo.widget',
          label: { 'en-US': 'Widget' },
          kind: 'widget',
          widget: { extensionId: 'demo.plugin', componentId: 'demo.widget.component' },
          order: 30
        }
      ]
    }
  ]
}

describe('forum.page.regions payload parsing', () => {
  it('parses host descriptors and keeps region order', () => {
    const parsed = parsePageRegionsPayload(validPayload, 'forum.home')
    expect(parsed).not.toBeNull()
    expect(parsed!.regions).toHaveLength(1)
    const items = parsed!.regions[0]!.items
    expect(items.map(item => item.kind)).toEqual(['link', 'action', 'widget'])
    expect(items[0]!.href).toBe('/tags')
    expect(items[1]!.method).toBe('POST')
    expect(items[2]!.widget).toEqual({ extensionId: 'demo.plugin', componentId: 'demo.widget.component' })
  })

  it('fails closed on schema or page mismatch', () => {
    expect(parsePageRegionsPayload({ ...validPayload, schemaVersion: 'v2' }, 'forum.home')).toBeNull()
    expect(parsePageRegionsPayload(validPayload, 'forum.topic.show')).toBeNull()
    expect(parsePageRegionsPayload(null, 'forum.home')).toBeNull()
  })

  it('drops unsafe or cross-extension items but keeps the rest', () => {
    const tampered = structuredClone(validPayload)
    tampered.regions[0]!.items[0]!.href = 'https://evil.example/'
    tampered.regions[0]!.items[1]!.path = '/api/secrets'
    tampered.regions[0]!.items[2]!.widget!.extensionId = 'other.plugin'
    const parsed = parsePageRegionsPayload(tampered, 'forum.home')
    // 全部条目非法 → 区域整体省略。
    expect(parsed).not.toBeNull()
    expect(parsed!.regions).toHaveLength(0)
  })

  it('collects deduplicated widget refs for CSP aggregation', () => {
    const parsed = parsePageRegionsPayload(validPayload, 'forum.home')
    expect(collectRegionWidgetRefs(parsed)).toEqual([
      { extensionId: 'demo.plugin', componentId: 'demo.widget.component' }
    ])
    expect(collectRegionWidgetRefs(null)).toEqual([])
  })

  it('whitelists host-relative hrefs only', () => {
    expect(safePageRegionHref('/tags')).toBe(true)
    expect(safePageRegionHref('//evil.example')).toBe(false)
    expect(safePageRegionHref('https://evil.example')).toBe(false)
    expect(safePageRegionHref('/api/secrets')).toBe(false)
    expect(safePageRegionHref('/a/../b')).toBe(false)
  })

  it('accepts a payload that fills every region the catalog declares for a page', () => {
    // MAX_REGIONS 由生成的区域目录推导：写死的 8 会把 forum.topic.show（9 个区域）
    // 的合法 payload 整页判成非法，于是页面静默失去所有区域内容。
    const page = 'forum.topic.show'
    const template = validPayload.regions[0]!
    const regions = PAGE_REGION_IDS[page]!.map(id => ({
      id,
      kind: 'content',
      items: template.items
    }))
    const parsed = parsePageRegionsPayload({
      schemaVersion: 'sforum.page-regions@1',
      page,
      regions
    }, page)
    expect(parsed?.regions).toHaveLength(regions.length)
  })

  it('mirrors the backend RegionCatalog page whitelist', () => {
    // PAGE_REGION_PAGES 现在由 scripts/v3-catalog/generate.mjs 从
    // apps/api/app/Support/RegionCatalog/catalog.go 生成，不再手工维护；
    // 这里钉住结果集合，让目录的增删必须显式经过这个测试。
    expect([...PAGE_REGION_PAGES].sort()).toEqual([
      'forum.category.index',
      'forum.category.show',
      'forum.home',
      'forum.notifications',
      'forum.profile.show',
      'forum.search',
      'forum.tag.index',
      'forum.tag.show',
      'forum.topic.create',
      'forum.topic.edit',
      'forum.topic.show'
    ])
  })
})

describe('region outlet wiring contracts', () => {
  it('SFRegionOutlet renders descriptors and delegates widgets to SFExtensionWidget', () => {
    const outlet = source('../../app/components/SFRegionOutlet.vue')
    expect(outlet).toContain('usePageRegionsState')
    expect(outlet).toContain('SFExtensionWidget')
    expect(outlet).toContain('`${item.extensionId}:${item.contributionId}`')
    // 出口自身不做 CSP 聚合（严禁双写）。
    expect(outlet).not.toContain('applyPublicPageDocumentPolicy')
  })

  it('SFPageOutletResolver fetches regions at SSR and applies CSP exactly once per path', () => {
    const resolver = source('../../app/components/SFPageOutletResolver.vue')
    expect(resolver).toContain('fetchPageRegions')
    expect(resolver).toContain('usePageRegionsState')
    expect(resolver).toContain('collectRegionWidgetRefs')
    // 原生路径由 Resolver 聚合;主题模板路径把 refs 交给 SFThemeTemplate。
    expect(resolver).toContain('!willUseThemeTemplate.value')
    expect(resolver).toContain(':region-widget-refs="regionWidgetRefs"')
  })

  it('SFThemeTemplate merges region widget refs into its single CSP aggregation', () => {
    const template = source('../../app/components/SFThemeTemplate.vue')
    expect(template).toContain('extraL2Refs')
    expect(template).toContain('normalizePublicFrontendComponentRefs([...themeRefs, ...(props.extraL2Refs ?? [])])')
    expect(template.match(/applyPublicPageDocumentPolicy\(/g)).toHaveLength(1)
  })

  it('every declared page×region has a real render site', () => {
    // 挂载点只存在于模板里，所以这是不可避免的静态契约。用生成的区域目录驱动，
    // 而不是手写页面清单：手写清单曾经漏掉 forum.topic.reply，而那个页面组件只是
    // 一个跳转占位、没有任何 outlet——作者会声明出一个永远不渲染的插槽。
    const placements: Record<string, string> = {
      'forum.home': 'forum/SFHomePage.vue',
      // SFHomePage 同时承载首页与搜索页，页面 id 由 registryPage 决定。
      'forum.search': 'forum/SFHomePage.vue',
      'forum.category.index': 'forum/SFCategoryIndexPage.vue',
      'forum.category.show': 'forum/SFCategoryShowPage.vue',
      'forum.tag.index': 'forum/SFTagIndexPage.vue',
      'forum.tag.show': 'forum/SFTagShowPage.vue',
      'forum.topic.show': 'forum/SFTopicShowPage.vue',
      'forum.profile.show': 'profile/SFProfileShowPage.vue',
      'forum.topic.create': 'forum/SFTopicComposerPage.vue',
      'forum.topic.edit': 'forum/SFTopicEditPage.vue',
      'forum.notifications': 'notifications/SFNotificationsPage.vue'
    }
    const contents = new Map<string, string>()
    for (const file of new Set(Object.values(placements))) {
      contents.set(file, source(`../../app/components/${file}`))
    }

    const missing: string[] = []
    for (const [page, regions] of Object.entries(PAGE_REGION_IDS)) {
      const file = placements[page]
      if (!file) {
        missing.push(`${page} (no component mapping)`)
        continue
      }
      const content = contents.get(file) || ''
      for (const region of regions) {
        if (!content.includes(`region="${region}"`)) {
          missing.push(`${page}/${region} → ${file}`)
        }
      }
    }
    expect(missing).toEqual([])

    // 反向：outlet 不得引用目录之外的区域。
    const declared = new Set(Object.values(PAGE_REGION_IDS).flat())
    const orphans: string[] = []
    for (const [file, content] of contents) {
      for (const match of content.matchAll(/region="([a-z0-9_]+)"/g)) {
        if (!declared.has(match[1])) {
          orphans.push(`${file}: region="${match[1]}"`)
        }
      }
    }
    expect(orphans).toEqual([])

    // 跳转占位页面不得重新出现在目录里（重新声明等于给作者一个死插槽）。
    expect(Object.keys(PAGE_REGION_IDS)).not.toContain('forum.topic.reply')

    const home = source('../../app/components/forum/SFHomePage.vue')
    expect(home).toContain("const registryPage = computed(() => isSearchPage.value ? 'forum.search' : 'forum.home')")

    // 有右栏的页面提供 sidebar 区域。
    for (const file of [
      'forum/SFHomePage.vue', 'forum/SFTopicShowPage.vue', 'forum/SFCategoryIndexPage.vue',
      'forum/SFTagShowPage.vue', 'forum/SFTagIndexPage.vue', 'profile/SFProfileShowPage.vue'
    ]) {
      expect(contents.get(file) || source(`../../app/components/${file}`)).toContain('region="sidebar"')
    }
  })
})
