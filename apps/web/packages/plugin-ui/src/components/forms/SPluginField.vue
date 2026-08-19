<script setup lang="ts">
defineProps<{
  label: string
  for?: string
  description?: string
  error?: string
  required?: boolean
}>()
</script>

<template>
  <div class="splugin-field" :class="{ 'splugin-field--invalid': error }">
    <label class="splugin-field__label" :for="for">
      {{ label }}
      <span v-if="required" class="splugin-field__required" aria-hidden="true">*</span>
    </label>
    <p v-if="description" class="splugin-field__description">{{ description }}</p>
    <div class="splugin-field__control"><slot /></div>
    <p v-if="error" class="splugin-field__error" role="alert">{{ error }}</p>
  </div>
</template>
