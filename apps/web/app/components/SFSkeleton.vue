<script setup lang="ts">
const props = withDefaults(defineProps<{
  lines?: number
  avatar?: boolean
}>(), {
  lines: 3,
  avatar: false
})

const rows = computed(() => Array.from({ length: Math.max(1, props.lines) }, (_, index) => index))

function rowWidth(index: number) {
  if (index === rows.value.length - 1) {
    return '68%'
  }
  if (index === 0) {
    return '92%'
  }
  return '100%'
}
</script>

<template>
  <div class="sf-skeleton" aria-hidden="true">
    <span v-if="avatar" class="sf-skeleton__avatar" />
    <div class="sf-skeleton__content">
      <span
        v-for="row in rows"
        :key="row"
        class="sf-skeleton__line"
        :style="{ width: rowWidth(row) }"
      />
    </div>
  </div>
</template>
