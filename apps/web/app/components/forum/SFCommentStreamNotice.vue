<script setup lang="ts">
import type { CommentLiveNotice } from '~/composables/forum/useTopicCommentLive'

// 评论区实时提示：只在「有新评论但没自动追加」或「列表有可见变化」时出现。
// 纯展示组件：决策与数据都在 useTopicCommentLive，这里只负责文案、图标与无障碍。
const props = withDefaults(defineProps<{
  notice: CommentLiveNotice
  newCount?: number
  jumpToLastPage?: boolean
}>(), {
  newCount: 0,
  jumpToLastPage: false
})

const emit = defineEmits<{
  apply: []
  dismiss: []
}>()

const { t } = useI18n()

const visible = computed(() => props.notice === 'new' || props.notice === 'changed')

const label = computed(() => {
  if (props.notice === 'changed') {
    return t('topicDetail.live.changed')
  }
  if (props.jumpToLastPage) {
    return t('topicDetail.live.jumpToLastPage', { count: props.newCount })
  }
  return t('topicDetail.live.newComments', { count: props.newCount })
})

const icon = computed(() => (
  props.notice === 'changed' ? 'i-lucide-refresh-cw' : 'i-lucide-arrow-down'
))
</script>

<template>
  <Transition name="sf-comment-live">
    <div v-if="visible" class="sf-comment-live" role="status" aria-live="polite">
      <button
        type="button"
        class="sf-comment-live__action"
        :title="label"
        @click="emit('apply')"
      >
        <UIcon :name="icon" class="sf-comment-live__icon" aria-hidden="true" />
        <span class="sf-comment-live__label">{{ label }}</span>
      </button>
      <button
        type="button"
        class="sf-comment-live__dismiss"
        :aria-label="t('topicDetail.live.dismiss')"
        :title="t('topicDetail.live.dismiss')"
        @click="emit('dismiss')"
      >
        <UIcon name="i-lucide-x" class="sf-comment-live__icon" aria-hidden="true" />
      </button>
    </div>
  </Transition>
</template>

<style scoped>
.sf-comment-live {
  display: flex;
  align-items: center;
  gap: 4px;
  width: fit-content;
  /* 桌面宽度下中心列是内层滚动容器：胶囊必须 sticky 在滚动视口底部，
     否则读者上滑后就看不到「有新评论」提示（提示只在列表末尾的文档流里）。 */
  position: sticky;
  bottom: 16px;
  z-index: 3;
  margin: 16px auto 4px;
  padding: 4px 6px 4px 4px;
  border: 1px solid var(--sf-public-border, var(--sf-border, #e2e8f0));
  border-radius: 999px;
  background: var(--sf-public-surface, var(--sf-surface, #fff));
  box-shadow: 0 6px 18px rgb(15 23 42 / 8%);
  color: var(--sf-public-text, var(--sf-fg, #25252b));
}

.sf-comment-live__action {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 34px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: var(--sf-accent-soft, var(--sf-public-surface-muted, var(--sf-muted, #f5f5f8)));
  color: var(--sf-accent, var(--sf-public-text, #25252b));
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
}

.sf-comment-live__action:hover {
  color: var(--sf-accent-hover, var(--sf-accent));
}

.sf-comment-live__dismiss {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--sf-public-text-muted, var(--sf-fg-tertiary, #a0a0aa));
  cursor: pointer;
}

.sf-comment-live__icon {
  width: 16px;
  height: 16px;
}

.sf-comment-live__label {
  white-space: nowrap;
}

.sf-comment-live-enter-active,
.sf-comment-live-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}

.sf-comment-live-enter-from,
.sf-comment-live-leave-to {
  opacity: 0;
  transform: translateY(6px);
}

/* 移动端底部导航是 fixed 的 60px 视口底栏（z-index 40）：胶囊必须抬到它上方，
   否则既看不见也点不到（点下去会命中「发帖」）。 */
@media (max-width: 980px) {
  .sf-comment-live {
    bottom: calc(60px + 12px);
  }
}

@media (prefers-reduced-motion: reduce) {
  .sf-comment-live-enter-active,
  .sf-comment-live-leave-active {
    transition: none;
  }
}
</style>
