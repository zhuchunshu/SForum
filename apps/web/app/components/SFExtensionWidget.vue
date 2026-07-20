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
const { t, locale } = useI18n()
const colorMode = useColorMode()
const route = useRoute()
const target = ref<HTMLElement>()
const ssrRoot = ref<HTMLElement>()
const state = ref<'idle' | 'loading' | 'mounted' | 'fallback' | 'quarantined'>('idle')
const descriptor = shallowRef<PublicFrontendComponentDescriptor>()
// 操作者可关闭当前页面上的披露；刷新后仍会再显示（不写 localStorage，避免隐藏安全提示）。
const honestyDismissed = ref(false)
let cleanup: PublicFrontendCleanup | undefined
let releaseAssets: (() => Promise<void>) | undefined
let loadQueue: Promise<void> = Promise.resolve()
let destroyed = false

const resolvedComponentId = computed(() => props.componentId || props.widgetId || props.name)
const showHonesty = computed(() =>
  state.value === 'mounted'
  && !honestyDismissed.value
  && Boolean(descriptor.value?.extensionId)
)
const honestyTitle = computed(() => t('public.extensions.l2Honesty.title'))
const honestyBody = computed(() => t('public.extensions.l2Honesty.body', {
  name: descriptor.value?.extensionId || resolvedComponentId.value,
  version: descriptor.value?.extensionVersion || ''
}))
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
    :data-l2-trust="descriptor ? PUBLIC_FRONTEND_TRUST_NOTICE : undefined"
    :aria-busy="state === 'loading' ? 'true' : undefined"
  >
    <div
      v-if="showHonesty"
      class="sf-extension-widget__honesty mb-2 rounded-md border border-amber-200/80 bg-amber-50 px-3 py-2 text-xs text-amber-950 dark:border-amber-900/60 dark:bg-amber-950/40 dark:text-amber-100"
      data-testid="public-l2-honesty"
      role="note"
    >
      <div class="flex items-start gap-2">
        <UIcon name="i-tabler-shield-exclamation" class="mt-0.5 size-4 shrink-0" />
        <div class="min-w-0 flex-1">
          <div class="font-medium">
            {{ honestyTitle }}
          </div>
          <p class="mt-0.5 break-all text-amber-900/90 dark:text-amber-100/90">
            {{ honestyBody }}
          </p>
        </div>
        <UButton
          icon="i-tabler-x"
          color="neutral"
          variant="link"
          size="xs"
          class="shrink-0"
          data-testid="public-l2-honesty-dismiss"
          :aria-label="t('public.extensions.l2Honesty.dismiss')"
          @click="() => { honestyDismissed = true }"
        />
      </div>
    </div>
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
