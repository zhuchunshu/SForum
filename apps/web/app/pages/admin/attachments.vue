<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'
import {
  createDefaultAttachmentSettings,
  isRecommendedAttachmentSettings,
  resetAttachmentSettingsToRecommended,
  splitAttachmentSettingList,
  type AttachmentProvider,
  type AttachmentSettings
} from '~/utils/attachmentSettings'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminAttachments'
})

type AttachmentTab = 'settings' | 'manager'

type AttachmentOwner = {
  id: number
  username: string
  displayName: string
}

type Attachment = {
  id: number
  publicId: string
  owner?: AttachmentOwner
  provider: string
  objectKey: string
  name: string
  contentType: string
  extension: string
  size: number
  sha256: string
  imageWidth?: number
  imageHeight?: number
  visibility: string
  status: string
  referenceCount: number
  url: string
  createdAt: string
  deletedAt?: string
}

type AttachmentReference = {
  id: number
  attachmentId: number
  resourceType: string
  resourceId: number
  context: string
  createdAt: string
}

type AttachmentDetail = Attachment & {
  references: AttachmentReference[]
}

type AttachmentList = {
  items: Attachment[]
  total: number
  page: number
  perPage: number
}

type ProbeResult = {
  provider: string
  ok: boolean
  message: string
}

const { t } = useI18n()
const toast = useToast()
const { request } = useApiClient()
const { can } = useAuthSession()
const adminPage = useAdminPage('/attachments')

const activeTab = ref<AttachmentTab>('settings')
const saving = ref(false)
const restoring = ref(false)
const testing = ref(false)
const loadingAttachments = ref(false)
const loadingDetail = ref(false)
const selected = ref<AttachmentDetail | null>(null)

const canManageSettings = computed(() => can('attachment.settings.manage'))
const canManageAttachments = computed(() => can('attachment.manage'))

const providerChoices = computed(() => [
  { label: t('admin.attachments.providers.local'), value: 'local' },
  { label: t('admin.attachments.providers.aliyunOss'), value: 'aliyun_oss' },
  { label: t('admin.attachments.providers.tencentCos'), value: 'tencent_cos' },
  { label: t('admin.attachments.providers.ftp'), value: 'ftp' },
  { label: t('admin.attachments.providers.sftp'), value: 'sftp' }
])

const tabs = computed<Array<{ id: AttachmentTab, label: string, icon: string, enabled: boolean }>>(() => [
  { id: 'settings', label: t('admin.attachments.tabs.settings'), icon: 'i-lucide-sliders-horizontal', enabled: canManageSettings.value },
  { id: 'manager', label: t('admin.attachments.tabs.manager'), icon: 'i-lucide-folder-search', enabled: canManageAttachments.value }
])

const form = reactive(createDefaultAttachmentSettings())
const list = ref<AttachmentList>({ items: [], total: 0, page: 1, perPage: 20 })
const filters = reactive({
  query: '',
  provider: '',
  status: '',
  contentType: '',
  referenceStatus: ''
})

const { pending, error, refresh } = await useAsyncData('admin-attachments', async () => {
  if (canManageSettings.value) {
    applySettings(await request<AttachmentSettings>('/admin/attachment-settings'))
  }
  if (canManageAttachments.value) {
    await fetchAttachments()
  }
  return true
})

useSeoMeta({
  title: t('admin.attachments.metaTitle')
})

const allowedExtensionsText = computed({
  get: () => form.allowedExtensions.join(','),
  set: (value: string) => {
    form.allowedExtensions = splitAttachmentSettingList(value)
  }
})

const allowedMimeTypesText = computed({
  get: () => form.allowedMimeTypes.join(','),
  set: (value: string) => {
    form.allowedMimeTypes = splitAttachmentSettingList(value)
  }
})

const recommendedApplied = computed(() => isRecommendedAttachmentSettings(form))

