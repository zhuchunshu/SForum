<script setup lang="ts">
/**
 * 发帖页右栏：任务型摘要 / 检查 / 规则 / 提示。
 * 视觉复用首页 sf-home-right-rail 卡片语言；桌面与移动抽屉共用。
 * 实时态：摘要填充/空态、检查就绪底色、值变化短暂高亮。
 */
export type ComposerPrePublishCheck = {
  key: string
  ok: boolean
  label: string
  text: string
  icon?: string
}

const props = defineProps<{
  categoryName?: string | null
  title: string
  titleCount: number
  titleMin: number
  titleMax: number
  tagCount: number
  tagPolicyLabel: string
  actorName: string
  publishVisibilityLabel: string
  checks: ComposerPrePublishCheck[]
  bodyMax: number
}>()

const { t } = useI18n()

const titleTrimmed = computed(() => props.title.trim())
const hasTitle = computed(() => Boolean(titleTrimmed.value))
const hasCategory = computed(() => Boolean(props.categoryName))
const hasTags = computed(() => props.tagCount > 0)

/** 标题长度状态：空 / 过短 / 正常 / 接近上限 / 超限 */
type TitleLengthStatus = 'empty' | 'short' | 'ok' | 'near' | 'over'

const titleStatus = computed<TitleLengthStatus>(() => {
  const count = props.titleCount
  const min = Math.max(0, props.titleMin)
  const max = Math.max(1, props.titleMax)
  if (!hasTitle.value || count <= 0) {
    return 'empty'
  }
  if (count > max) {
    return 'over'
  }
  if (count < min) {
    return 'short'
  }
  // 接近上限：达到上限 85% 且未超限
  if (count >= Math.ceil(max * 0.85)) {
    return 'near'
  }
  return 'ok'
})

const titleStatusLabel = computed(() => {
  switch (titleStatus.value) {
    case 'empty':
      return t('composer.rightRail.titleStatus.empty')
    case 'short':
      return t('composer.rightRail.titleStatus.short')
    case 'ok':
      return t('composer.rightRail.titleStatus.ok')
    case 'near':
      return t('composer.rightRail.titleStatus.near')
    case 'over':
      return t('composer.rightRail.titleStatus.over')
  }
})

const titleHintText = computed(() => {
  switch (titleStatus.value) {
    case 'empty':
      return t('composer.rightRail.titleHint.empty', { min: props.titleMin, max: props.titleMax })
    case 'short':
      return t('composer.rightRail.titleHint.short', { min: props.titleMin, count: props.titleCount })
    case 'ok':
      return t('composer.charCount', { count: props.titleCount, max: props.titleMax })
    case 'near':
      return t('composer.rightRail.titleHint.near', { count: props.titleCount, max: props.titleMax })
    case 'over':
      return t('composer.rightRail.titleHint.over', { count: props.titleCount, max: props.titleMax })
  }
})

/** 进度条相对上限（超限时夹到 100%，另用 over 样式区分） */
const titleProgressPercent = computed(() => {
  const max = Math.max(1, props.titleMax)
  return Math.min(100, Math.round((props.titleCount / max) * 100))
})

const titleIsOverflowingDisplay = computed(() => titleTrimmed.value.length > 36)

const summaryTitle = computed(() => titleTrimmed.value || t('composer.summary.untitled'))
const summaryCategoryName = computed(() => props.categoryName || t('composer.categoryDefaultShort'))
const summaryTagLabel = computed(() => (
  props.tagCount
    ? t('composer.summary.tagCount', { count: props.tagCount })
    : t('composer.summary.noTags')
))
const categoryMeta = computed(() => (
  props.categoryName
    ? t('composer.summary.category')
    : t('composer.summary.categoryDefault')
))

const checksReady = computed(() => props.checks.filter(item => item.ok).length)
const checksTotal = computed(() => props.checks.length)
const checksAllReady = computed(() => checksTotal.value > 0 && checksReady.value === checksTotal.value)
const checksProgressLabel = computed(() =>
  t('composer.rightRail.checksProgress', {
    ready: checksReady.value,
    total: checksTotal.value
  })
)

/** 值变化时短暂高亮，让实时更新更可感知 */
const flashKeys = ref<Record<string, number>>({})
const flashTimers = new Map<string, ReturnType<typeof setTimeout>>()

