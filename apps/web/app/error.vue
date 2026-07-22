<script setup lang="ts">
import type { NuxtError } from '#app'
import { resolveErrorPageContent } from '~/utils/errorPage'
import { useNotFoundPageResolve } from '~/composables/useNotFoundPageResolve'

const props = defineProps<{
  error: NuxtError
}>()
const nuxtError = computed(() => props.error)
const isNotFound = computed(() => Number(nuxtError.value?.statusCode) === 404)
const { t } = useI18n()
const localeHead = useLocaleHead({ dir: true, lang: true, seo: false })
const {
  siteName,
  resolvedAppearanceTheme,
  refresh: refreshWebOptions
} = useWebOptions()
const { refresh: refreshAuthSession } = useAuthSession()
const themeSkin = useActiveThemeSkin()
const startupTimeout = import.meta.dev ? 1000 : 1200
const hasServerSession = import.meta.server
  && /(?:^|;\s*)sforum_session=/.test(useRequestHeaders(['cookie']).cookie || '')
const notFoundContent = computed(() => resolveErrorPageContent(404))
const notFoundTitle = computed(() => t(notFoundContent.value.titleKey, { siteName: siteName.value }))
const notFoundDescription = computed(() => t(notFoundContent.value.descriptionKey, { siteName: siteName.value }))

if (isNotFound.value) {
  useSeoMeta({
    title: notFoundTitle,
    description: notFoundDescription,
    robots: 'noindex,nofollow'
  })

  // Nuxt 的硬导航错误文档不会经过 app.vue，404 必须在根错误页补齐当前主题 L0。
  useHead(() => {
    const htmlAttrs: Record<string, string | undefined> = {
      ...localeHead.value.htmlAttrs,
      'data-sforum-theme': resolvedAppearanceTheme.value.dataTheme
    }
    if (resolvedAppearanceTheme.value.style) {
      htmlAttrs.style = [htmlAttrs.style, resolvedAppearanceTheme.value.style]
        .filter(Boolean)
        .join('; ')
    }
    return {
      htmlAttrs,
      link: [
        ...(localeHead.value.link || []),
        ...themeSkin.links.value.map(href => ({
          rel: 'stylesheet',
          href,
          key: `sforum-theme-skin:${href}`,
          'data-sforum-theme-skin': '1'
        }))
      ],
      meta: localeHead.value.meta
    }
  })
}

// 先确认 system.not_found 可由当前主题解析；失败时不得再启动依赖同一故障 API 的 chrome 请求。
const notFoundResolve = isNotFound.value ? useNotFoundPageResolve() : null
async function prepareNotFoundPage() {
  const resolved = await notFoundResolve!.refresh()
  if (resolved.provider === 'core' || resolved.fallback) {
    return
  }

  const tasks: Array<Promise<unknown>> = [
    refreshWebOptions({ timeout: startupTimeout }).catch(() => null),
    themeSkin.refresh()
  ]
  if (hasServerSession) {
    tasks.push(refreshAuthSession({ timeout: startupTimeout }).catch(() => null))
  }
  await Promise.race([
    Promise.allSettled(tasks),
    new Promise(resolve => setTimeout(resolve, startupTimeout))
  ])
}
const notFoundStartup = isNotFound.value
  ? prepareNotFoundPage()
  : null
if (import.meta.server && notFoundStartup) {
  onServerPrefetch(() => notFoundStartup)
}
</script>

<template>
  <SFNotFoundPage
    v-if="isNotFound"
    :error="nuxtError"
    :resolved-page="notFoundResolve?.data.value"
    :resolving="notFoundResolve?.pending.value"
  />
  <UApp v-else>
    <SFErrorPageContent :error="nuxtError" />
  </UApp>
</template>
