<script setup lang="ts">
import { apiErrorMessage, apiErrorStatusCode } from '~/composables/useApiClient'
import {
  useSiteChromeApi,
  type SiteNavigationDocument,
  type SiteNavigationLocation,
  type SiteNavigationPreview
} from '~/composables/admin/useSiteChromeApi'

const props = defineProps<{
  revision: number
  activeLocation: SiteNavigationLocation
  locationOptions: Array<{ label: string, value: SiteNavigationLocation }>
  disabled?: boolean
}>()
const emit = defineEmits<{ applied: [document: SiteNavigationDocument] }>()
const { t } = useI18n()
const api = useSiteChromeApi()
const open = ref(false)
const scope = ref<'location' | 'all'>('location')
const location = ref<SiteNavigationLocation>(props.activeLocation)
const preview = ref<SiteNavigationPreview | null>(null)
const confirmed = ref(false)
const pending = ref(false)
const error = ref('')
const previewChanges = computed(() => preview.value?.changeEntries?.length
  ? preview.value.changeEntries
  : (preview.value?.changes || []).map(change => ({ kind: change === 'definitions' ? 'definitions' as const : 'location' as const, location: change === 'definitions' ? undefined : change as SiteNavigationLocation, beforeCount: -1, afterCount: -1 })))

watch(open, value => {
  if (value) location.value = props.activeLocation
  resetPreview()
})
watch([scope, location], resetPreview)

async function createPreview() {
  pending.value = true
  error.value = ''
  confirmed.value = false
  try {
    preview.value = await api.previewNavigationDefaults({
      expectedRevision: props.revision,
      scope: scope.value,
      ...(scope.value === 'location' ? { location: location.value } : {})
    })
  } catch (cause) {
    error.value = workflowError(cause, 'previewFailed')
  } finally {
    pending.value = false
  }
}

async function applyPreview() {
  if (!preview.value || !confirmed.value) return
  pending.value = true
  error.value = ''
  try {
    const document = await api.applyNavigationDefaults({
      expectedRevision: preview.value.expectedRevision,
      previewToken: preview.value.previewToken,
      reason: scope.value === 'all' ? 'operator_restore_all_defaults' : `operator_restore_defaults:${location.value}`
    })
    emit('applied', document)
    open.value = false
  } catch (cause) {
    error.value = workflowError(cause, 'applyFailed')
    if (apiErrorStatusCode(cause) === 409) preview.value = null
  } finally {
    pending.value = false
  }
}

function resetPreview() {
  preview.value = null
  confirmed.value = false
  error.value = ''
}

function workflowError(cause: unknown, fallback: string) {
  return apiErrorStatusCode(cause) === 409
    ? t('admin.navigationEditor.recovery.stalePreview')
    : apiErrorMessage(cause) || t(`admin.navigationEditor.recovery.defaults.${fallback}`)
}

function show() { open.value = true }
function close() { open.value = false }
</script>

<template>
  <UButton color="neutral" variant="outline" icon="i-lucide-rotate-ccw" :disabled="disabled" @click="show">
    {{ t('admin.navigationEditor.recovery.defaults.action') }}
  </UButton>

  <UModal v-model:open="open" :ui="{ content: 'sm:max-w-2xl' }">
    <template #content>
      <div class="max-h-[85vh] overflow-y-auto p-5">
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0"><h4 class="text-base font-semibold">{{ t('admin.navigationEditor.recovery.defaults.title') }}</h4><p class="mt-1 text-sm text-muted">{{ t('admin.navigationEditor.recovery.defaults.description') }}</p></div>
          <UButton color="neutral" variant="ghost" icon="i-lucide-x" :aria-label="t('common.close')" :title="t('common.close')" @click="close" />
        </div>

        <UAlert class="mt-4" color="warning" variant="soft" icon="i-lucide-shield-check" :title="t('admin.navigationEditor.recovery.preservedTitle')" :description="t('admin.navigationEditor.recovery.preservedBody')" />
        <div class="mt-4 grid gap-3">
          <UFormField :label="t('admin.navigationEditor.recovery.defaults.scope')">
            <URadioGroup v-model="scope" orientation="horizontal" :items="[{ label: t('admin.navigationEditor.recovery.defaults.oneLocation'), value: 'location' }, { label: t('admin.navigationEditor.recovery.defaults.allLocations'), value: 'all' }]" />
          </UFormField>
          <UFormField v-if="scope === 'location'" :label="t('admin.navigationEditor.locationLabel')">
            <USelect v-model="location" :items="locationOptions" />
          </UFormField>
          <UButton class="justify-self-start" icon="i-lucide-scan-search" :loading="pending && !preview" @click="createPreview">{{ t('admin.navigationEditor.recovery.preview') }}</UButton>
        </div>

        <UAlert v-if="error" class="mt-4" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="error" />
        <div v-if="preview" class="mt-4 border border-slate-200 p-4 dark:border-zinc-800">
          <h5 class="text-sm font-semibold">{{ t('admin.navigationEditor.recovery.previewTitle') }}</h5>
          <p v-if="!previewChanges.length" class="mt-2 text-sm text-muted">{{ t('admin.navigationEditor.recovery.noChanges') }}</p>
          <ul v-else class="mt-2 grid gap-1 text-sm"><li v-for="change in previewChanges" :key="`${change.kind}:${change.location || ''}`"><span>{{ change.kind === 'definitions' ? t('admin.navigationEditor.recovery.definitions') : locationOptions.find(item => item.value === change.location)?.label || change.location }}</span><span v-if="change.beforeCount >= 0" class="text-muted"> · {{ change.beforeCount }} → {{ change.afterCount }}</span></li></ul>
          <UCheckbox v-model="confirmed" class="mt-4" :label="t('admin.navigationEditor.recovery.defaults.confirm')" />
          <div class="mt-4 flex justify-end"><UButton color="error" icon="i-lucide-rotate-ccw" :loading="pending" :disabled="!confirmed" @click="applyPreview">{{ t('admin.navigationEditor.recovery.defaults.apply') }}</UButton></div>
        </div>
      </div>
    </template>
  </UModal>
</template>
