const supportedLocaleCodes = ['zh-CN', 'en'] as const

type LocaleCode = typeof supportedLocaleCodes[number]
type LocaleOption = {
  code: LocaleCode
  name?: string
}
type ConfiguredLocale = string | {
  code?: unknown
  name?: unknown
}

export type NavbarMenuItem = {
  label: string
  description?: string
  icon?: string
  to?: string
  /** 覆盖 ULink 的路由 active，语言项按 i18n locale 判定。 */
  active?: boolean
  type?: 'label' | 'checkbox'
  checked?: boolean
  color?: 'error'
  onSelect?: (event: Event) => void
  onUpdateChecked?: (checked: boolean) => void
  children?: NavbarMenuItem[]
}

function isLocaleCode(value: string): value is LocaleCode {
  return supportedLocaleCodes.includes(value as LocaleCode)
}

export function useNavbarLanguageMenu() {
  const { locale, locales, setLocale } = useI18n()
  const localeOptions = computed<LocaleOption[]>(() =>
    (locales.value as readonly ConfiguredLocale[]).flatMap((entry) => {
      const code = typeof entry === 'string' ? entry : entry.code
      if (typeof code !== 'string' || !isLocaleCode(code)) {
        return []
      }
      const name = typeof entry === 'string' ? entry : entry.name
      return [{ code, name: typeof name === 'string' && name.trim() ? name : code }]
    })
  )
  const currentLocaleName = computed(() =>
    localeOptions.value.find(entry => entry.code === locale.value)?.name || locale.value
  )

  // no_prefix：setLocale 只换文案与 cookie，URL 保持不变。
  const languageMenuItems = computed<NavbarMenuItem[]>(() =>
    localeOptions.value.map((entry) => {
      const isCurrent = entry.code === locale.value
      return {
        label: entry.name || entry.code,
        icon: isCurrent ? 'i-lucide-check' : 'i-tabler-language',
        active: isCurrent,
        onSelect: (event: Event) => {
          if (isCurrent) {
            event.preventDefault()
            return
          }
          void setLocale(entry.code)
        }
      }
    })
  )

  return { currentLocaleName, languageMenuItems }
}
