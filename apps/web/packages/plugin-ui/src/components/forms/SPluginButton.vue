<script setup lang="ts">
withDefaults(defineProps<{
  type?: 'button' | 'submit' | 'reset'
  variant?: 'primary' | 'secondary' | 'danger'
  size?: 'sm' | 'md'
  disabled?: boolean
  loading?: boolean
}>(), {
  type: 'button',
  variant: 'primary',
  size: 'md',
  disabled: false,
  loading: false
})

defineEmits<{
  click: [event: MouseEvent]
}>()
</script>

<template>
  <button
    class="splugin-button"
    :class="[`splugin-button--${variant}`, `splugin-button--${size}`]"
    :type="type"
    :disabled="disabled || loading"
    :aria-busy="loading || undefined"
    @click="$emit('click', $event)"
  >
    <span v-if="loading" class="splugin-button__spinner" aria-hidden="true" />
    <span class="splugin-button__label"><slot /></span>
  </button>
</template>
