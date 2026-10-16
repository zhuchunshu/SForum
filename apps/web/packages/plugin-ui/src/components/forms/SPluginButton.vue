<script setup lang="ts">
import SPluginIcon from '../icons/SPluginIcon.vue'

withDefaults(defineProps<{
  type?: 'button' | 'submit' | 'reset'
  variant?: 'primary' | 'secondary' | 'danger'
  size?: 'sm' | 'md'
  disabled?: boolean
  loading?: boolean
  /** 宿主图标名，例如 `i-lucide-save`；解析失败时静默隐藏。 */
  icon?: string
}>(), {
  type: 'button',
  variant: 'primary',
  size: 'md',
  disabled: false,
  loading: false,
  icon: ''
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
    <SPluginIcon v-else-if="icon" :name="icon" size="md" class="splugin-button__icon" />
    <span class="splugin-button__label"><slot /></span>
  </button>
</template>
