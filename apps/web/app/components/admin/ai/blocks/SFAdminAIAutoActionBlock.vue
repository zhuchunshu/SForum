<script setup lang="ts">
import type { AdminAISettings } from '~/composables/admin/useAdminAI'

// 动作白名单与后端 ValidAutoAction 一致：删除类动作永不进入可选项，
// 这是硬边界而不是可调项，因此这里也不提供它。
const AUTO_ACTIONS = ['hide', 'tag', 'reject'] as const

const { t } = useI18n()
const form = defineModel<AdminAISettings | null>('form', { required: true })
const disabled = defineModel<boolean>('disabled', { default: false })

const actionOptions = computed(() => [
  { value: 'hide', label: t('admin.ai.autoActionHide') },
  { value: 'tag', label: t('admin.ai.autoActionTag') },
  { value: 'reject', label: t('admin.ai.autoActionReject') }
])

// 后端在 enabled 且 allowedActions 为空时拒绝保存（422）。在这里提前说明，
// 而不是让运营者点了保存才看到一个没有解释的错误。
const needsActions = computed(() =>
  Boolean(form.value?.autoAction.enabled) && (form.value?.autoAction.allowedActions.length ?? 0) === 0
)

function toggleAction(action: string, checked: boolean) {
  const current = form.value
  if (!current) return
  const next = new Set(current.autoAction.allowedActions)
  if (checked) {
    next.add(action)
  } else {
    next.delete(action)
  }
  // 保持后端常量顺序，避免界面顺序随点击漂移。
  current.autoAction.allowedActions = AUTO_ACTIONS.filter(item => next.has(item))
}

function setThreshold(value: string | number) {
  const current = form.value
  if (!current) return
  const parsed = Number(value)
  if (Number.isNaN(parsed)) return
  current.autoAction.confidenceThreshold = Math.min(1, Math.max(0, parsed))
}
</script>

<template>
  <section v-if="form" class="min-w-0 rounded-lg border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
    <p class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.ai.autoAction') }}</p>
    <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.ai.autoActionIntro') }}</p>

    <UFormField class="mt-3">
      <UCheckbox
        v-model="form.autoAction.enabled"
        :label="t('admin.ai.autoActionEnabled')"
        :disabled="disabled"
      />
    </UFormField>

    <div class="mt-3 grid grid-cols-1 gap-3 lg:grid-cols-2">
      <UFormField :label="t('admin.ai.autoActionThreshold')" :help="t('admin.ai.autoActionThresholdHint')">
        <UInput
          :model-value="form.autoAction.confidenceThreshold"
          type="number"
          step="0.05"
          min="0"
          max="1"
          :disabled="disabled"
          class="w-full"
          @update:model-value="setThreshold"
        />
      </UFormField>
      <UFormField :label="t('admin.ai.autoActionActions')" :help="t('admin.ai.autoActionActionsHint')">
        <div class="flex flex-wrap gap-4 py-1">
          <UCheckbox
            v-for="option in actionOptions"
            :key="option.value"
            :model-value="form.autoAction.allowedActions.includes(option.value)"
            :label="option.label"
            :disabled="disabled"
            @update:model-value="value => toggleAction(option.value, value === true)"
          />
        </div>
      </UFormField>
    </div>

    <p v-if="needsActions" class="mt-2 text-xs text-amber-600 dark:text-amber-400">
      {{ t('admin.ai.autoActionNeedsActions') }}
    </p>
  </section>
</template>
