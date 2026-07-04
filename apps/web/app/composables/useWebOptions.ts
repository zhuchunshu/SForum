import type { ApiEnvelope } from '~/composables/useApiClient'

export type WebOption = {
  name: string
  value: string
}

export type AdminWebOption = WebOption & {
  public: boolean
  secret: boolean
  secretSet: boolean
}

type RefreshOptions = {
  timeout?: number
}

const fallbackOptions: Record<string, string> = {
  'site.name': 'SForum',
  'site.url': 'http://127.0.0.1:3000',
  'site.default_locale': 'zh-CN',
  'site.supported_locales': 'zh-CN,en-US',
  'human_verification.provider': 'disabled'
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

  async function fetchAdminEnvelope() {
    return await $fetch<ApiEnvelope<AdminWebOption[]>>(`${apiBaseUrl}/admin/web-options`, {
      credentials: 'include',
      headers: apiHeaders()
    })
  }

  async function saveMany(items: WebOption[]) {
    const adminItems = await request<AdminWebOption[]>('/admin/web-options', {
      method: 'PUT',
      body: { options: items }
    })

    const publicItems = adminItems.filter((item) => item.public && !item.secret)
    options.value = {
      ...options.value,
      ...Object.fromEntries(publicItems.map((item) => [item.name, item.value]))
    }
    return adminItems
  }

  function webOption(name: string, fallback = '') {
    return options.value[name] || fallbackOptions[name] || fallback
  }

  const siteName = computed(() => webOption('site.name', 'SForum'))
  const siteUrl = computed(() => webOption('site.url', 'http://127.0.0.1:3000'))
  const defaultLocale = computed(() => webOption('site.default_locale', 'zh-CN'))
  const supportedLocales = computed(() => parseSupportedLocales(webOption('site.supported_locales', 'zh-CN,en-US')))
  const humanVerificationProvider = computed(() => {
    const provider = webOption('human_verification.provider', 'disabled').trim().toLowerCase()
    return provider === 'altcha' ? 'altcha' : 'disabled'
  })

  return {
    options,
    siteName,
    siteUrl,
    defaultLocale,
    supportedLocales,
    humanVerificationProvider,
    webOption,
    refresh,
    save,
    saveMany,
    fetchEnvelope,
    fetchAdminEnvelope
  }
}

function parseSupportedLocales(value: string) {
  const locales = value.split(',').map((item) => item.trim()).filter(Boolean)
  return locales.length > 0 ? locales : ['zh-CN', 'en-US']
}
