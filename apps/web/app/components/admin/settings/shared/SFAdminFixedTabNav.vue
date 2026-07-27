<script setup lang="ts">
export type AdminFixedTabItem = {
  id: string
  label: string
  icon: string
  disabled?: boolean
}

defineProps<{
  items: AdminFixedTabItem[]
  modelValue: string
  ariaLabel: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>

<template>
  <div
    role="tablist"
    :aria-label="ariaLabel"
    class="relative z-0 mb-4 flex flex-wrap gap-2 border-b border-slate-200 pb-3 dark:border-zinc-800"
  >
    <UButton
      v-for="item in items"
      :key="item.id"
      type="button"
      role="tab"
      :icon="item.icon"
      :color="modelValue === item.id ? 'primary' : 'neutral'"
      :variant="modelValue === item.id ? 'solid' : 'ghost'"
      :disabled="item.disabled"
      :aria-selected="modelValue === item.id"
      @click="emit('update:modelValue', item.id)"
    >
      {{ item.label }}
    </UButton>
  </div>
</template>
