<script setup lang="ts">
import SFPersonalAccessTokensPage from '~/components/settings/SFPersonalAccessTokensPage.vue'

/** forum.settings.tokens 路由壳：主题 L1 可替换，个人访问令牌表单始终由 Host 岛拥有。 */
definePageMeta({ requiresAuth: true })
</script>

<template>
  <SFPageOutlet page="forum.settings.tokens">
    <SFPersonalAccessTokensPage />
  </SFPageOutlet>
</template>
