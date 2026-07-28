<script setup lang="ts">
import { apiErrorMessage, apiErrorStatusCode } from '~/composables/useApiClient'
import {
  useSiteChromeApi,
  type SiteNavigationDocument,
  type SiteNavigationLocation,
  type SiteNavigationSnapshot
} from '~/composables/admin/useSiteChromeApi'

const props = defineProps<{
  revision: number
  locationOptions: Array<{ label: string, value: SiteNavigationLocation }>
  disabled?: boolean
}>()
const emit = defineEmits<{ applied: [document: SiteNavigationDocument] }>()
const { t, locale } = useI18n()
const toast = useToast()
const api = useSiteChromeApi()
const snapshots = ref<SiteNavigationSnapshot[]>([])
const selected = ref<SiteNavigationSnapshot | null>(null)
const loading = ref(false)
const detailLoading = ref(false)
const restoring = ref(false)
const error = ref('')
const detailError = ref('')
const confirmed = ref(false)
const reason = ref('')

defineExpose({ refresh: load })
onMounted(load)

async function load() {
  loading.value = true
  error.value = ''
  try {
    snapshots.value = await api.listNavigationSnapshots()
  } catch (cause) {
    error.value = apiErrorMessage(cause) || t('admin.navigationEditor.recovery.history.loadFailed')
  } finally {
    loading.value = false
  }
}

async function inspect(snapshot: SiteNavigationSnapshot) {
  detailLoading.value = true
  detailError.value = ''
  confirmed.value = false
  reason.value = ''
  selected.value = { ...snapshot, document: undefined }
  try {
    selected.value = await api.getNavigationSnapshot(snapshot.id)
  } catch (cause) {
    detailError.value = apiErrorMessage(cause) || t('admin.navigationEditor.recovery.history.detailFailed')
  } finally {
    detailLoading.value = false
  }
}

async function restore() {
  if (!selected.value || !confirmed.value) return
  restoring.value = true
  detailError.value = ''
  try {
    const document = await api.restoreNavigationSnapshot(selected.value.id, {
      expectedRevision: props.revision,
      reason: reason.value.trim() || `operator_restore_snapshot:${selected.value.id}`
    })
    emit('applied', document)
    toast.add({ color: 'primary', icon: 'i-lucide-history', title: t('admin.navigationEditor.recovery.history.restored'), duration: 10000 })
    selected.value = null
    await load()
  } catch (cause) {
    detailError.value = apiErrorStatusCode(cause) === 409
      ? t('admin.navigationEditor.recovery.staleRevision')
      : apiErrorMessage(cause) || t('admin.navigationEditor.recovery.history.restoreFailed')
  } finally {
    restoring.value = false
  }
}

function closeDetail() {
  if (!restoring.value) selected.value = null
}

function actorLabel(snapshot: SiteNavigationSnapshot) {
  return snapshot.actorUserId ? t('admin.navigationEditor.recovery.history.actor', { id: snapshot.actorUserId }) : t('admin.navigationEditor.recovery.history.actorUnknown')
}

function formatDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat(String(locale.value), { dateStyle: 'medium', timeStyle: 'short' }).format(date)
}

function locationLabel(location: SiteNavigationLocation) {
  return props.locationOptions.find(item => item.value === location)?.label || location
}
</script>

