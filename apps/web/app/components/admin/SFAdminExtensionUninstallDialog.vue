<script setup lang="ts">
import {
  isLifecycleV2Plugin,
  type AdminExtension,
  type AdminLifecycleRemovalMode
} from '~/utils/admin/adminExtensions'

const props = defineProps<{
  extension: AdminExtension | null
  busy?: boolean
  error?: string
  cleanupDetails?: Record<string, unknown> | null
  finalizeBusy?: boolean
}>()

const emit = defineEmits<{
  cancel: []
  confirm: []
  finalize: []
}>()

const open = defineModel<boolean>('open', { required: true })
const removalMode = defineModel<AdminLifecycleRemovalMode>('removalMode', { required: true })
const { t } = useI18n()

const lifecycleV2 = computed(() => Boolean(props.extension && isLifecycleV2Plugin(props.extension)))
const removalOptions = computed(() => [
  {
    value: 'preserve' as const,
    label: t('admin.extensions.uninstallModes.preserve.label'),
    description: t('admin.extensions.uninstallModes.preserve.description')
  },
  {
    value: 'export_then_remove' as const,
    label: t('admin.extensions.uninstallModes.export.label'),
    description: t('admin.extensions.uninstallModes.export.description')
  },
  {
    value: 'complete_removal' as const,
    label: t('admin.extensions.uninstallModes.complete.label'),
    description: t('admin.extensions.uninstallModes.complete.description')
  }
])

const riskKey = computed(() => {
  if (removalMode.value === 'complete_removal') return 'admin.extensions.uninstallModes.complete.warning'
  if (removalMode.value === 'export_then_remove') return 'admin.extensions.uninstallModes.export.warning'
  return 'admin.extensions.uninstallModes.preserve.warning'
})

// 503 data.details.failedStep 的本地化展示；未知步骤回退到通用文案而非报错。
const cleanupStepLabel = computed(() => {
  const step = props.cleanupDetails?.failedStep
  if (typeof step !== 'string' || !step) return ''
  const key = `admin.extensions.uninstallCleanup.steps.${step}`
  const label = t(key)
  return label === key ? t('admin.extensions.uninstallCleanup.unknownStep') : label
})
const cleanupRetriable = computed(() => props.cleanupDetails?.retriable === true)
</script>

<template>
  <UModal v-model:open="open" :ui="{ content: 'sm:max-w-2xl' }">
    <template #content>
      <div class="p-5 sm:p-6">
        <div class="flex items-start gap-3">
          <div class="flex size-9 shrink-0 items-center justify-center rounded-md bg-red-50 text-red-600 dark:bg-red-950/40 dark:text-red-300">
            <UIcon name="i-lucide-trash-2" class="size-4" />
          </div>
          <div class="min-w-0">
            <h2 class="text-base font-semibold text-slate-900 dark:text-zinc-100">
              {{ t('admin.extensions.confirmUninstallTitle') }}
            </h2>
            <p class="mt-1 break-words text-sm leading-6 text-slate-600 dark:text-zinc-300">
              {{ lifecycleV2
                ? t('admin.extensions.confirmTrustedUninstallBody', { name: extension?.name?.trim() || extension?.id || '' })
                : t('admin.extensions.confirmUninstallBody', { name: extension?.name?.trim() || extension?.id || '' }) }}
            </p>
          </div>
        </div>

        <div v-if="lifecycleV2" class="mt-5 border-t border-slate-200 pt-5 dark:border-zinc-800">
          <URadioGroup
            v-model="removalMode"
            :legend="t('admin.extensions.uninstallModes.legend')"
            :items="removalOptions"
            value-key="value"
            color="primary"
          />
          <UAlert
            class="mt-4"
            :color="removalMode === 'complete_removal' ? 'error' : removalMode === 'export_then_remove' ? 'warning' : 'neutral'"
            :icon="removalMode === 'complete_removal' ? 'i-lucide-triangle-alert' : removalMode === 'export_then_remove' ? 'i-lucide-file-archive' : 'i-lucide-database-backup'"
            variant="subtle"
            :title="t(riskKey)"
          />
        </div>

        <UAlert
          v-if="extension?.pendingUninstallCleanup"
          class="mt-4"
          color="warning"
          icon="i-lucide-brush-cleaning"
          variant="subtle"
          :title="t('admin.extensions.uninstallCleanup.pendingBadge')"
        >
          <template #description>
            <div class="flex flex-col gap-2">
              <span>{{ t('admin.extensions.uninstallCleanup.pendingHint') }}</span>
              <UButton
                size="xs"
                color="warning"
                variant="outline"
                icon="i-lucide-check-circle-2"
                :disabled="busy || finalizeBusy"
                :loading="finalizeBusy"
                @click="emit('finalize')"
              >
                {{ t('admin.extensions.uninstallCleanup.finalize') }}
              </UButton>
            </div>
          </template>
        </UAlert>

        <UAlert
          v-if="error"
          class="mt-4"
          color="error"
          icon="i-lucide-triangle-alert"
          variant="subtle"
          :title="error"
        >
          <template v-if="cleanupStepLabel" #description>
            <div class="flex flex-col gap-2">
              <span>{{ t('admin.extensions.uninstallCleanup.failedStep', { step: cleanupStepLabel }) }}</span>
              <UButton
                v-if="cleanupRetriable"
                size="xs"
                color="error"
                variant="outline"
                icon="i-lucide-refresh-cw"
                :disabled="busy"
                :loading="busy"
                @click="emit('confirm')"
              >
                {{ t('admin.extensions.uninstallCleanup.retry') }}
              </UButton>
            </div>
          </template>
        </UAlert>

        <div class="mt-6 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <UButton color="neutral" variant="ghost" :disabled="busy" @click="emit('cancel')">
            {{ t('admin.extensions.confirmUninstallCancel') }}
          </UButton>
          <UButton color="error" icon="i-lucide-trash-2" :loading="busy" @click="emit('confirm')">
            {{ t('admin.extensions.confirmUninstallAction') }}
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
