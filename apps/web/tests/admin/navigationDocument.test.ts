import { describe, expect, test } from 'bun:test'
import type { SiteNavigationDocument } from '../../app/composables/admin/useSiteChromeApi'
import {
  moveNavigationItem,
  navigationItemsAt,
  reorderNavigationLocation,
  transferNavigationItem
} from '../../app/utils/admin/navigationDocument'

function document(): SiteNavigationDocument {
  return {
    revision: 8,
    definitions: [
      { sourceKey: 'operator.one', sourceKind: 'operator', linkKind: 'internalLink', labelZhCN: '一', href: '/one' },
      { sourceKey: 'operator.two', sourceKind: 'operator', linkKind: 'internalLink', labelZhCN: '二', href: '/two' }
    ],
    placements: [
      { sourceKey: 'operator.one', location: 'public.topbar.primary', order: 20, enabled: true, visibility: 'public' },
      { sourceKey: 'operator.two', location: 'public.topbar.primary', order: 10, enabled: true, visibility: 'public' }
    ],
    themeLocations: []
  }
}

describe('navigation document editor helpers', () => {
  test('reorders a location using stable ten-step positions', () => {
    const draft = document()
    reorderNavigationLocation(draft, 'public.topbar.primary', ['operator.one', 'operator.two'])
    expect(navigationItemsAt(draft, 'public.topbar.primary').map(item => [item.definition.sourceKey, item.placement.order])).toEqual([
      ['operator.one', 10], ['operator.two', 20]
    ])
    moveNavigationItem(draft, 'public.topbar.primary', 'operator.two', 0)
    expect(navigationItemsAt(draft, 'public.topbar.primary').map(item => item.definition.sourceKey)).toEqual(['operator.two', 'operator.one'])
  })

  test('moves and copies one source between independent locations without database ids', () => {
    const draft = document()
    transferNavigationItem(draft, 'operator.one', 'public.topbar.primary', 'public.footer.primary', true)
    expect(navigationItemsAt(draft, 'public.footer.primary').map(item => item.definition.sourceKey)).toEqual(['operator.one'])
    transferNavigationItem(draft, 'operator.two', 'public.topbar.primary', 'public.mobile.primary', false)
    expect(navigationItemsAt(draft, 'public.topbar.primary').map(item => item.definition.sourceKey)).toEqual(['operator.one'])
    expect(navigationItemsAt(draft, 'public.mobile.primary').map(item => item.definition.sourceKey)).toEqual(['operator.two'])
  })
})
