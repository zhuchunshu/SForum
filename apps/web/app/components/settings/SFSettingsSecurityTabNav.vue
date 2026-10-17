<script setup lang="ts">
/**
 * 账号安全三个固定子路由的 tab 导航：设备与会话 / 登录方式 / 登录密码。
 *
 * Core 固有 tab：每个 tab 都是独立路由 + 独立页面壳（各有自己的 SFPageOutlet 与
 * Host 岛），本组件只负责路由跳转、路径高亮与可访问性；不持有任何业务状态。
 * 三个页面壳（security/index、security/login-methods、security/password）都渲染它，
 * 因此合并后的侧栏高亮与 tab 高亮只有一个来源。
 */
const { t } = useI18n()
const localePath = useLocalePath()
const route = useRoute()

const tabs = computed(() => {
  const devices = localePath('/settings/security')
  const loginMethods = localePath('/settings/security/login-methods')
  const password = localePath('/settings/security/password')
  return [
    { key: 'devices', to: devices, label: t('accountSecurity.tabs.devices'), active: route.path === devices },
    { key: 'loginMethods', to: loginMethods, label: t('accountSecurity.tabs.loginMethods'), active: route.path === loginMethods },
    { key: 'password', to: password, label: t('accountSecurity.tabs.password'), active: route.path === password }
  ]
})
</script>

<template>
  <nav class="sforum-settings__tabs" :aria-label="t('accountSecurity.tabs.navAriaLabel')">
    <NuxtLink
      v-for="tab in tabs"
      :key="tab.key"
      :to="tab.to"
      class="sforum-settings__tab"
      :class="{ 'is-active': tab.active }"
      :aria-current="tab.active ? 'page' : undefined"
    >
      {{ tab.label }}
    </NuxtLink>
  </nav>
</template>
