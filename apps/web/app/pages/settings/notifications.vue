<script setup lang="ts">
import SFNotificationSettingsPage from '~/components/settings/SFNotificationSettingsPage.vue'

/** 账号通知偏好路由壳：主题 L1 可替换，实际偏好控制始终由 Host 岛拥有。 */
definePageMeta({ requiresAuth: true })
</script>

<template>
  <SFPageOutlet page="forum.settings.notifications">
    <SFNotificationSettingsPage />
  </SFPageOutlet>
</template>
