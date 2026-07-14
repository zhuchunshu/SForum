<script setup lang="ts">
import {
  isLifecycleV2Plugin,
  type AdminExtension,
  type AdminLifecycleRemovalMode
} from '~/utils/adminExtensions'

const props = defineProps<{
  extension: AdminExtension | null
  busy?: boolean
  error?: string
}>()

const emit = defineEmits<{
  cancel: []
  confirm: []
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
            <p class="mt-1 text-sm leading-6 text-slate-600 dark:text-zinc-300">
              {{ lifecycleV2
                ? t('admin.extensions.confirmUninstallV2Body', { name: extension?.name || '' })
                : t('admin.extensions.confirmUninstallBody', { name: extension?.name || '' }) }}
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
          <UButton color="error" icon="i-lucide-trash-2" :loading="busy" @click="emit('confirm')">
            {{ t('admin.extensions.confirmUninstallAction') }}
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
