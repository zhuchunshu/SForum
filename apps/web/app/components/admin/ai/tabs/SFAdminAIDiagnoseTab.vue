<script setup lang="ts">
import { ADMIN_AI_KEY } from '~/composables/admin/useAdminAI'

const { t, te } = useI18n()
// 数据由页面壳统一持有并注入；页签只渲染与触发。
const ai = inject(ADMIN_AI_KEY)!

const message = ref('')

const activeProfile = computed(() => ai.settings.value.profiles.find(profile => profile.enabled) ?? null)
const keyConfigured = computed(() => (activeProfile.value ? Boolean(ai.credentialStatus.value[activeProfile.value.id]) : false))
const configured = computed(() => ai.settings.value.enabled && Boolean(activeProfile.value) && keyConfigured.value)

// 失败原因优先用翻译；缺翻译时退回原始原因码，至少可诊断。
const reasonText = computed(() => {
  const reason = ai.diagnoseResult.value?.reason
  if (!reason) return ''
  const key = `admin.ai.diagnose.reasons.${reason}`
  return te(key) ? t(key) : reason
})

function runDiagnose() {
  void ai.diagnose(message.value)
}
</script>

<template>
  <div class="flex min-w-0 flex-col gap-4">
    <section class="min-w-0 rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <div class="border-b border-slate-200 px-4 py-3 dark:border-zinc-800">
        <p class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.ai.diagnose.current') }}</p>
      </div>
      <dl class="grid grid-cols-1 gap-3 px-4 py-4 sm:grid-cols-3">
        <div class="min-w-0">
          <dt class="text-xs text-slate-500">{{ t('admin.ai.diagnose.provider') }}</dt>
          <dd class="mt-1 truncate text-sm font-medium text-slate-900 dark:text-zinc-100">
            {{ activeProfile ? activeProfile.label : t('admin.ai.providerNone') }}
          </dd>
        </div>
        <div class="min-w-0">
          <dt class="text-xs text-slate-500">{{ t('admin.ai.diagnose.model') }}</dt>
          <dd class="mt-1 truncate text-sm text-slate-900 dark:text-zinc-100">{{ activeProfile?.model || '-' }}</dd>
        </div>
        <div class="min-w-0">
          <dt class="text-xs text-slate-500">{{ t('admin.ai.diagnose.credential') }}</dt>
          <dd class="mt-1">
            <UBadge :color="keyConfigured ? 'success' : 'neutral'" variant="soft" size="sm">
              {{ keyConfigured ? t('admin.ai.credentialConfigured') : t('admin.ai.credentialMissing') }}
            </UBadge>
          </dd>
        </div>
        <div class="min-w-0">
          <dt class="text-xs text-slate-500">{{ t('admin.ai.diagnose.replyBot') }}</dt>
          <dd class="mt-1 flex flex-wrap items-center gap-2">
            <span v-if="ai.replyBot.value.configured" class="truncate text-sm font-medium text-slate-900 dark:text-zinc-100">
              {{ ai.replyBot.value.username }}
            </span>
            <UBadge v-if="ai.replyBot.value.configured" color="success" variant="soft" size="sm">
              {{ t('admin.ai.diagnose.replyBotReady') }}
            </UBadge>
            <UBadge v-else color="warning" variant="soft" size="sm">
              {{ t('admin.ai.diagnose.replyBotMissing') }}
            </UBadge>
          </dd>
        </div>
      </dl>
      <p v-if="!ai.replyBot.value.configured" class="border-t border-slate-200 px-4 py-3 text-xs text-slate-500 dark:border-zinc-800 dark:text-zinc-400">
        {{ t('admin.ai.diagnose.replyBotHint') }}
      </p>
    </section>

    <UAlert
      v-if="!configured"
      color="warning"
      variant="soft"
      icon="i-lucide-circle-alert"
      :title="t('admin.ai.diagnose.notReady')"
    />

    <section class="min-w-0 rounded-lg border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
      <p class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.ai.diagnose.title') }}</p>
      <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.ai.diagnose.intro') }}</p>
      <div class="mt-3 flex min-w-0 flex-col gap-2 sm:flex-row sm:items-end">
        <UFormField :label="t('admin.ai.diagnose.message')" class="min-w-0 flex-1">
          <UInput v-model="message" :placeholder="t('admin.ai.diagnose.messagePlaceholder')" class="w-full" />
        </UFormField>
        <UButton
          icon="i-lucide-activity"
          :loading="ai.busy.value === 'diagnose'"
          :disabled="!configured"
          @click="runDiagnose"
        >
          {{ t('admin.ai.diagnose.run') }}
        </UButton>
      </div>
    </section>

    <section
      v-if="ai.diagnoseResult.value"
      class="min-w-0 rounded-lg border p-4"
      :class="ai.diagnoseResult.value.ok
        ? 'border-emerald-200 bg-emerald-50 dark:border-emerald-900 dark:bg-emerald-950/30'
        : 'border-rose-200 bg-rose-50 dark:border-rose-900 dark:bg-rose-950/30'"
    >
      <div class="flex flex-wrap items-center gap-2">
        <UIcon
          :name="ai.diagnoseResult.value.ok ? 'i-lucide-circle-check' : 'i-lucide-circle-x'"
          class="size-4"
          :class="ai.diagnoseResult.value.ok ? 'text-emerald-600' : 'text-rose-600'"
        />
        <p class="text-sm font-semibold">
          {{ ai.diagnoseResult.value.ok ? t('admin.ai.diagnose.success') : t('admin.ai.diagnose.failure') }}
        </p>
        <span v-if="ai.diagnoseResult.value.providerId" class="text-xs text-slate-500">
          {{ ai.diagnoseResult.value.providerId }} · {{ ai.diagnoseResult.value.model }}
        </span>
      </div>

      <p v-if="reasonText" class="mt-2 text-sm text-slate-700 dark:text-zinc-200">{{ reasonText }}</p>
      <p v-if="ai.diagnoseResult.value.error" class="mt-1 break-all font-mono text-xs text-slate-500">{{ ai.diagnoseResult.value.error }}</p>

      <p v-if="ai.diagnoseResult.value.ok" class="mt-2 rounded-md bg-white/70 px-3 py-2 text-sm dark:bg-zinc-900/60">
        {{ ai.diagnoseResult.value.reply }}
      </p>

      <dl v-if="ai.diagnoseResult.value.ok" class="mt-3 grid grid-cols-3 gap-3 text-xs">
        <div>
          <dt class="text-slate-500">{{ t('admin.ai.diagnose.latency') }}</dt>
          <dd class="mt-0.5 tabular-nums">{{ ai.diagnoseResult.value.latencyMs }} ms</dd>
        </div>
        <div>
          <dt class="text-slate-500">{{ t('admin.ai.diagnose.tokensIn') }}</dt>
          <dd class="mt-0.5 tabular-nums">{{ ai.diagnoseResult.value.inputTokens }}</dd>
        </div>
        <div>
          <dt class="text-slate-500">{{ t('admin.ai.diagnose.tokensOut') }}</dt>
          <dd class="mt-0.5 tabular-nums">{{ ai.diagnoseResult.value.outputTokens }}</dd>
        </div>
      </dl>
    </section>
  </div>
</template>