const beginnerDefaults = computed(() => [
  { icon: 'i-lucide-hard-drive', label: t('admin.attachments.beginner.defaults.local') },
  { icon: 'i-lucide-upload-cloud', label: t('admin.attachments.beginner.defaults.uploads') },
  { icon: 'i-lucide-shield-check', label: t('admin.attachments.beginner.defaults.safeTypes') }
])

const pathPreview = computed(() => {
  const now = new Date()
  return form.pathTemplate
    .replaceAll('{yyyy}', String(now.getFullYear()))
    .replaceAll('{mm}', String(now.getMonth() + 1).padStart(2, '0'))
    .replaceAll('{dd}', String(now.getDate()).padStart(2, '0'))
    .replaceAll('{public_id}', '7f4d9e2a')
    .replaceAll('{ext}', '.png')
})

watchEffect(() => {
  if (tabs.value.some(tab => tab.id === activeTab.value && tab.enabled)) {
    return
  }

  const fallback = tabs.value.find(tab => tab.enabled)
  if (fallback) {
    activeTab.value = fallback.id
  }
})

async function saveSettings() {
  saving.value = true
  try {
    await persistSettings(t('admin.attachments.settingsSaved'))
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.attachments.settingsSaveFailed') })
  } finally {
    saving.value = false
  }
}

async function restoreRecommendedSettings() {
  const previous = settingsPayload()
  Object.assign(form, resetAttachmentSettingsToRecommended(form))
  restoring.value = true
  try {
    await persistSettings(t('admin.attachments.recommendedRestored'))
  } catch (error) {
    applySettings(previous)
    toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.attachments.recommendedRestoreFailed') })
  } finally {
    restoring.value = false
  }
}

async function persistSettings(successTitle: string) {
  const updated = await request<AttachmentSettings>('/admin/attachment-settings', {
    method: 'PUT',
    body: settingsPayload()
  })
  applySettings(updated)
  toast.add({ color: 'success', icon: 'i-lucide-check', title: successTitle })
}

async function testConnection() {
  testing.value = true
  try {
    const result = await request<ProbeResult>('/admin/attachment-settings/test', { method: 'POST', body: {} })
    toast.add({
      color: result.ok ? 'success' : 'warning',
      icon: result.ok ? 'i-lucide-check' : 'i-lucide-triangle-alert',
      title: result.ok ? t('admin.attachments.testPassed') : t('admin.attachments.testFailed'),
      description: result.message
    })
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.attachments.testFailed') })
  } finally {
    testing.value = false
  }
}

async function fetchAttachments() {
  loadingAttachments.value = true
  try {
    const params = new URLSearchParams()
    params.set('page', String(list.value.page || 1))
    params.set('perPage', String(list.value.perPage || 20))
    for (const [key, value] of Object.entries(filters)) {
      if (value) {
        params.set(key, value)
      }
    }
    list.value = await request<AttachmentList>(`/admin/attachments?${params.toString()}`)
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.attachments.listLoadFailed') })
  } finally {
    loadingAttachments.value = false
  }
}

async function updateAttachmentStatus(item: Attachment, status: 'active' | 'disabled') {
  try {
    const updated = await request<Attachment>(`/admin/attachments/${item.id}`, {
      method: 'PATCH',
      body: { status }
    })
    replaceAttachment(updated)
    if (selected.value?.id === updated.id) {
      selected.value = { ...selected.value, ...updated }
    }
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.attachments.updateFailed') })
  }
}

async function deleteAttachment(item: Attachment) {
  try {
    const updated = await request<Attachment>(`/admin/attachments/${item.id}`, { method: 'DELETE' })
    replaceAttachment(updated)
    if (selected.value?.id === updated.id) {
      selected.value = { ...selected.value, ...updated }
    }
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
      title: t('admin.attachments.cleanupFinished', result)
    })
    await fetchAttachments()
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.attachments.cleanupFailed') })
  }
}

function replaceAttachment(item: Attachment) {
  list.value.items = list.value.items.map((current) => current.id === item.id ? item : current)
}

