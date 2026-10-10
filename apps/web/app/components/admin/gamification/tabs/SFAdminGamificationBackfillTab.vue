<script setup lang="ts">
import { ADMIN_GAMIFICATION_KEY, type AdminGamificationState } from '~/composables/admin/useAdminGamification'

/**
 * 历史回填：把存量发帖/评论数写入计量表并补发成长值。
 * 语义是权威覆盖（不是累加），重复执行安全。敏感操作，需显式确认。
 */
const gamification = inject(ADMIN_GAMIFICATION_KEY) as AdminGamificationState
const { t } = useI18n()

const payloadText = ref('')
const parseError = ref('')
const parsedCount = computed(() => {
  if (parseError.value) return 0
  try {
    const parsed = JSON.parse(payloadText.value || '{}') as { users?: unknown[] }
    return Array.isArray(parsed.users) ? parsed.users.length : 0
  } catch {
    return 0
  }
})

function validate() {
  parseError.value = ''
  if (!payloadText.value.trim()) {
    parseError.value = t('admin.gamification.backfill.required')
    return null
  }
  let parsed: { users?: Array<Record<string, unknown>> }
  try {
    parsed = JSON.parse(payloadText.value) as { users?: Array<Record<string, unknown>> }
  } catch {
    parseError.value = t('admin.gamification.backfill.invalidJson')
    return null
  }
  if (!Array.isArray(parsed.users) || parsed.users.length === 0) {
    parseError.value = t('admin.gamification.backfill.emptyUsers')
    return null
  }
  for (const entry of parsed.users) {
    const id = Number(entry.userId)
    if (!Number.isInteger(id) || id <= 0) {
      parseError.value = t('admin.gamification.backfill.invalidUser')
      return null
    }
  }
  return parsed.users.map(entry => ({
    userId: Number(entry.userId),
    topicCount: Number(entry.topicCount) || 0,
    commentCount: Number(entry.commentCount) || 0
  }))
}

async function run() {
  const users = validate()
  if (!users) return
  await gamification.backfill(users)
}

function fillExample() {
  payloadText.value = JSON.stringify({
    users: [{ userId: 42, topicCount: 17, commentCount: 4 }]
  }, null, 2)
}
</script>

<template>
  <UCard class="border-slate-200 bg-white text-slate-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100">
    <template #header>
      <div class="min-w-0">
        <h2 class="text-base font-bold">{{ t('admin.gamification.backfill.title') }}</h2>
        <p class="mt-1 text-xs text-muted">{{ t('admin.gamification.backfill.description') }}</p>
      </div>
    </template>

    <div class="space-y-4">
      <UAlert
        color="warning"
        variant="soft"
        icon="i-lucide-triangle-alert"
        :title="t('admin.gamification.backfill.warningTitle')"
        :description="t('admin.gamification.backfill.warning')"
      />

      <UFormField
        :label="t('admin.gamification.backfill.payload')"
        :description="t('admin.gamification.backfill.payloadHint')"
        name="backfill-payload"
      >
        <UTextarea v-model="payloadText" :rows="8" class="w-full font-mono text-xs" />
      </UFormField>

      <UAlert v-if="parseError" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="parseError" />

      <div class="flex flex-wrap items-center gap-2">
        <UButton
          color="neutral"
          variant="ghost"
          leading-icon="i-lucide-file-json"
          @click="fillExample"
        >
          {{ t('admin.gamification.backfill.fillExample') }}
        </UButton>
        <UButton
          leading-icon="i-lucide-history"
          :disabled="!!parseError || !parsedCount || gamification.saving.value"
          :loading="gamification.saving.value"
          class="bg-[var(--sf-accent)] hover:bg-[var(--sf-accent-hover)] font-semibold text-white"
          @click="run"
        >
          {{ t('admin.gamification.backfill.run', { count: parsedCount }) }}
        </UButton>
      </div>
    </div>
  </UCard>
</template>