<template>
  <section aria-labelledby="navigation-history-title">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div><h4 id="navigation-history-title" class="text-sm font-semibold">{{ t('admin.navigationEditor.recovery.history.title') }}</h4><p class="mt-1 text-xs text-muted">{{ t('admin.navigationEditor.recovery.history.description') }}</p></div>
      <UButton color="neutral" variant="ghost" icon="i-lucide-refresh-cw" :loading="loading" :aria-label="t('admin.navigationEditor.recovery.history.refresh')" :title="t('admin.navigationEditor.recovery.history.refresh')" @click="load" />
    </div>
    <UAlert v-if="error" class="mt-3" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="error" />
    <p v-if="loading && !snapshots.length" class="mt-3 text-sm text-muted">{{ t('admin.common.loading') }}</p>
    <p v-else-if="!snapshots.length" class="mt-3 text-sm text-muted">{{ t('admin.navigationEditor.recovery.history.empty') }}</p>
    <ul v-else class="mt-3 divide-y divide-slate-200 border-y border-slate-200 dark:divide-zinc-800 dark:border-zinc-800">
      <li v-for="snapshot in snapshots" :key="snapshot.id" class="flex min-w-0 flex-wrap items-center justify-between gap-3 py-3">
        <div class="min-w-0"><p class="truncate text-sm font-medium">{{ t(`admin.navigationEditor.recovery.operations.${snapshot.operation}`, snapshot.operation) }}</p><p class="mt-1 text-xs text-muted">{{ formatDate(snapshot.createdAt) }} · {{ actorLabel(snapshot) }} · {{ t('admin.navigationEditor.recovery.history.revision', { revision: snapshot.revision }) }}</p><p v-if="snapshot.reason" class="mt-1 truncate text-xs text-muted">{{ snapshot.reason }}</p></div>
        <UButton color="neutral" variant="outline" icon="i-lucide-eye" :disabled="disabled" :aria-label="t('admin.navigationEditor.recovery.history.inspect')" :title="t('admin.navigationEditor.recovery.history.inspect')" @click="inspect(snapshot)" />
      </li>
    </ul>
  </section>

  <UModal :open="Boolean(selected)" :ui="{ content: 'sm:max-w-2xl' }" @update:open="value => !value && closeDetail()">
    <template #content>
      <div class="max-h-[85vh] overflow-y-auto p-5">
        <div class="flex items-start justify-between gap-3"><div><h4 class="text-base font-semibold">{{ t('admin.navigationEditor.recovery.history.previewTitle') }}</h4><p v-if="selected" class="mt-1 text-sm text-muted">{{ formatDate(selected.createdAt) }} · {{ actorLabel(selected) }}</p></div><UButton color="neutral" variant="ghost" icon="i-lucide-x" :disabled="restoring" :aria-label="t('common.close')" :title="t('common.close')" @click="closeDetail" /></div>
        <p v-if="detailLoading" class="mt-4 text-sm text-muted">{{ t('admin.common.loading') }}</p>
        <UAlert v-if="detailError" class="mt-4" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="detailError" />
        <template v-if="selected?.document">
          <dl class="mt-4 grid gap-3 text-sm sm:grid-cols-2"><div><dt class="text-muted">{{ t('admin.navigationEditor.recovery.history.definitions') }}</dt><dd class="font-medium">{{ selected.document.definitions.length }}</dd></div><div><dt class="text-muted">{{ t('admin.navigationEditor.recovery.history.placements') }}</dt><dd class="font-medium">{{ selected.document.placements.length }}</dd></div></dl>
          <div class="mt-4"><p class="text-sm font-medium">{{ t('admin.navigationEditor.recovery.history.affected') }}</p><div class="mt-2 flex flex-wrap gap-2"><UBadge v-for="location in selected.affectedLocations" :key="location" color="neutral" variant="subtle">{{ locationLabel(location) }}</UBadge><span v-if="!selected.affectedLocations.length" class="text-sm text-muted">{{ t('admin.navigationEditor.recovery.history.none') }}</span></div></div>
          <UAlert class="mt-4" color="warning" variant="soft" icon="i-lucide-shield-check" :title="t('admin.navigationEditor.recovery.preservedTitle')" :description="t('admin.navigationEditor.recovery.preservedBody')" />
          <UFormField class="mt-4" :label="t('admin.navigationEditor.recovery.history.reason')"><UInput v-model="reason" :maxlength="240" /></UFormField>
          <UCheckbox v-model="confirmed" class="mt-4" :label="t('admin.navigationEditor.recovery.history.confirm')" />
          <div class="mt-4 flex justify-end"><UButton color="error" icon="i-lucide-history" :loading="restoring" :disabled="!confirmed" @click="restore">{{ t('admin.navigationEditor.recovery.history.restore') }}</UButton></div>
        </template>
      </div>
    </template>
  </UModal>
</template>