function triggerFlash(key: string) {
  if (!import.meta.client) {
    return
  }
  const prev = flashTimers.get(key)
  if (prev) {
    clearTimeout(prev)
  }
  flashKeys.value = { ...flashKeys.value, [key]: (flashKeys.value[key] || 0) + 1 }
  flashTimers.set(key, setTimeout(() => {
    const next = { ...flashKeys.value }
    delete next[key]
    flashKeys.value = next
    flashTimers.delete(key)
  }, 520))
}

function isFlashing(key: string) {
  return Boolean(flashKeys.value[key])
}

watch(() => props.title, () => triggerFlash('title'))
watch(() => props.categoryName, () => triggerFlash('category'))
watch(() => props.tagCount, () => triggerFlash('tags'))
watch(() => props.checks.map(item => `${item.key}:${item.ok}:${item.text}`).join('|'), (next, prev) => {
  if (!prev) {
    return
  }
  for (const item of props.checks) {
    // 任一检查文案或就绪态变化时高亮该行
    if (!prev.includes(`${item.key}:${item.ok}:${item.text}`)) {
      triggerFlash(`check-${item.key}`)
    }
  }
})

onBeforeUnmount(() => {
  for (const timer of flashTimers.values()) {
    clearTimeout(timer)
  }
  flashTimers.clear()
})
</script>

