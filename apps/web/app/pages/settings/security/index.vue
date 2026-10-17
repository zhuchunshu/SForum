<script setup lang="ts">
import SFSecuritySettingsPage from '~/components/settings/SFSecuritySettingsPage.vue'
/** forum.settings.security 路由壳：outlet + fail-closed；呈现由主题 L1 + SFSecuritySettingsPage 岛拥有。 */
definePageMeta({ requiresAuth: true })
</script>

<template>
  <SFPageOutlet page="forum.settings.security">
    <SFSecuritySettingsPage />
  </SFPageOutlet>
</template>
