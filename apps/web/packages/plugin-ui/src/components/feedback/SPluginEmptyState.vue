<script setup lang="ts">
defineProps<{
  title: string
  description?: string
}>()
</script>

<template>
  <div class="splugin-empty">
    <p class="splugin-empty__title">{{ title }}</p>
    <p v-if="description" class="splugin-empty__description">{{ description }}</p>
    <div v-if="$slots.actions" class="splugin-empty__actions"><slot name="actions" /></div>
  </div>
</template>