<template>
  <div class="sf-home-right-rail sforum-topic-composer__rail">
    <!-- 发布摘要：实时反映表单状态 -->
    <section class="sf-home-right-rail__card">
      <header class="sf-home-right-rail__head">
        <h3 class="sf-home-right-rail__title">{{ t('composer.summary.title') }}</h3>
        <span
          class="sf-home-right-rail__meta sforum-topic-composer__live-badge"
          :class="{ 'is-live': hasTitle || hasCategory || hasTags }"
        >
          {{ publishVisibilityLabel }}
        </span>
      </header>
      <div class="sforum-topic-composer__summary-list">
        <div
          class="sforum-topic-composer__summary-row"
          :class="{
            'is-filled': hasCategory,
            'is-empty': !hasCategory,
            'is-flash': isFlashing('category')
          }"
        >
          <span class="sforum-topic-composer__summary-icon" aria-hidden="true">
            <UIcon name="i-lucide-folder" class="size-4" />
          </span>
          <div>
            <strong>{{ summaryCategoryName }}</strong>
            <span>{{ categoryMeta }}</span>
          </div>
        </div>
        <!-- 标题：溢出截断 + 长度状态 + 进度条 -->
        <div
          class="sforum-topic-composer__summary-row sforum-topic-composer__title-row"
          :class="[
            `is-title-${titleStatus}`,
            {
              'is-filled': hasTitle && titleStatus !== 'over' && titleStatus !== 'short',
              'is-empty': titleStatus === 'empty',
              'is-flash': isFlashing('title')
            }
          ]"
        >
          <span class="sforum-topic-composer__summary-icon" aria-hidden="true">
            <UIcon
              :name="titleStatus === 'over' || titleStatus === 'short'
                ? 'i-lucide-triangle-alert'
                : 'i-lucide-heading-1'"
              class="size-4"
            />
          </span>
          <div class="sforum-topic-composer__title-body">
            <div class="sforum-topic-composer__title-head">
              <strong
                class="sforum-topic-composer__title-text"
                :class="{ 'is-clamped': hasTitle }"
                :title="hasTitle ? titleTrimmed : undefined"
              >
                {{ summaryTitle }}
              </strong>
              <span
                class="sforum-topic-composer__title-status"
                :data-status="titleStatus"
              >
                {{ titleStatusLabel }}
              </span>
            </div>
            <span
              v-if="hasTitle && titleIsOverflowingDisplay"
              class="sforum-topic-composer__title-overflow-hint"
            >
              <UIcon name="i-lucide-ellipsis" class="size-3.5" aria-hidden="true" />
              {{ t('composer.rightRail.titleOverflowHint') }}
            </span>
            <div
              class="sforum-topic-composer__title-meter"
              role="meter"
              :aria-valuenow="titleCount"
              :aria-valuemin="0"
              :aria-valuemax="titleMax"
              :aria-label="titleHintText"
            >
              <i
                class="sforum-topic-composer__title-meter-fill"
                :style="{ width: `${titleProgressPercent}%` }"
              />
            </div>
            <span class="sforum-topic-composer__count" :data-status="titleStatus">
              {{ titleHintText }}
            </span>
          </div>
        </div>
        <div
          class="sforum-topic-composer__summary-row"
          :class="{
            'is-filled': hasTags,
            'is-empty': !hasTags,
            'is-flash': isFlashing('tags')
          }"
        >
          <span class="sforum-topic-composer__summary-icon" aria-hidden="true">
            <UIcon name="i-lucide-tags" class="size-4" />
          </span>
          <div>
            <strong>{{ summaryTagLabel }}</strong>
            <span>{{ tagPolicyLabel }}</span>
          </div>
        </div>
        <div class="sforum-topic-composer__summary-row is-filled">
          <span class="sforum-topic-composer__summary-icon" aria-hidden="true">
            <UIcon name="i-lucide-user-round" class="size-4" />
          </span>
          <div>
            <strong>{{ actorName }}</strong>
            <span>{{ t('composer.summary.actor') }}</span>
          </div>
        </div>
      </div>
    </section>

    <!-- 发布前检查：就绪态用底色 + 勾选图标强化 -->
    <section
      class="sf-home-right-rail__card sforum-topic-composer__checks-card"
      :class="{ 'is-all-ready': checksAllReady }"
    >
      <header class="sf-home-right-rail__head">
        <h3 class="sf-home-right-rail__title">{{ t('composer.checks.heading') }}</h3>
        <span
          class="sf-home-right-rail__meta sforum-topic-composer__live-badge"
          :class="{ 'is-ready': checksAllReady, 'is-live': checksReady > 0 }"
        >
          {{ checksProgressLabel }}
        </span>
      </header>
      <ul class="sforum-topic-composer__checks" role="list">
        <li
          v-for="item in checks"
          :key="item.key"
          class="sforum-topic-composer__check"
          :class="{
            'is-ok': item.ok,
            'is-todo': !item.ok,
            'is-flash': isFlashing(`check-${item.key}`)
          }"
        >
          <span class="sforum-topic-composer__check-mark" aria-hidden="true">
            <UIcon
              :name="item.ok ? 'i-lucide-circle-check' : 'i-lucide-circle'"
              class="size-[18px]"
            />
          </span>
          <span class="sforum-topic-composer__check-body">
            <strong>{{ item.label }}</strong>
            <span class="sforum-topic-composer__check-text">{{ item.text }}</span>
          </span>
          <span class="sforum-topic-composer__check-status">
            {{ item.ok ? t('composer.leftRail.statusReady') : t('composer.leftRail.statusTodo') }}
          </span>
        </li>
      </ul>
    </section>

    <!-- 站点规则摘要（非可编辑设置） -->
    <section class="sf-home-right-rail__card">
      <header class="sf-home-right-rail__head">
        <h3 class="sf-home-right-rail__title">{{ t('composer.settings.title') }}</h3>
      </header>
      <dl class="sforum-topic-composer__settings">
        <div>
          <dt>{{ t('composer.settings.permission') }}</dt>
          <dd>{{ t('composer.settings.permissionValue') }}</dd>
        </div>
        <div :class="{ 'is-flash': isFlashing('category') }">
          <dt>{{ t('composer.settings.category') }}</dt>
          <dd :class="{ 'is-muted': !hasCategory }">{{ summaryCategoryName }}</dd>
        </div>
        <div :class="{ 'is-flash': isFlashing('tags') }">
          <dt>{{ t('composer.settings.tags') }}</dt>
          <dd>{{ tagPolicyLabel }}</dd>
        </div>
        <div>
          <dt>{{ t('composer.settings.limits') }}</dt>
          <dd>{{ t('composer.settings.limitValue', { titleMax, bodyMax }) }}</dd>
        </div>
      </dl>
    </section>

    <!-- 轻量提示 -->
    <section class="sf-home-right-rail__card">
      <div class="sforum-topic-composer__tip">
        <UIcon name="i-lucide-lightbulb" class="size-4" aria-hidden="true" />
        <span>{{ t('composer.rightRail.tip') }}</span>
      </div>
    </section>
  </div>
</template>

