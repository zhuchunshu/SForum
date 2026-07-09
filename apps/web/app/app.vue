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

async function refreshStartupState(options: { restoreAuth: boolean }) {
  // 开发热重载时 API 可能还在编译，首屏先使用本地默认状态。
  const tasks: Array<Promise<unknown>> = [
    refreshWebOptions({ timeout: startupOptionsTimeout }).catch(() => null)
  ]

  if (options.restoreAuth) {
    tasks.push(refreshAuthSession({ timeout: startupOptionsTimeout }))
  }

  await Promise.all(tasks)
  return true
}

if (import.meta.server) {
  // 公共页可能被 Nitro SWR 缓存，SSR 阶段不能把当前用户写进可复用 payload。
  await useAsyncData('app-startup', () => refreshStartupState({ restoreAuth: false }))
} else {
  // 浏览器挂载后再恢复会话，避免复用 SSR 的 app-startup payload 时跳过 auth 刷新。
  onMounted(() => {
    void refreshStartupState({ restoreAuth: true })
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
    <SFApiConnectionModal />
  </UApp>
</template>
