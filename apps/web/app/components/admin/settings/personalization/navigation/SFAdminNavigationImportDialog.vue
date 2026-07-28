<script setup lang="ts">
import { apiErrorMessage, apiErrorStatusCode } from '~/composables/useApiClient'
import {
  useSiteChromeApi,
  type SiteNavigationBackup,
  type SiteNavigationDocument,
  type SiteNavigationImportMode,
  type SiteNavigationLocation,
  type SiteNavigationPreview
} from '~/composables/admin/useSiteChromeApi'
import { NavigationBackupError, navigationBackupMaxBytes, parseNavigationBackup } from '~/utils/admin/navigationBackup'

const props = defineProps<{
  revision: number
  locationOptions: Array<{ label: string, value: SiteNavigationLocation }>
  disabled?: boolean
}>()
const emit = defineEmits<{ applied: [document: SiteNavigationDocument] }>()
const { t } = useI18n()
const api = useSiteChromeApi()
const open = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const fileName = ref('')
const backup = ref<SiteNavigationBackup | null>(null)
const mode = ref<SiteNavigationImportMode>('merge')
const preview = ref<SiteNavigationPreview | null>(null)
const confirmed = ref(false)
const pending = ref(false)
const error = ref('')
const previewChanges = computed(() => preview.value?.changeEntries?.length
  ? preview.value.changeEntries
  : (preview.value?.changes || []).map(change => ({ kind: change === 'definitions' ? 'definitions' as const : 'location' as const, location: change === 'definitions' ? undefined : change as SiteNavigationLocation, beforeCount: -1, afterCount: -1 })))

watch(open, reset)
watch(mode, () => {
  preview.value = null
  confirmed.value = false
  error.value = ''
})

async function chooseFile(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  backup.value = null
  preview.value = null
  confirmed.value = false
  error.value = ''
  fileName.value = file?.name || ''
  if (!file) return
  if (file.size > navigationBackupMaxBytes) {
    error.value = t('admin.navigationEditor.recovery.import.errors.oversized')
    return
  }
  try {
    backup.value = parseNavigationBackup(await file.text(), file.size)
  } catch (cause) {
    error.value = cause instanceof NavigationBackupError
      ? t(`admin.navigationEditor.recovery.import.errors.${cause.reason}`)
      : t('admin.navigationEditor.recovery.import.errors.malformed')
  }
}

async function createPreview() {
  if (!backup.value) return
  pending.value = true
  error.value = ''
  confirmed.value = false
  try {
    preview.value = await api.previewNavigationImport({ expectedRevision: props.revision, mode: mode.value, backup: backup.value })
  } catch (cause) {
    error.value = apiErrorStatusCode(cause) === 409
      ? t('admin.navigationEditor.recovery.staleRevision')
      : apiErrorMessage(cause) || t('admin.navigationEditor.recovery.import.previewFailed')
  } finally {
    pending.value = false
  }
}

async function applyPreview() {
  if (!preview.value || !confirmed.value) return
  pending.value = true
  error.value = ''
  try {
    const document = await api.applyNavigationImport({
      expectedRevision: preview.value.expectedRevision,
      previewToken: preview.value.previewToken,
      reason: `operator_import_${mode.value}`
    })
    emit('applied', document)
    open.value = false
  } catch (cause) {
    error.value = apiErrorStatusCode(cause) === 409
      ? t('admin.navigationEditor.recovery.stalePreview')
      : apiErrorMessage(cause) || t('admin.navigationEditor.recovery.import.applyFailed')
    if (apiErrorStatusCode(cause) === 409) preview.value = null
  } finally {
    pending.value = false
  }
}

function reset(value: boolean) {
  if (value) return
  fileName.value = ''
  backup.value = null
  mode.value = 'merge'
  preview.value = null
  confirmed.value = false
  pending.value = false
  error.value = ''
  if (fileInput.value) fileInput.value.value = ''
}

function locationLabel(change: string) {
  return props.locationOptions.find(item => item.value === change)?.label || change
}

function warningLabel(code: string, extensionId?: string, sourceKey?: string) {
  if (code === 'extension_reference_inert') return t('admin.navigationEditor.recovery.import.extensionInert', { extension: extensionId || sourceKey || t('admin.navigationEditor.recovery.import.unknownExtension') })
  return code
}

function show() { open.value = true }
function close() { open.value = false }
</script>

