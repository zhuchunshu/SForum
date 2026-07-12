<script setup lang="ts">
/**
 * 宿主岛：首页核心 UI。供 L1 模板 <sf-home-page> 与 pages/index 复用。
 * 实际列表逻辑仍在 pages/index.vue；此组件作为语义岛占位与未来抽取点。
 */
</script>

<template>
  <div class="sf-home-page-island">
    <slot />
  </div>
</template>
