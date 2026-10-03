import { parsePublicContentStyleCatalog } from '~/runtime/public-extensions/contentStyles'

const CONTENT_STYLE_TIMEOUT_MS = import.meta.dev ? 5000 : 8000

export type PublicContentStyleLink = {
  href: string
  integrity: string
  handle: string
}

export function usePublicContentStyles() {
  const { request } = useApiClient()
  const links = useState<PublicContentStyleLink[]>('sforum-public-content-styles', () => [])
  let revision = 0

  async function refresh() {
    const requestedRevision = ++revision
    try {
      const response = await request<unknown>('/extensions/runtime/content-styles', {
        timeout: CONTENT_STYLE_TIMEOUT_MS,
        serverInternal: import.meta.server
      })
      if (requestedRevision !== revision) return false
      const catalog = parsePublicContentStyleCatalog(response)
      links.value = catalog.styles.map(style => ({
        href: style.assetPath,
        integrity: style.integrity,
        handle: style.handle
      }))
      return true
    } catch {
      if (requestedRevision === revision) links.value = []
      return false
    }
  }

  function clear() {
    revision++
    links.value = []
  }

  return { links, refresh, clear }
}
