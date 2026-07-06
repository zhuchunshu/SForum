<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import {
  extensionAdminPageRoute,
  findExtensionAdminPage,
  normalizeExtensionPagePath,
  type AdminExtension,
  type AdminExtensionSettings
} from '~/utils/adminExtensions'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminExtensionDynamicPage'
})

const { t } = useI18n()
const route = useRoute()
const adminRoutes = useAdminRoutes()
const adminTabs = useAdminTabs()
const toast = useToast()
const { request } = useApiClient()

const extensionId = computed(() => {
  const value = route.params.extensionId
  return Array.isArray(value) ? value[0] || '' : `${value || ''}`
})
const currentPagePath = computed(() => normalizeExtensionPagePath(route.params.pagePath as string[] | string | undefined))

const {
  data: extensions,
  pending,
  error,
  refresh
} = await useAsyncData<AdminExtension[]>('admin-extension-dynamic-list', () => request<AdminExtension[]>('/admin/extensions'), {
  default: (): AdminExtension[] => []
})

const extension = computed(() => extensions.value.find(item => item.id === extensionId.value))
const adminPage = computed(() => extension.value ? findExtensionAdminPage(extension.value, currentPagePath.value) : undefined)
const settings = ref<AdminExtensionSettings | null>(null)
const formValues = reactive<Record<string, string>>({})
const loadingSettings = ref(false)
const savingSettings = ref(false)

const pageTitle = computed(() => adminPage.value?.label || extension.value?.name || t('admin.extensions.dynamic.notFoundTitle'))
const pageDescription = computed(() => adminPage.value?.description || extension.value?.manifest.description || '')
const isSettingsView = computed(() => adminPage.value?.view === 'settings')

useSeoMeta({
  title: pageTitle
})

watch([extension, adminPage], ([item, page]) => {
  if (!item || !page) {
    return
  }
  const id = extensionAdminPageRoute(item.id, page.path)
  adminTabs.openCustomTab({
    id,
    label: page.label,
    to: adminRoutes.path(id),
    icon: page.icon || (item.type === 'theme' ? 'i-lucide-palette' : 'i-lucide-plug'),
    closable: true,
    componentName: 'AdminExtensionDynamicPage'
  })
}, { immediate: true })

watch([extensionId, isSettingsView], async () => {
  if (isSettingsView.value) {
    await loadSettings()
  }
}, { immediate: true })

async function loadSettings() {
  if (!extensionId.value) {
    return
  }
  loadingSettings.value = true
  try {
    const next = await request<AdminExtensionSettings>(`/admin/extensions/${extensionId.value}/settings`)
    settings.value = next
    Object.keys(formValues).forEach(key => {
      delete formValues[key]
    })
    for (const item of next.items) {
      formValues[item.key] = item.value
    }
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.extensions.dynamic.settingsLoadFailed') })
  } finally {
    loadingSettings.value = false
  }
}

async function saveSettings() {
  savingSettings.value = true
  try {
    const updated = await request<AdminExtensionSettings>(`/admin/extensions/${extensionId.value}/settings`, {
      method: 'PUT',
      body: { values: { ...formValues } }
    })
    settings.value = updated
    for (const item of updated.items) {
      formValues[item.key] = item.value
    }
    toast.add({ color: 'success', icon: 'i-lucide-save', title: t('admin.extensions.dynamic.settingsSaved') })
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.extensions.dynamic.settingsSaveFailed') })
  } finally {
    savingSettings.value = false
  }
}

async function resetSettings() {
  savingSettings.value = true
  try {
    const updated = await request<AdminExtensionSettings>(`/admin/extensions/${extensionId.value}/settings/reset`, {
      method: 'POST',
      body: {}
    })
    settings.value = updated
    for (const item of updated.items) {
      formValues[item.key] = item.value
    }
    toast.add({ color: 'success', icon: 'i-lucide-rotate-ccw', title: t('admin.extensions.dynamic.settingsReset') })
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.extensions.dynamic.settingsSaveFailed') })
  } finally {
    savingSettings.value = false
  }
}

function updateBooleanSetting(key: string, checked: boolean) {
  formValues[key] = checked ? 'true' : 'false'
}

function onBooleanSettingChange(key: string, event: Event) {
  updateBooleanSetting(key, (event.target as HTMLInputElement | null)?.checked === true)
}
</script>

