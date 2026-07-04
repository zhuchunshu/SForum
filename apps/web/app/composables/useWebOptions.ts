import type { ApiEnvelope } from '~/composables/useApiClient'

export type WebOption = {
  name: string
  value: string
}

type RefreshOptions = {
  timeout?: number
}

const fallbackOptions: Record<string, string> = {
  'site.name': 'SForum'
}

export const useWebOptions = () => {
  const { apiBaseUrl, apiHeaders, request } = useApiClient()
  const options = useState<Record<string, string>>('web-options', () => ({ ...fallbackOptions }))

  async function refresh(requestOptions: RefreshOptions = {}) {
    const items = await request<WebOption[]>('/web-options', {
      timeout: requestOptions.timeout
    })
    options.value = {
      ...fallbackOptions,
      ...Object.fromEntries(items.map((item) => [item.name, item.value]))
    }
    return options.value
  }

  async function save(name: string, value: string) {
    const item = await request<WebOption>('/web-options', {
      method: 'PUT',
      body: { name, value }
    })
    options.value = {
      ...options.value,
      [item.name]: item.value
    }
    return item
  }

  async function fetchEnvelope() {
    return await $fetch<ApiEnvelope<WebOption[]>>(`${apiBaseUrl}/web-options`, {
      credentials: 'include',
      headers: apiHeaders()
    })
  }

  function webOption(name: string, fallback = '') {
    return options.value[name] || fallbackOptions[name] || fallback
  }

  const siteName = computed(() => webOption('site.name', 'SForum'))

  return {
    options,
    siteName,
    webOption,
    refresh,
    save,
    fetchEnvelope
  }
}
