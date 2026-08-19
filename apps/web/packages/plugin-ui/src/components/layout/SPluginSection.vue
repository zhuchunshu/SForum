<script setup lang="ts">
defineProps<{
  title?: string
  description?: string
}>()
</script>

<template>
  <section class="splugin-section">
    <header v-if="title || description || $slots.actions" class="splugin-section__header">
      <div class="splugin-section__heading">
        <h2 v-if="title" class="splugin-section__title">{{ title }}</h2>
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
