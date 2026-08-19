<script setup lang="ts">
import type { SPluginSelectOption } from '../../types'

defineOptions({ inheritAttrs: false })

withDefaults(defineProps<{
  modelValue?: string
  options: readonly SPluginSelectOption[]
  placeholder?: string
  disabled?: boolean
}>(), {
  modelValue: '',
  placeholder: '',
  disabled: false
})

defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>

<template>
  <select
    v-bind="$attrs"
    class="splugin-select"
    :value="modelValue"
    :disabled="disabled"
    @change="$emit('update:modelValue', ($event.target as HTMLSelectElement).value)"
  >
    <option v-if="placeholder" value="" disabled>{{ placeholder }}</option>
    <option
      v-for="option in options"
      :key="option.value"
      :value="option.value"
      :disabled="option.disabled"
    >
      {{ option.label }}
    </option>
  </select>
</template>
