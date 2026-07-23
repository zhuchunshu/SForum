<script setup lang="ts">
import type { NuxtError } from '#app'
import { resolveErrorPageContent } from '~/utils/errorPage'

const props = defineProps<{
  error: NuxtError
}>()
const nuxtError = computed(() => props.error)
const isNotFound = computed(() => Number(nuxtError.value?.statusCode) === 404)
const { t } = useI18n()
const localeHead = useLocaleHead({ dir: true, lang: true, seo: false })
const {
  siteName,
  resolvedAppearanceTheme
} = useWebOptions()
const themeSkin = useActiveThemeSkin()
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
      bodyAttrs: {
        'data-sforum-error': '404'
      },
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

// 服务端错误插件和资源路由在进入 error.vue 前准备最终快照；这里保持同步消费。
const notFoundPresentation = isNotFound.value ? useNotFoundPagePresentation() : null
const resolvedNotFoundPage = shallowRef(notFoundPresentation?.resolvedPage.value || null)
if (import.meta.client && isNotFound.value && !resolvedNotFoundPage.value) {
  void notFoundPresentation!.prepare().then((resolved) => {
    resolvedNotFoundPage.value = resolved
  })
}
</script>

<template>
  <SFNotFoundPage
    v-if="isNotFound"
    :error="nuxtError"
    :resolved-page="resolvedNotFoundPage"
  />
  <UApp v-else>
    <SFErrorPageContent :error="nuxtError" />
  </UApp>
</template>
