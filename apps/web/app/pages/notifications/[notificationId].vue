<script setup lang="ts">
import SFNotificationDetailPage from '~/components/notifications/detail/SFNotificationDetailPage.vue'

/** forum.notification.show 路由壳：登录中间件 + Page Registry + Core 详情岛。 */
definePageMeta({ requiresAuth: true })
</script>

<template>
  <SFPageOutlet page="forum.notification.show">
    <SFNotificationDetailPage />
  </SFPageOutlet>
</template>
