<script setup lang="ts">
import SFAdminFormFooter from '~/components/admin/SFAdminFormFooter.vue'
import { apiErrorMessage } from '~/composables/useApiClient'
import {
  createDefaultAttachmentSettings,
  isRecommendedAttachmentSettings,
  resetAttachmentSettingsToRecommended,
  splitAttachmentSettingList,
  type AttachmentSettings
} from '~/utils/attachments/attachmentSettings'

type ProbeResult = {
  provider: string
  ok: boolean
  message: string
  reason?: string
}

const { t } = useI18n()
const toast = useToast()
const { request } = useApiClient()
const form = reactive(createDefaultAttachmentSettings())
const saving = ref(false)
const restoring = ref(false)
const testing = ref(false)

const coreProviderLabels: Record<string, string> = {
  local: t('admin.attachments.providers.local'),
  aliyun_oss: t('admin.attachments.providers.aliyunOss'),
  tencent_cos: t('admin.attachments.providers.tencentCos'),
  ftp: t('admin.attachments.providers.ftp'),
  sftp: t('admin.attachments.providers.sftp')
}

const providerChoices = computed(() => {
  const candidates = form.candidates?.filter(item => item.available !== false) ?? []
  if (candidates.length > 0) {
    return candidates.map(item => ({
      label: item.kind === 'core' && coreProviderLabels[item.value]
        ? coreProviderLabels[item.value]
        : item.label || item.value,
      value: item.value,
      kind: item.kind,
      settingsPath: item.settingsPath
    }))
  }
  return Object.entries(coreProviderLabels).map(([value, label]) => ({
    label,
    value,
    kind: 'core' as const,
    settingsPath: undefined
  }))
})

const selectedProviderIsPlugin = computed(() => form.provider.startsWith('plugin:'))
const selectedPluginSettingsPath = computed(() => providerChoices.value.find(item => item.value === form.provider)?.settingsPath || '')
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

const { data: settings, pending, error, refresh } = await useAsyncData(
  'admin-attachment-settings-tab',
  () => request<AttachmentSettings>('/admin/attachment-settings')
)

watch(settings, value => {
  if (value) applySettings(value)
}, { immediate: true })

defineExpose({ refresh, pending })

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
  toast.add({ color: 'success', icon: 'i-lucide-check', title: successTitle, duration: 10000 })
}

async function testConnection() {
  testing.value = true
  try {
    const result = await request<ProbeResult>('/admin/attachment-settings/test', { method: 'POST', body: {} })
    const detailParts = [result.message, result.reason].filter(Boolean)
    toast.add({
      color: result.ok ? 'success' : 'warning',
      icon: result.ok ? 'i-lucide-check' : 'i-lucide-triangle-alert',
      title: result.ok ? t('admin.attachments.testPassed') : t('admin.attachments.testFailed'),
      description: detailParts.join(result.message && result.reason ? ' · ' : ''),
      duration: 10000
    })
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.attachments.testFailed') })
  } finally {
    testing.value = false
  }
}