<style scoped>
/* 行级样式：卡片壳来自全局 sf-home-right-rail */

.sforum-topic-composer__summary-list,
.sforum-topic-composer__settings,
.sforum-topic-composer__checks {
  display: grid;
  gap: 8px;
  margin: 0;
  padding: 0 12px 12px;
}

/* —— 实时徽标 —— */
.sforum-topic-composer__live-badge {
  display: inline-flex;
  align-items: center;
  min-height: 20px;
  border-radius: 999px;
  padding: 0 8px;
  background: var(--sf-public-surface-muted);
  color: var(--sf-public-text-muted);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0;
  transition:
    background 0.18s ease,
    color 0.18s ease;
}

.sforum-topic-composer__live-badge.is-live {
  background: var(--sf-accent-soft);
  color: var(--sf-accent);
}

.sforum-topic-composer__live-badge.is-ready {
  background: var(--sf-accent);
  color: var(--sf-accent-contrast, #fff);
}

/* —— 摘要行：空态 / 填充态 + 闪动 —— */
.sforum-topic-composer__summary-row {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr);
  gap: 8px;
  align-items: start;
  border: 1px solid transparent;
  border-radius: 8px;
  padding: 8px 8px;
  color: var(--sf-public-text-muted);
  font-size: 0.75rem;
  line-height: 1.5;
  transition:
    background 0.18s ease,
    border-color 0.18s ease,
    box-shadow 0.18s ease;
}

.sforum-topic-composer__summary-row.is-empty {
  background: var(--sf-public-surface-muted);
  border-color: var(--sf-public-border);
}

.sforum-topic-composer__summary-row.is-empty strong {
  color: var(--sf-public-text-muted);
  font-weight: 650;
}

.sforum-topic-composer__summary-row.is-filled {
  background: color-mix(in srgb, var(--sf-accent-soft) 55%, var(--sf-public-surface));
  border-color: color-mix(in srgb, var(--sf-accent) 22%, var(--sf-public-border));
}

.sforum-topic-composer__summary-row.is-filled .sforum-topic-composer__summary-icon {
  background: var(--sf-accent-soft);
  color: var(--sf-accent);
}

.sforum-topic-composer__summary-row.is-filled strong {
  color: var(--sf-public-text);
}

.sforum-topic-composer__summary-icon {
  display: grid;
  width: 28px;
  height: 28px;
  place-items: center;
  border-radius: 7px;
  background: var(--sf-public-surface-muted);
  color: var(--sf-public-text-muted);
  transition:
    background 0.18s ease,
    color 0.18s ease;
}

.sforum-topic-composer__summary-row strong,
.sforum-topic-composer__summary-row span {
  display: block;
  overflow-wrap: anywhere;
}

.sforum-topic-composer__summary-row strong {
  color: var(--sf-public-text);
  font-size: 0.8rem;
  font-weight: 700;
  line-height: 1.35;
}

/* —— 标题专用：截断、状态、进度 —— */
.sforum-topic-composer__title-row {
  align-items: stretch;
}

.sforum-topic-composer__title-body {
  min-width: 0;
  display: grid;
  gap: 5px;
}

.sforum-topic-composer__title-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;
}

.sforum-topic-composer__title-text {
  min-width: 0;
  flex: 1 1 auto;
  color: var(--sf-public-text);
  font-size: 0.8rem;
  font-weight: 700;
  line-height: 1.35;
}

.sforum-topic-composer__title-text.is-clamped {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
  overflow-wrap: anywhere;
  word-break: break-word;
}

.sforum-topic-composer__title-status {
  flex: none;
  align-self: flex-start;
  min-height: 18px;
  border-radius: 999px;
  padding: 1px 7px;
  background: var(--sf-public-surface-muted);
  color: var(--sf-public-text-muted);
  font-size: 10px;
  font-weight: 750;
  line-height: 1.4;
  white-space: nowrap;
}

