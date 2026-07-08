<script setup lang="ts">
import { useAdminTabs } from '~/composables/useAdminTabs'

const localeHead = useLocaleHead({
  dir: true,
  lang: true,
  seo: true
})
const { siteName, resolvedAppearanceTheme, seoSettings, refresh: refreshWebOptions } = useWebOptions()
const { refresh: refreshAuthSession } = useAuthSession()
const startupOptionsTimeout = import.meta.dev ? 800 : 2000

// 引入页签缓存控制列表
const { cachedTabNames } = useAdminTabs()

async function refreshStartupState() {
  // 开发热重载时 API 可能还在编译，首屏先使用本地默认状态。
  await Promise.all([
    refreshWebOptions({ timeout: startupOptionsTimeout }).catch(() => null),
    refreshAuthSession({ timeout: startupOptionsTimeout })
  ])
  return true
}

if (import.meta.server) {
  await useAsyncData('app-startup', refreshStartupState)
} else {
  // 认证页和设置页是 SPA，客户端启动刷新不能挡住第一屏挂载。
  void useAsyncData('app-startup', refreshStartupState, {
    server: false,
    lazy: true
  })
}

useHead(() => {
  const htmlAttrs: Record<string, string | undefined> = {
    ...localeHead.value.htmlAttrs,
    'data-sforum-theme': resolvedAppearanceTheme.value.dataTheme
  }
  const themeStyle = resolvedAppearanceTheme.value.style
  if (themeStyle) {
    htmlAttrs.style = [htmlAttrs.style, themeStyle].filter(Boolean).join('; ')
  }

  return {
    htmlAttrs,
    link: localeHead.value.link,
    meta: localeHead.value.meta,
    titleTemplate: (title) => title
      ? applySEOTitleTemplate(title, seoSettings.value.metaTitleTemplate, siteName.value)
      : siteName.value
  }
})
</script>

<template>
  <UApp>
    <NuxtLayout>
      <NuxtPage :keepalive="{ include: cachedTabNames }" />
    </NuxtLayout>
  </UApp>
</template>
