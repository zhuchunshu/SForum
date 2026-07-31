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
  ariaLabel?: string
  kbd?: string
}>(), {
  modelValue: '',
  placeholder: '搜索主题、用户或标签',
  filters: () => [],
  selectedFilter: undefined,
  ariaLabel: undefined,
  kbd: undefined
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'update:selectedFilter': [value: string]
  'submit': [value: string]
}>()
const inputElement = ref<HTMLInputElement | null>(null)

function onInput(event: Event) {
  emit('update:modelValue', (event.target as HTMLInputElement).value)
}

function submit() {
  emit('submit', (inputElement.value?.value ?? props.modelValue).trim())
}

function onKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter' || event.isComposing) {
    return
  }
  event.preventDefault()
  event.stopPropagation()
  submit()
}
</script>

<template>
  <div class="sf-search">
    <form role="search" @submit.prevent="submit">
      <div class="sf-search__box">
        <button
          type="button"
          class="sf-search__submit"
          :aria-label="ariaLabel || placeholder"
          :title="ariaLabel || placeholder"
          @click="submit"
        >
          <UIcon name="i-lucide-search" class="sf-search__icon size-4.5 shrink-0" aria-hidden="true" />
        </button>
        <input
          ref="inputElement"
          class="sf-search__input"
          type="search"
          :value="modelValue"
          :placeholder="placeholder"
          :aria-label="ariaLabel || placeholder"
          @input="onInput"
          @keydown="onKeydown"
        >
        <span v-if="kbd" class="sf-search__kbd">{{ kbd }}</span>
      </div>
    </form>
    <div v-if="filters.length" class="sf-search__filters" aria-label="搜索过滤" role="group">
      <button
        v-for="filter in filters"
        :key="filter.value"
        type="button"
        class="sf-badge"
        :class="selectedFilter === filter.value ? 'sf-badge--primary' : 'sf-badge--neutral'"
        :aria-pressed="selectedFilter === filter.value"
        @click="emit('update:selectedFilter', filter.value)"
      >
        {{ filter.label }}
      </button>
    </div>
  </div>
</template>
