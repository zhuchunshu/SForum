<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useSiteChromeApi, type SiteNavigationDocument, type SiteNavigationLocation } from '~/composables/admin/useSiteChromeApi'
import SFAdminNavigationDefaultsDialog from './SFAdminNavigationDefaultsDialog.vue'
import SFAdminNavigationHistory from './SFAdminNavigationHistory.vue'
import SFAdminNavigationImportDialog from './SFAdminNavigationImportDialog.vue'
import { navigationBackupFilename, serializeNavigationBackup } from '~/utils/admin/navigationBackup'

const props = withDefaults(defineProps<{
  revision: number
  activeLocation: SiteNavigationLocation
  locationOptions: Array<{ label: string, value: SiteNavigationLocation }>
  disabled?: boolean
  showHeading?: boolean
}>(), {
  disabled: false,
  showHeading: true
})
const emit = defineEmits<{ applied: [document: SiteNavigationDocument] }>()
const { t } = useI18n()
const toast = useToast()
const api = useSiteChromeApi()
const history = ref<{ refresh: () => Promise<void> } | null>(null)
const exporting = ref(false)
const exportError = ref('')

async function exportBackup() {
  exporting.value = true
  exportError.value = ''
  try {
    const backup = await api.exportNavigationBackup()
    const blob = new Blob([serializeNavigationBackup(backup)], { type: 'application/json;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = navigationBackupFilename(backup.exportedAt)
    anchor.click()
    URL.revokeObjectURL(url)
    toast.add({ color: 'primary', icon: 'i-lucide-download', title: t('admin.navigationEditor.recovery.exported'), duration: 10000 })
  } catch (cause) {
    exportError.value = apiErrorMessage(cause) || t('admin.navigationEditor.recovery.exportFailed')
  } finally {
    exporting.value = false
  }
}

async function applied(document: SiteNavigationDocument, workflow: 'defaults' | 'import' | 'history') {
  emit('applied', document)
  if (workflow !== 'history') {
    toast.add({ color: 'primary', icon: 'i-lucide-check', title: t(`admin.navigationEditor.recovery.${workflow}.applied`), duration: 10000 })
  }
  await nextTick()
  await history.value?.refresh()
}
</script>

<template>
  <section :aria-label="t('admin.navigationEditor.recovery.title')">
    <div class="flex flex-wrap items-start gap-3" :class="showHeading ? 'justify-between' : 'justify-end'">
      <div v-if="showHeading"><h3 class="text-sm font-semibold">{{ t('admin.navigationEditor.recovery.title') }}</h3><p class="mt-1 text-xs text-muted">{{ t('admin.navigationEditor.recovery.description') }}</p></div>
      <div class="flex flex-wrap gap-2">
        <SFAdminNavigationDefaultsDialog :revision="revision" :active-location="activeLocation" :location-options="locationOptions" :disabled="disabled" @applied="document => applied(document, 'defaults')" />
        <UButton color="neutral" variant="outline" icon="i-lucide-download" :loading="exporting" :disabled="disabled" @click="exportBackup">{{ t('admin.navigationEditor.recovery.export') }}</UButton>
        <SFAdminNavigationImportDialog :revision="revision" :location-options="locationOptions" :disabled="disabled" @applied="document => applied(document, 'import')" />
      </div>
    </div>
    <UAlert v-if="disabled" class="mt-3" color="warning" variant="soft" icon="i-lucide-file-pen-line" :title="t('admin.navigationEditor.recovery.saveDraftFirst')" />
    <UAlert v-if="exportError" class="mt-3" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="exportError" />
    <div class="mt-5"><SFAdminNavigationHistory ref="history" :revision="revision" :location-options="locationOptions" :disabled="disabled" @applied="document => applied(document, 'history')" /></div>
  </section>
</template>
