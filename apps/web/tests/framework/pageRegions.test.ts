import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import {
  PAGE_REGION_PAGES,
  collectRegionWidgetRefs,
  parsePageRegionsPayload,
  safePageRegionHref
} from '../../app/composables/pages/usePageRegions'

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

  it('mirrors the backend RegionCatalog page whitelist', () => {
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
      'forum.topic.reply',
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

  it('every region page ships its outlets', () => {
    const placements: Record<string, string[]> = {
      'forum/SFTopicShowPage.vue': ['forum.topic.show'],
      'forum/SFCategoryShowPage.vue': ['forum.category.show'],
      'forum/SFCategoryIndexPage.vue': ['forum.category.index'],
      'forum/SFTagShowPage.vue': ['forum.tag.show'],
      'forum/SFTagIndexPage.vue': ['forum.tag.index'],
      'profile/SFProfileShowPage.vue': ['forum.profile.show'],
      'forum/SFTopicComposerPage.vue': ['forum.topic.create'],
      'forum/SFTopicEditPage.vue': ['forum.topic.edit'],
      'notifications/SFNotificationsPage.vue': ['forum.notifications']
    }
    for (const [file, pages] of Object.entries(placements)) {
      const content = source(`../../app/components/${file}`)
      for (const page of pages) {
        expect(content).toContain(`<SFRegionOutlet page="${page}" region="content_before"`)
        expect(content).toContain(`<SFRegionOutlet page="${page}" region="content_after"`)
      }
    }
    const home = source('../../app/components/forum/SFHomePage.vue')
    expect(home).toContain("const registryPage = computed(() => isSearchPage.value ? 'forum.search' : 'forum.home')")
    expect(home).toContain('<SFRegionOutlet :page="registryPage" region="content_before"')
    expect(home).toContain('<SFRegionOutlet :page="registryPage" region="content_after"')
    // 有右栏的页面提供 sidebar 区域。
    for (const file of [
      'forum/SFHomePage.vue', 'forum/SFTopicShowPage.vue', 'forum/SFCategoryIndexPage.vue',
      'forum/SFTagShowPage.vue', 'forum/SFTagIndexPage.vue', 'profile/SFProfileShowPage.vue'
    ]) {
      expect(source(`../../app/components/${file}`)).toContain('region="sidebar"')
    }
  })
})