async function selectAttachment(item: Attachment) {
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
  if (!selected.value?.url || !import.meta.client) {
    return
  }
  await navigator.clipboard.writeText(selected.value.url)
  toast.add({ color: 'success', icon: 'i-lucide-copy-check', title: t('admin.attachments.copied') })
}

function applySettings(settings: AttachmentSettings) {
  Object.assign(form, createDefaultAttachmentSettings(), settings)
}

function settingsPayload(): AttachmentSettings {
  return {
    ...form,
    allowedExtensions: [...form.allowedExtensions],
    allowedMimeTypes: [...form.allowedMimeTypes],
    local: { ...form.local },
    aliyunOss: { ...form.aliyunOss },
    tencentCos: { ...form.tencentCos },
    ftp: { ...form.ftp },
    sftp: { ...form.sftp }
  }
}

function setActiveTab(tab: AttachmentTab) {
  activeTab.value = tab
}

function statusColor(status: string) {
  if (status === 'active') return 'success'
  if (status === 'disabled') return 'warning'
  return 'neutral'
}

function humanFileSize(size: number) {
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

function providerLabel(provider: string) {
  return providerChoices.value.find(item => item.value === provider)?.label || provider
}

function isPreviewableImage(item: Attachment) {
  return item.contentType.startsWith('image/') && item.url
}
</script>

<template>
  <div class="mb-4 flex flex-col gap-1">
    <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ t('admin.attachments.title') }}
    </h2>
    <p class="text-sm text-slate-500 dark:text-zinc-400">
      {{ t('admin.attachments.intro') }}
    </p>
  </div>

  <UDashboardToolbar class="mb-6 rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-500 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm">
        <UIcon name="i-lucide-paperclip" class="size-4" />
        <span class="truncate">{{ t('admin.attachments.toolbar') }}</span>
      </div>
    </template>
    <template #right>
      <UButton color="neutral" variant="outline" leading-icon="i-lucide-refresh-cw" :loading="pending || loadingAttachments" @click="refresh()">
        {{ t('admin.attachments.refresh') }}
      </UButton>
    </template>
  </UDashboardToolbar>

  <UAlert
    v-if="error"
    color="error"
    variant="soft"
    icon="i-lucide-triangle-alert"
    :title="t('admin.attachments.settingsLoadFailed')"
    class="mb-4"
  />

  <div role="tablist" :aria-label="t('admin.attachments.tabs.label')" class="mb-4 flex flex-wrap gap-2 border-b border-slate-200 pb-3 dark:border-zinc-800">
    <UButton
      v-for="tab in tabs.filter(item => item.enabled)"
      :key="tab.id"
      :color="activeTab === tab.id ? 'primary' : 'neutral'"
      :variant="activeTab === tab.id ? 'solid' : 'ghost'"
      :leading-icon="tab.icon"
      @click="setActiveTab(tab.id)"
    >
      {{ tab.label }}
    </UButton>
  </div>

  <form v-if="activeTab === 'settings' && canManageSettings" class="flex flex-col" @submit.prevent="saveSettings">
    <section class="mb-4 rounded-lg border border-emerald-200 bg-emerald-50/80 p-4 text-sm text-emerald-950 dark:border-emerald-900/60 dark:bg-emerald-950/30 dark:text-emerald-100">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div class="flex gap-3">
          <div class="grid size-10 shrink-0 place-items-center rounded-lg bg-white text-emerald-700 shadow-sm dark:bg-emerald-900/60 dark:text-emerald-200">
            <UIcon name="i-lucide-sparkles" class="size-5" />
          </div>
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <h3 class="text-base font-bold text-emerald-950 dark:text-emerald-50">
                {{ t('admin.attachments.beginner.title') }}
              </h3>
              <UBadge v-if="recommendedApplied" color="success" variant="soft">
                {{ t('admin.attachments.beginner.currentRecommended') }}
              </UBadge>
            </div>
            <p class="mt-1 max-w-3xl text-sm text-emerald-800 dark:text-emerald-200">
              {{ t('admin.attachments.beginner.description') }}
            </p>
          </div>
        </div>
        <UButton
          type="button"
          color="primary"
          variant="solid"
          leading-icon="i-lucide-rotate-ccw"
          :loading="restoring"
          :disabled="saving || pending"
          class="shrink-0"
          @click="restoreRecommendedSettings"
        >
          {{ t('admin.attachments.restoreRecommended') }}
        </UButton>
      </div>

      <div class="mt-4 grid gap-2 md:grid-cols-3">
        <div
          v-for="item in beginnerDefaults"
          :key="item.label"
          class="flex items-center gap-2 rounded-md border border-emerald-200 bg-white/80 px-3 py-2 dark:border-emerald-900/60 dark:bg-emerald-950/40"
        >
          <UIcon :name="item.icon" class="size-4 shrink-0 text-emerald-700 dark:text-emerald-200" />
          <span class="text-xs font-medium text-emerald-900 dark:text-emerald-100">{{ item.label }}</span>
        </div>
      </div>

      <p class="mt-3 text-xs text-emerald-700 dark:text-emerald-200">
        {{ t('admin.attachments.beginner.secretNote') }}
      </p>
    </section>

    <UCard class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900" :ui="{ footer: 'sticky bottom-0 z-20 bg-white/95 dark:bg-zinc-900/95 backdrop-blur-sm border-t border-slate-200 dark:border-zinc-800 p-4 sm:px-6' }">
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h3 class="text-base font-bold text-slate-900 dark:text-zinc-100">
              {{ t('admin.attachments.settingsTitle') }}
            </h3>
            <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.attachments.settingsDescription') }}
            </p>
          </div>
          <UButton type="button" color="neutral" variant="outline" leading-icon="i-lucide-plug-zap" :loading="testing" @click="testConnection">
            {{ t('admin.attachments.testConnection') }}
          </UButton>
        </div>
      </template>

      <div class="grid gap-5 xl:grid-cols-[minmax(0,1fr)_320px]">
        <div class="grid gap-4">
          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="t('admin.attachments.provider')" name="attachment-provider">
              <select v-model="form.provider" class="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-900 outline-none transition focus:border-[var(--sf-accent)] focus:ring-2 focus:ring-[var(--sf-accent-focus)] dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100">
                <option v-for="choice in providerChoices" :key="choice.value" :value="choice.value">
                  {{ choice.label }}
                </option>
              </select>
            </UFormField>
            <UFormField :label="t('admin.attachments.maxFileSize')" name="attachment-max-size">
              <UInput v-model.number="form.maxFileSizeMb" type="number" min="1" max="1024" icon="i-lucide-hard-drive-upload" class="w-full" />
            </UFormField>
          </div>

          <label class="flex items-start gap-3 rounded-lg border border-slate-200 p-4 dark:border-zinc-800">
            <input v-model="form.uploadEnabled" type="checkbox" class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]" />
            <span>
              <span class="block text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.attachments.uploadEnabled') }}</span>
              <span class="mt-1 block text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.attachments.uploadEnabledDescription') }}</span>
            </span>
          </label>

          <UFormField :label="t('admin.attachments.pathTemplate')" name="attachment-path-template">
            <UInput v-model="form.pathTemplate" icon="i-lucide-route" class="w-full font-mono" />
            <p class="mt-2 break-all text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.attachments.pathPreview') }} {{ pathPreview }}
            </p>
          </UFormField>

          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="t('admin.attachments.publicBaseUrl')" name="attachment-public-base-url">
              <UInput v-model="form.publicBaseUrl" type="url" icon="i-lucide-link" class="w-full" />
            </UFormField>
            <UFormField :label="t('admin.attachments.defaultVisibility')" name="attachment-visibility">
              <select v-model="form.defaultVisibility" class="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-900 outline-none transition focus:border-[var(--sf-accent)] focus:ring-2 focus:ring-[var(--sf-accent-focus)] dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100">
                <option value="public">{{ t('admin.attachments.visibility.public') }}</option>
                <option value="private">{{ t('admin.attachments.visibility.private') }}</option>
              </select>
            </UFormField>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="t('admin.attachments.allowedExtensions')" name="attachment-extensions">
              <UTextarea v-model="allowedExtensionsText" :rows="4" class="w-full font-mono text-xs" />
            </UFormField>
            <UFormField :label="t('admin.attachments.allowedMimeTypes')" name="attachment-mime-types">
              <UTextarea v-model="allowedMimeTypesText" :rows="4" class="w-full font-mono text-xs" />
            </UFormField>
          </div>

          <UFormField :label="t('admin.attachments.cleanupDays')" name="attachment-cleanup-days">
            <UInput v-model.number="form.cleanupOrphanAfterDays" type="number" min="1" max="3650" icon="i-lucide-calendar-clock" class="w-full md:max-w-xs" />
          </UFormField>

          <div v-if="form.provider === 'local'" class="grid gap-4 md:grid-cols-2">
            <UFormField :label="t('admin.attachments.localRoot')" name="attachment-local-root">
              <UInput v-model="form.local.root" icon="i-lucide-folder-tree" class="w-full font-mono" />
              <p class="mt-2 text-xs text-slate-500 dark:text-zinc-400">
                {{ t('admin.attachments.localRootDescription') }}
              </p>
            </UFormField>
            <UFormField :label="t('admin.attachments.localPublicPrefix')" name="attachment-local-prefix">
              <UInput v-model="form.local.publicPrefix" type="url" icon="i-lucide-folder" class="w-full" />
            </UFormField>
          </div>

          <div v-else-if="form.provider === 'aliyun_oss'" class="grid gap-4 md:grid-cols-2">
            <UInput v-model="form.aliyunOss.endpoint" :placeholder="t('admin.attachments.aliyun.endpoint')" icon="i-lucide-cloud" />
            <UInput v-model="form.aliyunOss.bucket" :placeholder="t('admin.attachments.aliyun.bucket')" icon="i-lucide-archive" />
            <UInput v-model="form.aliyunOss.region" :placeholder="t('admin.attachments.aliyun.region')" icon="i-lucide-map-pin" />
            <UInput v-model="form.aliyunOss.accessKeyId" :placeholder="t('admin.attachments.aliyun.accessKeyId')" icon="i-lucide-key-round" />
            <UInput v-model="form.aliyunOss.accessKeySecret" type="password" :placeholder="form.aliyunOss.accessKeySecretSet ? t('admin.attachments.keepSecret') : t('admin.attachments.aliyun.accessKeySecret')" icon="i-lucide-lock-keyhole" />
          </div>

          <div v-else-if="form.provider === 'tencent_cos'" class="grid gap-4 md:grid-cols-2">
            <UInput v-model="form.tencentCos.region" :placeholder="t('admin.attachments.tencent.region')" icon="i-lucide-map-pin" />
            <UInput v-model="form.tencentCos.bucket" :placeholder="t('admin.attachments.tencent.bucket')" icon="i-lucide-archive" />
            <UInput v-model="form.tencentCos.secretId" :placeholder="t('admin.attachments.tencent.secretId')" icon="i-lucide-key-round" />
            <UInput v-model="form.tencentCos.secretKey" type="password" :placeholder="form.tencentCos.secretKeySet ? t('admin.attachments.keepSecret') : t('admin.attachments.tencent.secretKey')" icon="i-lucide-lock-keyhole" />
            <UInput v-model="form.tencentCos.cdnDomain" type="url" :placeholder="t('admin.attachments.tencent.cdnDomain')" icon="i-lucide-globe" />
          </div>

          <div v-else-if="form.provider === 'ftp'" class="grid gap-4 md:grid-cols-2">
            <UInput v-model="form.ftp.host" :placeholder="t('admin.attachments.remote.host')" icon="i-lucide-server" />
            <UInput v-model.number="form.ftp.port" type="number" min="1" max="65535" :placeholder="t('admin.attachments.remote.port')" icon="i-lucide-hash" />
            <UInput v-model="form.ftp.username" :placeholder="t('admin.attachments.remote.username')" icon="i-lucide-user" />
            <UInput v-model="form.ftp.password" type="password" :placeholder="form.ftp.passwordSet ? t('admin.attachments.keepSecret') : t('admin.attachments.remote.password')" icon="i-lucide-lock-keyhole" />
            <UInput v-model="form.ftp.rootPath" :placeholder="t('admin.attachments.remote.rootPath')" icon="i-lucide-folder-tree" />
            <UInput v-model="form.ftp.publicBaseUrl" type="url" :placeholder="t('admin.attachments.remote.publicBaseUrl')" icon="i-lucide-link" />
            <label class="flex items-center gap-2 text-sm"><input v-model="form.ftp.passive" type="checkbox" class="size-4 rounded" />{{ t('admin.attachments.ftp.passive') }}</label>
            <label class="flex items-center gap-2 text-sm"><input v-model="form.ftp.explicitTls" type="checkbox" class="size-4 rounded" />{{ t('admin.attachments.ftp.explicitTls') }}</label>
          </div>

          <div v-else class="grid gap-4 md:grid-cols-2">
            <UInput v-model="form.sftp.host" :placeholder="t('admin.attachments.remote.host')" icon="i-lucide-server" />
            <UInput v-model.number="form.sftp.port" type="number" min="1" max="65535" :placeholder="t('admin.attachments.remote.port')" icon="i-lucide-hash" />
            <UInput v-model="form.sftp.username" :placeholder="t('admin.attachments.remote.username')" icon="i-lucide-user" />
            <UInput v-model="form.sftp.password" type="password" :placeholder="form.sftp.passwordSet ? t('admin.attachments.keepSecret') : t('admin.attachments.remote.password')" icon="i-lucide-lock-keyhole" />
            <UTextarea v-model="form.sftp.privateKey" :rows="4" :placeholder="form.sftp.privateKeySet ? t('admin.attachments.keepSecret') : t('admin.attachments.sftp.privateKey')" class="md:col-span-2 font-mono text-xs" />
            <UInput v-model="form.sftp.passphrase" type="password" :placeholder="form.sftp.passphraseSet ? t('admin.attachments.keepSecret') : t('admin.attachments.sftp.passphrase')" icon="i-lucide-lock" />
            <UInput v-model="form.sftp.rootPath" :placeholder="t('admin.attachments.remote.rootPath')" icon="i-lucide-folder-tree" />
            <UInput v-model="form.sftp.hostKeyFingerprint" :placeholder="t('admin.attachments.sftp.hostKeyFingerprint')" icon="i-lucide-fingerprint" />
            <UInput v-model="form.sftp.publicBaseUrl" type="url" :placeholder="t('admin.attachments.remote.publicBaseUrl')" icon="i-lucide-link" />
          </div>
        </div>

        <aside class="rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm dark:border-zinc-800 dark:bg-zinc-950/60">
          <h3 class="font-bold text-slate-900 dark:text-zinc-100">{{ t('admin.attachments.summary') }}</h3>
          <dl class="mt-3 space-y-3">
            <div><dt class="text-xs text-slate-500">{{ t('admin.attachments.provider') }}</dt><dd class="font-medium">{{ providerLabel(form.provider) }}</dd></div>
            <div><dt class="text-xs text-slate-500">{{ t('admin.attachments.maxFileSize') }}</dt><dd class="font-medium">{{ form.maxFileSizeMb }} MB</dd></div>
            <div><dt class="text-xs text-slate-500">{{ t('admin.attachments.defaultVisibility') }}</dt><dd class="font-medium">{{ t(`admin.attachments.visibility.${form.defaultVisibility}`) }}</dd></div>
            <div><dt class="text-xs text-slate-500">{{ t('admin.attachments.allowedExtensions') }}</dt><dd class="break-words font-mono text-xs">{{ form.allowedExtensions.join(', ') }}</dd></div>
          </dl>
        </aside>
      </div>

      <template #footer>
        <SFAdminFormFooter
          :saving="saving || restoring"
          :disabled="pending"
          :submit-text="t('admin.attachments.saveSettings')"
          :reset-text="t('admin.attachments.restoreRecommended')"
          reset-icon="i-lucide-rotate-ccw"
          @reset="restoreRecommendedSettings"
        >
          <template #left>
            <div class="flex items-center gap-2">
              <UIcon
                :name="recommendedApplied ? 'i-lucide-circle-check' : 'i-lucide-info'"
                :class="recommendedApplied ? 'size-4 text-emerald-500' : 'size-4 text-slate-400'"
              />
              <span>
                {{ recommendedApplied ? t('admin.attachments.beginner.currentRecommended') : t('admin.attachments.beginner.restoreHint') }}
              </span>
            </div>
          </template>
        </SFAdminFormFooter>
      </template>
    </UCard>
  </form>

  <div v-else-if="activeTab === 'manager' && canManageAttachments" class="grid gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
    <UCard class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <template #header>
        <div class="grid gap-3 lg:grid-cols-[1fr_150px_150px_150px_auto]">
          <UInput v-model="filters.query" icon="i-lucide-search" :placeholder="t('admin.attachments.searchPlaceholder')" @keyup.enter="fetchAttachments" />
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
                    <div class="max-w-xs truncate font-mono text-xs text-slate-500">{{ item.contentType }}</div>
                  </div>
                </div>
              </td>
              <td class="px-3 py-3">{{ providerLabel(item.provider) }}</td>
              <td class="px-3 py-3">{{ humanFileSize(item.size) }}</td>
              <td class="px-3 py-3">{{ item.referenceCount }}</td>
              <td class="px-3 py-3"><UBadge :color="statusColor(item.status)" variant="soft">{{ t(`admin.attachments.status.${item.status}`) }}</UBadge></td>
              <td class="px-3 py-3 text-xs text-slate-500">{{ new Date(item.createdAt).toLocaleString() }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <SFEmptyState v-if="list.items.length === 0" :title="t('admin.attachments.empty')" />
    </UCard>

    <aside class="rounded-lg border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
      <template v-if="selected">
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <h3 class="truncate text-base font-bold text-slate-900 dark:text-zinc-100">{{ selected.name }}</h3>
            <p class="mt-1 font-mono text-xs text-slate-500">{{ selected.publicId }}</p>
          </div>
          <UBadge :color="statusColor(selected.status)" variant="soft">{{ t(`admin.attachments.status.${selected.status}`) }}</UBadge>
        </div>
        <div v-if="loadingDetail" class="mt-3 flex items-center gap-2 text-xs text-slate-500">
          <UIcon name="i-lucide-loader-circle" class="size-4 animate-spin" />
          {{ t('admin.attachments.loadingDetail') }}
        </div>
        <dl class="mt-4 space-y-3 text-sm">
          <div><dt class="text-xs text-slate-500">{{ t('admin.attachments.objectKey') }}</dt><dd class="break-all font-mono text-xs">{{ selected.objectKey }}</dd></div>
          <div><dt class="text-xs text-slate-500">{{ t('admin.attachments.sha256') }}</dt><dd class="break-all font-mono text-xs">{{ selected.sha256 }}</dd></div>
          <div><dt class="text-xs text-slate-500">{{ t('admin.attachments.owner') }}</dt><dd>{{ selected.owner?.displayName || selected.owner?.username || '-' }}</dd></div>
          <div><dt class="text-xs text-slate-500">{{ t('admin.attachments.url') }}</dt><dd class="break-all font-mono text-xs">{{ selected.url }}</dd></div>
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
              <div class="mt-1 text-slate-500">{{ reference.context || t('admin.attachments.noReferenceContext') }}</div>
            </div>
          </div>
          <p v-else class="mt-3 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.attachments.noReferences') }}</p>
        </section>
      </template>
      <SFEmptyState v-else :title="t('admin.attachments.selectOne')" />
    </aside>
  </div>
</template>
