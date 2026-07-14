<script setup lang="ts">
const props = withDefaults(defineProps<{
  page?: number
  totalPages?: number
  siblingCount?: number
  pageTo?: (page: number) => string | { path: string, query?: Record<string, string> }
}>(), {
  page: 1,
  totalPages: 1,
  siblingCount: 1
})

const emit = defineEmits<{
  'update:page': [value: number]
}>()

const pages = computed(() => {
  const total = Math.max(1, props.totalPages)
  const current = Math.min(total, Math.max(1, props.page))
  const siblings = Math.max(0, props.siblingCount)
  const start = Math.max(1, current - siblings)
  const end = Math.min(total, current + siblings)
  const values: Array<number | 'ellipsis-start' | 'ellipsis-end'> = []

  values.push(1)

  if (start > 2) {
    values.push('ellipsis-start')
  }

  for (let value = Math.max(2, start); value <= Math.min(total - 1, end); value += 1) {
    values.push(value)
  }

  if (end < total - 1) {
    values.push('ellipsis-end')
  }

  if (total > 1) {
    values.push(total)
  }

  return values
})

function go(value: number) {
  const total = Math.max(1, props.totalPages)
  emit('update:page', Math.min(total, Math.max(1, value)))
}

function linkTo(value: number) {
  return props.pageTo?.(Math.min(Math.max(1, props.totalPages), Math.max(1, value)))
}
</script>

<template>
  <nav class="sf-pagination" aria-label="分页">
    <NuxtLink
      v-if="pageTo && page > 1"
      :to="linkTo(page - 1)"
      class="sf-pagination__item"
      aria-label="上一页"
      aria-current="false"
    >
      <UIcon name="i-lucide-chevron-left" class="size-4" />
    </NuxtLink>
    <button
      v-else
      type="button"
      class="sf-pagination__item"
      :disabled="page <= 1"
      aria-label="上一页"
      @click="go(page - 1)"
    >
      <UIcon name="i-lucide-chevron-left" class="size-4" />
    </button>
    <template v-for="item in pages" :key="item">
      <span v-if="typeof item !== 'number'" class="sf-pagination__item" aria-hidden="true">
        ...
      </span>
      <NuxtLink
        v-else-if="pageTo"
        :to="linkTo(item)"
        class="sf-pagination__item"
        :aria-current="item === page ? 'page' : 'false'"
      >
        {{ item }}
      </NuxtLink>
      <button
        v-else
        type="button"
        class="sf-pagination__item"
        :aria-current="item === page ? 'page' : undefined"
        @click="go(item)"
      >
        {{ item }}
      </button>
    </template>
    <NuxtLink
      v-if="pageTo && page < totalPages"
      :to="linkTo(page + 1)"
      class="sf-pagination__item"
      aria-label="下一页"
      aria-current="false"
    >
      <UIcon name="i-lucide-chevron-right" class="size-4" />
    </NuxtLink>
    <button
      v-else
      type="button"
      class="sf-pagination__item"
      :disabled="page >= totalPages"
      aria-label="下一页"
      @click="go(page + 1)"
    >
      <UIcon name="i-lucide-chevron-right" class="size-4" />
    </button>
  </nav>
</template>
