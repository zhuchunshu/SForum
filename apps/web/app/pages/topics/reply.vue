<script setup lang="ts">
import SFTopicReplyPage from '~/components/forum/SFTopicReplyPage.vue'
/** forum.topic.reply 路由壳：outlet + fail-closed；呈现由主题 L1 + SFTopicReplyPage 岛拥有。 */
definePageMeta({
  requiresAuth: true
})
</script>

<template>
  <SFPageOutlet page="forum.topic.reply">
    <SFTopicReplyPage />
  </SFPageOutlet>
</template>