<template>
  <UButton color="neutral" variant="outline" icon="i-lucide-upload" :disabled="disabled" @click="show">{{ t('admin.navigationEditor.recovery.import.action') }}</UButton>
  <UModal v-model:open="open" :ui="{ content: 'sm:max-w-2xl' }">
    <template #content>
      <div class="max-h-[85vh] overflow-y-auto p-5">
        <div class="flex items-start justify-between gap-3"><div><h4 class="text-base font-semibold">{{ t('admin.navigationEditor.recovery.import.title') }}</h4><p class="mt-1 text-sm text-muted">{{ t('admin.navigationEditor.recovery.import.description') }}</p></div><UButton color="neutral" variant="ghost" icon="i-lucide-x" :aria-label="t('common.close')" :title="t('common.close')" @click="close" /></div>
        <div class="mt-4 grid gap-4">
          <div><input ref="fileInput" class="hidden" type="file" accept="application/json,.json" @change="chooseFile"><UButton color="neutral" variant="outline" icon="i-lucide-file-json" @click="fileInput?.click()">{{ t('admin.navigationEditor.recovery.import.chooseFile') }}</UButton><p class="mt-2 break-all text-xs text-muted">{{ fileName || t('admin.navigationEditor.recovery.import.noFile') }}</p></div>
          <UFormField :label="t('admin.navigationEditor.recovery.import.mode')"><URadioGroup v-model="mode" orientation="horizontal" :items="[{ label: t('admin.navigationEditor.recovery.import.merge'), value: 'merge' }, { label: t('admin.navigationEditor.recovery.import.replace'), value: 'replace' }]" /></UFormField>
          <UAlert :color="mode === 'replace' ? 'warning' : 'neutral'" variant="soft" :icon="mode === 'replace' ? 'i-lucide-triangle-alert' : 'i-lucide-info'" :title="t(`admin.navigationEditor.recovery.import.${mode}Title`)" :description="t(`admin.navigationEditor.recovery.import.${mode}Body`)" />
          <UButton class="justify-self-start" icon="i-lucide-scan-search" :loading="pending && !preview" :disabled="!backup" @click="createPreview">{{ t('admin.navigationEditor.recovery.preview') }}</UButton>
        </div>
        <UAlert v-if="error" class="mt-4" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="error" />
        <div v-if="preview" class="mt-4 border border-slate-200 p-4 dark:border-zinc-800">
          <h5 class="text-sm font-semibold">{{ t('admin.navigationEditor.recovery.previewTitle') }}</h5>
          <p v-if="!previewChanges.length" class="mt-2 text-sm text-muted">{{ t('admin.navigationEditor.recovery.noChanges') }}</p>
          <ul v-else class="mt-2 grid gap-1 text-sm"><li v-for="change in previewChanges" :key="`${change.kind}:${change.location || ''}`"><span>{{ change.kind === 'definitions' ? t('admin.navigationEditor.recovery.definitions') : locationLabel(change.location || '') }}</span><span v-if="change.beforeCount >= 0" class="text-muted"> · {{ change.beforeCount }} → {{ change.afterCount }}</span></li></ul>
          <UAlert v-if="preview.warningEntries?.length || preview.warnings.length" class="mt-4" color="warning" variant="soft" icon="i-lucide-plug-zap" :title="t('admin.navigationEditor.recovery.import.warningTitle')"><template #description><ul class="mt-1 grid gap-1"><template v-if="preview.warningEntries?.length"><li v-for="warning in preview.warningEntries" :key="`${warning.code}:${warning.sourceKey || ''}`" class="break-words">{{ warningLabel(warning.code, warning.extensionId, warning.sourceKey) }}</li></template><template v-else><li v-for="warning in preview.warnings" :key="warning" class="break-words">{{ warning }}</li></template></ul></template></UAlert>
          <UAlert class="mt-4" color="neutral" variant="soft" icon="i-lucide-shield-check" :title="t('admin.navigationEditor.recovery.preservedTitle')" :description="t('admin.navigationEditor.recovery.preservedBody')" />
          <UCheckbox v-model="confirmed" class="mt-4" :label="t(`admin.navigationEditor.recovery.import.${mode}Confirm`)" />
          <div class="mt-4 flex justify-end"><UButton :color="mode === 'replace' ? 'error' : 'primary'" icon="i-lucide-upload" :loading="pending" :disabled="!confirmed" @click="applyPreview">{{ t('admin.navigationEditor.recovery.import.apply') }}</UButton></div>
        </div>
      </div>
    </template>
  </UModal>
</template>
