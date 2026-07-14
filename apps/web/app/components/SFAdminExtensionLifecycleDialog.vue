<script setup lang="ts">
import {
  canRecoverLifecycleOperation,
  canSubmitLifecycleRecovery,
  type AdminExtension,
  type AdminLifecycleOperation,
  type AdminLifecycleOperationDetail,
  type AdminLifecycleRecoveryDecision,
  type AdminLifecycleRecoveryInput
} from '~/utils/adminExtensions'

const props = defineProps<{
  extension: AdminExtension | null
  operations: AdminLifecycleOperation[]
  operation: AdminLifecycleOperationDetail | null
  loading?: boolean
  recoveryBusy?: boolean
  error?: string
  isSuperAdmin: boolean
}>()

const emit = defineEmits<{
  select: [operationId: number]
  recover: [input: AdminLifecycleRecoveryInput]
}>()

const open = defineModel<boolean>('open', { required: true })
const { t } = useI18n()
const { format: formatSiteDateTime } = useSiteDateTime()
const decision = ref<AdminLifecycleRecoveryDecision>('retry')
const reason = ref('')
const escalateForced = ref(false)
const residualRiskAcknowledged = ref(false)

const extensionIdentity = computed(() => {
  const id = props.extension?.id || ''
  const name = props.extension?.name?.trim()
  return name && name !== id ? `${name} · ${id}` : id || name || ''
})

const recoveryOptions = computed(() => [
  {
    value: 'retry' as const,
    label: t('admin.extensions.lifecycle.recovery.retry'),
    description: t('admin.extensions.lifecycle.recovery.retryDescription')
  },
  {
    value: 'skip_step' as const,
    label: t('admin.extensions.lifecycle.recovery.skip'),
    description: t('admin.extensions.lifecycle.recovery.skipDescription')
  }
])

const recoveryInput = computed<AdminLifecycleRecoveryInput>(() => ({
  decision: decision.value,
  reason: reason.value,
  escalateForced: escalateForced.value,
  residualRiskAcknowledged: residualRiskAcknowledged.value
}))

const canRecover = computed(() => canRecoverLifecycleOperation(props.operation))
const canSubmitRecovery = computed(() => canSubmitLifecycleRecovery(
  props.operation,
  recoveryInput.value,
  props.isSuperAdmin
))

watch(() => props.operation?.id, () => {
  decision.value = 'retry'
  reason.value = ''
  escalateForced.value = false
  residualRiskAcknowledged.value = false
})

watch(escalateForced, (forced) => {
  if (!forced) residualRiskAcknowledged.value = false
})

function operationStatusColor(operation: AdminLifecycleOperation) {
  if (operation.terminalResult === 'succeeded' || operation.terminalResult === 'skipped') return 'success'
  if (operation.terminalResult === 'failed' || operation.terminalResult === 'cancelled') return 'error'
  return 'warning'
}

function stepStatusColor(status: string) {
  if (status === 'succeeded' || status === 'skipped') return 'success'
  if (status === 'failed' || status === 'cancelled') return 'error'
  if (status === 'running' || status === 'waiting') return 'warning'
  return 'neutral'
}

function submitRecovery() {
  if (canSubmitRecovery.value) emit('recover', recoveryInput.value)
}

function closeDialog() {
  open.value = false
}
</script>

