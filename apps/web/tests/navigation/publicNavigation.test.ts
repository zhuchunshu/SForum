import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import {
  emptyPublicNavigation,
  isCoreDynamicCategories,
  isExternalNavigationItem,
  isInternalNavigationItem,
  limitDynamicNavigationItems,
  PUBLIC_NAVIGATION_LOCATIONS,
  PUBLIC_NAVIGATION_SCHEMA,
  publicNavigationItems,
  renderablePublicNavigationItems,
  type PublicNavigationDocument,
  type PublicNavigationItem
} from '../../app/utils/navigation/publicNavigation'

const source = (relative: string) => readFileSync(new URL(relative, import.meta.url), 'utf8')
const composableSource = source('../../app/composables/navigation/usePublicNavigation.ts')
const linksSource = source('../../app/components/navigation/SFPublicNavigationLinks.vue')
const mobileSource = source('../../app/components/navigation/SFPublicMobileNavigation.vue')
const sidebarSource = source('../../app/components/forum/SFHomeNavigation.vue')
const footerSource = source('../../app/components/SFFooter.vue')

function item(overrides: Partial<PublicNavigationItem> = {}): PublicNavigationItem {
  return {
    sourceKey: 'core.home',
    sourceKind: 'core',
    linkKind: 'coreRoute',
    label: 'Home',
    href: '/',
    ...overrides
  }
}

describe('public navigation document utilities', () => {
  test('selects only the requested canonical location and defaults missing locations to empty', () => {
    const document: PublicNavigationDocument = {
      schemaVersion: PUBLIC_NAVIGATION_SCHEMA,
      revision: 7,
      locations: [
        { location: PUBLIC_NAVIGATION_LOCATIONS.topbar, supported: true, items: [item()] },
        {
          location: PUBLIC_NAVIGATION_LOCATIONS.mobile,
          supported: true,
          items: [item({ sourceKey: 'operator.help', label: 'Help', href: '/help' })]
        }
      ]
    }

    expect(publicNavigationItems(document, PUBLIC_NAVIGATION_LOCATIONS.topbar).map(entry => entry.sourceKey)).toEqual(['core.home'])
    expect(publicNavigationItems(document, PUBLIC_NAVIGATION_LOCATIONS.mobile).map(entry => entry.sourceKey)).toEqual(['operator.help'])
    expect(publicNavigationItems(document, 'public.footer.primary')).toEqual([])
    expect(publicNavigationItems(null, PUBLIC_NAVIGATION_LOCATIONS.topbar)).toEqual([])
  })

  test('provides a schema-valid, fail-closed empty document', () => {
    expect(emptyPublicNavigation()).toEqual({
      schemaVersion: PUBLIC_NAVIGATION_SCHEMA,
      revision: 0,
      locations: []
    })
  })

  test('accepts safe internal and HTTP(S) external destinations only', () => {
    expect(isInternalNavigationItem(item({ href: '/categories' }))).toBe(true)
    expect(isInternalNavigationItem(item({ linkKind: 'extensionRoute', href: '/extensions/demo' }))).toBe(true)
    expect(isInternalNavigationItem(item({ href: '//evil.example/path' }))).toBe(false)
    expect(isInternalNavigationItem(item({ href: 'relative/path' }))).toBe(false)
    expect(isExternalNavigationItem(item({ linkKind: 'externalLink', href: 'https://example.com/docs' }))).toBe(true)
    expect(isExternalNavigationItem(item({ linkKind: 'externalLink', href: 'http://example.com' }))).toBe(true)
    expect(isExternalNavigationItem(item({ linkKind: 'externalLink', href: 'javascript:alert(1)' }))).toBe(false)
    expect(isExternalNavigationItem(item({ linkKind: 'externalLink', href: '//example.com' }))).toBe(false)
  })

  test('fails closed for dynamic blocks, unsafe URLs, and blank labels', () => {
    const validInternal = item()
    const validExternal = item({
      sourceKey: 'operator.docs',
      linkKind: 'externalLink',
      href: 'https://example.com',
      label: 'Docs'
    })
    const filtered = renderablePublicNavigationItems([
      validInternal,
      validExternal,
      item({ sourceKey: 'dynamic.categories', sourceKind: 'dynamic', linkKind: 'dynamicBlock', href: '/categories' }),
      item({ sourceKey: 'unsafe', href: '//evil.example' }),
      item({ sourceKey: 'blank', label: '   ' })
    ])

    expect(filtered).toEqual([validInternal, validExternal])
  })

  test('recognizes only the bounded Core category block', () => {
    expect(isCoreDynamicCategories(item({
      sourceKey: 'core.dynamic.categories',
      sourceKind: 'dynamic',
      linkKind: 'dynamicBlock',
      href: undefined
    }))).toBe(true)
    expect(isCoreDynamicCategories(item({
      sourceKey: 'plugin.dynamic.categories',
      sourceKind: 'extension',
      linkKind: 'dynamicBlock',
      href: undefined
    }))).toBe(false)
    expect(isCoreDynamicCategories(item({
      sourceKey: 'core.dynamic.categories',
      sourceKind: 'dynamic',
      linkKind: 'coreRoute'
    }))).toBe(false)
  })

  test('limits dynamic categories while retaining the selected category', () => {
    const categories = Array.from({ length: 6 }, (_, index) => ({ slug: `category-${index + 1}` }))

    expect(limitDynamicNavigationItems(categories, 0)).toEqual(categories)
    expect(limitDynamicNavigationItems(categories, undefined)).toEqual(categories)
    expect(limitDynamicNavigationItems(categories, 3).map(item => item.slug)).toEqual(['category-1', 'category-2', 'category-3'])
    expect(limitDynamicNavigationItems(categories, 3, 'category-6').map(item => item.slug)).toEqual(['category-1', 'category-2', 'category-6'])
  })
})

