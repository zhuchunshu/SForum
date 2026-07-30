<script setup lang="ts">
import SFAppearanceSettingsPage from '~/components/settings/SFAppearanceSettingsPage.vue'

/** 个人外观设置路由壳：主题只拥有呈现，偏好读写由 Host 岛负责。 */
definePageMeta({ requiresAuth: true })
</script>

<template>
  <SFPageOutlet page="forum.settings.appearance">
    <SFAppearanceSettingsPage />
  </SFPageOutlet>
</template>
