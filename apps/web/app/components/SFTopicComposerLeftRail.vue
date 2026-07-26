<script setup lang="ts">
/**
 * 发帖页左栏工具区：撰写进度 / 快捷操作 / 写作要点。
 * 挂在 SFHomeNavigation #after-navigation，对齐通知页 type-nav 模式。
 */
import type { ComposerPrePublishCheck } from '~/components/SFTopicComposerRightRail.vue'

export type ComposerFocusField = 'title' | 'body' | 'category' | 'tags'

const props = defineProps<{
  checks: ComposerPrePublishCheck[]
  draftSaving?: boolean
  draftStatusLabel?: string
  /** 无发帖权限时隐藏进度与草稿操作 */
  canCreate?: boolean
  /** 编辑页无本地草稿概念，隐藏"保存草稿"动作（状态文案仍显示） */
  showDraftAction?: boolean
}>()

const emit = defineEmits<{
  'focus-field': [field: ComposerFocusField]
  'save-draft': []
}>()

const { t } = useI18n()
const localePath = useLocalePath()

const tips = computed(() => [
  t('composer.tip1'),
  t('composer.tip2'),
  t('composer.tip3'),
  t('composer.tip4')
])

const progressItems = computed(() => {
  const order: ComposerFocusField[] = ['title', 'body', 'category', 'tags']
  const byKey = new Map(props.checks.map(item => [item.key, item]))
  return order.map((key) => {
    const check = byKey.get(key)
    return {
      key,
      label: check?.label || key,
      ok: Boolean(check?.ok),
      icon: check?.icon || progressIcon(key)
    }
  })
})

function progressIcon(key: ComposerFocusField) {
  switch (key) {
    case 'title':
      return 'i-lucide-heading-1'
    case 'body':
      return 'i-lucide-file-text'
    case 'category':
      return 'i-lucide-folder'
    case 'tags':
      return 'i-lucide-tags'
  }
}

function statusLabel(ok: boolean) {
  return ok ? t('composer.leftRail.statusReady') : t('composer.leftRail.statusTodo')
}

function onFocus(field: ComposerFocusField) {
  emit('focus-field', field)
}

function onSaveDraft() {
  emit('save-draft')
}
</script>

<template>
  <div class="sforum-topic-composer__left-rail">
    <template v-if="canCreate">
      <!-- 撰写进度：点击跳到主列字段 -->
      <nav class="sforum-topic-composer__type-nav" :aria-label="t('composer.leftRail.progressTitle')">
        <div class="sforum-topic-composer__rail-label">{{ t('composer.leftRail.progressTitle') }}</div>
        <button
          v-for="item in progressItems"
          :key="item.key"
          type="button"
          class="sforum-topic-composer__rail-link"
          :class="{ 'is-ok': item.ok }"
          :aria-label="t('composer.leftRail.focusAria', { field: item.label })"
          @click="onFocus(item.key)"
        >
          <span class="sforum-topic-composer__rail-link-main">
            <UIcon :name="item.icon" class="size-[18px]" aria-hidden="true" />
            {{ item.label }}
          </span>
          <span class="sforum-topic-composer__rail-count">{{ statusLabel(item.ok) }}</span>
        </button>
      </nav>

      <!-- 快捷操作 -->
      <div class="sforum-topic-composer__type-nav" :aria-label="t('composer.leftRail.actionsTitle')">
        <div class="sforum-topic-composer__rail-label">{{ t('composer.leftRail.actionsTitle') }}</div>
        <button
          v-if="showDraftAction !== false"
          type="button"
          class="sforum-topic-composer__rail-link"
          :disabled="draftSaving"
          @click="onSaveDraft"
        >
          <span class="sforum-topic-composer__rail-link-main">
            <UIcon name="i-lucide-save" class="size-[18px]" aria-hidden="true" />
            {{ draftSaving ? t('composer.draft.saving') : t('composer.leftRail.saveDraft') }}
          </span>
        </button>
        <p v-if="draftStatusLabel" class="sforum-topic-composer__left-meta">
          {{ draftStatusLabel }}
        </p>
        <NuxtLink
          :to="localePath('/guidelines')"
          class="sforum-topic-composer__rail-link sforum-topic-composer__rail-link--anchor"
        >
          <span class="sforum-topic-composer__rail-link-main">
            <UIcon name="i-lucide-book-open" class="size-[18px]" aria-hidden="true" />
            {{ t('composer.leftRail.guidelines') }}
          </span>
        </NuxtLink>
      </div>
    </template>

    <!-- 写作要点（只读） -->
    <section class="sforum-topic-composer__type-nav" :aria-label="t('composer.leftRail.tipsTitle')">
      <div class="sforum-topic-composer__rail-label">{{ t('composer.leftRail.tipsTitle') }}</div>
      <ul class="sforum-topic-composer__tips">
        <li v-for="(tip, index) in tips" :key="index">
          <UIcon name="i-lucide-lightbulb" class="size-3.5" aria-hidden="true" />
          <span>{{ tip }}</span>
        </li>
      </ul>
    </section>
  </div>
</template>

<style scoped>
.sforum-topic-composer__left-rail {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding-bottom: 8px;
}

.sforum-topic-composer__type-nav {
  padding: 0;
}

.sforum-topic-composer__rail-label {
  padding: 10px 16px 6px;
  color: var(--sf-public-text-muted);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0;
}

.sforum-topic-composer__rail-link {
  width: 100%;
  min-height: 34px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  border: 0;
  border-left: 2px solid transparent;
  border-radius: 0;
  padding: 7px 16px;
  background: transparent;
  color: var(--sf-public-text-secondary);
  font-size: 13px;
  font-weight: 400;
  text-align: left;
  cursor: pointer;
  text-decoration: none;
}

.sforum-topic-composer__rail-link:hover:not(:disabled) {
  background: var(--sf-public-surface-muted);
  color: var(--sf-public-text);
}

.sforum-topic-composer__rail-link:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.sforum-topic-composer__rail-link.is-ok {
  background: var(--sf-accent-soft);
  border-left-color: var(--sf-accent);
  color: var(--sf-public-text);
  font-weight: 600;
}

.sforum-topic-composer__rail-link.is-ok .sforum-topic-composer__rail-link-main :deep(svg) {
  color: var(--sf-accent);
}

.sforum-topic-composer__rail-link.is-ok .sforum-topic-composer__rail-count {
  border-radius: 999px;
  padding: 1px 7px;
  background: var(--sf-accent);
  color: var(--sf-accent-contrast, #fff);
  font-weight: 750;
}

.sforum-topic-composer__rail-link-main {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 11px;
}

.sforum-topic-composer__rail-count {
  flex: none;
  color: var(--sf-public-text-muted);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.sforum-topic-composer__left-meta {
  margin: 0;
  padding: 0 16px 6px;
  color: var(--sf-public-text-muted);
  font-size: 11px;
  line-height: 1.45;
}

.sforum-topic-composer__tips {
  list-style: none;
  margin: 0;
  padding: 2px 16px 10px;
  display: grid;
  gap: 8px;
}

.sforum-topic-composer__tips li {
  display: grid;
  grid-template-columns: 14px minmax(0, 1fr);
  gap: 8px;
  color: var(--sf-public-text-muted);
  font-size: 12px;
  line-height: 1.55;
}

.sforum-topic-composer__tips li :deep(svg) {
  margin-top: 2px;
  color: var(--sf-accent);
}
</style>
