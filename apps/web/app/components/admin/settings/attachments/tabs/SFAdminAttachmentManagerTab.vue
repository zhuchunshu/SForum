<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import {
  attachmentStatusColor,
  buildAttachmentListQuery,
  humanFileSize,
  type AdminAttachment,
  type AttachmentDetail,
  type AttachmentFilters,
  type AttachmentList
} from '../model'

const { format: formatSiteDateTime } = useSiteDateTime()
const { t } = useI18n()
const toast = useToast()
const { request } = useApiClient()
const loadingAttachments = ref(false)
const loadingDetail = ref(false)
const selected = ref<AttachmentDetail | null>(null)
const list = ref<AttachmentList>({ items: [], total: 0, page: 1, perPage: 20 })
const filters = reactive<AttachmentFilters>({
  query: '',
  provider: '',
  status: '',
  contentType: '',
  referenceStatus: ''
})

const coreProviderLabels = computed<Record<string, string>>(() => ({
  local: t('admin.attachments.providers.local'),
  aliyun_oss: t('admin.attachments.providers.aliyunOss'),
  tencent_cos: t('admin.attachments.providers.tencentCos'),
  ftp: t('admin.attachments.providers.ftp'),
  sftp: t('admin.attachments.providers.sftp')
}))
const providerChoices = computed(() => {
  const values = new Set([...Object.keys(coreProviderLabels.value), ...list.value.items.map(item => item.provider)])
  if (filters.provider) values.add(filters.provider)
  return [...values].map(value => ({ value, label: coreProviderLabels.value[value] || value }))
})

onMounted(fetchAttachments)
defineExpose({ refresh: fetchAttachments, pending: loadingAttachments })

async function fetchAttachments() {
  loadingAttachments.value = true
  try {
    list.value = await request<AttachmentList>(`/admin/attachments?${buildAttachmentListQuery(list.value, filters)}`)
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.attachments.listLoadFailed') })
  } finally {
    loadingAttachments.value = false
  }
}

async function updateAttachmentStatus(item: AdminAttachment, status: 'active' | 'disabled') {
  try {
    const updated = await request<AdminAttachment>(`/admin/attachments/${item.id}`, {
      method: 'PATCH',
      body: { status }
    })
    replaceAttachment(updated)
    if (selected.value?.id === updated.id) selected.value = { ...selected.value, ...updated }
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('admin.attachments.statusUpdated'), duration: 10000 })
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.attachments.updateFailed') })
  }
}

async function deleteAttachment(item: AdminAttachment) {
  try {
    const updated = await request<AdminAttachment>(`/admin/attachments/${item.id}`, { method: 'DELETE' })
    replaceAttachment(updated)
    if (selected.value?.id === updated.id) selected.value = { ...selected.value, ...updated }
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('admin.attachments.deleted'), duration: 10000 })
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.attachments.deleteFailed') })
  }
}

async function cleanupAttachments() {
  try {
    const result = await request<{ deleted: number, failed: number }>('/admin/attachments/cleanup', {
      method: 'POST',
      body: { limit: 100 }
    })
    toast.add({
      color: result.failed > 0 ? 'warning' : 'success',
      icon: result.failed > 0 ? 'i-lucide-triangle-alert' : 'i-lucide-check',
      title: t('admin.attachments.cleanupFinished', result),
      duration: 10000
    })
    await fetchAttachments()
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.attachments.cleanupFailed') })
  }
}

function replaceAttachment(item: AdminAttachment) {
  list.value.items = list.value.items.map(current => current.id === item.id ? item : current)
}

async function selectAttachment(item: AdminAttachment) {
  selected.value = { ...item, references: selected.value?.id === item.id ? selected.value.references : [] }
  loadingDetail.value = true
  try {
    selected.value = await request<AttachmentDetail>(`/admin/attachments/${item.id}`)
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.attachments.detailLoadFailed') })
  } finally {
    loadingDetail.value = false
  }
}

async function copySelectedURL() {
  if (!selected.value?.url || !import.meta.client) return
  await navigator.clipboard.writeText(selected.value.url)
  toast.add({ color: 'success', icon: 'i-lucide-copy-check', title: t('admin.attachments.copied'), duration: 10000 })
}

