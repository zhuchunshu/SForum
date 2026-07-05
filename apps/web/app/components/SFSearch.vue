<script setup lang="ts">
type SearchFilter = {
  label: string
  value: string
}

const props = withDefaults(defineProps<{
  modelValue?: string
  placeholder?: string
  filters?: SearchFilter[]
  selectedFilter?: string
  kbd?: string
}>(), {
  modelValue: '',
  placeholder: '搜索主题、用户或标签',
  filters: () => [],
  selectedFilter: undefined,
  kbd: '⌘K'
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'update:selectedFilter': [value: string]
}>()

function onInput(event: Event) {
  emit('update:modelValue', (event.target as HTMLInputElement).value)
}
</script>

<template>
  <div class="sf-search">
    <label class="sf-search__box">
      <UIcon name="i-lucide-search" class="sf-search__icon size-4.5 shrink-0" aria-hidden="true" />
      <input
        class="sf-search__input"
        type="search"
        :value="modelValue"
        :placeholder="placeholder"
        @input="onInput"
      >
      <span v-if="kbd" class="sf-search__kbd">{{ kbd }}</span>
    </label>
    <div v-if="filters.length" class="sf-search__filters" aria-label="搜索过滤">
      <button
        v-for="filter in filters"
        :key="filter.value"
        type="button"
        class="sf-badge"
        :class="selectedFilter === filter.value ? 'sf-badge--primary' : 'sf-badge--neutral'"
        @click="emit('update:selectedFilter', filter.value)"
      >
        {{ filter.label }}
      </button>
    </div>
  </div>
</template>
