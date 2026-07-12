/** 当前激活主题的公开设置（非 secret），供默认主题 layer 消费。 */

export type PublicActiveThemeSettings = {
  themeId: string
  settings: Record<string, string>
}

const emptyThemeSettings = (): PublicActiveThemeSettings => ({
  themeId: '',
  settings: {}
})

function normalizeEnabled(value: string | undefined, fallback = true) {
  const raw = `${value ?? ''}`.trim().toLowerCase()
  if (!raw) {
    return fallback
  }
  if (raw === 'enabled' || raw === 'true' || raw === '1' || raw === 'yes' || raw === 'on') {
    return true
  }
  if (raw === 'disabled' || raw === 'false' || raw === '0' || raw === 'no' || raw === 'off') {
    return false
  }
  return fallback
}

function clampInt(value: string | undefined, fallback: number, min: number, max: number) {
  const parsed = Number.parseInt(`${value ?? ''}`.trim(), 10)
  if (!Number.isFinite(parsed)) {
    return fallback
  }
  return Math.min(max, Math.max(min, Math.floor(parsed)))
}

export function useActiveThemeSettings() {
  const { request } = useApiClient()
  const { locale } = useI18n()

  const { data, pending, error, refresh } = useAsyncData(
    'site-active-theme-settings',
    async () => {
      try {
        return await request<PublicActiveThemeSettings>('/site/active-theme/settings')
      } catch {
        return emptyThemeSettings()
      }
    },
    { default: emptyThemeSettings }
  )

  const settings = computed(() => data.value?.settings || {})
  const themeId = computed(() => data.value?.themeId || '')

  function setting(key: string, fallback = '') {
    const value = settings.value[key]
    if (value === undefined || value === null) {
      return fallback
    }
    return `${value}`
  }

  function settingEnabled(key: string, fallback = true) {
    return normalizeEnabled(settings.value[key], fallback)
  }

  // 按当前 locale 取文案型设置：先精确 key，再 zh-CN / en-US 回退。
  function localizedSetting(baseKey: string, fallback = '') {
    const code = String(locale.value || 'zh-CN')
    const exact = setting(`${baseKey}.${code}`, '')
    if (exact.trim()) {
      return exact.trim()
    }
    if (code.startsWith('zh')) {
      return setting(`${baseKey}.zh-CN`, fallback).trim() || fallback
    }
    if (code.startsWith('en')) {
      return setting(`${baseKey}.en-US`, fallback).trim() || fallback
    }
    return (
      setting(`${baseKey}.zh-CN`, '').trim()
      || setting(`${baseKey}.en-US`, '').trim()
      || fallback
    )
  }

  const homeNotice = computed(() => localizedSetting('home.notice', ''))
  // 空状态标题/说明：有主题覆盖用覆盖，否则由页面回退 i18n
  const homeEmptyTitle = computed(() => localizedSetting('home.empty_title', ''))
  const homeEmptyDescription = computed(() => localizedSetting('home.empty_description', ''))
  const rightRailEnabled = computed(() => settingEnabled('home.right_rail.enabled', true))
  const rightRailShowHot = computed(() => settingEnabled('home.right_rail.show_hot', true))
  const rightRailShowStats = computed(() => settingEnabled('home.right_rail.show_stats', true))
  const rightRailShowTags = computed(() => settingEnabled('home.right_rail.show_tags', true))
  const rightRailShowAuthCard = computed(() => settingEnabled('home.right_rail.show_auth_card', true))
  const rightRailHotLimit = computed(() => clampInt(settings.value['home.right_rail.hot_limit'], 5, 3, 15))
  const rightRailTagLimit = computed(() => clampInt(settings.value['home.right_rail.tag_limit'], 10, 3, 20))
  const rightRailWelcome = computed(() => localizedSetting('home.right_rail.welcome', ''))
  const navShowCompose = computed(() => settingEnabled('home.nav.show_compose', true))
  const navShowCounts = computed(() => settingEnabled('home.nav.show_counts', true))
  const layoutShowFooter = computed(() => settingEnabled('layout.show_footer', true))
  const layoutShowAnnouncements = computed(() => settingEnabled('layout.show_announcements', true))

  return {
    themeId,
    settings,
    pending,
    error,
    refresh,
    setting,
    settingEnabled,
    localizedSetting,
    homeNotice,
    homeEmptyTitle,
    homeEmptyDescription,
    rightRailEnabled,
    rightRailShowHot,
    rightRailShowStats,
    rightRailShowTags,
    rightRailShowAuthCard,
    rightRailHotLimit,
    rightRailTagLimit,
    rightRailWelcome,
    navShowCompose,
    navShowCounts,
    layoutShowFooter,
    layoutShowAnnouncements
  }
}
