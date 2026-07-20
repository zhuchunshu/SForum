<script setup lang="ts">
/** forum.tag.show 路由壳：outlet + fail-closed 回退；SEO 在 body 岛内（依赖列表数据）。 */
</script>

<template>
  <SFPageOutlet page="forum.tag.show">
    <SFTagShowPage />
  </SFPageOutlet>
</template>
