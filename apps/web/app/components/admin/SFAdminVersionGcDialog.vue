<script setup lang="ts">
import { formatPluginMemoryBytes } from '~/utils/admin/adminExtensions'
import {
  versionGcBlockedSummary,
  versionGcReclaimable,
  type AdminVersionGCBlockedReason,
  type AdminVersionGCCandidate,
  type AdminVersionGCConfirmation,
  type AdminVersionGCPolicy,
  type AdminVersionGCSweepResult
} from '~/utils/admin/adminVersionGc'

const props = defineProps<{
  busy?: boolean
  error?: string
  candidates: AdminVersionGCCandidate[]
  confirmation: AdminVersionGCConfirmation | null
  result: AdminVersionGCSweepResult | null
}>()

const emit = defineEmits<{
  cancel: []
  refresh: []
  confirmPlan: []
  execute: []
}>()

const open = defineModel<boolean>('open', { required: true })
const policy = defineModel<AdminVersionGCPolicy>('policy', { required: true })
const { t } = useI18n()
const { format: formatSiteDateTime } = useSiteDateTime()

const reclaimable = computed(() => versionGcReclaimable(props.candidates))
const blocked = computed(() => versionGcBlockedSummary(props.candidates))
// 阶段机：结果 → 确认 → 计划。execute 之后停在结果视图直到关闭。
const phase = computed<'result' | 'confirmation' | 'plan'>(() => {
  if (props.result) return 'result'
  if (props.confirmation) return 'confirmation'
  return 'plan'
})

function blockedLabel(reason: AdminVersionGCBlockedReason | undefined) {
  if (!reason) return ''
  return t(`admin.extensions.versionGc.blocked.${reason}`)
}

function candidateSubtitle(item: AdminVersionGCCandidate) {
  return `${item.extensionId} · ${t(`admin.extensions.types.${item.extensionType}`)} · v${item.version}`
}
</script>

