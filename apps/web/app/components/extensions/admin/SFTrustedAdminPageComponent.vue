<script setup lang="ts">
import { ADMIN_MICRO_FRONTEND_API_VERSION } from '@sforum/admin-sdk'
import type {
  AdminMicroFrontendCleanup,
  AdminPageBridgeV1,
  AdminPageModuleV1
} from '@sforum/admin-sdk'
import { useColorModePreference } from '~/composables/appearance/useColorModePreference'
import { useAdminRoutes } from '~/composables/admin/useAdminRoutes'
import { useAdminFrontendTrust } from '~/composables/admin/useAdminFrontendTrust'
import SFAdminFrontendTrustPanel from '~/components/admin/SFAdminFrontendTrustPanel.vue'
import type { AdminExtension, AdminExtensionAdminPage } from '~/utils/admin/adminExtensions'
import { assertAdminExtensionRelativePath, extensionRequestPath } from '~/runtime/admin-extensions/types'
import {
  clearContributionFailures,
  contributionFailureKey,
  contributionFailureState,
  recordContributionFailure
} from '~/runtime/admin-extensions/quarantine'

const props = defineProps<{
  extension: AdminExtension
  page: AdminExtensionAdminPage
}>()

const extension = computed(() => props.extension)
const component = computed(() => props.page.component)
const { status, load: loadTrust } = useAdminFrontendTrust(extension)
const { request } = useApiClient()
const { t, locale } = useI18n()
const { resolvedMode } = useColorModePreference()
const toast = useToast()
const adminRoutes = useAdminRoutes()
const target = ref<HTMLElement>()
const loading = ref(false)
const mounted = ref(false)
const failed = ref(false)
const failureMessage = ref('')
const quarantined = ref(false)
let cleanup: AdminMicroFrontendCleanup | undefined
let styleElement: HTMLStyleElement | undefined
let loadQueue: Promise<void> = Promise.resolve()
let loadGeneration = 0
let disposed = true

const digest = computed(() => status.value?.digest || '')
const trusted = computed(() => ['trusted', 'source_trusted'].includes(status.value?.trustState || ''))
const failureKey = computed(() => contributionFailureKey(
  `admin-page:${digest.value}`,
  props.extension.id,
  component.value?.id || props.page.path
))

function assetURL(name: 'entry' | 'style') {
  const current = component.value
  if (!current) return ''
  return `/_sforum/private-assets/extensions/${encodeURIComponent(props.extension.id)}/${digest.value}/${encodeURIComponent(current.id)}/${name}`
}

function currentStorage() {
  return import.meta.client ? window.sessionStorage : undefined
}

async function dispose() {
  const currentCleanup = cleanup
  cleanup = undefined
  if (currentCleanup) {
    try {
      await currentCleanup()
    } catch {
      // cleanup 失败仅影响当前插件页面，不得破坏 Host 管理后台壳层。
    }
  }
  styleElement?.remove()
  styleElement = undefined
  target.value?.replaceChildren()
  mounted.value = false
}

function appearance() {
  const styles = getComputedStyle(document.documentElement)
  return {
    colorMode: resolvedMode.value,
    accent: styles.getPropertyValue('--sf-accent').trim(),
    accentContrast: styles.getPropertyValue('--sf-accent-contrast').trim()
  }
}

function bridge(): AdminPageBridgeV1 {
  return {
    apiVersion: ADMIN_MICRO_FRONTEND_API_VERSION,
    extensionId: props.extension.id,
    extensionVersion: props.extension.version,
    locale: locale.value,
    appearance: appearance(),
    page: {
      path: props.page.path,
      label: props.page.label,
      description: props.page.description,
      icon: props.page.icon
    },
    request: <T>(path: string, options: Record<string, unknown> = {}) => request<T>(
      extensionRequestPath(props.extension.id, path),
      options as Parameters<typeof request>[1]
    ),
    toast: (input) => {
      const error = input.kind === 'error'
      toast.add({
        title: input.title,
        description: input.description,
        color: error ? 'error' : 'success',
        icon: error ? 'i-lucide-triangle-alert' : 'i-lucide-check',
        duration: error ? Number.POSITIVE_INFINITY : 10000
      })
    },
    t,
    navigate: async (adminPath: string) => {
      await navigateTo(adminRoutes.path(assertAdminExtensionRelativePath(adminPath)))
    }
  }
}

