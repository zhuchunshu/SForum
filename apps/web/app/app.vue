<script setup lang="ts">
import { useAdminTabs } from '~/composables/useAdminTabs'

const localeHead = useLocaleHead({
  dir: true,
  lang: true,
  seo: true
})
const {
  siteName,
  resolvedAppearanceTheme,
  seoSettings,
  siteFaviconUrl,
  siteAppleTouchIconUrl,
  refresh: refreshWebOptions
} = useWebOptions()
const { refresh: refreshAuthSession } = useAuthSession()
const route = useRoute()
const adminRoutes = useAdminRoutes()
const themeSkin = useActiveThemeSkin()
const startupOptionsTimeout = import.meta.dev ? 800 : 2000
const hasServerSession = import.meta.server
  && /(?:^|;\s*)sforum_session=/.test(useRequestHeaders(['cookie']).cookie || '')

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

async function syncThemeSkin() {
  if (adminRoutes.routeId(route.path) !== null) {
    themeSkin.clear()
    return
  }
  await themeSkin.refresh()
}

if (import.meta.server) {
  // 带会话的页面由 public-session-cache 禁止共享缓存，可以安全输出登录态首屏。
  await useAsyncData('app-startup', async () => {
    await Promise.all([
      refreshStartupState({ restoreAuth: hasServerSession }),
      syncThemeSkin()
    ])
    return true
  })
} else {

  watch(() => route.path, () => { void syncThemeSkin() }, { flush: 'post' })
  // 浏览器挂载后再恢复会话，避免复用 SSR 的 app-startup payload 时跳过 auth 刷新。
  onMounted(() => {
    void refreshStartupState({ restoreAuth: true })
    // 公共主题 L0 皮肤不得进入独立的管理端样式边界。
    void syncThemeSkin()
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

  // Wave 2 品牌：运营配置的 favicon / apple-touch；空则不注入，沿用浏览器默认。
  const brandLinks: Array<Record<string, string>> = []
  if (siteFaviconUrl.value) {
    brandLinks.push({ rel: 'icon', href: siteFaviconUrl.value })
  }
  if (siteAppleTouchIconUrl.value) {
    brandLinks.push({ rel: 'apple-touch-icon', href: siteAppleTouchIconUrl.value })
  }
  const themeLinks = themeSkin.links.value.map(href => ({
    rel: 'stylesheet',
    href,
    key: `sforum-theme-skin:${href}`,
    'data-sforum-theme-skin': '1'
  }))

  return {
    htmlAttrs,
    link: [...(localeHead.value.link || []), ...brandLinks, ...themeLinks],
    meta: localeHead.value.meta,
    titleTemplate: (title) => title
      ? applySEOTitleTemplate(title, seoSettings.value.metaTitleTemplate, siteName.value)
      : siteName.value
  }
})
</script>

<template>
  <UApp>
    <NuxtLoadingIndicator
      color="var(--sf-accent)"
      :height="3"
      :throttle="80"
    />
    <NuxtLayout>
      <NuxtPage :keepalive="{ include: cachedTabNames }" />
    </NuxtLayout>
    <SFApiConnectionModal />
  </UApp>
</template>
