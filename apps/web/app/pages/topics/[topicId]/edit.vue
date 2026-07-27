<script setup lang="ts">
import SFTopicEditPage from '~/components/forum/SFTopicEditPage.vue'
/** forum.topic.edit 路由壳：outlet + fail-closed；呈现由主题 L1 + SFTopicEditPage 岛拥有。 */
definePageMeta({
  requiresAuth: true
})
</script>

<template>
  <SFPageOutlet page="forum.topic.edit">
    <SFTopicEditPage />
  </SFPageOutlet>
</template>
