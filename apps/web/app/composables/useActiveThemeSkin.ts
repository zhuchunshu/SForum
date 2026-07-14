/**
 * L0 皮肤：从 API 拉取当前主题 CSS URL 并注入 document。
 * 不触发 Nuxt rebuild / Nitro 切换。
 */
export function useActiveThemeSkin() {
  const links = useState<string[]>('sforum-active-theme-css', () => [])
  let revision = 0

  async function refresh() {
    const requestedRevision = ++revision
    try {
      const { request } = useApiClient()
      const skin = await request<{ css?: string[], tokens?: string }>('/site/active-theme/skin')
      if (requestedRevision !== revision) {
        return
      }
      const hrefs = [...(skin?.css || [])]
      if (skin?.tokens) {
        hrefs.unshift(skin.tokens)
      }
      links.value = hrefs
      if (import.meta.client) {
        applyStylesheets(hrefs)
      }
    } catch {
      if (requestedRevision !== revision) {
        return
      }
      links.value = []
    }
  }

  function clear() {
    revision++
    links.value = []
    if (import.meta.client) {
      applyStylesheets([])
    }
  }

  function applyStylesheets(hrefs: string[]) {
    const marker = 'data-sforum-theme-skin'
    document.querySelectorAll(`link[${marker}]`).forEach(el => el.remove())
    for (const href of hrefs) {
      const link = document.createElement('link')
      link.rel = 'stylesheet'
      link.href = href
      link.setAttribute(marker, '1')
      document.head.appendChild(link)
    }
  }

  return { links, refresh, clear }
}
