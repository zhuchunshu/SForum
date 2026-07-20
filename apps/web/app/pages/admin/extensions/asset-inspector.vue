<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import {
  type AssetInspectorSnapshot,
  useAdminAssetInspector
} from '~/composables/useAdminAssetInspector'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminExtensionAssetInspector' })

const { t } = useI18n()
const toast = useToast()
useAdminPage('/extensions/asset-inspector')
const { inspect } = useAdminAssetInspector()

const limit = ref(50)
const pending = ref(true)
const error = ref('')
const snapshot = ref<AssetInspectorSnapshot | null>(null)
const limitItems = [25, 50, 100, 200].map(value => ({ label: String(value), value }))

function shortDigest(value: string) {
  return value.length > 16 ? `${value.slice(0, 12)}…${value.slice(-4)}` : value
}

async function load() {
  pending.value = true
  error.value = ''
  try {
    snapshot.value = await inspect(limit.value)
  } catch (err) {
    error.value = apiErrorMessage(err)
    snapshot.value = null
  } finally {
    pending.value = false
  }
}

async function refresh() {
  await load()
  if (!error.value) {
    toast.add({ title: t('admin.extensions.assetInspector.refreshed'), color: 'primary', duration: 10000 })
  }
}

onMounted(load)
watch(limit, load)
useSeoMeta({ title: t('admin.extensions.assetInspector.metaTitle') })
</script>

<template>
  <div data-testid="admin-asset-inspector" class="min-w-0 w-full max-w-full">
    <div class="mb-6 flex min-w-0 flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
      <div class="min-w-0">
        <h1 class="truncate text-xl font-semibold text-slate-900 dark:text-white">
          {{ t('admin.extensions.assetInspector.title') }}
        </h1>
        <p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">
          {{ t('admin.extensions.assetInspector.intro') }}
        </p>
      </div>
      <div class="flex shrink-0 items-center gap-2">
        <USelect
          v-model="limit"
          :items="limitItems"
          value-key="value"
          class="w-28"
          data-testid="asset-inspector-limit"
        />
        <UButton
          icon="i-tabler-refresh"
          color="neutral"
          variant="soft"
          :loading="pending"
          data-testid="asset-inspector-refresh"
          @click="refresh"
        >
          {{ t('admin.extensions.assetInspector.refresh') }}
        </UButton>
      </div>
    </div>

    <UAlert
      v-if="error"
      color="error"
      variant="subtle"
      icon="i-tabler-alert-triangle"
      :title="error"
      class="mb-4"
      data-testid="asset-inspector-error"
      :close-button="{ icon: 'i-tabler-x', color: 'neutral', variant: 'link' }"
      @close="error = ''"
    />

    <div
      v-if="pending && !snapshot"
      class="text-sm text-slate-500 dark:text-zinc-400"
      data-testid="asset-inspector-loading"
    >
      {{ t('admin.extensions.assetInspector.loading') }}
    </div>

    <template v-if="snapshot">
      <div class="mb-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <div class="rounded-lg border border-slate-200 p-3 dark:border-zinc-800">
          <div class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.assetInspector.revision') }}
          </div>
          <div class="text-lg font-semibold text-slate-900 dark:text-white">{{ snapshot.revision }}</div>
        </div>
        <div class="rounded-lg border border-slate-200 p-3 dark:border-zinc-800">
          <div class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.assetInspector.publications') }}
          </div>
          <div class="text-lg font-semibold text-slate-900 dark:text-white">{{ snapshot.publicationCount }}</div>
        </div>
        <div class="rounded-lg border border-slate-200 p-3 dark:border-zinc-800">
          <div class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.assetInspector.assets') }}
          </div>
          <div class="text-lg font-semibold text-slate-900 dark:text-white">{{ snapshot.assetCount }}</div>
        </div>
        <div class="rounded-lg border border-slate-200 p-3 dark:border-zinc-800">
          <div class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.assetInspector.digest') }}
          </div>
          <div class="break-all text-sm font-semibold text-slate-900 dark:text-white" data-testid="asset-inspector-digest">
            {{ shortDigest(snapshot.digest) }}
          </div>
        </div>
      </div>

      <p class="mb-4 text-xs text-slate-500 dark:text-zinc-400">
        {{ t('admin.extensions.assetInspector.redactionHint') }}
      </p>

      <div class="rounded-lg border border-slate-200 dark:border-zinc-800">
        <div class="border-b border-slate-200 px-4 py-3 font-medium dark:border-zinc-800">
          {{ t('admin.extensions.assetInspector.publicationsTitle') }}
          ({{ snapshot.publications.length }}/{{ snapshot.publicationCount }})
        </div>
        <div class="p-4">
          <p
            v-if="!snapshot.publications.length"
            class="text-sm text-slate-500 dark:text-zinc-400"
            data-testid="asset-inspector-empty-publications"
          >
            {{ t('admin.extensions.assetInspector.noPublications') }}
          </p>
          <div v-else class="space-y-4" data-testid="asset-inspector-publications">
            <div
              v-for="publication in snapshot.publications"
              :key="publication.extensionId + publication.packageDigest"
              class="rounded-md border border-slate-100 p-3 dark:border-zinc-800"
            >
              <div class="mb-2 flex min-w-0 flex-wrap items-center gap-2">
                <span class="break-all font-medium text-slate-900 dark:text-white">
                  {{ publication.extensionId }}
                </span>
                <UBadge color="neutral" variant="subtle">{{ publication.extensionVersion }}</UBadge>
                <UBadge color="neutral" variant="outline">{{ publication.ownerKind }}</UBadge>
              </div>
              <div class="mb-2 break-all text-xs text-slate-500 dark:text-zinc-400">
                {{ t('admin.extensions.assetInspector.packageDigest') }}:
                {{ shortDigest(publication.packageDigest) }}
              </div>
              <div class="overflow-x-auto">
                <table class="min-w-full text-left text-xs">
                  <thead class="text-slate-500 dark:text-zinc-400">
                    <tr>
                      <th class="py-1 pr-3 font-medium">{{ t('admin.extensions.assetInspector.handle') }}</th>
                      <th class="py-1 pr-3 font-medium">{{ t('admin.extensions.assetInspector.type') }}</th>
                      <th class="py-1 pr-3 font-medium">{{ t('admin.extensions.assetInspector.path') }}</th>
                      <th class="py-1 pr-3 font-medium">{{ t('admin.extensions.assetInspector.loadingMode') }}</th>
                      <th class="py-1 font-medium">{{ t('admin.extensions.assetInspector.scope') }}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="asset in publication.assets"
                      :key="asset.handle"
                      class="border-t border-slate-100 dark:border-zinc-800"
                    >
                      <td class="break-all py-1.5 pr-3 font-medium text-slate-800 dark:text-zinc-100">{{ asset.handle }}</td>
                      <td class="py-1.5 pr-3">
                        <UBadge color="neutral" variant="subtle">{{ asset.type }}</UBadge>
                        <UBadge v-if="asset.module" color="primary" variant="subtle" class="ml-1">module</UBadge>
                      </td>
                      <td class="break-all py-1.5 pr-3 text-slate-600 dark:text-zinc-300">{{ asset.path || '—' }}</td>
                      <td class="py-1.5 pr-3 text-slate-600 dark:text-zinc-300">{{ asset.loading || '—' }}</td>
                      <td class="break-all py-1.5 text-slate-600 dark:text-zinc-300">
                        {{ asset.scope.length ? asset.scope.join(', ') : '—' }}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
