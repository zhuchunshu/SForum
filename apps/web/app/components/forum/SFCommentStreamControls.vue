<script setup lang="ts">
defineProps<{
  count: number
  modelValue: 'tree' | 'flat'
}>()

const emit = defineEmits<{
  'update:modelValue': [value: 'tree' | 'flat']
}>()

const { t } = useI18n()

const items = computed(() => [
  { label: t('topicDetail.viewTree'), value: 'tree' },
  { label: t('topicDetail.viewFlat'), value: 'flat' }
])

function updateView(value: string) {
  if (value === 'tree' || value === 'flat') {
    emit('update:modelValue', value)
  }
}
</script>

<template>
  <header class="sf-comment-stream-controls">
    <h2>{{ t('topicDetail.commentsTitle', { count }) }}</h2>
    <SFTabs
      :model-value="modelValue"
      :items="items"
      :aria-label="t('topicDetail.progress.viewMode')"
      @update:model-value="updateView"
    />
  </header>
</template>
