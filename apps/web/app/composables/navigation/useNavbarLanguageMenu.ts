import { useUserLanguage } from '~/composables/identity/useUserLanguage'

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

export function useNavbarLanguageMenu() {
  const { locale } = useI18n()
  const { languageOptions, updateLanguage } = useUserLanguage()
  const currentLocaleName = computed(() =>
    languageOptions.value.find(entry => entry.appCode === locale.value)?.label || locale.value
  )

  // no_prefix：切换后 URL 保持不变；登录用户会同步更新默认语言和 i18n cookie。
  const languageMenuItems = computed<NavbarMenuItem[]>(() =>
    languageOptions.value.map((entry) => {
      const isCurrent = entry.appCode === locale.value
      return {
        label: entry.label,
        icon: isCurrent ? 'i-lucide-check' : 'i-tabler-language',
        active: isCurrent,
        onSelect: (event: Event) => {
          if (isCurrent) {
            event.preventDefault()
            return
          }
          void updateLanguage(entry.value)
        }
      }
    })
  )

  return { currentLocaleName, languageMenuItems }
}
