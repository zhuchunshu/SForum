<script setup lang="ts">
import type { AdminExtensionSettingsContext } from '@sforum/admin-sdk'
import { apiErrorMessage } from '~/composables/useApiClient'
import {
  extensionAdminPageRoute,
  extensionLocalizedDisplay,
  findExtensionAdminPage,
  normalizeExtensionPagePath,
  pluginWebReleaseProgress,
  recommendedExtensionSettingValues,
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

const { t, locale } = useI18n()
const route = useRoute()
const adminRoutes = useAdminRoutes()
const adminTabs = useAdminTabs()
const toast = useToast()
const { request } = useApiClient()
const { contributionsFor, adminFrontendInjected } = useAdminExtensionRegistry()

const extensionId = computed(() => {
  const value = route.params.extensionId
  return Array.isArray(value) ? value[0] || '' : `${value || ''}`
})
const currentPagePath = computed(() => normalizeExtensionPagePath(route.params.pagePath as string[] | string | undefined))

// 与插件列表共用同一缓存：启用/轮询结果会立刻反映到本页，避免「已启用仍显示禁用」。
const {
  data: extensions,
  pending,
  error,
  refresh
} = await useAsyncData<AdminExtension[]>('admin-extensions', () => request<AdminExtension[]>('/admin/extensions'), {
  default: (): AdminExtension[] => []
})

const extension = computed(() => extensions.value.find(item => item.id === extensionId.value))
const extensionDisplay = computed(() => extension.value ? extensionLocalizedDisplay(extension.value, locale.value) : null)
const adminPage = computed(() => extension.value ? findExtensionAdminPage(extension.value, currentPagePath.value, locale.value) : undefined)
const settings = ref<AdminExtensionSettings | null>(null)
const formValues = reactive<Record<string, string>>({})
const loadingSettings = ref(false)
const savingSettings = ref(false)

const pageTitle = computed(() => adminPage.value?.label || extensionDisplay.value?.name || t('admin.extensions.dynamic.notFoundTitle'))
const pageDescription = computed(() => adminPage.value?.description || extensionDisplay.value?.description || '')
const isSettingsView = computed(() => adminPage.value?.view === 'settings')
// 插件/主题未启用时：允许查看 about，但功能性设置页与贡献组件不可用。
const isExtensionActive = computed(() => extension.value?.status === 'enabled')
// 受信任插件启用会先排队 Web Release，DB status 在激活阶段才变为 enabled。
const releaseProgress = computed(() => pluginWebReleaseProgress(extension.value?.webRelease))
const isLifecyclePending = computed(() => Boolean(releaseProgress.value?.active))

// 仅当前扩展可贡献设置页/页眉/页脚组件。
const hasCustomSettingsPage = computed(() => {
  if (!extensionId.value || !isExtensionActive.value) {
    return false
  }
  return contributionsFor('admin.extension.settings.page').some(item => item.extensionId === extensionId.value)
})

// manifest 声明了自定义设置页，但当前 Nuxt 进程嵌入的 registry 还没有该贡献。
// （Web Release 完成后 API 不再挂 webRelease；不能靠 status===active 判断。）
const expectsCustomSettingsPage = computed(() => {
  const item = extension.value
  if (!item) {
    return false
  }
  return Boolean(
    item.manifest.frontend?.admin
    || item.manifest.contributions?.some(entry => entry.point === 'admin.extension.settings.page')
  )
})
// 插件已启用且声明了自定义 UI，但本进程 registry 里没有对应贡献。
const missingCustomSettingsUI = computed(() => {
  return isExtensionActive.value
    && isSettingsView.value
    && expectsCustomSettingsPage.value
    && !hasCustomSettingsPage.value
})
// dev:plain 从不注入 SFORUM_ADMIN_REGISTRY_ROOT：刷新无效，需改用完整 dev。
const registryUnavailable = computed(() => missingCustomSettingsUI.value && !adminFrontendInjected)
// 完整 supervisor 已注入 registry，但本会话仍是旧包：刷新页面即可。
const needsFrontendReload = computed(() => missingCustomSettingsUI.value && adminFrontendInjected)

function reloadFrontend() {
  if (import.meta.client) {
    globalThis.location.reload()
  }
}

// 进入本页时若列表可能陈旧（从其它标签切来），主动拉一次最新 status / webRelease。
onMounted(() => {
  void refresh()
})

// Web Release 进行中时轮询，避免长期停在「已禁用」假象。
let lifecyclePollTimer: ReturnType<typeof setInterval> | null = null
function startLifecyclePolling() {
  if (lifecyclePollTimer || !import.meta.client) {
    return
  }
  lifecyclePollTimer = setInterval(() => {
    if (!pending.value) {
      void refresh()
    }
  }, 2000)
}
function stopLifecyclePolling() {
  if (!lifecyclePollTimer) {
    return
  }
  clearInterval(lifecyclePollTimer)
  lifecyclePollTimer = null
}
watch(isLifecyclePending, (active) => {
  if (active) {
    startLifecyclePolling()
  } else {
    stopLifecyclePolling()
  }
}, { immediate: true })
onUnmounted(() => {
  stopLifecyclePolling()
})

const recommendedApplied = computed(() => {
  const items = settings.value?.items || []
  if (!items.length) {
    return true
  }
  const recommended = recommendedExtensionSettingValues(items)
  return Object.entries(recommended).every(([key, value]) => (formValues[key] ?? '') === value)
})

const hasSecretFields = computed(() => (settings.value?.items || []).some(item => item.type === 'secret'))

const settingsSlotContext = computed<AdminExtensionSettingsContext>(() => ({
  extensionId: extensionId.value,
  items: settings.value?.items || [],
  values: formValues,
  loading: loadingSettings.value,
  saving: savingSettings.value,
  recommendedApplied: recommendedApplied.value,
  updateValue: (key: string, value: string) => {
    formValues[key] = value
  },
  save: saveSettings,
  reset: resetSettings,
  openMailCenter: async () => {
    await navigateTo(adminRoutes.path('/settings/mail'))
  }
}))

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

watch([extensionId, isSettingsView, isExtensionActive], async () => {
  if (isSettingsView.value && isExtensionActive.value) {
    await loadSettings()
    return
  }
  // 禁用后清空已加载的设置，避免界面仍显示可编辑表单。
  settings.value = null
  Object.keys(formValues).forEach((key) => {
    delete formValues[key]
  })
}, { immediate: true })

async function loadSettings() {
  if (!extensionId.value || !isExtensionActive.value) {
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
    toast.add({ color: 'success', icon: 'i-lucide-save', title: t('admin.extensions.dynamic.settingsSaved'), duration: 10000 })
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.extensions.dynamic.settingsSaveFailed') })
  } finally {
    savingSettings.value = false
  }
}

async function resetSettings() {
  savingSettings.value = true
  try {
    const updated = await request<AdminExtensionSettings>(`/admin/extensions/${extensionId.value}/settings`, {
      method: 'PUT',
      body: { values: recommendedExtensionSettingValues(settings.value?.items || []) }
    })
    settings.value = updated
    for (const item of updated.items) {
      formValues[item.key] = item.value
    }
    toast.add({ color: 'success', icon: 'i-lucide-rotate-ccw', title: t('admin.extensions.dynamic.settingsReset'), duration: 10000 })
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

function secretPlaceholder(item: { type: string, secretSet?: boolean, placeholder?: string }) {
  if (item.type === 'secret' && item.secretSet) {
    return t('admin.extensions.dynamic.secretSetPlaceholder')
  }
  return item.placeholder
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

  <!-- Web Release 排队/构建/激活中：status 尚未变为 enabled，勿误导成「已禁用」 -->
  <div
    v-else-if="isLifecyclePending && isSettingsView"
    class="rounded-lg border border-sky-200 bg-sky-50/80 p-8 dark:border-sky-900/60 dark:bg-sky-950/30"
  >
    <SFEmptyState
      icon-label="…"
      :title="t('admin.extensions.dynamic.enablingTitle')"
      :description="t('admin.extensions.dynamic.enablingDescription')"
    />
    <div class="mx-auto mt-5 max-w-md space-y-2">
      <UProgress :model-value="releaseProgress?.percent || 8" color="primary" />
      <p class="text-center text-xs text-slate-500 dark:text-zinc-400">
        {{ t(releaseProgress?.detailKey || 'admin.extensions.webReleaseProgress.queued') }}
      </p>
    </div>
    <div class="mt-5 flex flex-wrap justify-center gap-2">
      <UButton icon="i-lucide-plug" color="neutral" variant="subtle" :to="adminRoutes.path('/extensions/plugins')">
        {{ t('admin.extensions.dynamic.openPlugins') }}
      </UButton>
      <UButton icon="i-lucide-rotate-cw" color="neutral" variant="subtle" :loading="pending" @click="refresh()">
        {{ t('admin.extensions.refresh') }}
      </UButton>
    </div>
  </div>

  <!-- 未启用：功能性页面（含 settings / 自定义贡献页）全部停用，引导回插件列表启用 -->
  <div v-else-if="!isExtensionActive && isSettingsView" class="rounded-lg border border-amber-200 bg-amber-50/80 p-8 dark:border-amber-900/60 dark:bg-amber-950/30">
    <SFEmptyState
      icon-label="OFF"
      :title="t('admin.extensions.dynamic.disabledTitle')"
      :description="t('admin.extensions.dynamic.disabledDescription')"
    />
    <div class="mt-5 flex flex-wrap justify-center gap-2">
      <UButton icon="i-lucide-plug" color="primary" :to="adminRoutes.path('/extensions/plugins')">
        {{ t('admin.extensions.dynamic.openPlugins') }}
      </UButton>
      <UButton icon="i-lucide-rotate-cw" color="neutral" variant="subtle" :loading="pending" @click="refresh()">
        {{ t('admin.extensions.refresh') }}
      </UButton>
    </div>
  </div>

  <div v-else-if="isSettingsView" class="space-y-4">
    <!-- plain 开发：后端可用，下方是宿主通用表单；插件自定义组件不会被注入 -->
    <UAlert
      v-if="registryUnavailable"
      color="info"
      variant="subtle"
      icon="i-lucide-info"
      :title="t('admin.extensions.dynamic.plainDevTitle')"
      :description="t('admin.extensions.dynamic.plainDevDescription')"
      class="mb-2"
    />
    <!-- 完整 dev：registry 已有但本会话未加载该插件包 → 刷新即可 -->
    <UAlert
      v-else-if="needsFrontendReload"
      color="warning"
      variant="subtle"
      icon="i-lucide-refresh-cw"
      :title="t('admin.extensions.dynamic.reloadRequiredTitle')"
      :description="t('admin.extensions.dynamic.reloadRequiredDescription')"
      :actions="[{ label: t('admin.extensions.releaseNotice.reload'), onClick: reloadFrontend }]"
      class="mb-2"
    />

    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="min-w-0">
        <div class="flex flex-wrap items-center gap-2">
          <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
            {{ t('admin.extensions.dynamic.settingsTitle') }}
          </h3>
          <UBadge v-if="hasCustomSettingsPage" color="neutral" variant="subtle" size="sm">
            {{ t('admin.extensions.dynamic.customPageBadge') }}
          </UBadge>
        </div>
        <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
          {{ t('admin.extensions.dynamic.settingsIntro') }}
        </p>
      </div>
      <UButton icon="i-lucide-rotate-cw" color="neutral" variant="ghost" :loading="loadingSettings" @click="loadSettings">
        {{ t('admin.extensions.refresh') }}
      </UButton>
    </div>

    <!-- 插件自定义整页：宿主只提供 chrome + 上下文，文案与布局由插件负责 -->
    <template v-if="hasCustomSettingsPage">
      <div v-if="loadingSettings && !settings" class="rounded-lg border border-slate-200 bg-white p-8 text-sm text-slate-500 dark:border-zinc-800 dark:bg-zinc-900">
        {{ t('admin.extensions.dynamic.loading') }}
      </div>
      <SFAdminExtensionSlot
        v-else
        point="admin.extension.settings.page"
        :extension-id="extensionId"
        :context="settingsSlotContext"
      />
    </template>

    <!-- 宿主通用表单：字段标签/说明来自 API 已按 locale 解析的 settings -->
    <template v-else>
      <section class="rounded-lg border border-emerald-200 bg-emerald-50/80 p-4 text-sm text-emerald-950 dark:border-emerald-900/60 dark:bg-emerald-950/30 dark:text-emerald-100">
        <h3 class="text-base font-bold">{{ t('admin.extensions.dynamic.recommendedTitle') }}</h3>
        <p class="mt-1 max-w-3xl text-sm text-emerald-800 dark:text-emerald-200">
          {{ t('admin.extensions.dynamic.recommendedDescription') }}
        </p>
        <p v-if="hasSecretFields" class="mt-2 text-xs text-emerald-700 dark:text-emerald-300">
          {{ t('admin.extensions.dynamic.secretsPreserved') }}
        </p>
      </section>

      <SFAdminExtensionSlot
        point="admin.extension.settings.header"
        :extension-id="extensionId"
        :context="settingsSlotContext"
      />

      <div class="overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <form v-if="settings?.items.length" class="divide-y divide-slate-200 dark:divide-zinc-800" @submit.prevent="saveSettings">
          <template v-for="(item, index) in settings.items" :key="item.key">
            <div
              v-if="item.group && item.group !== settings.items[index - 1]?.group"
              class="bg-slate-50 px-4 py-2 text-xs font-semibold text-slate-600 dark:bg-zinc-950 dark:text-zinc-300"
            >
              {{ item.group }}
            </div>
            <div class="grid gap-3 px-4 py-4 md:grid-cols-[220px_1fr]">
              <div class="min-w-0">
                <label :for="`extension-setting-${item.key}`" class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                  {{ item.label }}
                </label>
                <p class="mt-1 text-xs leading-5 text-slate-500 dark:text-zinc-400">
                  {{ item.description || item.key }}
                </p>
              </div>
              <div class="min-w-0">
                <label
                  v-if="item.type === 'boolean'"
                  class="inline-flex min-h-10 items-center gap-2 rounded-md border border-slate-200 px-3 text-sm text-slate-700 dark:border-zinc-800 dark:text-zinc-200"
                >
                  <input
                    :id="`extension-setting-${item.key}`"
                    type="checkbox"
                    class="size-4 accent-[var(--sf-accent)]"
                    :checked="formValues[item.key] === 'true'"
                    @change="onBooleanSettingChange(item.key, $event)"
                  >
                  <span>{{ t('admin.extensions.dynamic.enabled') }}</span>
                </label>
                <USelect
                  v-else-if="item.options?.length"
                  :id="`extension-setting-${item.key}`"
                  v-model="formValues[item.key]"
                  class="max-w-xl"
                  value-key="value"
                  label-key="label"
                  :items="item.options"
                  :placeholder="item.placeholder"
                />
                <UInput
                  v-else
                  :id="`extension-setting-${item.key}`"
                  v-model="formValues[item.key]"
                  class="max-w-xl"
                  :type="item.type === 'secret' ? 'password' : item.type === 'number' ? 'number' : 'text'"
                  :placeholder="secretPlaceholder(item)"
                />
                <div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-slate-500 dark:text-zinc-400">
                  <span>{{ t('admin.extensions.dynamic.defaultValue', { value: item.recommendedValue || item.default || t('admin.extensions.dynamic.emptyValue') }) }}</span>
                  <UBadge v-if="item.type === 'secret' && item.secretSet" color="success" variant="soft" size="sm">
                    {{ t('admin.extensions.dynamic.secretConfigured') }}
                  </UBadge>
                </div>
              </div>
            </div>
          </template>

          <div class="border-t border-slate-200 px-4 py-4 dark:border-zinc-800">
            <SFAdminFormFooter
              :saving="savingSettings"
              :disabled="loadingSettings"
              :submit-text="t('admin.extensions.dynamic.saveSettings')"
              :reset-text="t('admin.extensions.dynamic.resetDefaults')"
              @submit="saveSettings"
              @reset="resetSettings"
            >
              <template #left>
                <span>{{ hasSecretFields ? t('admin.extensions.dynamic.footerSecretHint') : t('admin.extensions.dynamic.footerHint') }}</span>
              </template>
            </SFAdminFormFooter>
          </div>
        </form>

        <div v-else-if="!loadingSettings" class="p-10">
          <SFEmptyState
            icon-label="CFG"
            :title="t('admin.extensions.dynamic.emptySettingsTitle')"
            :description="t('admin.extensions.dynamic.emptySettingsDescription')"
          />
        </div>
        <div v-else class="p-8 text-sm text-slate-500 dark:text-zinc-400">
          {{ t('admin.extensions.dynamic.loading') }}
        </div>
      </div>

      <SFAdminExtensionSlot
        point="admin.extension.settings.footer"
        :extension-id="extensionId"
        :context="settingsSlotContext"
      />
    </template>
  </div>

  <div v-else class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_320px]">
    <div class="rounded-lg border border-slate-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
      <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
        {{ extensionDisplay?.name }}
      </h3>
      <p class="mt-3 text-sm leading-6 text-slate-600 dark:text-zinc-300">
        {{ extensionDisplay?.description }}
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
