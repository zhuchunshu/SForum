<script setup lang="ts">
import SFCategoryShowPage from '~/components/forum/SFCategoryShowPage.vue'
/** forum.category.show 路由壳：outlet + fail-closed 回退；SEO 在 body 岛内（依赖列表数据）。 */
</script>

<template>
  <SFPageOutlet page="forum.category.show">
    <SFCategoryShowPage />
  </SFPageOutlet>
</template>
