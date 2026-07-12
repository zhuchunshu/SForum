<script setup lang="ts">
/**
 * F4.4：实体自定义字段定义管理（user / topic）。
 * 值读写走公开/实体 API；本页只管理字段目录。
 */
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminEntityMeta'
})

type FieldDefinition = {
  id: number
  fieldKey: string
  entityType: 'user' | 'topic'
  valueType: 'string' | 'text' | 'number' | 'boolean'
  visibility: 'public' | 'owner' | 'admin'
  label: Record<string, string>
  description?: Record<string, string>
  ownerExtensionId?: string
  required: boolean
  enabled: boolean
  sortOrder: number
}

const { t, locale } = useI18n()
const toast = useToast()
const { request } = useApiClient()
const adminPage = useAdminPage('/entity-meta')
const { can } = useAuthSession()
const canManage = computed(() => can('entity_meta.manage') || can('settings.manage'))

const pending = ref(true)
const saving = ref(false)
const loadError = ref('')
const items = ref<FieldDefinition[]>([])
const filterEntity = ref<'all' | 'user' | 'topic'>('all')

const form = reactive({
  fieldKey: '',
  entityType: 'user' as 'user' | 'topic',
  valueType: 'string' as FieldDefinition['valueType'],
  visibility: 'public' as FieldDefinition['visibility'],
  labelZh: '',
  labelEn: '',
  required: false,
  enabled: true,
  sortOrder: 100
})

const entityOptions = [
  { value: 'user', label: 'user' },
  { value: 'topic', label: 'topic' }
]
const valueTypeOptions = [
  { value: 'string', label: 'string' },
  { value: 'text', label: 'text' },
  { value: 'number', label: 'number' },
  { value: 'boolean', label: 'boolean' }
]
const visibilityOptions = [
  { value: 'public', label: 'public' },
  { value: 'owner', label: 'owner' },
  { value: 'admin', label: 'admin' }
]

const filteredItems = computed(() => {
  if (filterEntity.value === 'all') return items.value
  return items.value.filter(item => item.entityType === filterEntity.value)
})

function labelOf(item: FieldDefinition) {
  const labels = item.label || {}
  return labels[String(locale.value)] || labels['zh-CN'] || labels['en-US'] || item.fieldKey
}

async function load() {
  pending.value = true
  loadError.value = ''
  try {
    const qs = filterEntity.value === 'all' ? '' : `?entityType=${filterEntity.value}`
    const data = await request<{ items: FieldDefinition[] }>(`/admin/entity-meta/definitions${qs}`)
    items.value = data.items || []
  } catch (error) {
    loadError.value = apiErrorMessage(error, t('admin.entityMeta.loadFailed'))
  } finally {
    pending.value = false
  }
}

