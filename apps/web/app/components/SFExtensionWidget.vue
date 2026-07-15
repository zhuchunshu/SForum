<script setup lang="ts">
import type {
  PublicFrontendBridgeV1,
  PublicFrontendCleanup,
  PublicFrontendComponentDescriptor
} from '~/runtime/public-extensions/types'
import {
  PUBLIC_FRONTEND_API_VERSION,
  PUBLIC_FRONTEND_DESCRIPTOR_TIMEOUT_MS,
  PUBLIC_FRONTEND_TRUST_NOTICE,
  assertPublicNavigationPath,
  parsePublicFrontendDescriptor,
  publicComponentPath,
  publicExtensionRequestPath,
  publicFrontendRequestOptions
} from '~/runtime/public-extensions/types'
import { loadPublicFrontendRelease } from '~/runtime/public-extensions/assets'
import { createPublicFrontendLeaseMonitor } from '~/runtime/public-extensions/lease'
import { mountPublicFrontendModule } from '~/runtime/public-extensions/mount'
import {
  clearPublicContributionFailures,
  publicContributionFailureKey,
  publicContributionFailureState,
  recordPublicContributionFailure
} from '~/runtime/public-extensions/quarantine'

const props = withDefaults(defineProps<{
  extensionId?: string
  componentId?: string
  widgetId?: string
  name?: string
  componentProps?: Readonly<Record<string, unknown>>
}>(), {
  extensionId: '',
  componentId: '',
  widgetId: '',
  name: '',
  componentProps: () => ({})
})

const { apiBaseUrl, request } = useApiClient()
const { locale } = useI18n()
const colorMode = useColorMode()
const route = useRoute()
const target = ref<HTMLElement>()
const ssrRoot = ref<HTMLElement>()
const state = ref<'idle' | 'loading' | 'mounted' | 'fallback' | 'quarantined'>('idle')
const descriptor = shallowRef<PublicFrontendComponentDescriptor>()
let cleanup: PublicFrontendCleanup | undefined
let releaseAssets: (() => Promise<void>) | undefined
let loadQueue: Promise<void> = Promise.resolve()
let destroyed = false

const resolvedComponentId = computed(() => props.componentId || props.widgetId || props.name)
const failureKey = computed(() => descriptor.value
  ? publicContributionFailureKey(descriptor.value.impactDigest, descriptor.value.extensionId, descriptor.value.componentId)
  : '')

function currentStorage() {
  return import.meta.client ? window.sessionStorage : undefined
}

async function dispose() {
  const currentCleanup = cleanup
  const currentRelease = releaseAssets
  cleanup = undefined
  releaseAssets = undefined
  try {
    await currentCleanup?.()
  } catch {
    // 第三方 cleanup 失败不能阻止宿主恢复 L1/SSR 回退。
  }
  try {
    if (currentRelease) await currentRelease()
  } catch {
    // CSS lease cleanup is best effort; the ref-count owner remains Host code.
  }
  target.value?.replaceChildren()
}

function appearance() {
  const styles = getComputedStyle(document.documentElement)
  return {
    colorMode: colorMode.value === 'dark' ? 'dark' as const : 'light' as const,
    accent: styles.getPropertyValue('--sf-accent').trim(),
    accentContrast: styles.getPropertyValue('--sf-accent-contrast').trim()
  }
}

function enqueue(task: () => Promise<void>) {
  const next = loadQueue.then(task, task)
  loadQueue = next.catch(() => {})
  return next
}

const leaseMonitor = createPublicFrontendLeaseMonitor({
  read: current => request<unknown>(
    publicComponentPath(current.extensionId, current.componentId),
    publicFrontendRequestOptions(current, {
      timeout: PUBLIC_FRONTEND_DESCRIPTOR_TIMEOUT_MS
    }) as Parameters<typeof request>[1]
  ),
  onChanged: () => load(),
  onUnavailable: () => enqueue(async () => {
    await dispose()
    state.value = 'fallback'
  })
})

