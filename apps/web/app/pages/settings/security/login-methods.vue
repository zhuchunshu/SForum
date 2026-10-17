<script setup lang="ts">
import SFLoginMethodsSettingsPage from '~/components/settings/SFLoginMethodsSettingsPage.vue'

/** 登录方式设置路由壳：主题 L1 可替换，外部身份/密码控制始终由 Host 岛拥有。 */
definePageMeta({ requiresAuth: true })
</script>

<template>
  <SFPageOutlet page="forum.settings.login_methods">
    <SFLoginMethodsSettingsPage />
  </SFPageOutlet>
</template>