function loadIsCurrent(generation: number) {
  return !disposed && generation === loadGeneration
}

async function performLoad(generation: number) {
  await dispose()
  if (!loadIsCurrent(generation)) return
  loading.value = false
  failed.value = false
  failureMessage.value = ''
  quarantined.value = false
  if (!import.meta.client || !trusted.value || !component.value || !digest.value) return
  const storage = currentStorage()
  if (storage && contributionFailureState(storage, failureKey.value).quarantined) {
    failed.value = true
    quarantined.value = true
    return
  }
  await nextTick()
  if (!loadIsCurrent(generation) || !target.value) return
  loading.value = true
  try {
    if (component.value.css) {
      const response = await fetch(assetURL('style'), { credentials: 'include' })
      if (!loadIsCurrent(generation)) return
      if (!response.ok || !response.headers.get('content-type')?.toLowerCase().startsWith('text/css')) {
        throw new Error(`Admin page stylesheet request failed (${response.status})`)
      }
      const stylesheet = document.createElement('style')
      stylesheet.dataset.sforumExtension = props.extension.id
      stylesheet.dataset.sforumComponent = component.value.id
      stylesheet.textContent = await response.text()
      if (!loadIsCurrent(generation)) return
      styleElement = stylesheet
      document.head.append(stylesheet)
    }
    const module = await import(/* @vite-ignore */ assetURL('entry')) as AdminPageModuleV1
    if (!loadIsCurrent(generation)) return
    if (module.apiVersion !== ADMIN_MICRO_FRONTEND_API_VERSION || typeof module.mount !== 'function') {
      throw new Error('Admin page component contract mismatch')
    }
    const mountTarget = target.value
    if (!mountTarget) return
    const result = await module.mount(mountTarget, bridge())
    if (result !== undefined && typeof result !== 'function') {
      throw new Error('Admin page component cleanup must be a function')
    }
    if (!loadIsCurrent(generation)) {
      if (typeof result === 'function') await result()
      return
    }
    cleanup = typeof result === 'function' ? result : undefined
    mounted.value = true
    if (storage) clearContributionFailures(storage, failureKey.value)
  } catch (error) {
    if (!loadIsCurrent(generation)) return
    failureMessage.value = error instanceof Error ? error.message : String(error)
    await dispose()
    failed.value = true
    if (storage) quarantined.value = recordContributionFailure(storage, failureKey.value).quarantined
  } finally {
    if (loadIsCurrent(generation)) loading.value = false
  }
}

function load() {
  const generation = ++loadGeneration
  const next = loadQueue.then(() => performLoad(generation), () => performLoad(generation))
  loadQueue = next.catch(() => {})
  return next
}

async function retry() {
  const storage = currentStorage()
  if (storage) clearContributionFailures(storage, failureKey.value)
  await loadTrust()
  await load()
}

watch([() => status.value?.trustState, digest, component, locale, resolvedMode], () => void load())
onMounted(() => {
  disposed = false
  void load()
})
onBeforeUnmount(() => {
  disposed = true
  loadGeneration += 1
  void dispose()
})
</script>

<template>
  <section
    class="min-w-0"
    :data-extension-id="extension.id"
    :data-component-id="component?.id"
    :aria-busy="loading ? 'true' : undefined"
  >
    <div ref="target" v-show="mounted || loading" />
    <UAlert
      v-if="loading && !mounted"
      color="info"
      variant="subtle"
      icon="i-lucide-loader-circle"
      :title="t('admin.extensions.dynamic.componentLoading')"
    />
    <UAlert
      v-else-if="failed"
      color="error"
      variant="subtle"
      icon="i-lucide-triangle-alert"
      :title="quarantined ? t('admin.extensions.dynamic.componentQuarantined') : t('admin.extensions.dynamic.componentLoadFailed')"
      :description="failureMessage || t('admin.extensions.dynamic.componentFallback')"
      :actions="[{ label: t('admin.extensions.dynamic.componentRetry'), onClick: retry }]"
    />
    <template v-else-if="!trusted">
      <UAlert
        color="warning"
        variant="subtle"
        icon="i-lucide-shield-alert"
        :title="t('admin.extensions.dynamic.componentTrustRequired')"
        :description="t('admin.extensions.frontend.prebuiltDescription')"
      />
      <SFAdminFrontendTrustPanel :extension="extension" @changed="loadTrust" />
    </template>
  </section>
</template>
