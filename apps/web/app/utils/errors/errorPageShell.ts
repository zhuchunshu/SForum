/**
 * 错误页外壳的 i18n 就绪判定与兜底文本。
 *
 * 错误页是"应用已经出问题"时唯一的出口，因此它不能假设可选插件可用。
 * `@nuxtjs/i18n` 的 runtime 插件把 `vueApp.use(i18n)` 放在 setup 末尾，并
 * 通过 `provide('localeHead')` 暴露 useLocaleHead；任何提前失败（消息加载、
 * 请求上下文缺失、模块解析失败）都会留下一个"没有 i18n 的 app"，此时
 * `useI18n()` 抛 NOT_INSTALLED，错误页二次崩溃并掩盖原始错误。
 */

export type ErrorPageTranslator = (key: string, named?: Record<string, unknown>) => string

export type ErrorPageLocaleHead = {
  htmlAttrs: Record<string, string | undefined>
  link: unknown[]
  meta: unknown[]
}

export type ErrorPageShellSources = {
  /** `nuxtApp.$i18n`：只有 i18n 插件执行完 `vueApp.use(i18n)` 才存在。 */
  i18n?: unknown
  /** `nuxtApp.$localeHead`：只有 i18n 插件 `provide('localeHead')` 后才存在。 */
  localeHead?: unknown
}

/** 双条件判定：半安装的 i18n（只有 composer 没有 localeHead）同样不可信。 */
export function isErrorPageI18nReady(sources: ErrorPageShellSources): boolean {
  return Boolean(sources.i18n) && typeof sources.localeHead === 'function'
}

export function createEmptyErrorPageLocaleHead(): ErrorPageLocaleHead {
  return { htmlAttrs: {}, link: [], meta: [] }
}

export function interpolateErrorPageText(text: string, named?: Record<string, unknown>): string {
  if (!named) {
    return text
  }
  return text.replace(/\{(\w+)\}/g, (match, name: string) => {
    const value = named[name]
    return value === undefined || value === null ? match : String(value)
  })
}

/** 兜底翻译器：i18n 不可用时返回 key 末段，保证错误页仍有文本而不是抛错。 */
export function createFallbackErrorPageTranslator(
  warn: (message: string) => void = message => console.warn(message)
): ErrorPageTranslator {
  let warned = false
  return (key, named) => {
    if (!warned) {
      warned = true
      warn('[SForum] i18n unavailable on the error page; falling back to raw message keys')
    }
    return interpolateErrorPageText(key.split('.').pop() || key, named)
  }
}
