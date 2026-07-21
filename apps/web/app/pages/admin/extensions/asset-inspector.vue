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
const adminPage = useAdminPage('/extensions/asset-inspector')
const { inspect } = useAdminAssetInspector()

const limit = ref(100)
const pending = ref(true)
const error = ref('')
const snapshot = ref<AssetInspectorSnapshot | null>(null)
const limitItems = [25, 50, 100, 200].map(value => ({ label: String(value), value }))

function shortDigest(value: string) {
  return value.length > 16 ? `${value.slice(0, 12)}…${value.slice(-4)}` : value
}

function formatCount(value: number) {
  return new Intl.NumberFormat().format(value)
}

function mapLoadError(cause: unknown) {
  const fromApi = apiErrorMessage(cause)
  if (fromApi) return fromApi
  if (cause instanceof Error && cause.message.trim()) return cause.message
  return t('admin.extensions.assetInspector.loadFailed')
}

async function load(manual = false) {
  pending.value = true
  error.value = ''
  try {
    snapshot.value = await inspect(limit.value)
    if (manual) {
      toast.add({
        color: 'primary',
        icon: 'i-lucide-check',
        title: t('admin.extensions.assetInspector.refreshSuccess'),
        duration: 10000
      })
    }
  } catch (cause) {
    snapshot.value = null
    error.value = mapLoadError(cause)
  } finally {
    pending.value = false
  }
}

watch(limit, () => void load())
useSeoMeta({ title: t('admin.extensions.assetInspector.metaTitle') })
void load()
</script>