function bridge(current: PublicFrontendComponentDescriptor): PublicFrontendBridgeV1 {
  if (!ssrRoot.value) throw new Error('public component SSR root is unavailable')
  return Object.freeze({
    apiVersion: PUBLIC_FRONTEND_API_VERSION,
    trust: PUBLIC_FRONTEND_TRUST_NOTICE,
    extensionId: current.extensionId,
    extensionVersion: current.extensionVersion,
    packageDigest: current.packageDigest,
    impactDigest: current.impactDigest,
    componentId: current.componentId,
    locale: locale.value,
    appearance: Object.freeze(appearance()),
    props: Object.freeze({ ...props.componentProps }),
    ssrRoot: ssrRoot.value,
    request: async <T>(path: string, options: Record<string, unknown> = {}) => {
      try {
        return await request<T>(
          publicExtensionRequestPath(current.extensionId, path),
          publicFrontendRequestOptions(current, options) as Parameters<typeof request>[1]
        )
      } catch (error) {
        // A rejected exact request may be the first observable revoke/upgrade
        // signal; revalidate immediately instead of waiting for the lease tick.
        void leaseMonitor.trigger()
        throw error
      }
    },
    navigate: async (path: string) => {
      await navigateTo(assertPublicNavigationPath(path))
    }
  })
}

async function performLoad() {
  leaseMonitor.stop()
  await dispose()
  descriptor.value = undefined
  state.value = 'fallback'
  const extensionId = props.extensionId.trim()
  const componentId = resolvedComponentId.value.trim()
  if (!import.meta.client || destroyed || !extensionId || !componentId || !target.value || !ssrRoot.value) return
  state.value = 'loading'
  try {
    const raw = await request<unknown>(publicComponentPath(extensionId, componentId), {
      timeout: PUBLIC_FRONTEND_DESCRIPTOR_TIMEOUT_MS
    })
    const current = parsePublicFrontendDescriptor(raw, extensionId, componentId)
    descriptor.value = current
    const storage = currentStorage()
    const key = failureKey.value
    if (storage && publicContributionFailureState(storage, key).quarantined) {
      state.value = 'quarantined'
      // 隔离只绑定当前 impact digest；继续租约检查，升级后的新产物可自动恢复。
      leaseMonitor.start(current)
      return
    }
    const loaded = await loadPublicFrontendRelease(current, apiBaseUrl || '')
    if (destroyed || !target.value) {
      await loaded.release()
      return
    }
    releaseAssets = loaded.release
    const currentTarget = target.value
    const currentBridge = bridge(current)
    cleanup = await mountPublicFrontendModule(loaded.module, currentTarget, currentBridge)
    state.value = 'mounted'
    leaseMonitor.start(current)
    if (storage) clearPublicContributionFailures(storage, key)
  } catch {
    await dispose()
    const storage = currentStorage()
    if (storage && failureKey.value) {
      const failure = recordPublicContributionFailure(storage, failureKey.value)
      state.value = failure.quarantined ? 'quarantined' : 'fallback'
    } else {
      state.value = 'fallback'
    }
  }
}

function load() {
  leaseMonitor.stop()
  return enqueue(performLoad)
}

watch([
  () => props.extensionId,
  resolvedComponentId,
  () => props.componentProps,
  () => locale.value,
  () => colorMode.value,
  () => route.fullPath
], () => void load(), { deep: true })

function revalidateVisibleLease() {
  if (document.visibilityState === 'visible') void leaseMonitor.trigger()
}

onMounted(() => {
  window.addEventListener('focus', revalidateVisibleLease)
  document.addEventListener('visibilitychange', revalidateVisibleLease)
  void load()
})
onBeforeUnmount(() => {
  destroyed = true
  leaseMonitor.stop()
  window.removeEventListener('focus', revalidateVisibleLease)
  document.removeEventListener('visibilitychange', revalidateVisibleLease)
  void loadQueue.then(dispose, dispose)
})

defineExpose({ reload: load })
</script>

<template>
  <div
    class="sf-extension-widget"
    :data-extension-id="extensionId"
    :data-component-id="resolvedComponentId"
    :data-l2-state="state"
    :aria-busy="state === 'loading' ? 'true' : undefined"
  >
    <div
      v-show="state !== 'mounted'"
      ref="ssrRoot"
      class="sf-extension-widget__fallback"
      data-l2-fallback=""
    >
      <slot />
    </div>
    <div
      ref="target"
      class="sf-extension-widget__target"
      data-l2-target=""
    />
  </div>
</template>
