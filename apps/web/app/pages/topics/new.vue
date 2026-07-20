<script setup lang="ts">
/** forum.topic.create 路由壳：outlet + fail-closed；呈现由主题 L1 + SFTopicComposerPage 岛拥有。 */
definePageMeta({
  // 发帖需要登录；路由中间件由全局 auth 守卫处理，这里只声明意图。
  requiresAuth: true
})
</script>

<template>
  <SFPageOutlet page="forum.topic.create">
    <SFTopicComposerPage />
  </SFPageOutlet>
</template>
