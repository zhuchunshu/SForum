import {
  formatSiteDateTime,
  resolveSiteDateTimeSettings,
  type SiteDateTimeSettings
} from '~/utils/siteDateTime'

/**
 * 读取 public web_options 中的站点时区/日期时间配置，并提供统一格式化入口。
 * 各页面应优先用 format()，避免再手写 toLocaleString / 硬编码 UTC。
 */
export function useSiteDateTime() {
  const { locale } = useI18n()
  const { options, webOption } = useWebOptions()

  const settings = computed<SiteDateTimeSettings>(() => resolveSiteDateTimeSettings(options.value))

  const timezone = computed(() => settings.value.timezone)
  const dateFormat = computed(() => settings.value.dateFormat)
  const timeFormat = computed(() => settings.value.timeFormat)
  const startOfWeek = computed(() => settings.value.startOfWeek)

  function format(
    value: string | number | Date | null | undefined,
    overrides?: Partial<{ includeTime: boolean, now: Date }>
  ): string {
    return formatSiteDateTime(value, {
      settings: settings.value,
      locale: String(locale.value || webOption('site.default_locale', 'zh-CN')),
      includeTime: overrides?.includeTime,
      now: overrides?.now
    })
  }

  /** 仅日期（即使配置了时间格式也省略时间）。 */
  function formatDateOnly(value: string | number | Date | null | undefined): string {
    return format(value, { includeTime: false })
  }

  return {
    settings,
    timezone,
    dateFormat,
    timeFormat,
    startOfWeek,
    format,
    formatDateOnly
  }
}