<template>
  <UModal v-model:open="open" :ui="{ content: 'sm:max-w-2xl' }">
    <template #content>
      <div class="p-5 sm:p-6">
        <div class="flex items-start gap-3">
          <div class="flex size-9 shrink-0 items-center justify-center rounded-md bg-amber-50 text-amber-600 dark:bg-amber-950/40 dark:text-amber-300">
            <UIcon name="i-lucide-archive-x" class="size-4" />
          </div>
          <div class="min-w-0">
            <h2 class="text-base font-semibold text-slate-900 dark:text-zinc-100">
              {{ t('admin.extensions.versionGc.title') }}
            </h2>
            <p class="mt-1 text-sm leading-6 text-slate-600 dark:text-zinc-300">
              {{ t('admin.extensions.versionGc.description') }}
            </p>
          </div>
        </div>

        <template v-if="phase === 'plan'">
          <div class="mt-5 border-t border-slate-200 pt-5 dark:border-zinc-800">
            <h3 class="text-sm font-medium text-slate-900 dark:text-zinc-100">
              {{ t('admin.extensions.versionGc.policyTitle') }}
            </h3>
            <div class="mt-3 grid gap-3 sm:grid-cols-2">
              <UFormField :label="t('admin.extensions.versionGc.retainLabel')" name="gc-retain">
                <UInputNumber v-model="policy.retain" :min="0" class="w-full" />
              </UFormField>
              <UFormField :label="t('admin.extensions.versionGc.retainDaysLabel')" name="gc-retain-days">
                <UInputNumber v-model="policy.retainDays" :min="0" class="w-full" />
              </UFormField>
            </div>
            <UButton
              class="mt-3"
              color="neutral"
              variant="soft"
              icon="i-lucide-refresh-cw"
              :loading="busy"
              @click="emit('refresh')"
            >
              {{ t('admin.extensions.versionGc.refreshPlan') }}
            </UButton>
          </div>

          <div class="mt-5">
            <div class="flex items-center justify-between gap-3">
              <h3 class="text-sm font-medium text-slate-900 dark:text-zinc-100">
                {{ t('admin.extensions.versionGc.listTitle') }}
              </h3>
              <span class="text-xs text-slate-500 dark:text-zinc-400">
                {{ t('admin.extensions.versionGc.reclaimableCount', { count: reclaimable.length }) }}
              </span>
            </div>
            <p v-if="candidates.length === 0" class="mt-2 text-sm text-slate-500 dark:text-zinc-400">
              {{ t('admin.extensions.versionGc.empty') }}
            </p>
            <ul
              v-else
              class="mt-2 max-h-64 divide-y divide-slate-200 overflow-y-auto rounded-md border border-slate-200 dark:divide-zinc-800 dark:border-zinc-800"
            >
              <li
                v-for="item in candidates"
                :key="item.versionId"
                class="flex items-start gap-3 px-3 py-3"
              >
                <UIcon
                  :name="item.extensionType === 'theme' ? 'i-lucide-palette' : 'i-lucide-plug'"
                  class="mt-0.5 size-4 shrink-0"
                  :class="item.reclaimable ? 'text-amber-500' : 'text-slate-400 dark:text-zinc-500'"
                />
                <div class="min-w-0 flex-1">
                  <p class="truncate text-sm font-medium text-slate-900 dark:text-zinc-100">
                    {{ item.extensionName || item.extensionId }}
                  </p>
                  <p class="mt-0.5 break-all text-xs text-slate-500 dark:text-zinc-400">
                    {{ candidateSubtitle(item) }} · {{ t('admin.extensions.versionGc.rank', { rank: item.historyRank }) }}
                  </p>
                </div>
                <UBadge
                  :color="item.reclaimable ? 'warning' : 'neutral'"
                  variant="subtle"
                  class="shrink-0"
                >
                  {{ item.reclaimable
                    ? t('admin.extensions.versionGc.reclaimableBadge')
                    : blockedLabel(item.blockedReason) }}
                </UBadge>
              </li>
            </ul>
            <div
              v-if="blocked.size"
              class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500 dark:text-zinc-400"
            >
              <span v-for="[reason, count] in blocked" :key="reason">
                {{ t('admin.extensions.versionGc.blockedCount', { reason: blockedLabel(reason), count }) }}
              </span>
            </div>
          </div>

          <UAlert
            class="mt-4"
            color="warning"
            icon="i-lucide-triangle-alert"
            variant="subtle"
            :title="t('admin.extensions.versionGc.planWarning')"
          />
        </template>

        <template v-else-if="phase === 'confirmation' && confirmation">
          <div class="mt-5 border-t border-slate-200 pt-5 dark:border-zinc-800">
            <h3 class="text-sm font-medium text-slate-900 dark:text-zinc-100">
              {{ t('admin.extensions.versionGc.confirmationTitle') }}
            </h3>
            <p class="mt-1 text-sm leading-6 text-slate-600 dark:text-zinc-300">
              {{ t('admin.extensions.versionGc.confirmationSummary', {
                count: confirmation.candidateCount,
                expires: formatSiteDateTime(confirmation.expiresAt)
              }) }}
            </p>
            <UAlert
              class="mt-4"
              color="error"
              icon="i-lucide-triangle-alert"
              variant="subtle"
              :title="t('admin.extensions.versionGc.executeWarning')"
            />
          </div>
        </template>

        <template v-else-if="phase === 'result' && result">
          <div class="mt-5 border-t border-slate-200 pt-5 dark:border-zinc-800">
            <h3 class="text-sm font-medium text-slate-900 dark:text-zinc-100">
              {{ t('admin.extensions.versionGc.resultTitle') }}
            </h3>
            <p class="mt-1 text-sm leading-6 text-slate-600 dark:text-zinc-300">
              {{ t('admin.extensions.versionGc.resultSummary', {
                removed: result.removed.length,
                bytes: formatPluginMemoryBytes(result.bytesFreed)
              }) }}
            </p>
            <UAlert
              v-if="result.failed.length"
              class="mt-3"
              color="error"
              icon="i-lucide-triangle-alert"
              variant="subtle"
              :title="t('admin.extensions.versionGc.failedSummary', { count: result.failed.length })"
            />
            <UAlert
              v-if="result.warnings && result.warnings.length"
              class="mt-3"
              color="warning"
              icon="i-lucide-info"
              variant="subtle"
              :title="t('admin.extensions.versionGc.warningNotice')"
              :description="result.warnings.join(' ')"
            />
          </div>
        </template>

        <UAlert
          v-if="error"
          class="mt-4"
          color="error"
          icon="i-lucide-triangle-alert"
          variant="subtle"
          :title="error"
        />

        <div class="mt-6 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <UButton color="neutral" variant="ghost" :disabled="busy" @click="emit('cancel')">
            {{ t('admin.extensions.confirmUninstallCancel') }}
          </UButton>
          <UButton
            v-if="phase === 'plan'"
            color="warning"
            icon="i-lucide-key-round"
            :loading="busy"
            :disabled="reclaimable.length === 0"
            @click="emit('confirmPlan')"
          >
            {{ t('admin.extensions.versionGc.confirmPlan') }}
          </UButton>
          <template v-else-if="phase === 'confirmation'">
            <UButton color="neutral" variant="soft" :disabled="busy" @click="emit('refresh')">
              {{ t('admin.extensions.versionGc.backToPlan') }}
            </UButton>
            <UButton color="error" icon="i-lucide-trash-2" :loading="busy" @click="emit('execute')">
              {{ t('admin.extensions.versionGc.execute') }}
            </UButton>
          </template>
        </div>
      </div>
    </template>
  </UModal>
</template>