<template>
  <div class="mb-4 flex flex-col gap-1">
    <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage?.icon || 'i-lucide-blocks'" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ pageTitle }}
    </h2>
    <p v-if="pageDescription" class="text-sm text-slate-500 dark:text-zinc-400">
      {{ pageDescription }}
    </p>
  </div>

  <UAlert
    v-if="error"
    color="error"
    icon="i-lucide-triangle-alert"
    variant="subtle"
    :title="apiErrorMessage(error) || t('admin.extensions.loadFailed')"
    class="mb-6"
  />

  <div v-if="pending" class="rounded-lg border border-slate-200 bg-white p-8 text-sm text-slate-500 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400">
    {{ t('admin.extensions.dynamic.loading') }}
  </div>

  <div v-else-if="!extension || !adminPage" class="rounded-lg border border-slate-200 bg-white p-10 dark:border-zinc-800 dark:bg-zinc-900">
    <SFEmptyState icon-label="EXT" :title="t('admin.extensions.dynamic.notFoundTitle')" :description="t('admin.extensions.dynamic.notFoundDescription')" />
    <div class="mt-5 flex justify-center">
      <UButton icon="i-lucide-rotate-cw" color="neutral" variant="subtle" :loading="pending" @click="refresh()">
        {{ t('admin.extensions.refresh') }}
      </UButton>
    </div>
  </div>

  <div v-else-if="isSettingsView" class="overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
    <div class="flex items-center justify-between gap-4 border-b border-slate-200 px-4 py-3 dark:border-zinc-800">
      <div class="min-w-0">
        <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
          {{ t('admin.extensions.dynamic.settingsTitle') }}
        </h3>
        <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
          {{ t('admin.extensions.dynamic.settingsIntro') }}
        </p>
      </div>
      <UButton icon="i-lucide-rotate-cw" color="neutral" variant="ghost" :loading="loadingSettings" @click="loadSettings">
        {{ t('admin.extensions.refresh') }}
      </UButton>
    </div>

    <form v-if="settings?.items.length" class="divide-y divide-slate-200 dark:divide-zinc-800" @submit.prevent="saveSettings">
      <div v-for="item in settings.items" :key="item.key" class="grid gap-3 px-4 py-4 md:grid-cols-[220px_1fr]">
        <div class="min-w-0">
          <label :for="`extension-setting-${item.key}`" class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
            {{ item.label }}
          </label>
          <p class="mt-1 text-xs leading-5 text-slate-500 dark:text-zinc-400">
            {{ item.description || item.key }}
          </p>
        </div>
        <div class="min-w-0">
          <label v-if="item.type === 'boolean'" class="inline-flex min-h-10 items-center gap-2 rounded-md border border-slate-200 px-3 text-sm text-slate-700 dark:border-zinc-800 dark:text-zinc-200">
            <input
              :id="`extension-setting-${item.key}`"
              type="checkbox"
              class="size-4 accent-[var(--sf-accent)]"
              :checked="formValues[item.key] === 'true'"
              @change="onBooleanSettingChange(item.key, $event)"
            >
            <span>{{ t('admin.extensions.dynamic.enabled') }}</span>
          </label>
          <UInput
            v-else
            :id="`extension-setting-${item.key}`"
            v-model="formValues[item.key]"
            class="max-w-xl"
          />
          <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.dynamic.defaultValue', { value: item.default || t('admin.extensions.dynamic.emptyValue') }) }}
          </p>
        </div>
      </div>

      <div class="flex flex-wrap items-center justify-end gap-2 px-4 py-4">
        <UButton type="button" color="neutral" variant="subtle" icon="i-lucide-rotate-ccw" :loading="savingSettings" @click="resetSettings">
          {{ t('admin.extensions.dynamic.resetDefaults') }}
        </UButton>
        <UButton type="submit" icon="i-lucide-save" :loading="savingSettings">
          {{ t('admin.extensions.dynamic.saveSettings') }}
        </UButton>
      </div>
    </form>

    <div v-else class="p-10">
      <SFEmptyState icon-label="CFG" :title="t('admin.extensions.dynamic.emptySettingsTitle')" :description="t('admin.extensions.dynamic.emptySettingsDescription')" />
    </div>
  </div>

  <div v-else class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_320px]">
    <div class="rounded-lg border border-slate-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
      <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
        {{ extension.name }}
      </h3>
      <p class="mt-3 text-sm leading-6 text-slate-600 dark:text-zinc-300">
        {{ extension.manifest.description }}
      </p>
      <div class="mt-5 flex flex-wrap gap-2">
        <UButton :to="extension.manifest.url" target="_blank" color="neutral" variant="subtle" icon="i-lucide-external-link">
          {{ t('admin.extensions.dynamic.website') }}
        </UButton>
        <UButton
          v-if="extension.manifest.author?.url"
          :to="extension.manifest.author.url"
          target="_blank"
          color="neutral"
          variant="ghost"
          icon="i-lucide-user-round"
        >
          {{ extension.manifest.author.name }}
        </UButton>
      </div>
    </div>

    <div class="rounded-lg border border-slate-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
      <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
        {{ t('admin.extensions.dynamic.packageInfo') }}
      </h3>
      <dl class="mt-4 space-y-3 text-sm">
        <div class="flex items-center justify-between gap-4">
          <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.dynamic.type') }}</dt>
          <dd class="font-medium text-slate-900 dark:text-zinc-100">{{ extension.type }}</dd>
        </div>
        <div class="flex items-center justify-between gap-4">
          <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.dynamic.version') }}</dt>
          <dd class="font-medium text-slate-900 dark:text-zinc-100">{{ extension.version }}</dd>
        </div>
        <div class="flex items-center justify-between gap-4">
          <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.dynamic.status') }}</dt>
          <dd class="font-medium text-slate-900 dark:text-zinc-100">{{ extension.status }}</dd>
        </div>
        <div class="flex items-center justify-between gap-4">
          <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.dynamic.author') }}</dt>
          <dd class="truncate font-medium text-slate-900 dark:text-zinc-100">{{ extension.manifest.author?.name }}</dd>
        </div>
      </dl>
    </div>
  </div>
</template>
