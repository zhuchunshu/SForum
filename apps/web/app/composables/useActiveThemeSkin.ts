/**
 * L0 皮肤：从 API 拉取当前主题 CSS URL，交给 app.vue 的 useHead 输出。
 * SSR 首屏必须已经包含 link，避免 hydration 后注入导致整页重排。
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
  }

  return { links, refresh, clear }
}
