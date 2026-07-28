<script setup lang="ts">
import { onBeforeRouteLeave } from 'vue-router'
import { apiErrorMessage } from '~/composables/useApiClient'
import {
  useSiteChromeApi,
  type SiteNavigationDefinition,
  type SiteNavigationDocument,
  type SiteNavigationLinkKind,
  type SiteNavigationLocation,
  type SiteNavigationVisibility
} from '~/composables/admin/useSiteChromeApi'
import SFAdminNavigationLocationList from '~/components/admin/settings/personalization/SFAdminNavigationLocationList.vue'
import SFAdminNavigationRecoveryPanel from '~/components/admin/settings/personalization/navigation/SFAdminNavigationRecoveryPanel.vue'
import SFAdminFixedTabNav from '~/components/admin/settings/shared/SFAdminFixedTabNav.vue'
import {
  cloneNavigationDocument,
  moveNavigationItem,
  navigationDocumentsEqual,
  navigationItemsAt,
  navigationLocations,
  removeNavigationDefinition,
  reorderNavigationLocation,
  transferNavigationItem
} from '~/utils/admin/navigationDocument'

const { t } = useI18n()
const toast = useToast()
const api = useSiteChromeApi()
const loading = ref(false)
const saving = ref(false)
const persistentError = ref('')
const document = ref<SiteNavigationDocument | null>(null)
const baseline = ref<SiteNavigationDocument | null>(null)
const activeSection = ref<'navigation' | 'recovery'>('navigation')
const activeLocation = ref<SiteNavigationLocation>('public.topbar.primary')
const editorModalOpen = ref(false)
const editingKey = ref<string | null>(null)
const formError = ref('')
const form = reactive({ labelZhCN: '', labelEnUS: '', href: '/', icon: '', iconHidden: false, maxItems: 0, openInNewTab: false, visibility: 'public' as SiteNavigationVisibility, permission: '', linkKind: 'internalLink' as SiteNavigationLinkKind })

const locationLabelKeys: Record<SiteNavigationLocation, string> = {
  'public.topbar.primary': 'topbar',
  'public.sidebar.primary': 'sidebar',
  'public.mobile.primary': 'mobile',
  'public.footer.primary': 'footer'
}
const locationOptions = computed(() => navigationLocations.map(location => ({ value: location, label: t(`admin.navigationEditor.locations.${locationLabelKeys[location]}`) })))
const sectionOptions = computed(() => [
  { id: 'navigation', label: t('admin.navigationEditor.title'), icon: 'i-lucide-menu' },
  { id: 'recovery', label: t('admin.navigationEditor.recovery.title'), icon: 'i-lucide-history' }
])
const dirty = computed(() => !navigationDocumentsEqual(document.value, baseline.value))
const selectedItems = computed(() => document.value ? navigationItemsAt(document.value, activeLocation.value) : [])
const activeThemeSupported = computed(() => document.value?.themeLocations.find(item => item.location === activeLocation.value)?.supported !== false)
const editingCore = computed(() => document.value?.definitions.find(item => item.sourceKey === editingKey.value)?.sourceKind === 'core')
const editingDynamic = computed(() => document.value?.definitions.find(item => item.sourceKey === editingKey.value)?.sourceKind === 'dynamic')
const editingBuiltIn = computed(() => editingCore.value || editingDynamic.value)
const editorHintKey = computed(() => editingDynamic.value ? 'admin.navigationEditor.dynamicFormHint' : editingCore.value ? 'admin.navigationEditor.coreFormHint' : 'admin.navigationEditor.formHint')

defineExpose({ refresh: load, loading })
onMounted(load)