function providerLabel(provider: string) {
  return coreProviderLabels.value[provider] || provider
}

function isPreviewableImage(item: AdminAttachment) {
  return item.contentType.startsWith('image/') && item.url
}
</script>

<template>
  <div class="grid gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
    <UCard class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <template #header>
        <div class="grid gap-3 lg:grid-cols-[1fr_150px_150px_150px_auto]">
          <UInput v-model="filters.query" size="lg" icon="i-lucide-search" :placeholder="t('admin.attachments.searchPlaceholder')" @keyup.enter="fetchAttachments" />
          <select v-model="filters.provider" class="h-10 rounded-md border border-slate-200 bg-white px-3 text-sm dark:border-zinc-700 dark:bg-zinc-950">
            <option value="">{{ t('admin.attachments.allProviders') }}</option>
            <option v-for="choice in providerChoices" :key="choice.value" :value="choice.value">{{ choice.label }}</option>
          </select>
          <select v-model="filters.status" class="h-10 rounded-md border border-slate-200 bg-white px-3 text-sm dark:border-zinc-700 dark:bg-zinc-950">
            <option value="">{{ t('admin.attachments.allStatuses') }}</option>
            <option value="active">{{ t('admin.attachments.status.active') }}</option>
            <option value="disabled">{{ t('admin.attachments.status.disabled') }}</option>
            <option value="deleted">{{ t('admin.attachments.status.deleted') }}</option>
          </select>
          <select v-model="filters.referenceStatus" class="h-10 rounded-md border border-slate-200 bg-white px-3 text-sm dark:border-zinc-700 dark:bg-zinc-950">
            <option value="">{{ t('admin.attachments.allReferences') }}</option>
            <option value="referenced">{{ t('admin.attachments.referenced') }}</option>
            <option value="orphan">{{ t('admin.attachments.orphan') }}</option>
          </select>
          <div class="flex gap-2">
            <UButton color="primary" leading-icon="i-lucide-search" :loading="loadingAttachments" @click="fetchAttachments">{{ t('admin.attachments.filter') }}</UButton>
            <UButton color="neutral" variant="outline" leading-icon="i-lucide-trash-2" @click="cleanupAttachments">{{ t('admin.attachments.cleanup') }}</UButton>
          </div>
        </div>
      </template>

      <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-slate-200 text-sm dark:divide-zinc-800">
          <thead class="text-left text-xs uppercase text-slate-500 dark:text-zinc-400">
            <tr>
              <th class="px-3 py-2">{{ t('admin.attachments.file') }}</th>
              <th class="px-3 py-2">{{ t('admin.attachments.provider') }}</th>
              <th class="px-3 py-2">{{ t('admin.attachments.size') }}</th>
              <th class="px-3 py-2">{{ t('admin.attachments.references') }}</th>
              <th class="px-3 py-2">{{ t('admin.attachments.statusLabel') }}</th>
              <th class="px-3 py-2">{{ t('admin.attachments.createdAt') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100 dark:divide-zinc-800">
            <tr v-for="item in list.items" :key="item.id" class="cursor-pointer hover:bg-slate-50 dark:hover:bg-zinc-950/60" @click="selectAttachment(item)">
              <td class="px-3 py-3">
                <div class="flex min-w-[220px] items-center gap-3">
                  <img v-if="isPreviewableImage(item)" :src="item.url" :alt="item.name" class="size-10 rounded-md border border-slate-200 object-cover dark:border-zinc-800" loading="lazy">
                  <div v-else class="grid size-10 place-items-center rounded-md border border-slate-200 bg-slate-50 text-slate-500 dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-400">
                    <UIcon name="i-lucide-file" class="size-5" />
                  </div>
                  <div class="min-w-0">
                    <div class="truncate font-medium text-slate-900 dark:text-zinc-100">{{ item.name }}</div>
                    <div class="max-w-xs truncate font-mono text-xs text-slate-500 dark:text-zinc-400">{{ item.contentType }}</div>
                  </div>
                </div>
              </td>
              <td class="px-3 py-3">{{ providerLabel(item.provider) }}</td>
              <td class="px-3 py-3">{{ humanFileSize(item.size) }}</td>
              <td class="px-3 py-3">{{ item.referenceCount }}</td>
              <td class="px-3 py-3"><UBadge :color="attachmentStatusColor(item.status)" variant="soft">{{ t(`admin.attachments.status.${item.status}`) }}</UBadge></td>
              <td class="px-3 py-3 text-xs text-slate-500 dark:text-zinc-400">{{ formatSiteDateTime(item.createdAt) }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <SFEmptyState v-if="!loadingAttachments && list.items.length === 0" :title="t('admin.attachments.empty')" />
    </UCard>

    <aside class="rounded-lg border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
      <template v-if="selected">
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <h3 class="truncate text-base font-bold text-slate-900 dark:text-zinc-100">{{ selected.name }}</h3>
            <p class="mt-1 font-mono text-xs text-slate-500 dark:text-zinc-400">{{ selected.publicId }}</p>
          </div>
          <UBadge :color="attachmentStatusColor(selected.status)" variant="soft">{{ t(`admin.attachments.status.${selected.status}`) }}</UBadge>
        </div>
        <div v-if="loadingDetail" class="mt-3 flex items-center gap-2 text-xs text-slate-500 dark:text-zinc-400">
          <UIcon name="i-lucide-loader-circle" class="size-4 animate-spin" />
          {{ t('admin.attachments.loadingDetail') }}
        </div>
        <dl class="mt-4 space-y-3 text-sm">
          <div><dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.attachments.objectKey') }}</dt><dd class="break-all font-mono text-xs">{{ selected.objectKey }}</dd></div>
          <div><dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.attachments.sha256') }}</dt><dd class="break-all font-mono text-xs">{{ selected.sha256 }}</dd></div>
          <div><dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.attachments.owner') }}</dt><dd>{{ selected.owner?.displayName || selected.owner?.username || '-' }}</dd></div>
          <div><dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.attachments.url') }}</dt><dd class="break-all font-mono text-xs">{{ selected.url }}</dd></div>
        </dl>
        <div class="mt-4 flex flex-wrap gap-2">
          <UButton color="neutral" variant="outline" leading-icon="i-lucide-external-link" :to="selected.url" target="_blank">{{ t('admin.attachments.open') }}</UButton>
          <UButton color="neutral" variant="outline" leading-icon="i-lucide-copy" @click="copySelectedURL">{{ t('admin.attachments.copyLink') }}</UButton>
          <UButton v-if="selected.status !== 'disabled'" color="warning" variant="soft" leading-icon="i-lucide-eye-off" @click="updateAttachmentStatus(selected, 'disabled')">{{ t('admin.attachments.disable') }}</UButton>
          <UButton v-if="selected.status !== 'active'" color="success" variant="soft" leading-icon="i-lucide-rotate-ccw" @click="updateAttachmentStatus(selected, 'active')">{{ t('admin.attachments.restore') }}</UButton>
          <UButton color="error" variant="soft" leading-icon="i-lucide-trash-2" :disabled="selected.referenceCount > 0" @click="deleteAttachment(selected)">{{ t('admin.attachments.delete') }}</UButton>
        </div>
        <section class="mt-6 border-t border-slate-200 pt-4 dark:border-zinc-800">
          <h4 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.attachments.referencesTitle') }}</h4>
          <div v-if="selected.references.length > 0" class="mt-3 space-y-2">
            <div v-for="reference in selected.references" :key="reference.id" class="rounded-md border border-slate-200 p-3 text-xs dark:border-zinc-800">
              <div class="font-medium text-slate-900 dark:text-zinc-100">{{ reference.resourceType }} #{{ reference.resourceId }}</div>
              <div class="mt-1 text-slate-500 dark:text-zinc-400">{{ reference.context || t('admin.attachments.noReferenceContext') }}</div>
            </div>
          </div>
          <p v-else class="mt-3 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.attachments.noReferences') }}</p>
        </section>
      </template>
      <SFEmptyState v-else :title="t('admin.attachments.selectOne')" />
    </aside>
  </div>
</template>
