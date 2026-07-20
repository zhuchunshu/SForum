<script setup lang="ts">
/** forum.topic.show 路由壳：outlet + fail-closed；呈现由主题 L1 + SFTopicShowPage 岛拥有。 */
definePageMeta({
  // 主题详情对所有人可见（公开读限定 active/locked）。
  public: true
})
</script>

<template>
  <SFPageOutlet page="forum.topic.show">
    <SFTopicShowPage />
  </SFPageOutlet>
</template>
