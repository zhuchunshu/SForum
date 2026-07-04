<script setup lang="ts">
import { useAdminTabs } from '~/composables/useAdminTabs'

const localeHead = useLocaleHead({
  dir: true,
  lang: true,
  seo: true
})
const { siteName, appearanceTheme, refresh: refreshWebOptions } = useWebOptions()
const { refresh: refreshAuthSession } = useAuthSession()
const startupOptionsTimeout = import.meta.dev ? 800 : 2000

// 引入页签缓存控制列表
const { cachedTabNames } = useAdminTabs()

await useAsyncData('app-startup', async () => {
  // 开发热重载时 API 可能还在编译，首屏先使用本地默认状态。
  await Promise.all([
    refreshWebOptions({ timeout: startupOptionsTimeout }).catch(() => null),
    refreshAuthSession({ timeout: startupOptionsTimeout })
  ])
  return true
})

useHead(() => ({
  htmlAttrs: {
    ...localeHead.value.htmlAttrs,
    'data-sforum-theme': appearanceTheme.value
  },
  link: localeHead.value.link,
  meta: localeHead.value.meta,
  titleTemplate: (title) => title ? `${title} - ${siteName.value}` : siteName.value
}))
</script>

<template>
  <UApp>
    <NuxtLayout>
      <NuxtPage :keepalive="{ include: cachedTabNames }" />
    </NuxtLayout>
  </UApp>
</template>
