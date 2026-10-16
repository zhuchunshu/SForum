<script setup lang="ts">
import SPluginIcon from '../icons/SPluginIcon.vue'

defineProps<{
  title: string
  description?: string
  /** 宿主图标名，例如 `i-lucide-inbox`；解析失败时静默隐藏。 */
  icon?: string
}>()
</script>

<template>
  <div class="splugin-empty">
    <SPluginIcon v-if="icon" :name="icon" size="lg" class="splugin-empty__icon" />
    <p class="splugin-empty__title">{{ title }}</p>
    <p v-if="description" class="splugin-empty__description">{{ description }}</p>
    <div v-if="$slots.actions" class="splugin-empty__actions"><slot name="actions" /></div>
  </div>
</template>