<template>
  <UModal v-model:open="open" :ui="{ content: 'sm:max-w-5xl' }">
    <template #content>
      <div class="flex max-h-[min(84vh,860px)] flex-col">
        <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-5 py-4 dark:border-zinc-800 sm:px-6">
          <div class="min-w-0">
            <h2 class="flex items-center gap-2 text-base font-semibold text-slate-900 dark:text-zinc-100">
              <UIcon name="i-lucide-history" class="size-4 text-[var(--sf-accent)]" />
              {{ t('admin.extensions.lifecycle.title') }}
            </h2>
            <p class="mt-1 truncate text-sm text-slate-500 dark:text-zinc-400">
              {{ extensionIdentity }}
            </p>
          </div>
          <UButton
            color="neutral"
            variant="ghost"
            icon="i-lucide-x"
            :aria-label="t('common.close')"
            :title="t('common.close')"
            @click="closeDialog"
          />
        </div>

        <div class="min-h-0 flex-1 overflow-y-auto p-5 sm:p-6">
          <UAlert
            v-if="error"
            class="mb-4"
            color="error"
            icon="i-lucide-triangle-alert"
            variant="subtle"
            :title="error"
          />

          <div v-if="loading && operations.length === 0" class="flex min-h-52 items-center justify-center text-sm text-slate-500 dark:text-zinc-400">
            <UIcon name="i-lucide-loader-circle" class="mr-2 size-4 animate-spin" />
            {{ t('admin.extensions.lifecycle.loading') }}
          </div>

          <SFEmptyState
            v-else-if="operations.length === 0"
            icon-label="HST"
            :title="t('admin.extensions.lifecycle.emptyTitle')"
            :description="t('admin.extensions.lifecycle.emptyDescription')"
          />

          <div v-else class="grid min-h-0 gap-5 md:grid-cols-[minmax(210px,280px)_minmax(0,1fr)]">
            <nav class="space-y-2" :aria-label="t('admin.extensions.lifecycle.history')">
              <button
                v-for="item in operations"
                :key="item.id"
                type="button"
                class="w-full rounded-md border px-3 py-3 text-left transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--sf-accent)]"
                :class="operation?.id === item.id
                  ? 'border-[var(--sf-accent)] bg-[var(--sf-accent-soft)] dark:border-[var(--sf-accent-dark)] dark:bg-zinc-800'
                  : 'border-slate-200 bg-white hover:bg-slate-50 dark:border-zinc-800 dark:bg-zinc-900 dark:hover:bg-zinc-800'"
                @click="emit('select', item.id)"
              >
                <span class="flex items-center justify-between gap-2">
                  <span class="text-sm font-medium text-slate-900 dark:text-zinc-100">
                    {{ t(`admin.extensions.lifecycle.operations.${item.operation}`) }}
                  </span>
                  <UBadge :color="operationStatusColor(item)" variant="subtle" size="xs">
                    {{ t(`admin.extensions.lifecycle.results.${item.terminalResult || 'running'}`) }}
                  </UBadge>
                </span>
                <span class="mt-1 block text-xs text-slate-500 dark:text-zinc-400">
                  #{{ item.id }} · {{ formatSiteDateTime(item.createdAt) }}
                </span>
              </button>
            </nav>

            <div v-if="operation" class="min-w-0 space-y-5">
              <section class="border-b border-slate-200 pb-5 dark:border-zinc-800">
                <div class="flex flex-wrap items-center gap-2">
                  <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                    {{ t(`admin.extensions.lifecycle.operations.${operation.operation}`) }} #{{ operation.id }}
                  </h3>
                  <UBadge :color="operationStatusColor(operation)" variant="subtle">
                    {{ t(`admin.extensions.lifecycle.results.${operation.terminalResult || 'running'}`) }}
                  </UBadge>
                  <UBadge v-if="operation.forced" color="error" variant="outline" icon="i-lucide-shield-alert">
                    {{ t('admin.extensions.lifecycle.forced') }}
                  </UBadge>
                </div>
                <dl class="mt-3 grid gap-3 text-xs sm:grid-cols-2">
                  <div>
                    <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.lifecycle.version') }}</dt>
                    <dd class="mt-1 text-slate-900 dark:text-zinc-100">v{{ operation.extensionVersion }}</dd>
                  </div>
                  <div>
                    <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.lifecycle.attempts') }}</dt>
                    <dd class="mt-1 text-slate-900 dark:text-zinc-100">{{ operation.attemptCount }}</dd>
                  </div>
                  <div v-if="operation.removalMode">
                    <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.lifecycle.removalMode') }}</dt>
                    <dd class="mt-1 text-slate-900 dark:text-zinc-100">{{ t(`admin.extensions.uninstallModes.${operation.removalMode === 'export_then_remove' ? 'export' : operation.removalMode === 'complete_removal' ? 'complete' : 'preserve'}.label`) }}</dd>
                  </div>
                  <div v-if="operation.auditEventId">
                    <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.lifecycle.audit') }}</dt>
                    <dd class="mt-1 text-slate-900 dark:text-zinc-100">#{{ operation.auditEventId }}</dd>
                  </div>
                </dl>
                <UAlert
                  v-if="operation.error?.message"
                  class="mt-4"
                  color="error"
                  icon="i-lucide-circle-x"
                  variant="subtle"
                  :title="operation.error.message"
                />
              </section>

              <section>
                <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                  {{ t('admin.extensions.lifecycle.steps') }}
                </h3>
                <div class="mt-3 divide-y divide-slate-200 border-y border-slate-200 dark:divide-zinc-800 dark:border-zinc-800">
                  <div v-for="step in operation.steps" :key="`${step.id}-${step.attempt}`" class="py-3">
                    <div class="flex flex-wrap items-center justify-between gap-2">
                      <span class="break-all text-sm font-medium text-slate-900 dark:text-zinc-100">{{ step.lifecycleAction }}</span>
                      <UBadge :color="stepStatusColor(step.status)" variant="subtle" size="xs">
                        {{ t(`admin.extensions.lifecycle.stepResults.${step.status}`) }}
                      </UBadge>
                    </div>
                    <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
                      {{ t('admin.extensions.lifecycle.stepAttempt', { attempt: step.attempt, completed: step.completedUnits, total: step.totalUnits }) }}
                    </p>
                    <p v-if="step.skipReason" class="mt-1 text-xs leading-5 text-amber-700 dark:text-amber-300">
                      {{ step.skipReason }}
                    </p>
                    <p v-if="step.error?.message" class="mt-1 text-xs leading-5 text-red-600 dark:text-red-300">
                      {{ step.error.message }}
                    </p>
                  </div>
                </div>
              </section>

              <section v-if="operation.recoveries.length">
                <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                  {{ t('admin.extensions.lifecycle.recoveryHistory') }}
                </h3>
                <div class="mt-3 space-y-3">
                  <div v-for="item in operation.recoveries" :key="item.id" class="border-l-2 border-slate-300 pl-3 text-xs dark:border-zinc-700">
                    <div class="flex flex-wrap items-center gap-2 text-slate-900 dark:text-zinc-100">
                      <span class="font-medium">{{ t(`admin.extensions.lifecycle.recovery.${item.decision === 'skip_step' ? 'skip' : 'retry'}`) }}</span>
                      <UBadge v-if="item.escalateForced" color="error" variant="subtle" size="xs">
                        {{ t('admin.extensions.lifecycle.forced') }}
                      </UBadge>
                      <span class="text-slate-500 dark:text-zinc-400">#{{ item.auditEventId }} · {{ formatSiteDateTime(item.createdAt) }}</span>
                    </div>
                    <p v-if="item.reason" class="mt-1 leading-5 text-slate-600 dark:text-zinc-300">{{ item.reason }}</p>
                  </div>
                </div>
              </section>

              <section v-if="canRecover" class="border-t border-slate-200 pt-5 dark:border-zinc-800">
                <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                  {{ t('admin.extensions.lifecycle.recovery.title') }}
                </h3>
                <URadioGroup
                  v-model="decision"
                  class="mt-3"
                  :items="recoveryOptions"
                  value-key="value"
                  color="primary"
                />

                <UAlert
                  v-if="decision === 'skip_step'"
                  class="mt-4"
                  color="warning"
                  icon="i-lucide-skip-forward"
                  variant="subtle"
                  :title="t('admin.extensions.lifecycle.recovery.skipRisk')"
                />

                <div v-if="operation.operation === 'uninstall'" class="mt-4 border-t border-slate-200 pt-4 dark:border-zinc-800">
                  <UCheckbox
                    v-model="escalateForced"
                    :disabled="operation.forced || !isSuperAdmin"
                    :label="t('admin.extensions.lifecycle.recovery.forceLabel')"
                  />
                  <p v-if="!isSuperAdmin" class="mt-2 text-xs text-slate-500 dark:text-zinc-400">
                    {{ t('admin.extensions.lifecycle.recovery.forceSuperAdmin') }}
                  </p>
                  <UAlert
                    v-if="escalateForced"
                    class="mt-3"
                    color="error"
                    icon="i-lucide-shield-alert"
                    variant="subtle"
                    :title="t('admin.extensions.lifecycle.recovery.forceRiskTitle')"
                    :description="t('admin.extensions.lifecycle.recovery.forceRiskDescription')"
                  />
                  <UCheckbox
                    v-if="escalateForced"
                    v-model="residualRiskAcknowledged"
                    class="mt-3"
                    :label="t('admin.extensions.lifecycle.recovery.forceAcknowledge')"
                  />
                </div>

                <UFormField
                  class="mt-4"
                  :label="t('admin.extensions.lifecycle.recovery.reason')"
                  :description="t('admin.extensions.lifecycle.recovery.reasonDescription')"
                  :required="decision === 'skip_step' || escalateForced"
                >
                  <UTextarea v-model="reason" class="w-full" :rows="3" :maxlength="4096" />
                </UFormField>

                <div class="mt-4 flex justify-end">
                  <UButton
                    :color="escalateForced ? 'error' : 'primary'"
                    :icon="decision === 'retry' ? 'i-lucide-refresh-cw' : 'i-lucide-skip-forward'"
                    :loading="recoveryBusy"
                    :disabled="!canSubmitRecovery"
                    @click="submitRecovery"
                  >
                    {{ t(decision === 'retry' ? 'admin.extensions.lifecycle.recovery.retryAction' : 'admin.extensions.lifecycle.recovery.skipAction') }}
                  </UButton>
                </div>
              </section>
            </div>
          </div>
        </div>
      </div>
    </template>
  </UModal>
</template>