watch(() => form.linkKind, kind => {
  if (kind === 'internalLink' && !form.href.startsWith('/')) form.href = '/'
  if (kind === 'externalLink' && !/^https?:\/\//.test(form.href)) form.href = 'https://'
})

onBeforeRouteLeave(() => !dirty.value || window.confirm(t('admin.navigationEditor.leaveConfirm')))
onMounted(() => window.addEventListener('beforeunload', warnBeforeUnload))
onBeforeUnmount(() => window.removeEventListener('beforeunload', warnBeforeUnload))

async function load() {
  loading.value = true
  persistentError.value = ''
  try {
    const loaded = await api.getAdminNavigation()
    document.value = cloneNavigationDocument(loaded)
    baseline.value = cloneNavigationDocument(loaded)
    editorModalOpen.value = false
    resetForm()
  } catch (error) {
    persistentError.value = apiErrorMessage(error) || t('admin.navigationEditor.loadFailed')
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!document.value || saving.value) return
  saving.value = true
  persistentError.value = ''
  try {
    const saved = await api.applyAdminNavigation({ expectedRevision: document.value.revision, reason: 'operator_editor_save', document: document.value })
    document.value = cloneNavigationDocument(saved)
    baseline.value = cloneNavigationDocument(saved)
    clearNuxtData(key => key.startsWith('site-public-navigation:'))
    toast.add({ color: 'primary', icon: 'i-lucide-check', title: t('admin.navigationEditor.saved'), duration: 10000 })
  } catch (error) {
    persistentError.value = isConflict(error) ? t('admin.navigationEditor.stale') : apiErrorMessage(error) || t('admin.navigationEditor.saveFailed')
  } finally {
    saving.value = false
  }
}

function createOrUpdate() {
  if (!document.value) return
  formError.value = validateForm()
  if (formError.value) return
  if (editingKey.value) {
    const definition = document.value.definitions.find(item => item.sourceKey === editingKey.value)
    const placement = document.value.placements.find(item => item.sourceKey === editingKey.value && item.location === activeLocation.value)
    if (!definition || !placement) return
    if (definition.sourceKind === 'operator') {
      Object.assign(definition, normalizedForm())
    } else if (definition.sourceKind === 'core' || definition.sourceKind === 'dynamic') {
      Object.assign(placement, {
        labelZhCN: presentationOverride(form.labelZhCN, definition.labelZhCN),
        labelEnUS: presentationOverride(form.labelEnUS, definition.labelEnUS),
        icon: form.iconHidden ? '' : presentationOverride(form.icon, definition.icon),
        iconHidden: form.iconHidden,
        maxItems: definition.sourceKind === 'dynamic' ? form.maxItems : 0
      })
    } else {
      return
    }
    Object.assign(placement, { visibility: form.visibility, permission: form.visibility === 'permission' ? form.permission.trim() : '' })
  } else {
    const definition: SiteNavigationDefinition = { sourceKey: operatorKey(), sourceKind: 'operator', ...normalizedForm() }
    document.value.definitions.push(definition)
    document.value.placements.push({ sourceKey: definition.sourceKey, location: activeLocation.value, order: (selectedItems.value.length + 1) * 10, enabled: true, visibility: form.visibility, permission: form.visibility === 'permission' ? form.permission.trim() : '' })
  }
  editorModalOpen.value = false
  resetForm()
}

function beginCreate() {
  resetForm()
  editorModalOpen.value = true
}

function beginEdit(sourceKey: string) {
  const definition = document.value?.definitions.find(item => item.sourceKey === sourceKey)
  const placement = document.value?.placements.find(item => item.sourceKey === sourceKey && item.location === activeLocation.value)
  if (!definition || !['operator', 'core', 'dynamic'].includes(definition.sourceKind) || !placement) return
  editingKey.value = sourceKey
  Object.assign(form, {
    labelZhCN: placement.labelZhCN || definition.labelZhCN || '',
    labelEnUS: placement.labelEnUS || definition.labelEnUS || '',
    href: definition.href || '',
    icon: placement.icon || definition.icon || '',
    iconHidden: Boolean(placement.iconHidden),
    maxItems: placement.maxItems || 0,
    openInNewTab: Boolean(definition.openInNewTab),
    linkKind: definition.linkKind,
    visibility: placement.visibility,
    permission: placement.permission || ''
  })
  formError.value = ''
  editorModalOpen.value = true
}

function restoreBuiltInDefaults() {
  const definition = document.value?.definitions.find(item => item.sourceKey === editingKey.value)
  if (!definition || !['core', 'dynamic'].includes(definition.sourceKind)) return
  Object.assign(form, {
    labelZhCN: definition.labelZhCN || '',
    labelEnUS: definition.labelEnUS || '',
    icon: definition.icon || '',
    iconHidden: false,
    maxItems: 0,
    visibility: 'public',
    permission: ''
  })
  formError.value = ''
}

function closeEditor() {
  editorModalOpen.value = false
  resetForm()
}

function remove(sourceKey: string) {
  if (!document.value || !window.confirm(t('admin.navigationEditor.deleteConfirm'))) return
  removeNavigationDefinition(document.value, sourceKey)
  if (editingKey.value === sourceKey) resetForm()
}

function toggle(sourceKey: string) {
  const placement = document.value?.placements.find(item => item.sourceKey === sourceKey && item.location === activeLocation.value)
  if (placement) placement.enabled = !placement.enabled
}

function reorder(sourceKeys: string[]) {
  if (document.value) reorderNavigationLocation(document.value, activeLocation.value, sourceKeys)
}

function move(sourceKey: string, index: number) {
  if (document.value) moveNavigationItem(document.value, activeLocation.value, sourceKey, index)
}

function transfer(sourceKey: string, target: SiteNavigationLocation, copy: boolean) {
  if (!document.value) return
  transferNavigationItem(document.value, sourceKey, activeLocation.value, target, copy)
}

function resetForm() {
  editingKey.value = null
  formError.value = ''
  Object.assign(form, { labelZhCN: '', labelEnUS: '', href: '/', icon: '', iconHidden: false, maxItems: 0, openInNewTab: false, visibility: 'public', permission: '', linkKind: 'internalLink' })
}

function normalizedForm() {
  return { labelZhCN: form.labelZhCN.trim(), labelEnUS: form.labelEnUS.trim(), href: form.href.trim(), icon: form.icon.trim(), openInNewTab: form.openInNewTab, linkKind: form.linkKind }
}

function presentationOverride(value: string, defaultValue?: string) {
  const normalized = value.trim()
  return normalized === (defaultValue || '').trim() ? '' : normalized
}

function validateForm() {
  if (!form.labelZhCN.trim() && !form.labelEnUS.trim()) return t('admin.navigationEditor.validation.label')
  if (form.linkKind === 'internalLink' && (!form.href.startsWith('/') || form.href.startsWith('//') || form.href.startsWith('/api') || form.href.startsWith('/admin'))) return t('admin.navigationEditor.validation.internal')
  if (form.linkKind === 'externalLink' && !/^https?:\/\/[^\s/$.?#].[^\s]*$/i.test(form.href.trim())) return t('admin.navigationEditor.validation.external')
  if (form.visibility === 'permission' && !form.permission.trim()) return t('admin.navigationEditor.validation.permission')
  if (editingDynamic.value && (!Number.isInteger(form.maxItems) || form.maxItems < 0 || form.maxItems > 100)) return t('admin.navigationEditor.validation.maxItems')
  return ''
}

function operatorKey() {
  const random = globalThis.crypto?.randomUUID?.().replaceAll('-', '') || `${Date.now()}${Math.random().toString(16).slice(2)}`
  return `operator.${random.slice(0, 48)}`
}

function warnBeforeUnload(event: BeforeUnloadEvent) {
  if (!dirty.value) return
  event.preventDefault()
  event.returnValue = ''
}

function isConflict(error: unknown) {
  const status = Number((error as { statusCode?: number, status?: number })?.statusCode || (error as { status?: number })?.status || 0)
  return status === 409 || String(apiErrorMessage(error)).includes('site_chrome.conflict')
}

function acceptRecoveryDocument(recovered: SiteNavigationDocument) {
  document.value = cloneNavigationDocument(recovered)
  baseline.value = cloneNavigationDocument(recovered)
  editingKey.value = null
  resetForm()
}

function selectLocation(location: SiteNavigationLocation) {
  activeLocation.value = location
}

function selectSection(section: string) {
  if (section === 'navigation' || section === 'recovery') activeSection.value = section
}
</script>

<template>
  <div class="min-w-0">
    <SFAdminFixedTabNav :items="sectionOptions" :model-value="activeSection" :aria-label="t('admin.navigationEditor.sectionLabel')" @update:model-value="selectSection" />

    <UCard v-if="activeSection === 'navigation'" class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <template #header>
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div><h3 class="text-base font-bold">{{ t('admin.navigationEditor.title') }}</h3><p class="mt-1 text-xs text-muted">{{ t('admin.navigationEditor.description') }}</p></div>
          <UBadge color="neutral" variant="subtle">{{ document?.revision ? `${t('admin.navigationEditor.revision')} ${document.revision}` : t('admin.common.loading') }}</UBadge>
        </div>
      </template>

      <div v-if="persistentError" class="mb-4 flex flex-wrap items-center justify-between gap-3"><UAlert class="min-w-0 flex-1" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="persistentError" /><UButton color="neutral" variant="outline" icon="i-lucide-refresh-cw" :aria-label="t('admin.navigationEditor.reload')" :title="t('admin.navigationEditor.reload')" @click="load" /></div>
      <div v-if="loading" class="py-12 text-center text-sm text-muted">{{ t('admin.common.loading') }}</div>
      <template v-else-if="document">
        <div class="mb-4 flex flex-wrap gap-2" role="tablist" :aria-label="t('admin.navigationEditor.locationLabel')">
          <UButton v-for="option in locationOptions" :key="option.value" role="tab" :aria-selected="activeLocation === option.value" :color="activeLocation === option.value ? 'primary' : 'neutral'" :variant="activeLocation === option.value ? 'solid' : 'outline'" @click="selectLocation(option.value)">{{ option.label }}</UButton>
        </div>

        <UAlert v-if="!activeThemeSupported" class="mb-4" color="warning" variant="soft" icon="i-lucide-eye-off" :title="t('admin.navigationEditor.unsupportedTitle')" :description="t('admin.navigationEditor.unsupportedBody')" />
        <SFAdminNavigationLocationList :location="activeLocation" :label="locationOptions.find(option => option.value === activeLocation)?.label || activeLocation" :supported="activeThemeSupported" :items="selectedItems" :locations="locationOptions" @create="beginCreate" @reorder="reorder" @move="move" @edit="beginEdit" @remove="remove" @toggle="toggle" @transfer="transfer" />
        <div class="mt-4 border-t border-slate-200 pt-4 dark:border-zinc-800"><p v-if="dirty" class="mb-3 text-sm text-amber-700 dark:text-amber-300">{{ t('admin.navigationEditor.unsaved') }}</p><div class="flex flex-wrap justify-end gap-2"><UButton color="neutral" variant="outline" icon="i-lucide-rotate-ccw" :disabled="!dirty || saving" @click="load">{{ t('admin.navigationEditor.discard') }}</UButton><UButton icon="i-lucide-save" :loading="saving" :disabled="!dirty" @click="save">{{ t('admin.navigationEditor.save') }}</UButton></div></div>
      </template>
    </UCard>

    <UCard v-else class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <template #header>
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div><h3 class="text-base font-bold">{{ t('admin.navigationEditor.recovery.title') }}</h3><p class="mt-1 text-xs text-muted">{{ t('admin.navigationEditor.recovery.description') }}</p></div>
          <UBadge color="neutral" variant="subtle">{{ document?.revision ? `${t('admin.navigationEditor.revision')} ${document.revision}` : t('admin.common.loading') }}</UBadge>
        </div>
      </template>

      <div v-if="persistentError" class="mb-4 flex flex-wrap items-center justify-between gap-3"><UAlert class="min-w-0 flex-1" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="persistentError" /><UButton color="neutral" variant="outline" icon="i-lucide-refresh-cw" :aria-label="t('admin.navigationEditor.reload')" :title="t('admin.navigationEditor.reload')" @click="load" /></div>
      <div v-if="loading" class="py-12 text-center text-sm text-muted">{{ t('admin.common.loading') }}</div>
      <SFAdminNavigationRecoveryPanel v-else-if="document" :revision="document.revision" :active-location="activeLocation" :location-options="locationOptions" :disabled="dirty || saving" :show-heading="false" @applied="acceptRecoveryDocument" />
    </UCard>

    <UModal v-model:open="editorModalOpen" :ui="{ content: 'sm:max-w-3xl' }" @update:open="open => { if (!open) closeEditor() }">
      <template #content>
        <form class="flex max-h-[85vh] flex-col" @submit.prevent="createOrUpdate">
          <div class="flex items-start justify-between gap-3 border-b border-slate-200 px-5 py-4 dark:border-zinc-800">
            <div class="min-w-0"><h3 class="text-base font-bold text-slate-900 dark:text-white">{{ editingKey ? t('admin.navigationEditor.edit') : t('admin.navigationEditor.add') }}</h3><p class="mt-1 text-xs text-muted">{{ t(editorHintKey) }}</p></div>
            <UButton type="button" color="neutral" variant="ghost" icon="i-lucide-x" :aria-label="t('admin.navigationEditor.cancel')" :title="t('admin.navigationEditor.cancel')" @click="closeEditor" />
          </div>

          <div class="grid flex-1 gap-4 overflow-y-auto px-5 py-4">
            <UAlert v-if="formError" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="formError" />
            <div class="grid gap-4 md:grid-cols-2">
              <UFormField :label="t('admin.navigationEditor.labelZh')"><UInput v-model="form.labelZhCN" class="w-full" /></UFormField>
              <UFormField :label="t('admin.navigationEditor.labelEn')"><UInput v-model="form.labelEnUS" class="w-full" /></UFormField>
              <UFormField :label="t('admin.navigationEditor.targetType')"><USelect v-model="form.linkKind" class="w-full" :disabled="editingBuiltIn" :items="editingDynamic ? [{ label: t('admin.navigationEditor.dynamicBlock'), value: 'dynamicBlock' }] : editingCore ? [{ label: t('admin.navigationEditor.coreRoute'), value: 'coreRoute' }] : [{ label: t('admin.navigationEditor.internal'), value: 'internalLink' }, { label: t('admin.navigationEditor.external'), value: 'externalLink' }]" /></UFormField>
              <UFormField :label="t('admin.navigationEditor.href')"><UInput v-model="form.href" class="w-full" icon="i-lucide-link" :disabled="editingBuiltIn" /></UFormField>
            </div>
            <LazySFIconPicker v-model="form.icon" :label="t('admin.navigationEditor.iconPicker')" :hint="t('admin.navigationEditor.iconPickerHint')" :disabled="editingBuiltIn && form.iconHidden" :show-custom-input="false" />
            <UCheckbox v-if="editingBuiltIn" v-model="form.iconHidden" :label="t('admin.navigationEditor.hideIcon')" />
            <UFormField v-if="editingDynamic" :label="t('admin.navigationEditor.maxItems')">
              <UInputNumber v-model="form.maxItems" class="w-full" :min="0" :max="100" />
              <p class="mt-2 text-xs text-muted">{{ t('admin.navigationEditor.maxItemsHint') }}</p>
            </UFormField>
            <div class="grid gap-4 md:grid-cols-2">
              <UFormField :label="t('admin.navigationEditor.visibility')"><USelect v-model="form.visibility" class="w-full" :items="[{ label: t('admin.navigationEditor.public'), value: 'public' }, { label: t('admin.navigationEditor.anonymous'), value: 'anonymous' }, { label: t('admin.navigationEditor.authenticated'), value: 'authenticated' }, { label: t('admin.navigationEditor.permission'), value: 'permission' }]" /></UFormField>
              <UFormField v-if="form.visibility === 'permission'" :label="t('admin.navigationEditor.permissionKey')"><UInput v-model="form.permission" class="w-full" /></UFormField>
            </div>
            <UCheckbox v-model="form.openInNewTab" :disabled="editingBuiltIn" :label="t('admin.navigationEditor.newTab')" />
          </div>

          <div class="flex flex-wrap justify-end gap-2 border-t border-slate-200 px-5 py-4 dark:border-zinc-800">
            <UButton v-if="editingBuiltIn" type="button" class="mr-auto" color="neutral" variant="outline" icon="i-lucide-rotate-ccw" @click="restoreBuiltInDefaults">{{ t('admin.navigationEditor.restoreDefault') }}</UButton>
            <UButton type="button" color="neutral" variant="ghost" @click="closeEditor">{{ t('admin.navigationEditor.cancel') }}</UButton>
            <UButton type="submit" :icon="editingKey ? 'i-lucide-save' : 'i-lucide-plus'">{{ editingKey ? t('admin.navigationEditor.update') : t('admin.navigationEditor.add') }}</UButton>
          </div>
        </form>
      </template>
    </UModal>
  </div>
</template>
