import { useAuthSession } from '~/composables/identity/useAuthSession'
import { disableSharedPageCacheForPageResolve, type MutableRouteRulesContext } from '~/utils/pageResolve'
import {
  emptyPublicNavigation,
  PUBLIC_NAVIGATION_LOCATIONS,
  publicNavigationItems,
  type PublicNavigationDocument
} from '~/utils/navigation/publicNavigation'

type PublicNavigationState = {
  document: PublicNavigationDocument
  failed: boolean
}

const requestedLocations = [
  PUBLIC_NAVIGATION_LOCATIONS.topbar,
  PUBLIC_NAVIGATION_LOCATIONS.sidebar,
  PUBLIC_NAVIGATION_LOCATIONS.footer
].join(',')

function emptyState(failed = false): PublicNavigationState {
  return { document: emptyPublicNavigation(), failed }
}

export function usePublicNavigation(enabled = true) {
  const { request } = useApiClient()
  const { locale } = useI18n()
  const { user } = useAuthSession()
  const responseCacheControl = import.meta.server ? useResponseHeader('cache-control') : undefined
  const requestEvent = import.meta.server ? useRequestEvent() : undefined

  const actorKey = computed(() => {
    if (!user.value) return 'guest'
    const roles = [...(user.value.roleKeys || [])].sort().join(',')
    const permissions = [...(user.value.permissions || [])].sort().join(',')
    return `user:${user.value.id}:${user.value.status}:${roles}:${permissions}`
  })
  const navigationKey = computed(() => `site-public-navigation:${locale.value}:${actorKey.value}`)

  function disableSharedDocumentCache() {
    if (!import.meta.server) return
    disableSharedPageCacheForPageResolve(
      requestEvent?.context as MutableRouteRulesContext | undefined,
      value => {
        if (responseCacheControl) responseCacheControl.value = value
      }
    )
  }

  if (!enabled) {
    const state = shallowRef(emptyState())
    return {
      document: computed(() => state.value.document),
      topbarItems: computed(() => []),
      sidebarItems: computed(() => []),
      footerItems: computed(() => []),
      pending: readonly(ref(false)),
      failed: computed(() => false),
      refresh: async () => state.value
    }
  }

  const { data, pending, refresh } = useAsyncData(
    navigationKey,
    async (): Promise<PublicNavigationState> => {
      try {
        const document = await request<PublicNavigationDocument>(
          `/site/navigation?locations=${encodeURIComponent(requestedLocations)}`,
          { serverInternal: import.meta.server }
        )
        return { document, failed: false }
      } catch {
        disableSharedDocumentCache()
        return emptyState(true)
      }
    },
    {
      default: () => emptyState(),
      watch: [() => locale.value, actorKey]
    }
  )

  const document = computed(() => data.value?.document || emptyPublicNavigation())
  const topbarItems = computed(() => publicNavigationItems(document.value, PUBLIC_NAVIGATION_LOCATIONS.topbar))
  const sidebarItems = computed(() => publicNavigationItems(document.value, PUBLIC_NAVIGATION_LOCATIONS.sidebar))
  const footerItems = computed(() => publicNavigationItems(document.value, PUBLIC_NAVIGATION_LOCATIONS.footer))

  return {
    document,
    topbarItems,
    sidebarItems,
    footerItems,
    pending,
    failed: computed(() => data.value?.failed === true),
    refresh
  }
}
