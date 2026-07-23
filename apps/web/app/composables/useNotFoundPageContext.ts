import type { NuxtError } from '#app'
import type { ComputedRef } from 'vue'

type NotFoundPageContext = {
  error: ComputedRef<NuxtError>
}

const notFoundPageContextKey = Symbol('sforum.not-found-page-context')

/** 当前 Nuxt 错误仅在该次页面树内传递，主题岛不保存也不修改它。 */
export function provideNotFoundPageContext(error: ComputedRef<NuxtError>) {
  provideSystemErrorPageContext(error)
  provide(notFoundPageContextKey, { error })
}

export function useNotFoundPageContext() {
  return inject<NotFoundPageContext | null>(notFoundPageContextKey, null)
    || useSystemErrorPageContext()
}
