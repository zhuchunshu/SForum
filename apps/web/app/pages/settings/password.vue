<script setup lang="ts">
import SFLocalPasswordSettingsPage from '~/components/settings/SFLocalPasswordSettingsPage.vue'

/** forum.settings.password 路由壳：主题 L1 可替换，密码修改表单始终由 Host 岛拥有。 */
definePageMeta({ requiresAuth: true })
</script>

<template>
  <SFPageOutlet page="forum.settings.password">
    <SFLocalPasswordSettingsPage />
  </SFPageOutlet>
</template>
