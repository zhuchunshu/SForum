<script setup lang="ts">
import type { TopicActionMenuItem } from '~/utils/forum/forumTopicPresentation'

type TopicMenuEntry = {
  label: string
  icon: string
  color?: 'error'
  onSelect: (event: Event) => void
}

const props = withDefaults(defineProps<{
  items: TopicActionMenuItem[]
  pending?: boolean
  runningId?: string
}>(), {
  pending: false,
  runningId: ''
})

const emit = defineEmits<{
  select: [id: string]
}>()

const { t } = useI18n()

const menuItems = computed<TopicMenuEntry[][]>(() => [props.items.map((item: TopicActionMenuItem) => ({
  label: item.label,
  icon: item.id === props.runningId ? 'i-lucide-loader-circle' : item.icon,
  color: item.tone === 'danger' ? 'error' : undefined,
  onSelect: () => emit('select', item.id)
}))])
</script>

<template>
  <UDropdownMenu v-if="items.length" :items="menuItems" :content="{ align: 'end' }">
    <UButton
      color="neutral"
      variant="ghost"
      square
      class="sf-topic-action-menu"
      :disabled="pending"
      :aria-label="t('topicDetail.progress.actions')"
      :title="t('topicDetail.progress.actions')"
    >
      <UIcon
        :name="pending ? 'i-lucide-loader-circle' : 'i-lucide-ellipsis'"
        class="size-5"
        :class="{ 'animate-spin': pending }"
        aria-hidden="true"
      />
    </UButton>
  </UDropdownMenu>
</template>
