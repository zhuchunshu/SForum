import type { NuxtError } from '#app'
import type { ComputedRef } from 'vue'

type SystemErrorPageContext = {
  error: ComputedRef<NuxtError>
}

const systemErrorPageContextKey = Symbol('sforum.system-error-page-context')

/** 当前 Nuxt 错误仅在本次页面树内传递；主题岛只能读取安全语义，不能修改状态。 */
export function provideSystemErrorPageContext(error: ComputedRef<NuxtError>) {
  provide(systemErrorPageContextKey, { error })
}

export function useSystemErrorPageContext() {
  return inject<SystemErrorPageContext | null>(systemErrorPageContextKey, null)
}
