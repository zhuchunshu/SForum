import type {
  SiteNavigationDefinition,
  SiteNavigationDocument,
  SiteNavigationLocation,
  SiteNavigationPlacement
} from '~/composables/admin/useSiteChromeApi'

export const navigationLocations: SiteNavigationLocation[] = [
  'public.topbar.primary',
  'public.sidebar.primary',
  'public.mobile.primary',
  'public.footer.primary'
]

export type NavigationEditorItem = { definition: SiteNavigationDefinition, placement: SiteNavigationPlacement }

export function cloneNavigationDocument(document: SiteNavigationDocument): SiteNavigationDocument {
  return JSON.parse(JSON.stringify(document)) as SiteNavigationDocument
}

export function navigationItemsAt(document: SiteNavigationDocument, location: SiteNavigationLocation): NavigationEditorItem[] {
  const definitions = new Map(document.definitions.map(definition => [definition.sourceKey, definition]))
  return document.placements
    .filter(placement => placement.location === location)
    .map(placement => ({ placement, definition: definitions.get(placement.sourceKey) }))
    .filter((item): item is NavigationEditorItem => Boolean(item.definition))
    .sort((left, right) => left.placement.order - right.placement.order || left.placement.sourceKey.localeCompare(right.placement.sourceKey))
}

export function reorderNavigationLocation(document: SiteNavigationDocument, location: SiteNavigationLocation, sourceKeys: string[]) {
  const positions = new Map(sourceKeys.map((sourceKey, index) => [sourceKey, (index + 1) * 10]))
  for (const placement of document.placements) {
    if (placement.location === location && positions.has(placement.sourceKey)) placement.order = positions.get(placement.sourceKey)!
  }
}

export function moveNavigationItem(document: SiteNavigationDocument, location: SiteNavigationLocation, sourceKey: string, targetIndex: number) {
  const keys = navigationItemsAt(document, location).map(item => item.definition.sourceKey).filter(key => key !== sourceKey)
  keys.splice(Math.max(0, Math.min(targetIndex, keys.length)), 0, sourceKey)
  reorderNavigationLocation(document, location, keys)
}

export function transferNavigationItem(document: SiteNavigationDocument, sourceKey: string, from: SiteNavigationLocation, to: SiteNavigationLocation, copy: boolean) {
  const existing = document.placements.find(placement => placement.sourceKey === sourceKey && placement.location === from)
  if (!existing || from === to) return
  if (!copy) document.placements = document.placements.filter(placement => !(placement.sourceKey === sourceKey && placement.location === from))
  const target = document.placements.find(placement => placement.sourceKey === sourceKey && placement.location === to)
  if (!target) document.placements.push({ ...existing, location: to, order: (navigationItemsAt(document, to).length + 1) * 10 })
  reorderNavigationLocation(document, to, navigationItemsAt(document, to).map(item => item.definition.sourceKey))
}

export function removeNavigationDefinition(document: SiteNavigationDocument, sourceKey: string) {
  document.definitions = document.definitions.filter(definition => definition.sourceKey !== sourceKey)
  document.placements = document.placements.filter(placement => placement.sourceKey !== sourceKey)
}

export function navigationDocumentsEqual(left: SiteNavigationDocument | null, right: SiteNavigationDocument | null) {
  return JSON.stringify(left) === JSON.stringify(right)
}
