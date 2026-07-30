import { useApiClient } from '~/composables/useApiClient'
import { useWebOptions } from '~/composables/useWebOptions'
import { useAuthSession, type CurrentUser } from './useAuthSession'

type ConfiguredLocale = string | {
  code?: unknown
  language?: unknown
  name?: unknown
}

export type UserLanguageOption = {
  appCode: string
  label: string
  value: string
}

// The API stores BCP 47 language tags (for example en-US), while Nuxt i18n
// uses its own locale keys (en). Keep that mapping at this boundary.
export function useUserLanguage() {
  const { locales, setLocale } = useI18n()
  type AppLocaleCode = Parameters<typeof setLocale>[0]
  const { request } = useApiClient()
  const { user, setUser } = useAuthSession()
  const { defaultLocale, supportedLocales } = useWebOptions()

  const configuredLocales = computed(() => (locales.value as readonly ConfiguredLocale[]).flatMap((entry) => {
    const code = typeof entry === 'string' ? entry : entry.code
    if (typeof code !== 'string' || !code.trim()) {
      return []
    }
    const language = typeof entry === 'string' ? entry : entry.language
    const name = typeof entry === 'string' ? entry : entry.name
    return [{
      code,
      language: normalizeLocale(typeof language === 'string' ? language : code),
      name: typeof name === 'string' && name.trim() ? name : code
    }]
  }))

  const languageOptions = computed<UserLanguageOption[]>(() => supportedLocales.value.flatMap((item) => {
    const value = normalizeLocale(item)
    const configured = configuredLocales.value.find(entry => entry.language === value)
    return configured ? [{ appCode: configured.code, label: configured.name, value }] : []
  }))

  const siteDefaultLanguage = computed(() => resolveSupportedLocale(defaultLocale.value, languageOptions.value) || languageOptions.value[0]?.value || 'zh-CN')
  const currentLanguage = computed(() => resolveSupportedLocale(user.value?.locale, languageOptions.value) || siteDefaultLanguage.value)

  async function applyLanguage(language: string) {
    const configured = configuredLocales.value.find(entry => entry.language === normalizeLocale(language))
    if (configured) {
      await setLocale(configured.code as AppLocaleCode)
    }
  }

  async function updateLanguage(language: string) {
    const next = resolveSupportedLocale(language, languageOptions.value)
    if (!next) {
      throw new Error('identity.locale_invalid')
    }
    if (user.value) {
      const updated = await request<CurrentUser>('/auth/locale', {
        method: 'PUT',
        body: { locale: next }
      })
      setUser(updated)
    }
    await applyLanguage(next)
    return next
  }

  async function applyStoredLanguage() {
    if (user.value) {
      await applyLanguage(currentLanguage.value)
    }
  }

  return {
    currentLanguage,
    siteDefaultLanguage,
    languageOptions,
    updateLanguage,
    applyStoredLanguage
  }
}

function normalizeLocale(value: string | null | undefined) {
  const locale = value?.trim().toLowerCase()
  if (locale === 'en' || locale === 'en-us') return 'en-US'
  if (locale === 'zh' || locale === 'zh-cn' || locale === 'cn') return 'zh-CN'
  return value?.trim() || ''
}

function resolveSupportedLocale(value: string | null | undefined, options: UserLanguageOption[]) {
  const normalized = normalizeLocale(value)
  return options.find(option => option.value === normalized)?.value || ''
}
