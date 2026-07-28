export const PUBLIC_NAVIGATION_SCHEMA = 'sforum.site-navigation@1' as const

export const PUBLIC_NAVIGATION_LOCATIONS = {
  topbar: 'public.topbar.primary',
  sidebar: 'public.sidebar.primary',
  mobile: 'public.mobile.primary',
  footer: 'public.footer.primary'
} as const

export type PublicNavigationLocation =
  | 'public.topbar.primary'
  | 'public.sidebar.primary'
  | 'public.mobile.primary'
  | 'public.footer.primary'

export type PublicNavigationLinkKind =
  | 'coreRoute'
  | 'internalLink'
  | 'externalLink'
  | 'extensionHostLink'
  | 'extensionRoute'
  | 'dynamicBlock'

export type PublicNavigationItem = {
  sourceKey: string
  sourceKind: 'core' | 'operator' | 'extension' | 'dynamic'
  linkKind: PublicNavigationLinkKind
  label: string
  href?: string
  icon?: string
  iconHidden?: boolean
  maxItems?: number
  openInNewTab?: boolean
}

export type PublicNavigationLocationView = {
  location: PublicNavigationLocation
  supported: boolean
  items: PublicNavigationItem[]
}

export type PublicNavigationDocument = {
  schemaVersion: typeof PUBLIC_NAVIGATION_SCHEMA
  revision: number
  locations: PublicNavigationLocationView[]
}

export function emptyPublicNavigation(): PublicNavigationDocument {
  return {
    schemaVersion: PUBLIC_NAVIGATION_SCHEMA,
    revision: 0,
    locations: []
  }
}

export function publicNavigationItems(
  document: PublicNavigationDocument | null | undefined,
  location: PublicNavigationLocation
) {
  return document?.locations.find(entry => entry.location === location)?.items || []
}

export function isExternalNavigationItem(item: PublicNavigationItem) {
  return item.linkKind === 'externalLink'
    && /^https?:\/\/[^/]/i.test((item.href || '').trim())
}

export function isInternalNavigationItem(item: PublicNavigationItem) {
  if (!['coreRoute', 'internalLink', 'extensionHostLink', 'extensionRoute'].includes(item.linkKind)) {
    return false
  }
  const href = (item.href || '').trim()
  return href.startsWith('/') && !href.startsWith('//')
}

export function renderablePublicNavigationItems(items: PublicNavigationItem[]) {
  return items.filter(item => Boolean(item.label.trim()))
    .filter(item => isExternalNavigationItem(item) || isInternalNavigationItem(item))
}

export function isCoreDynamicCategories(item: PublicNavigationItem) {
  return item.sourceKey === 'core.dynamic.categories'
    && item.sourceKind === 'dynamic'
    && item.linkKind === 'dynamicBlock'
    && !item.href
}

export function limitDynamicNavigationItems<T extends { slug: string }>(items: readonly T[], maxItems?: number, selectedSlug = '') {
  const limit = Math.min(100, Math.max(0, Math.trunc(Number(maxItems) || 0)))
  if (limit === 0 || items.length <= limit) return [...items]
  const visible = items.slice(0, limit)
  const selected = items.find(item => item.slug === selectedSlug)
  if (selected && !visible.some(item => item.slug === selected.slug)) {
    visible[limit - 1] = selected
  }
  return visible
}
