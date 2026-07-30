<script setup lang="ts">
import { useActiveThemeSkin } from '~/composables/themes/useActiveThemeSkin'
import { useAppliedAppearance } from '~/composables/appearance/useAppliedAppearance'
import { useSystemErrorPagePresentation } from '~/composables/errors/useSystemErrorPagePresentation'
import SFSystemErrorPage from '~/components/errors/SFSystemErrorPage.vue'
import SFErrorPageContent from '~/components/errors/SFErrorPageContent.vue'
import type { NuxtError } from '#app'
import {
  isThemeableSystemErrorStatus,
  resolveErrorPageContent,
  systemErrorPageIdForStatus
} from '~/utils/errors/errorPage'

const props = defineProps<{
  error: NuxtError
}>()
const nuxtError = computed(() => props.error)
const systemPage = computed(() => resolveErrorPageContent(nuxtError.value?.statusCode))
const isThemeableSystemError = computed(() => isThemeableSystemErrorStatus(nuxtError.value?.statusCode))
const systemPageId = computed(() => systemErrorPageIdForStatus(nuxtError.value?.statusCode))
const { t } = useI18n()
const localeHead = useLocaleHead({ dir: true, lang: true, seo: false })
const {
  siteName
} = useWebOptions()
const {
  appliedAppearanceTheme,
  appliedLightBackground,
  savedUserAppearance,
  userAppearancePreview
} = useAppliedAppearance()
const themeSkin = useActiveThemeSkin()
const systemTitle = computed(() => t(systemPage.value.titleKey, { siteName: siteName.value }))
const systemDescription = computed(() => t(systemPage.value.descriptionKey, { siteName: siteName.value }))

if (isThemeableSystemError.value) {
  useSeoMeta({
    title: systemTitle,
    description: systemDescription,
    robots: 'noindex,nofollow'
  })

  // Nuxt 的硬导航错误文档不会经过 app.vue，404 必须在根错误页补齐当前主题 L0。
  useHead(() => {
    const htmlAttrs: Record<string, string | undefined> = {
      ...localeHead.value.htmlAttrs,
      'data-sforum-theme': appliedAppearanceTheme.value.dataTheme,
      'data-sforum-light-background': appliedLightBackground.value,
      'data-sforum-user-appearance': savedUserAppearance.value ? 'custom' : 'site-default',
      'data-sforum-user-appearance-preview': userAppearancePreview.value ? 'active' : undefined
    }
    if (appliedAppearanceTheme.value.style) {
      htmlAttrs.style = [htmlAttrs.style, appliedAppearanceTheme.value.style]
        .filter(Boolean)
        .join('; ')
    }
    return {
      htmlAttrs,
      bodyAttrs: {
        'data-sforum-error': String(systemPage.value.statusCode)
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
const systemPresentation = isThemeableSystemError.value ? useSystemErrorPagePresentation(systemPageId) : null
const resolvedSystemPage = shallowRef(systemPresentation?.resolvedPage.value || null)
if (import.meta.client && isThemeableSystemError.value && !resolvedSystemPage.value) {
  void systemPresentation!.prepare().then((resolved) => {
    resolvedSystemPage.value = resolved
  })
}
</script>

<template>
  <SFSystemErrorPage
    v-if="isThemeableSystemError"
    :error="nuxtError"
    :resolved-page="resolvedSystemPage"
  />
  <UApp v-else :toaster="{ position: 'top-right' }">
    <SFErrorPageContent :error="nuxtError" />
  </UApp>
</template>
