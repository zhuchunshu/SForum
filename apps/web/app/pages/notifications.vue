<script setup lang="ts">
import SFNotificationsPage from '~/components/notifications/SFNotificationsPage.vue'
/** forum.notifications 路由壳：outlet + fail-closed；呈现由主题 L1 + SFNotificationsPage 岛拥有。 */
definePageMeta({ requiresAuth: true })
</script>

<template>
  <SFPageOutlet page="forum.notifications">
    <SFNotificationsPage />
  </SFPageOutlet>
</template>