.sforum-topic-composer__title-status[data-status='ok'] {
  background: var(--sf-accent);
  color: var(--sf-accent-contrast, #fff);
}

.sforum-topic-composer__title-status[data-status='near'] {
  background: color-mix(in srgb, var(--sf-accent) 78%, #f59e0b);
  color: var(--sf-accent-contrast, #fff);
}

.sforum-topic-composer__title-status[data-status='short'],
.sforum-topic-composer__title-status[data-status='over'] {
  background: var(--sf-danger, #dc2626);
  color: #fff;
}

.sforum-topic-composer__title-status[data-status='empty'] {
  background: var(--sf-public-surface);
  border: 1px solid var(--sf-public-border);
  color: var(--sf-public-text-muted);
}

.sforum-topic-composer__title-overflow-hint {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--sf-public-text-muted);
  font-size: 10px;
  font-weight: 650;
  line-height: 1.3;
}

.sforum-topic-composer__title-meter {
  position: relative;
  width: 100%;
  height: 4px;
  overflow: hidden;
  border-radius: 999px;
  background: var(--sf-public-border);
}

.sforum-topic-composer__title-meter-fill {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--sf-public-text-muted);
  transition:
    width 0.18s ease,
    background 0.18s ease;
}

.sforum-topic-composer__title-row.is-title-ok .sforum-topic-composer__title-meter-fill {
  background: var(--sf-accent);
}

.sforum-topic-composer__title-row.is-title-near .sforum-topic-composer__title-meter-fill {
  background: color-mix(in srgb, var(--sf-accent) 55%, #f59e0b);
}

.sforum-topic-composer__title-row.is-title-short .sforum-topic-composer__title-meter-fill,
.sforum-topic-composer__title-row.is-title-over .sforum-topic-composer__title-meter-fill {
  background: var(--sf-danger, #dc2626);
}

.sforum-topic-composer__title-row.is-title-empty {
  background: var(--sf-public-surface-muted);
  border-color: var(--sf-public-border);
}

.sforum-topic-composer__title-row.is-title-ok {
  background: color-mix(in srgb, var(--sf-accent-soft) 55%, var(--sf-public-surface));
  border-color: color-mix(in srgb, var(--sf-accent) 22%, var(--sf-public-border));
}

.sforum-topic-composer__title-row.is-title-near {
  background: color-mix(in srgb, #f59e0b 12%, var(--sf-public-surface));
  border-color: color-mix(in srgb, #f59e0b 40%, var(--sf-public-border));
}

.sforum-topic-composer__title-row.is-title-short,
.sforum-topic-composer__title-row.is-title-over {
  background: color-mix(in srgb, var(--sf-danger, #dc2626) 10%, var(--sf-public-surface));
  border-color: color-mix(in srgb, var(--sf-danger, #dc2626) 35%, var(--sf-public-border));
}

.sforum-topic-composer__title-row.is-title-short .sforum-topic-composer__summary-icon,
.sforum-topic-composer__title-row.is-title-over .sforum-topic-composer__summary-icon {
  background: color-mix(in srgb, var(--sf-danger, #dc2626) 14%, var(--sf-public-surface));
  color: var(--sf-danger, #dc2626);
}

.sforum-topic-composer__title-row.is-title-near .sforum-topic-composer__summary-icon {
  background: color-mix(in srgb, #f59e0b 16%, var(--sf-public-surface));
  color: #b45309;
}

.sforum-topic-composer__title-row.is-title-ok .sforum-topic-composer__summary-icon {
  background: var(--sf-accent-soft);
  color: var(--sf-accent);
}

.sforum-topic-composer__count {
  font-variant-numeric: tabular-nums;
  font-size: 0.72rem;
  line-height: 1.4;
}

.sforum-topic-composer__count[data-status='ok'] {
  color: var(--sf-public-text-secondary);
  font-weight: 650;
}

.sforum-topic-composer__count[data-status='near'] {
  color: #b45309;
  font-weight: 700;
}

.sforum-topic-composer__count[data-status='short'],
.sforum-topic-composer__count[data-status='over'] {
  color: var(--sf-danger, #dc2626);
  font-weight: 700;
}

.sforum-topic-composer__count[data-status='empty'] {
  color: var(--sf-public-text-muted);
}

/* —— 检查项：待补虚框 / 就绪实底 —— */
.sforum-topic-composer__checks {
  list-style: none;
  gap: 6px;
}

.sforum-topic-composer__check {
  display: grid;
  grid-template-columns: 22px minmax(0, 1fr) auto;
  gap: 8px;
  align-items: start;
  border: 1px solid var(--sf-public-border);
  border-radius: 8px;
  padding: 9px 10px;
  background: var(--sf-public-surface);
  color: var(--sf-public-text-muted);
  font-size: 0.75rem;
  line-height: 1.5;
  transition:
    background 0.2s ease,
    border-color 0.2s ease,
    box-shadow 0.2s ease,
    color 0.2s ease;
}

.sforum-topic-composer__check.is-todo {
  border-style: dashed;
  background: var(--sf-public-surface-muted);
}

.sforum-topic-composer__check.is-ok {
  border-style: solid;
  border-color: color-mix(in srgb, var(--sf-accent) 35%, var(--sf-public-border));
  background: var(--sf-accent-soft);
  color: var(--sf-public-text);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--sf-accent) 12%, transparent);
}

.sforum-topic-composer__check-mark {
  display: grid;
  place-items: start center;
  padding-top: 1px;
  color: var(--sf-public-text-muted);
  transition: color 0.18s ease, transform 0.18s ease;
}

.sforum-topic-composer__check.is-ok .sforum-topic-composer__check-mark {
  color: var(--sf-accent);
  transform: scale(1.06);
}

.sforum-topic-composer__check-body {
  min-width: 0;
  display: grid;
  gap: 2px;
}

.sforum-topic-composer__check strong {
  color: var(--sf-public-text);
  font-size: 0.8rem;
  font-weight: 700;
}

.sforum-topic-composer__check.is-todo strong {
  color: var(--sf-public-text-secondary);
}

.sforum-topic-composer__check-text {
  color: var(--sf-public-text-muted);
  font-size: 0.72rem;
  line-height: 1.45;
  overflow-wrap: anywhere;
}

.sforum-topic-composer__check.is-ok .sforum-topic-composer__check-text {
  color: var(--sf-public-text-secondary);
}

.sforum-topic-composer__check-status {
  flex: none;
  align-self: center;
  min-width: 2.5rem;
  border-radius: 999px;
  padding: 2px 7px;
  background: var(--sf-public-surface);
  color: var(--sf-public-text-muted);
  font-size: 10px;
  font-weight: 750;
  line-height: 1.4;
  text-align: center;
  transition:
    background 0.18s ease,
    color 0.18s ease;
}

.sforum-topic-composer__check.is-ok .sforum-topic-composer__check-status {
  background: var(--sf-accent);
  color: var(--sf-accent-contrast, #fff);
}

.sforum-topic-composer__checks-card.is-all-ready {
  border-color: color-mix(in srgb, var(--sf-accent) 40%, var(--sf-public-border));
  box-shadow:
    var(--sf-public-shadow),
    0 0 0 1px color-mix(in srgb, var(--sf-accent) 18%, transparent);
}

/* —— 值变化闪动 —— */
.sforum-topic-composer__summary-row.is-flash,
.sforum-topic-composer__check.is-flash,
.sforum-topic-composer__settings > div.is-flash {
  animation: sforum-composer-flash 0.52s ease;
}

@keyframes sforum-composer-flash {
  0% {
    box-shadow: 0 0 0 0 color-mix(in srgb, var(--sf-accent) 0%, transparent);
  }
  35% {
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--sf-accent) 28%, transparent);
  }
  100% {
    box-shadow: 0 0 0 0 color-mix(in srgb, var(--sf-accent) 0%, transparent);
  }
}

/* —— 规则表 —— */
.sforum-topic-composer__settings div {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  border-radius: 6px;
  padding: 4px 2px;
  color: var(--sf-public-text-muted);
  font-size: 0.75rem;
  transition: background 0.18s ease, box-shadow 0.18s ease;
}

.sforum-topic-composer__settings dd {
  margin: 0;
  color: var(--sf-public-text);
  font-weight: 700;
  text-align: right;
}

.sforum-topic-composer__settings dd.is-muted {
  color: var(--sf-public-text-muted);
  font-weight: 650;
}

.sforum-topic-composer__tip {
  display: flex;
  gap: 8px;
  margin: 0;
  padding: 12px 14px 14px;
  color: var(--sf-public-notice-text);
  font-size: 0.72rem;
  line-height: 1.6;
}

.sforum-topic-composer__tip :deep(svg) {
  flex: none;
  margin-top: 1px;
  color: var(--sf-accent);
}
</style>