async function createField() {
  if (!canManage.value) return
  saving.value = true
  loadError.value = ''
  try {
    await request('/admin/entity-meta/definitions', {
      method: 'POST',
      body: {
        fieldKey: form.fieldKey.trim(),
        entityType: form.entityType,
        valueType: form.valueType,
        visibility: form.visibility,
        label: {
          'zh-CN': form.labelZh.trim() || form.labelEn.trim(),
          'en-US': form.labelEn.trim() || form.labelZh.trim()
        },
        required: form.required,
        enabled: form.enabled,
        sortOrder: form.sortOrder
      }
    })
    toast.add({ title: t('admin.entityMeta.created'), color: 'primary' })
    form.fieldKey = ''
    form.labelZh = ''
    form.labelEn = ''
    await load()
  } catch (error) {
    loadError.value = apiErrorMessage(error, t('admin.entityMeta.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function toggleEnabled(item: FieldDefinition) {
  if (!canManage.value) return
  try {
    await request(`/admin/entity-meta/definitions/${encodeURIComponent(item.fieldKey)}`, {
      method: 'PATCH',
      body: { enabled: !item.enabled }
    })
    toast.add({ title: t('admin.entityMeta.updated'), color: 'primary' })
    await load()
  } catch (error) {
    loadError.value = apiErrorMessage(error, t('admin.entityMeta.saveFailed'))
  }
}

async function removeField(item: FieldDefinition) {
  if (!canManage.value) return
  if (!confirm(t('admin.entityMeta.deleteConfirm', { key: item.fieldKey }))) return
  try {
    await request(`/admin/entity-meta/definitions/${encodeURIComponent(item.fieldKey)}`, {
      method: 'DELETE'
    })
    toast.add({ title: t('admin.entityMeta.deleted'), color: 'primary' })
    await load()
  } catch (error) {
    loadError.value = apiErrorMessage(error, t('admin.entityMeta.saveFailed'))
  }
}

watch(filterEntity, () => { load() })
onMounted(load)
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-bold text-slate-900 dark:text-white">
        {{ t('admin.entityMeta.title') }}
      </h1>
      <p class="mt-1 max-w-2xl text-sm text-slate-600 dark:text-zinc-400">
        {{ t('admin.entityMeta.description') }}
      </p>
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
        <h2 class="font-semibold text-slate-900 dark:text-white">
          {{ t('admin.entityMeta.createTitle') }}
        </h2>
      </template>
      <div class="grid gap-4 md:grid-cols-2">
        <UFormField :label="t('admin.entityMeta.fieldKey')">
          <UInput v-model="form.fieldKey" placeholder="demo.extra_field" :disabled="!canManage || saving" />
        </UFormField>
        <UFormField :label="t('admin.entityMeta.entityType')">
          <USelect v-model="form.entityType" :items="entityOptions" value-key="value" label-key="label" :disabled="!canManage || saving" />
        </UFormField>
        <UFormField :label="t('admin.entityMeta.valueType')">
          <USelect v-model="form.valueType" :items="valueTypeOptions" value-key="value" label-key="label" :disabled="!canManage || saving" />
        </UFormField>
        <UFormField :label="t('admin.entityMeta.visibility')">
          <USelect v-model="form.visibility" :items="visibilityOptions" value-key="value" label-key="label" :disabled="!canManage || saving" />
        </UFormField>
        <UFormField :label="t('admin.entityMeta.labelZh')">
          <UInput v-model="form.labelZh" :disabled="!canManage || saving" />
        </UFormField>
        <UFormField :label="t('admin.entityMeta.labelEn')">
          <UInput v-model="form.labelEn" :disabled="!canManage || saving" />
        </UFormField>
      </div>
      <div class="mt-4 flex flex-wrap items-center gap-4">
        <UCheckbox v-model="form.required" :label="t('admin.entityMeta.required')" :disabled="!canManage || saving" />
        <UCheckbox v-model="form.enabled" :label="t('admin.entityMeta.enabled')" :disabled="!canManage || saving" />
        <UButton color="primary" :loading="saving" :disabled="!canManage" @click="createField">
          {{ t('admin.entityMeta.create') }}
        </UButton>
      </div>
    </UCard>

    <UCard class="border-slate-200 dark:border-zinc-800">
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <h2 class="font-semibold text-slate-900 dark:text-white">
            {{ t('admin.entityMeta.listTitle') }}
          </h2>
          <div class="flex gap-2">
            <UButton size="sm" :variant="filterEntity === 'all' ? 'solid' : 'soft'" color="neutral" @click="filterEntity = 'all'">all</UButton>
            <UButton size="sm" :variant="filterEntity === 'user' ? 'solid' : 'soft'" color="neutral" @click="filterEntity = 'user'">user</UButton>
            <UButton size="sm" :variant="filterEntity === 'topic' ? 'solid' : 'soft'" color="neutral" @click="filterEntity = 'topic'">topic</UButton>
          </div>
        </div>
      </template>

      <div v-if="pending" class="py-8 text-center text-sm text-slate-500">
        {{ t('admin.common.loading') }}
      </div>
      <div v-else-if="!filteredItems.length" class="py-8 text-center text-sm text-slate-500">
        {{ t('admin.entityMeta.empty') }}
      </div>
      <div v-else class="divide-y divide-slate-100 dark:divide-zinc-800">
        <div
          v-for="item in filteredItems"
          :key="item.fieldKey"
          class="flex flex-wrap items-center justify-between gap-3 py-3"
        >
          <div class="min-w-0">
            <p class="font-medium text-slate-900 dark:text-white">
              {{ labelOf(item) }}
              <span class="ml-2 font-mono text-xs text-slate-400">{{ item.fieldKey }}</span>
            </p>
            <p class="text-xs text-slate-500 dark:text-zinc-400">
              {{ item.entityType }} · {{ item.valueType }} · {{ item.visibility }}
              <span v-if="!item.enabled" class="ml-2 text-amber-600">{{ t('admin.entityMeta.disabled') }}</span>
            </p>
          </div>
          <div class="flex gap-2">
            <UButton size="sm" color="neutral" variant="soft" :disabled="!canManage" @click="toggleEnabled(item)">
              {{ item.enabled ? t('admin.entityMeta.disable') : t('admin.entityMeta.enable') }}
            </UButton>
            <UButton size="sm" color="error" variant="soft" :disabled="!canManage" @click="removeField(item)">
              {{ t('admin.entityMeta.delete') }}
            </UButton>
          </div>
        </div>
      </div>
    </UCard>
  </div>
</template>
