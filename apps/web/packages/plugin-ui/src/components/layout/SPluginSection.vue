<script setup lang="ts">
import SPluginIcon from '../icons/SPluginIcon.vue'

defineProps<{
  title?: string
  description?: string
  /** 宿主图标名，例如 `i-lucide-settings`；解析失败时静默隐藏。 */
  icon?: string
}>()
</script>

<template>
  <section class="splugin-section">
    <header v-if="title || description || icon || $slots.actions" class="splugin-section__header">
      <div class="splugin-section__heading">
        <h2 v-if="title" class="splugin-section__title">
          <SPluginIcon v-if="icon" :name="icon" size="md" class="splugin-section__title-icon" />
          {{ title }}
        </h2>
        <p v-if="description" class="splugin-section__description">{{ description }}</p>
      </div>
      <div v-if="$slots.actions" class="splugin-section__actions">
        <slot name="actions" />
      </div>
    </header>
    <div class="splugin-section__body">
      <slot />
    </div>
  </section>
</template>
