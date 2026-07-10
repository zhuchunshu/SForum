<script setup lang="ts">
import { ADMIN_HOST_INJECTION_KEY } from '@sforum/admin-sdk/internal'
import { markRaw, onErrorCaptured, onMounted, provide, shallowRef, type Component } from 'vue'

import type { AdminComponentMetadata } from '~/runtime/admin-extensions/types'
import {
  clearContributionFailures,
  contributionFailureKey,
  contributionFailureState,
  recordContributionFailure
} from '~/runtime/admin-extensions/quarantine'

const props = defineProps<{
  metadata: AdminComponentMetadata
  context: unknown
}>()

const { locale } = useI18n()
const adminRoutes = useAdminRoutes()
const { releaseId, loaderFor, hostFor } = useAdminExtensionRegistry()
const loadedComponent = shallowRef<Component>()
const loading = ref(false)
const failed = ref(false)
const quarantined = ref(false)
const failureKey = contributionFailureKey(releaseId, props.metadata.extensionId, props.metadata.contributionId)

provide(ADMIN_HOST_INJECTION_KEY, hostFor(props.metadata.extensionId))

const copy = computed(() => locale.value === 'en'
  ? {
      unavailable: 'Extension component unavailable',
      retry: 'Retry',
      manage: 'Manage plugins',
      quarantined: 'Paused after repeated failures'
    }
  : {
      unavailable: '扩展组件暂不可用',
      retry: '重试',
      manage: '管理插件',
      quarantined: '多次失败后已暂停'
    })

function currentStorage() {
  return import.meta.client ? window.sessionStorage : undefined
}

function captureFailure() {
  loadedComponent.value = undefined
  failed.value = true
  const storage = currentStorage()
  if (storage) {
    quarantined.value = recordContributionFailure(storage, failureKey).quarantined
  }
}

async function load() {
  const storage = currentStorage()
  if (!storage) {
    return
  }
  const state = contributionFailureState(storage, failureKey)
  if (state.quarantined) {
    failed.value = true
    quarantined.value = true
    return
  }

  loading.value = true
  failed.value = false
  try {
    const loader = await loaderFor(props.metadata.extensionId, props.metadata.contributionId)
    if (!loader) {
      throw new Error('Admin extension component loader is missing')
    }
    const module = await loader()
    loadedComponent.value = markRaw(module.default)
  } catch {
    captureFailure()
  } finally {
    loading.value = false
  }
}

async function retry() {
  const storage = currentStorage()
  if (storage && quarantined.value) {
    clearContributionFailures(storage, failureKey)
    quarantined.value = false
  }
  await load()
}

onErrorCaptured(() => {
  captureFailure()
  return false
})
onMounted(load)
</script>

<template>
  <div
    class="sf-admin-extension-contribution"
    :data-extension-id="metadata.extensionId"
    :data-contribution-id="metadata.contributionId"
  >
    <component
      :is="loadedComponent"
      v-if="loadedComponent"
      :context="context"
      :options="metadata.options"
      :extension-id="metadata.extensionId"
      :contribution-id="metadata.contributionId"
    />

    <div
      v-else-if="failed"
      class="flex min-h-10 flex-wrap items-center gap-2 border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-200"
      role="alert"
    >
      <UIcon name="i-lucide-triangle-alert" class="size-4 shrink-0" />
      <span>{{ quarantined ? copy.quarantined : copy.unavailable }}</span>
      <code class="text-xs">{{ metadata.extensionId }} / {{ metadata.contributionId }}</code>
      <UButton size="xs" variant="soft" color="error" icon="i-lucide-rotate-cw" @click="retry">
        {{ copy.retry }}
      </UButton>
      <NuxtLink :to="adminRoutes.path('/extensions/plugins')" class="text-xs font-medium underline">
        {{ copy.manage }}
      </NuxtLink>
    </div>

    <div v-else class="min-h-6" aria-hidden="true" :data-loading="loading || undefined" />
  </div>
</template>
