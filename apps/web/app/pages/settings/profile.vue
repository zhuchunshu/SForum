<script setup lang="ts">
import SFProfileSettingsPage from '~/components/settings/SFProfileSettingsPage.vue'
/** forum.settings.profile 路由壳：outlet + fail-closed；呈现由主题 L1 + SFProfileSettingsPage 岛拥有。 */
definePageMeta({ requiresAuth: true })
</script>

<template>
  <SFPageOutlet page="forum.settings.profile">
    <SFProfileSettingsPage />
  </SFPageOutlet>
</template>
