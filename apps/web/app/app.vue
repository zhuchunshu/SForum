<script setup lang="ts">
const localeHead = useLocaleHead({
  dir: true,
  lang: true,
  seo: true
})
const { siteName, refresh } = useWebOptions()
const startupOptionsTimeout = import.meta.dev ? 800 : 2000

await useAsyncData('web-options', async () => {
  // 开发热重载时 API 可能还在编译，首屏先使用本地默认站点配置。
  await refresh({ timeout: startupOptionsTimeout }).catch(() => null)
  return true
})

useHead(() => ({
  htmlAttrs: localeHead.value.htmlAttrs,
  link: localeHead.value.link,
  meta: localeHead.value.meta,
  titleTemplate: (title) => title ? `${title} - ${siteName.value}` : siteName.value
}))
</script>

<template>
  <UApp>
    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>
  </UApp>
</template>
