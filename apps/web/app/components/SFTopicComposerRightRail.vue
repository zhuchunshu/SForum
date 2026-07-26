<script setup lang="ts">
/**
 * 发帖页右栏：摘要 / 检查 / 规则 / 提示。
 * 桌面与移动抽屉共用；优先紧凑密度，避免检查项占满视口。
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
  tagPolicyLabel: string
  actorName: string
  publishVisibilityLabel: string
  checks: ComposerPrePublishCheck[]
  titleMax: number
  bodyMax: number
  /** 默认展示发帖权限；编辑等复用场景可覆盖为对应权限语义。 */
  permissionValueLabel?: string
}>()

const { t } = useI18n()

const hasCategory = computed(() => Boolean(props.categoryName))

const summaryCategoryName = computed(() => props.categoryName || t('composer.categoryDefaultShort'))

const checksReady = computed(() => props.checks.filter(item => item.ok).length)
const checksTotal = computed(() => props.checks.length)
const checksAllReady = computed(() => checksTotal.value > 0 && checksReady.value === checksTotal.value)
const checksProgressLabel = computed(() =>
  t('composer.rightRail.checksProgress', {
    ready: checksReady.value,
    total: checksTotal.value
  })
)

/** 值变化时短暂高亮 */
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

watch(() => props.categoryName, () => triggerFlash('category'))
watch(() => props.tagPolicyLabel, () => triggerFlash('tags'))
watch(() => props.checks.map(item => `${item.key}:${item.ok}:${item.text}`).join('|'), (_next, prev) => {
  if (!prev) {
    return
  }
  for (const item of props.checks) {
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
    <!-- 发布摘要：一行分类 + 一行发布者 -->
    <section class="sf-home-right-rail__card">
      <header class="sf-home-right-rail__head">
        <h3 class="sf-home-right-rail__title">{{ t('composer.summary.title') }}</h3>
        <span
          class="sf-home-right-rail__meta sforum-topic-composer__live-badge"
          :class="{ 'is-live': hasCategory }"
        >
          {{ publishVisibilityLabel }}
        </span>
      </header>
      <dl class="sforum-topic-composer__meta-list">
        <div
          class="sforum-topic-composer__meta-row"
          :class="{
            'is-filled': hasCategory,
            'is-empty': !hasCategory,
            'is-flash': isFlashing('category')
          }"
        >
          <dt>
            <UIcon name="i-lucide-folder" class="size-3.5" aria-hidden="true" />
            {{ t('composer.settings.category') }}
          </dt>
          <dd :class="{ 'is-muted': !hasCategory }">{{ summaryCategoryName }}</dd>
        </div>
        <div class="sforum-topic-composer__meta-row is-filled">
          <dt>
            <UIcon name="i-lucide-user-round" class="size-3.5" aria-hidden="true" />
            {{ t('composer.summary.actor') }}
          </dt>
          <dd>{{ actorName }}</dd>
        </div>
      </dl>
    </section>

    <!-- 发布前检查：紧凑单行；仅未就绪时展开说明 -->
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
              class="size-4"
            />
          </span>
          <span class="sforum-topic-composer__check-body">
            <strong>{{ item.label }}</strong>
            <!-- 已就绪只保留标签；待补时展示下一步提示 -->
            <span
              v-if="!item.ok"
              class="sforum-topic-composer__check-text"
            >{{ item.text }}</span>
          </span>
        </li>
      </ul>
    </section>

    <!-- 站点规则摘要 -->
    <section class="sf-home-right-rail__card">
      <header class="sf-home-right-rail__head">
        <h3 class="sf-home-right-rail__title">{{ t('composer.settings.title') }}</h3>
      </header>
      <dl class="sforum-topic-composer__settings">
        <div>
          <dt>{{ t('composer.settings.permission') }}</dt>
          <dd>{{ permissionValueLabel || t('composer.settings.permissionValue') }}</dd>
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
        <UIcon name="i-lucide-lightbulb" class="size-3.5" aria-hidden="true" />
        <span>{{ t('composer.rightRail.tip') }}</span>
      </div>
    </section>
  </div>
</template>

<style scoped>
/* 行级样式：卡片壳来自全局 sf-home-right-rail */

.sforum-topic-composer__meta-list,
.sforum-topic-composer__settings,
.sforum-topic-composer__checks {
  display: grid;
  gap: 4px;
  margin: 0;
  padding: 0 10px 10px;
}

/* —— 实时徽标 —— */
.sforum-topic-composer__live-badge {
  display: inline-flex;
  align-items: center;
  min-height: 18px;
  border-radius: 999px;
  padding: 0 7px;
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

/* —— 摘要：紧凑键值行 —— */
.sforum-topic-composer__meta-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  border-radius: 6px;
  padding: 5px 6px;
  color: var(--sf-public-text-muted);
  font-size: 0.72rem;
  line-height: 1.35;
  transition:
    background 0.18s ease,
    box-shadow 0.18s ease;
}

.sforum-topic-composer__meta-row.is-empty {
  background: var(--sf-public-surface-muted);
}

.sforum-topic-composer__meta-row.is-filled {
  background: color-mix(in srgb, var(--sf-accent-soft) 40%, transparent);
}

.sforum-topic-composer__meta-row dt {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-width: 0;
  color: var(--sf-public-text-muted);
  font-weight: 650;
}

.sforum-topic-composer__meta-row dd {
  margin: 0;
  min-width: 0;
  color: var(--sf-public-text);
  font-weight: 700;
  text-align: right;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sforum-topic-composer__meta-row dd.is-muted {
  color: var(--sf-public-text-muted);
  font-weight: 650;
}

/* —— 检查项：默认单行；待补时才展开说明 —— */
.sforum-topic-composer__checks {
  list-style: none;
  gap: 3px;
}

.sforum-topic-composer__check {
  display: grid;
  grid-template-columns: 16px minmax(0, 1fr);
  gap: 6px;
  align-items: start;
  border-radius: 6px;
  padding: 5px 6px;
  background: transparent;
  color: var(--sf-public-text-muted);
  font-size: 0.72rem;
  line-height: 1.35;
  transition:
    background 0.18s ease,
    box-shadow 0.18s ease,
    color 0.18s ease;
}

.sforum-topic-composer__check.is-todo {
  background: var(--sf-public-surface-muted);
}

.sforum-topic-composer__check.is-ok {
  background: color-mix(in srgb, var(--sf-accent-soft) 45%, transparent);
  color: var(--sf-public-text);
}

.sforum-topic-composer__check-mark {
  display: grid;
  place-items: center;
  height: 1.15rem;
  color: var(--sf-public-text-muted);
  transition: color 0.18s ease;
}

.sforum-topic-composer__check.is-ok .sforum-topic-composer__check-mark {
  color: var(--sf-accent);
}

.sforum-topic-composer__check-body {
  min-width: 0;
  display: grid;
  gap: 1px;
}

.sforum-topic-composer__check strong {
  color: var(--sf-public-text);
  font-size: 0.75rem;
  font-weight: 700;
  line-height: 1.3;
}

.sforum-topic-composer__check.is-todo strong {
  color: var(--sf-public-text-secondary);
}

.sforum-topic-composer__check-text {
  color: var(--sf-public-text-muted);
  font-size: 0.68rem;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.sforum-topic-composer__checks-card.is-all-ready {
  border-color: color-mix(in srgb, var(--sf-accent) 36%, var(--sf-public-border));
}

/* —— 值变化闪动 —— */
.sforum-topic-composer__meta-row.is-flash,
.sforum-topic-composer__check.is-flash,
.sforum-topic-composer__settings > div.is-flash {
  animation: sforum-composer-flash 0.52s ease;
}

@keyframes sforum-composer-flash {
  0% {
    box-shadow: 0 0 0 0 color-mix(in srgb, var(--sf-accent) 0%, transparent);
  }
  35% {
    box-shadow: 0 0 0 2px color-mix(in srgb, var(--sf-accent) 26%, transparent);
  }
  100% {
    box-shadow: 0 0 0 0 color-mix(in srgb, var(--sf-accent) 0%, transparent);
  }
}

/* —— 规则表 —— */
.sforum-topic-composer__settings div {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  border-radius: 4px;
  padding: 3px 2px;
  color: var(--sf-public-text-muted);
  font-size: 0.72rem;
  line-height: 1.4;
  transition: background 0.18s ease, box-shadow 0.18s ease;
}

.sforum-topic-composer__settings dt {
  flex: none;
  max-width: 42%;
}

.sforum-topic-composer__settings dd {
  margin: 0;
  min-width: 0;
  color: var(--sf-public-text);
  font-weight: 700;
  text-align: right;
  overflow-wrap: anywhere;
}

.sforum-topic-composer__settings dd.is-muted {
  color: var(--sf-public-text-muted);
  font-weight: 650;
}

.sforum-topic-composer__tip {
  display: flex;
  gap: 6px;
  margin: 0;
  padding: 8px 10px 10px;
  color: var(--sf-public-notice-text);
  font-size: 0.68rem;
  line-height: 1.5;
}

.sforum-topic-composer__tip :deep(svg) {
  flex: none;
  margin-top: 1px;
  color: var(--sf-accent);
}
</style>