function applySettings(value: AttachmentSettings) {
  Object.assign(form, createDefaultAttachmentSettings(), value)
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

function providerLabel(provider: string) {
  return providerChoices.value.find(item => item.value === provider)?.label || provider
}
</script>

<template>
  <form class="flex flex-col" @submit.prevent="saveSettings">
    <UAlert
      v-if="error"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="t('admin.attachments.settingsLoadFailed')"
      class="mb-4"
    />

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
              <p v-if="selectedProviderIsPlugin" class="mt-1.5 text-xs text-slate-500 dark:text-zinc-400">
                {{ t('admin.attachments.pluginProviderHint') }}
                <NuxtLink
                  v-if="selectedPluginSettingsPath"
                  :to="`/admin${selectedPluginSettingsPath}`"
                  class="ml-1 font-medium text-[var(--sf-accent)] underline-offset-2 hover:underline dark:text-[var(--sf-accent-dark)]"
                >
                  {{ t('admin.attachments.openPluginSettings') }}
                </NuxtLink>
              </p>
            </UFormField>
            <UFormField :label="t('admin.attachments.maxFileSize')" name="attachment-max-size">
              <UInput v-model.number="form.maxFileSizeMb" size="lg" type="number" min="1" max="1024" icon="i-lucide-hard-drive-upload" class="w-full" />
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
            <UInput v-model="form.pathTemplate" size="lg" icon="i-lucide-route" class="w-full font-mono" />
            <p class="mt-2 break-all text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.attachments.pathPreview') }} {{ pathPreview }}
            </p>
          </UFormField>

          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="t('admin.attachments.publicBaseUrl')" name="attachment-public-base-url">
              <UInput v-model="form.publicBaseUrl" size="lg" type="url" icon="i-lucide-link" class="w-full" />
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
              <UTextarea v-model="allowedExtensionsText" size="lg" :rows="4" class="w-full font-mono text-xs" />
            </UFormField>
            <UFormField :label="t('admin.attachments.allowedMimeTypes')" name="attachment-mime-types">
              <UTextarea v-model="allowedMimeTypesText" size="lg" :rows="4" class="w-full font-mono text-xs" />
            </UFormField>
          </div>

          <UFormField :label="t('admin.attachments.cleanupDays')" name="attachment-cleanup-days">
            <UInput v-model.number="form.cleanupOrphanAfterDays" size="lg" type="number" min="1" max="3650" icon="i-lucide-calendar-clock" class="w-full" />
          </UFormField>

          <div v-if="form.provider === 'local'" class="grid gap-4 md:grid-cols-2">
            <UFormField :label="t('admin.attachments.localRoot')" name="attachment-local-root">
              <UInput v-model="form.local.root" size="lg" icon="i-lucide-folder-tree" class="w-full font-mono" />
              <p class="mt-2 text-xs text-slate-500 dark:text-zinc-400">
                {{ t('admin.attachments.localRootDescription') }}
              </p>
            </UFormField>
            <UFormField :label="t('admin.attachments.localPublicPrefix')" name="attachment-local-prefix">
              <UInput v-model="form.local.publicPrefix" size="lg" type="url" icon="i-lucide-folder" class="w-full" />
            </UFormField>
          </div>

          <div v-else-if="form.provider === 'aliyun_oss'" class="grid gap-4 md:grid-cols-2">
            <UInput v-model="form.aliyunOss.endpoint" size="lg" :placeholder="t('admin.attachments.aliyun.endpoint')" icon="i-lucide-cloud" />
            <UInput v-model="form.aliyunOss.bucket" size="lg" :placeholder="t('admin.attachments.aliyun.bucket')" icon="i-lucide-archive" />
            <UInput v-model="form.aliyunOss.region" size="lg" :placeholder="t('admin.attachments.aliyun.region')" icon="i-lucide-map-pin" />
            <UInput v-model="form.aliyunOss.accessKeyId" size="lg" :placeholder="t('admin.attachments.aliyun.accessKeyId')" icon="i-lucide-key-round" />
            <UInput v-model="form.aliyunOss.accessKeySecret" size="lg" type="password" :placeholder="form.aliyunOss.accessKeySecretSet ? t('admin.attachments.keepSecret') : t('admin.attachments.aliyun.accessKeySecret')" icon="i-lucide-lock-keyhole" />
          </div>

          <div v-else-if="form.provider === 'tencent_cos'" class="grid gap-4 md:grid-cols-2">
            <UInput v-model="form.tencentCos.region" size="lg" :placeholder="t('admin.attachments.tencent.region')" icon="i-lucide-map-pin" />
            <UInput v-model="form.tencentCos.bucket" size="lg" :placeholder="t('admin.attachments.tencent.bucket')" icon="i-lucide-archive" />
            <UInput v-model="form.tencentCos.secretId" size="lg" :placeholder="t('admin.attachments.tencent.secretId')" icon="i-lucide-key-round" />
            <UInput v-model="form.tencentCos.secretKey" size="lg" type="password" :placeholder="form.tencentCos.secretKeySet ? t('admin.attachments.keepSecret') : t('admin.attachments.tencent.secretKey')" icon="i-lucide-lock-keyhole" />
            <UInput v-model="form.tencentCos.cdnDomain" size="lg" type="url" :placeholder="t('admin.attachments.tencent.cdnDomain')" icon="i-lucide-globe" />
          </div>

          <div v-else-if="form.provider === 'ftp'" class="grid gap-4 md:grid-cols-2">
            <UInput v-model="form.ftp.host" size="lg" :placeholder="t('admin.attachments.remote.host')" icon="i-lucide-server" />
            <UInput v-model.number="form.ftp.port" size="lg" type="number" min="1" max="65535" :placeholder="t('admin.attachments.remote.port')" icon="i-lucide-hash" />
            <UInput v-model="form.ftp.username" size="lg" :placeholder="t('admin.attachments.remote.username')" icon="i-lucide-user" />
            <UInput v-model="form.ftp.password" size="lg" type="password" :placeholder="form.ftp.passwordSet ? t('admin.attachments.keepSecret') : t('admin.attachments.remote.password')" icon="i-lucide-lock-keyhole" />
            <UInput v-model="form.ftp.rootPath" size="lg" :placeholder="t('admin.attachments.remote.rootPath')" icon="i-lucide-folder-tree" />
            <UInput v-model="form.ftp.publicBaseUrl" size="lg" type="url" :placeholder="t('admin.attachments.remote.publicBaseUrl')" icon="i-lucide-link" />
            <label class="flex items-center gap-2 text-sm"><input v-model="form.ftp.passive" type="checkbox" class="size-4 rounded" />{{ t('admin.attachments.ftp.passive') }}</label>
            <label class="flex items-center gap-2 text-sm"><input v-model="form.ftp.explicitTls" type="checkbox" class="size-4 rounded" />{{ t('admin.attachments.ftp.explicitTls') }}</label>
          </div>

          <div v-else-if="form.provider === 'sftp'" class="grid gap-4 md:grid-cols-2">
            <UInput v-model="form.sftp.host" size="lg" :placeholder="t('admin.attachments.remote.host')" icon="i-lucide-server" />
            <UInput v-model.number="form.sftp.port" size="lg" type="number" min="1" max="65535" :placeholder="t('admin.attachments.remote.port')" icon="i-lucide-hash" />
            <UInput v-model="form.sftp.username" size="lg" :placeholder="t('admin.attachments.remote.username')" icon="i-lucide-user" />
            <UInput v-model="form.sftp.password" size="lg" type="password" :placeholder="form.sftp.passwordSet ? t('admin.attachments.keepSecret') : t('admin.attachments.remote.password')" icon="i-lucide-lock-keyhole" />
            <UTextarea v-model="form.sftp.privateKey" size="lg" :rows="4" :placeholder="form.sftp.privateKeySet ? t('admin.attachments.keepSecret') : t('admin.attachments.sftp.privateKey')" class="md:col-span-2 font-mono text-xs" />
            <UInput v-model="form.sftp.passphrase" size="lg" type="password" :placeholder="form.sftp.passphraseSet ? t('admin.attachments.keepSecret') : t('admin.attachments.sftp.passphrase')" icon="i-lucide-lock" />
            <UInput v-model="form.sftp.rootPath" size="lg" :placeholder="t('admin.attachments.remote.rootPath')" icon="i-lucide-folder-tree" />
            <UInput v-model="form.sftp.hostKeyFingerprint" size="lg" :placeholder="t('admin.attachments.sftp.hostKeyFingerprint')" icon="i-lucide-fingerprint" />
            <UInput v-model="form.sftp.publicBaseUrl" size="lg" type="url" :placeholder="t('admin.attachments.remote.publicBaseUrl')" icon="i-lucide-link" />
          </div>

          <!-- 插件提供方的凭证只在通用扩展设置页管理。 -->
          <div
            v-else-if="selectedProviderIsPlugin"
            class="rounded-lg border border-dashed border-slate-300 bg-slate-50 p-4 dark:border-zinc-700 dark:bg-zinc-950/50"
          >
            <div class="flex items-start gap-3">
              <UIcon name="i-lucide-puzzle" class="mt-0.5 size-5 shrink-0 text-[var(--sf-accent)]" />
              <div class="min-w-0 space-y-2">
                <p class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                  {{ t('admin.attachments.pluginProviderTitle') }}
                </p>
                <p class="text-xs leading-relaxed text-slate-600 dark:text-zinc-400">
                  {{ t('admin.attachments.pluginProviderHint') }}
                </p>
                <p class="text-xs text-slate-500 dark:text-zinc-500">
                  {{ t('admin.attachments.pluginProviderSecretsNote') }}
                </p>
                <div class="flex flex-wrap gap-2 pt-1">
                  <UButton
                    v-if="selectedPluginSettingsPath"
                    :to="`/admin${selectedPluginSettingsPath}`"
                    size="sm"
                    color="primary"
                    variant="soft"
                    leading-icon="i-lucide-settings-2"
                  >
                    {{ t('admin.attachments.openPluginSettings') }}
                  </UButton>
                  <UButton
                    type="button"
                    size="sm"
                    color="neutral"
                    variant="outline"
                    leading-icon="i-lucide-plug-zap"
                    :loading="testing"
                    @click="testConnection"
                  >
                    {{ t('admin.attachments.testConnection') }}
                  </UButton>
                </div>
              </div>
            </div>
          </div>
        </div>

        <aside class="rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm dark:border-zinc-800 dark:bg-zinc-950/60">
          <h3 class="font-bold text-slate-900 dark:text-zinc-100">{{ t('admin.attachments.summary') }}</h3>
          <dl class="mt-3 space-y-3">
            <div><dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.attachments.provider') }}</dt><dd class="font-medium">{{ providerLabel(form.provider) }}</dd></div>
            <div><dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.attachments.maxFileSize') }}</dt><dd class="font-medium">{{ form.maxFileSizeMb }} MB</dd></div>
            <div><dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.attachments.defaultVisibility') }}</dt><dd class="font-medium">{{ t(`admin.attachments.visibility.${form.defaultVisibility}`) }}</dd></div>
            <div><dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.attachments.allowedExtensions') }}</dt><dd class="break-words font-mono text-xs">{{ form.allowedExtensions.join(', ') }}</dd></div>
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
                :class="recommendedApplied ? 'size-4 text-emerald-500' : 'size-4 text-slate-400 dark:text-zinc-500'"
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
</template>
