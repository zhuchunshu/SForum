<script setup lang="ts">
defineOptions({ inheritAttrs: false })

withDefaults(defineProps<{
  modelValue?: string
  type?: 'text' | 'email' | 'password' | 'search' | 'url' | 'number'
  disabled?: boolean
}>(), {
  modelValue: '',
  type: 'text',
  disabled: false
})

defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>

<template>
  <input
    v-bind="$attrs"
    class="splugin-input"
    :type="type"
    :value="modelValue"
    :disabled="disabled"
    @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
  >
</template>
