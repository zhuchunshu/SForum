<script setup lang="ts">
import type { AdminAISettings } from '~/composables/admin/useAdminAI'

const { t } = useI18n()
const form = defineModel<AdminAISettings | null>('form', { required: true })
const disabled = defineModel<boolean>('disabled', { default: false })

// 关闭总开关时把三个子项一并关掉，避免「脱敏已启用但什么都没脱」的自相矛盾状态。
function toggleEnabled(value: boolean) {
  const current = form.value
  if (!current) return
  current.redaction.enabled = value
  if (!value) {
    current.redaction.redactEmail = false
    current.redaction.redactPhone = false
    current.redaction.redactIp = false
  }
}
</script>

<template>
  <section v-if="form" class="min-w-0 rounded-lg border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
    <p class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.ai.redaction') }}</p>
    <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.ai.redactionIntro') }}</p>

    <UFormField class="mt-3">
      <UCheckbox
        :model-value="form.redaction.enabled"
        :label="t('admin.ai.redactionEnabled')"
        :disabled="disabled"
        @update:model-value="value => toggleEnabled(value === true)"
      />
    </UFormField>

    <div class="mt-3 flex flex-wrap gap-4">
      <UCheckbox
        v-model="form.redaction.redactEmail"
        :label="t('admin.ai.redactEmail')"
        :disabled="disabled || !form.redaction.enabled"
      />
      <UCheckbox
        v-model="form.redaction.redactPhone"
        :label="t('admin.ai.redactPhone')"
        :disabled="disabled || !form.redaction.enabled"
      />
      <UCheckbox
        v-model="form.redaction.redactIp"
        :label="t('admin.ai.redactIp')"
        :disabled="disabled || !form.redaction.enabled"
      />
    </div>
  </section>
</template>
