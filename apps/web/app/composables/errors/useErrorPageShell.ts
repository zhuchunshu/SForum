import { shallowRef } from 'vue'
import {
  createEmptyErrorPageLocaleHead,
  createFallbackErrorPageTranslator,
  isErrorPageI18nReady,
  type ErrorPageTranslator
} from '~/utils/errors/errorPageShell'

/**
 * `deps` 是测试接缝：生产调用不传，直接用 Nuxt 自动导入的实现；
 * 测试传入替身即可验证"未就绪时绝不调用 useI18n"。
 */
export type ErrorPageShellDeps = {
  useI18n: () => { t: unknown }
  useLocaleHead: (options: { dir: boolean, lang: boolean, seo: boolean }) => unknown
}

export type ErrorPageShell = {
  i18nReady: boolean
  t: ErrorPageTranslator
  localeHead: ReturnType<typeof useLocaleHead>
}

/**
 * 错误页外壳依赖解析。
 *
 * i18n 插件可能因模块解析失败或 setup 提前抛错而没跑完，此时 `useI18n()`
 * 会抛 NOT_INSTALLED，让错误页二次崩溃并掩盖原始错误。这里先判定就绪，
 * 未就绪则退回零依赖的翻译器与空 head，保证错误页始终可渲染。
 */
export function useErrorPageShell(
  nuxtApp: { $i18n?: unknown, $localeHead?: unknown } = useNuxtApp(),
  deps: ErrorPageShellDeps = { useI18n, useLocaleHead }
): ErrorPageShell {
  if (!isErrorPageI18nReady({ i18n: nuxtApp.$i18n, localeHead: nuxtApp.$localeHead })) {
    return {
      i18nReady: false,
      t: createFallbackErrorPageTranslator(),
      // 空 head 是合法的 head 输入，只是没有 locale 相关内容。
      localeHead: shallowRef(createEmptyErrorPageLocaleHead()) as unknown as ReturnType<typeof useLocaleHead>
    }
  }

  const { t } = deps.useI18n()
  return {
    i18nReady: true,
    // vue-i18n 的 t 有重载签名，这里统一成错误页实际使用的形态。
    t: t as unknown as ErrorPageTranslator,
    localeHead: deps.useLocaleHead({ dir: true, lang: true, seo: false }) as ReturnType<typeof useLocaleHead>
  }
}