describe('public navigation fetch and renderer contracts', () => {
  test('uses one canonical request for all four public locations', () => {
    expect(composableSource.match(/request<PublicNavigationDocument>/g)?.length).toBe(1)
    expect(composableSource).toContain('`/site/navigation?locations=${encodeURIComponent(requestedLocations)}`')
    expect(composableSource).toContain('PUBLIC_NAVIGATION_LOCATIONS.topbar')
    expect(composableSource).toContain('PUBLIC_NAVIGATION_LOCATIONS.sidebar')
    expect(composableSource).toContain('PUBLIC_NAVIGATION_LOCATIONS.mobile')
    expect(composableSource).toContain('PUBLIC_NAVIGATION_LOCATIONS.footer')
    expect(composableSource).toContain('const sidebarItems = computed')
    expect(composableSource).toContain('const footerItems = computed')
    expect(composableSource).not.toContain('/site/nav-items')
  })

  test('isolates async data by locale and authoritative actor attributes', () => {
    expect(composableSource).toContain("if (!user.value) return 'guest'")
    expect(composableSource).toContain('user.value.id')
    expect(composableSource).toContain('user.value.status')
    expect(composableSource).toContain('user.value.roleKeys')
    expect(composableSource).toContain('user.value.permissions')
    expect(composableSource).toContain('`site-public-navigation:${locale.value}:${actorKey.value}`')
  })

  test('returns before useAsyncData when remote chrome is disabled', () => {
    const disabledBranch = composableSource.slice(
      composableSource.indexOf('if (!enabled)'),
      composableSource.indexOf('const { data, pending, refresh } = useAsyncData')
    )
    expect(disabledBranch).toContain('return {')
    expect(disabledBranch).toContain('topbarItems: computed(() => [])')
    expect(disabledBranch).toContain('sidebarItems: computed(() => [])')
    expect(disabledBranch).toContain('mobileItems: computed(() => [])')
    expect(disabledBranch).toContain('footerItems: computed(() => [])')
    expect(disabledBranch).not.toContain('request<PublicNavigationDocument>')
  })

  test('fails closed and disables shared SSR cache after request failure', () => {
    expect(composableSource).toContain('disableSharedDocumentCache()')
    expect(composableSource).toContain('return emptyState(true)')
    expect(composableSource).toContain('disableSharedPageCacheForPageResolve')
  })

  test('bounds topbar overflow and makes external links safe', () => {
    expect(linksSource).toContain('visibleLimit: 4')
    expect(linksSource).toContain('safeItems.value.slice(0, props.visibleLimit)')
    expect(linksSource).toContain('safeItems.value.slice(props.visibleLimit)')
    expect(linksSource).toContain('<UDropdownMenu')
    expect(linksSource).toContain("t('nav.more')")
    expect(linksSource).toContain('target="_blank"')
    expect(linksSource).toContain('rel="noopener noreferrer"')
  })

  test('renders mobile from its own location without the topbar limit', () => {
    expect(linksSource).toContain("props.mode === 'topbar'")
    expect(linksSource).toContain(':class="`sf-public-navigation-links--${mode}`"')
    expect(mobileSource).toContain('data-navigation-location="public.mobile.primary"')
    expect(mobileSource).toContain('<SFPublicNavigationLinks mode="mobile"')
    expect(mobileSource).toContain('@navigate="emit(\'close\')"')
  })

  test('renders sidebar links and the bounded category block in resolver order', () => {
    expect(sidebarSource).toContain('const { sidebarItems } = usePublicNavigation()')
    expect(sidebarSource).toContain('v-for="item in resolvedSidebarItems"')
    expect(sidebarSource).toContain('v-if="isCoreDynamicCategories(item)"')
    expect(sidebarSource.indexOf('v-for="item in resolvedSidebarItems"')).toBeLessThan(
      sidebarSource.indexOf('v-if="isCoreDynamicCategories(item)"')
    )
    expect(sidebarSource).toContain("navigationMode?: 'filter' | 'route'")
    expect(sidebarSource).toContain("emit('select-category', slug)")
    expect(sidebarSource).toContain('void router.push(slug ? categoryTo(slug) : allTopicsTo())')
    expect(sidebarSource).toContain("return 'i-lucide-folder'")
    expect(sidebarSource).toContain('category.topicCount')
    expect(sidebarSource).toContain("selectedCategorySlug === category.slug")
    expect(sidebarSource).toContain('const visibleCategories = computed')
    expect(sidebarSource).toContain('dynamicCategoryItem.value?.maxItems')
    expect(sidebarSource).toContain('limitDynamicNavigationItems(props.categories')
    expect(sidebarSource).toContain('v-for="category in visibleCategories"')
    expect(sidebarSource).toContain('const hasHiddenCategories = computed')
    expect(sidebarSource).toContain("t('home.sidebar.viewAllCategories')")
    expect(sidebarSource).toContain(':to="localePath(\'/categories\')"')
  })

  test('fails the dynamic category block closed without hiding ordinary sidebar links', () => {
    expect(sidebarSource).toContain('if (isCoreDynamicCategories(item)) return props.showCategories')
    expect(sidebarSource).toContain('return Boolean(item.label.trim()) && (isExternalNavigationItem(item) || isInternalNavigationItem(item))')
    expect(sidebarSource).toContain('props.showCategories && Boolean(dynamicCategoryItem.value)')
    expect(sidebarSource).toContain('v-if="!desktopOnly && hasDynamicCategories"')
    expect(sidebarSource).not.toContain("t('home.sidebar.guidelines')")
  })

  test('isolates external sidebar and footer destinations from the opener', () => {
    for (const renderer of [sidebarSource, linksSource, footerSource]) {
      expect(renderer).toContain('target="_blank"')
      expect(renderer).toContain('rel="noopener noreferrer"')
    }
  })

  test('keeps canonical footer navigation beside copyright and friend-link owners', () => {
    expect(footerSource).toContain('const { footerItems } = usePublicNavigation(props.fetchRemoteChrome)')
    expect(footerSource).toContain('<SFPublicNavigationLinks')
    expect(footerSource).toContain('mode="footer"')
    expect(footerSource).toContain(':items="footerItems"')
    expect(footerSource).toContain('footerCopyrightTemplate(locale.value)')
    expect(footerSource).toContain('chromeApi.listPublicFriendLinks()')
    expect(footerSource).not.toContain('webOptions.footerLinks')
  })
})