<template>
  <div data-testid="admin-asset-inspector" class="min-w-0 w-full max-w-full">
    <div class="mb-4 flex min-w-0 flex-col gap-1">
      <h2 class="flex min-w-0 items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
        <UIcon :name="adminPage.icon" class="size-5 shrink-0 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
        <span class="min-w-0 truncate">{{ t('admin.extensions.assetInspector.title') }}</span>
      </h2>
      <p class="max-w-4xl text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.extensions.assetInspector.intro') }}
      </p>
    </div>

    <UDashboardToolbar class="mb-4 rounded-lg border border-slate-200 bg-white px-3 py-2.5 dark:border-zinc-800 dark:bg-zinc-900">
      <template #left>
        <div class="flex min-w-0 flex-wrap items-center gap-2 text-sm text-slate-600 dark:text-zinc-300">
          <UIcon name="i-lucide-package" class="size-4 shrink-0" />
          <template v-if="snapshot">
            <UBadge color="neutral" variant="subtle">
              {{ t('admin.extensions.assetInspector.revisionBadge', { revision: snapshot.revision }) }}
            </UBadge>
            <span>{{ t('admin.extensions.assetInspector.publicationCount', { count: snapshot.publicationCount }) }}</span>
            <span>{{ t('admin.extensions.assetInspector.assetCountLabel', { count: snapshot.assetCount }) }}</span>
          </template>
        </div>
      </template>
      <template #right>
        <div class="flex min-w-0 items-center gap-2">
          <label class="text-xs text-slate-500 dark:text-zinc-400" for="asset-inspector-limit">
            {{ t('admin.extensions.assetInspector.limit') }}
          </label>
          <USelect
            id="asset-inspector-limit"
            v-model="limit"
            :items="limitItems"
            value-key="value"
            label-key="label"
            class="w-20"
            data-testid="asset-inspector-limit"
          />
          <UButton
            icon="i-lucide-rotate-cw"
            color="neutral"
            variant="subtle"
            :loading="pending"
            data-testid="asset-inspector-refresh"
            @click="load(true)"
          >
            {{ t('admin.extensions.refresh') }}
          </UButton>
        </div>
      </template>
    </UDashboardToolbar>

    <SFAlert
      v-if="error"
      variant="danger"
      :title="error"
      class="mb-4"
      data-testid="asset-inspector-error"
    />

    <div v-if="pending" class="space-y-3" aria-busy="true" data-testid="asset-inspector-loading">
      <USkeleton class="h-24 w-full rounded-lg" />
      <USkeleton class="h-44 w-full rounded-lg" />
      <USkeleton class="h-52 w-full rounded-lg" />
    </div>

    <template v-else-if="snapshot">
      <section class="mb-4 overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex min-w-0 flex-col gap-2 border-b border-slate-200 px-3 py-2.5 dark:border-zinc-800 sm:flex-row sm:items-center sm:justify-between">
          <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
            {{ t('admin.extensions.assetInspector.summaryTitle') }}
          </h3>
          <p class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.assetInspector.redactionHint') }}
          </p>
        </div>
        <dl class="grid grid-cols-2 divide-x divide-y divide-slate-200 dark:divide-zinc-800 sm:grid-cols-2 lg:grid-cols-4">
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.assetInspector.fields.revision') }}</dt>
            <dd class="mt-1 text-lg font-semibold text-slate-900 dark:text-zinc-100">{{ formatCount(snapshot.revision) }}</dd>
          </div>
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.assetInspector.fields.publications') }}</dt>
            <dd class="mt-1 text-lg font-semibold text-slate-900 dark:text-zinc-100">{{ formatCount(snapshot.publicationCount) }}</dd>
          </div>
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.assetInspector.fields.assets') }}</dt>
            <dd class="mt-1 text-lg font-semibold text-slate-900 dark:text-zinc-100">{{ formatCount(snapshot.assetCount) }}</dd>
          </div>
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.assetInspector.fields.digest') }}</dt>
            <dd
              class="mt-1 break-all font-mono text-sm font-semibold text-slate-900 dark:text-zinc-100"
              data-testid="asset-inspector-digest"
              :title="snapshot.digest"
            >
              {{ shortDigest(snapshot.digest) }}
            </dd>
          </div>
        </dl>
        <dl class="grid gap-x-4 gap-y-2 border-t border-slate-200 px-3 py-3 text-xs dark:border-zinc-800 sm:grid-cols-2">
          <div class="min-w-0">
            <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.assetInspector.fields.schema') }}</dt>
            <dd class="break-all font-mono text-slate-700 dark:text-zinc-200">{{ snapshot.schemaVersion }}</dd>
          </div>
          <div class="min-w-0">
            <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.assetInspector.fields.shown') }}</dt>
            <dd class="text-slate-700 dark:text-zinc-200">
              {{ snapshot.publications.length }} / {{ snapshot.publicationCount }}
            </dd>
          </div>
        </dl>
      </section>

      <section class="overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex items-center justify-between border-b border-slate-200 px-3 py-2.5 dark:border-zinc-800">
          <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
            {{ t('admin.extensions.assetInspector.publicationsTitle') }}
          </h3>
          <UBadge color="neutral" variant="subtle">{{ snapshot.publications.length }}</UBadge>
        </div>
        <SFEmptyState
          v-if="snapshot.publications.length === 0"
          icon-label="AST"
          :title="t('admin.extensions.assetInspector.publicationsEmptyTitle')"
          :description="t('admin.extensions.assetInspector.publicationsEmptyDescription')"
          class="m-6"
          data-testid="asset-inspector-empty-publications"
        />
        <div v-else class="divide-y divide-slate-200 dark:divide-zinc-800" data-testid="asset-inspector-publications">
          <div
            v-for="publication in snapshot.publications"
            :key="publication.extensionId + publication.packageDigest"
            class="min-w-0 px-3 py-3"
          >
            <div class="mb-2 flex min-w-0 flex-wrap items-center gap-2">
              <span class="break-all font-medium text-slate-900 dark:text-zinc-100">
                {{ publication.extensionId }}
              </span>
              <UBadge color="neutral" variant="subtle">{{ publication.extensionVersion }}</UBadge>
              <UBadge color="neutral" variant="outline">{{ publication.ownerKind }}</UBadge>
              <span
                class="break-all font-mono text-xs text-slate-500 dark:text-zinc-400"
                :title="publication.packageDigest"
              >
                {{ t('admin.extensions.assetInspector.packageDigest') }}:
                {{ shortDigest(publication.packageDigest) }}
              </span>
            </div>
            <div class="overflow-x-auto">
              <table class="min-w-[760px] w-full text-left text-xs">
                <thead class="bg-slate-50 text-slate-500 dark:bg-zinc-950/70 dark:text-zinc-400">
                  <tr>
                    <th class="px-3 py-2 font-medium">{{ t('admin.extensions.assetInspector.handle') }}</th>
                    <th class="px-3 py-2 font-medium">{{ t('admin.extensions.assetInspector.type') }}</th>
                    <th class="px-3 py-2 font-medium">{{ t('admin.extensions.assetInspector.path') }}</th>
                    <th class="px-3 py-2 font-medium">{{ t('admin.extensions.assetInspector.loadingMode') }}</th>
                    <th class="px-3 py-2 font-medium">{{ t('admin.extensions.assetInspector.scope') }}</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
                  <tr v-for="asset in publication.assets" :key="asset.handle">
                    <td class="break-all px-3 py-2 font-mono font-medium text-slate-800 dark:text-zinc-100">
                      {{ asset.handle }}
                    </td>
                    <td class="px-3 py-2">
                      <UBadge color="neutral" variant="subtle">{{ asset.type }}</UBadge>
                      <UBadge v-if="asset.module" color="primary" variant="subtle" class="ml-1">
                        module
                      </UBadge>
                    </td>
                    <td class="break-all px-3 py-2 text-slate-600 dark:text-zinc-300">{{ asset.path || '—' }}</td>
                    <td class="px-3 py-2 text-slate-600 dark:text-zinc-300">{{ asset.loading || '—' }}</td>
                    <td class="break-all px-3 py-2 text-slate-600 dark:text-zinc-300">
                      {{ asset.scope.length ? asset.scope.join(', ') : '—' }}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>
