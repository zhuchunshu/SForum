<script setup lang="ts">
/**
 * F4.5：站点产品开关。与 RBAC 正交——只控制产品面是否开启，不授予权限。
 */
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminFeatureFlags'
})

type FeatureOption = {
  name: string
  value: string
  public: boolean
  secret: boolean
  secretSet: boolean
}

type FeatureCatalogItem = {
  name: string
  public: boolean
  recommendedDefault: string
  description: string
}

const { t, te } = useI18n()
const toast = useToast()
const { request } = useApiClient()
const adminPage = useAdminPage('/settings/features')
const { can } = useAuthSession()
const canManage = computed(() => can('settings.site.manage') || can('settings.manage'))

const pending = ref(true)
const saving = ref(false)
const restoring = ref(false)
const loadError = ref('')
const items = ref<FeatureOption[]>([])
const catalog = ref<FeatureCatalogItem[]>([])

const form = reactive<Record<string, boolean>>({})

const catalogByName = computed(() => {
  const map: Record<string, FeatureCatalogItem> = {}
  for (const item of catalog.value) {
    map[item.name] = item
  }
  return map
})

function featureLabel(name: string) {
  const key = `admin.features.flags.${name}`
  return te(key) ? t(key) : name
}

function featureHelp(name: string) {
  const key = `admin.features.flagsHelp.${name}`
  return te(key) ? t(key) : (catalogByName.value[name]?.description || '')
}

function isEnabled(value: string) {
  return value === 'enabled' || value === 'true' || value === '1'
}

async function load() {
  pending.value = true
  loadError.value = ''
  try {
    const data = await request<{ items: FeatureOption[], catalog: FeatureCatalogItem[] }>('/admin/features')
    items.value = data.items || []
    catalog.value = data.catalog || []
    for (const item of items.value) {
      form[item.name] = isEnabled(item.value)
    }
  } catch (error) {
    loadError.value = apiErrorMessage(error) || t('admin.features.loadFailed')
  } finally {
    pending.value = false
  }
}

async function save() {
  if (!canManage.value) return
  saving.value = true
  loadError.value = ''
  try {
    const options = Object.keys(form).map(name => ({
      name,
      value: form[name] ? 'enabled' : 'disabled'
    }))
    await request('/admin/features', {
      method: 'PUT',
      body: { options }
    })
    toast.add({ title: t('admin.features.saved'), color: 'primary' })
    await load()
  } catch (error) {
    loadError.value = apiErrorMessage(error) || t('admin.features.saveFailed')
  } finally {
    saving.value = false
  }
}

async function restoreRecommended() {
  if (!canManage.value) return
  restoring.value = true
  loadError.value = ''
  try {
    await request('/admin/features/restore-defaults', { method: 'POST' })
    toast.add({ title: t('admin.features.restored'), color: 'primary' })
    await load()
  } catch (error) {
    loadError.value = apiErrorMessage(error) || t('admin.features.saveFailed')
  } finally {
    restoring.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold text-slate-900 dark:text-white">
          {{ t('admin.features.title') }}
        </h1>
        <p class="mt-1 max-w-2xl text-sm text-slate-600 dark:text-zinc-400">
          {{ t('admin.features.description') }}
        </p>
      </div>
      <div class="flex flex-wrap gap-2">
        <UButton
          color="neutral"
          variant="soft"
          leading-icon="i-lucide-rotate-ccw"
          :loading="restoring"
          :disabled="!canManage || saving || pending"
          @click="restoreRecommended"
        >
          {{ t('admin.features.restoreRecommended') }}
        </UButton>
        <UButton
          color="primary"
          leading-icon="i-lucide-save"
          :loading="saving"
          :disabled="!canManage || restoring || pending"
          @click="save"
        >
          {{ t('admin.features.save') }}
        </UButton>
      </div>
    </div>

    <UAlert
      v-if="loadError"
      color="error"
      variant="subtle"
      :title="loadError"
      icon="i-lucide-circle-alert"
    />

    <UCard class="border-slate-200 dark:border-zinc-800">
      <template #header>
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-toggle-left" class="size-5 text-slate-500" />
          <div>
            <h2 class="font-semibold text-slate-900 dark:text-white">
              {{ t('admin.features.panelTitle') }}
            </h2>
            <p class="text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.features.panelHelp') }}
            </p>
          </div>
        </div>
      </template>

      <div v-if="pending" class="py-10 text-center text-sm text-slate-500">
        {{ t('admin.common.loading') }}
      </div>

      <div v-else class="divide-y divide-slate-100 dark:divide-zinc-800">
        <div
          v-for="item in items"
          :key="item.name"
          class="flex flex-wrap items-center justify-between gap-4 py-4 first:pt-0 last:pb-0"
        >
          <div class="min-w-0 flex-1">
            <p class="font-medium text-slate-900 dark:text-white">
              {{ featureLabel(item.name) }}
            </p>
            <p class="mt-0.5 text-sm text-slate-500 dark:text-zinc-400">
              {{ featureHelp(item.name) }}
            </p>
            <p class="mt-1 font-mono text-[11px] text-slate-400 dark:text-zinc-500">
              {{ item.name }}
              <span v-if="!item.public" class="ml-2 rounded bg-slate-100 px-1.5 py-0.5 text-[10px] dark:bg-zinc-800">
                {{ t('admin.features.adminOnly') }}
              </span>
            </p>
          </div>
          <USwitch
            v-model="form[item.name]"
            :disabled="!canManage || saving || restoring"
          />
        </div>
      </div>
    </UCard>
  </div>
</template>
