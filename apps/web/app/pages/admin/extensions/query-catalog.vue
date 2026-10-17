<script setup lang="ts">
import SFAdminQueryCatalogTable from '~/components/admin/SFAdminQueryCatalogTable.vue'
import { useAdminPage } from '~/composables/admin/useAdminPage'
import {
  queryCatalogDigestShort,
  useAdminQueryCatalog,
  type QueryCatalogSnapshot
} from '~/composables/admin/useAdminQueryCatalog'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminExtensionQueryCatalog'
})

const { t } = useI18n()
const toast = useToast()
const adminPage = useAdminPage('/extensions/query-catalog')
const { load } = useAdminQueryCatalog()

const pending = ref(false)
const error = ref('')
const snapshot = ref<QueryCatalogSnapshot | null>(null)

const stableCount = computed(() =>
  snapshot.value ? snapshot.value.queries.filter(item => !item.queryId.startsWith('core.query.shortcode.')).length : 0
)

async function loadCatalog() {
  pending.value = true
  error.value = ''
  try {
    snapshot.value = await load()
  } catch (cause) {
    error.value = apiErrorMessage(cause) || t('admin.extensions.queryCatalog.loadFailed')
  } finally {
    pending.value = false
  }
}

async function refresh() {
  await loadCatalog()
  if (!error.value) {
    toast.add({ title: t('admin.extensions.queryCatalog.refreshed'), color: 'success' })
  }
}

useSeoMeta({ title: t('admin.extensions.queryCatalog.metaTitle') })
void loadCatalog()
</script>

<template>
  <div data-testid="admin-query-catalog" class="min-w-0 shrink-0">
    <div class="mb-4 flex flex-col gap-1">
      <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
        <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
        {{ t('admin.extensions.queryCatalog.title') }}
      </h2>
      <p class="max-w-4xl text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.extensions.queryCatalog.intro') }}
      </p>
    </div>

    <UDashboardToolbar class="mb-5 rounded-lg border border-slate-200 bg-white px-4 py-2.5 dark:border-zinc-800 dark:bg-zinc-900">
      <template #left>
        <div class="flex min-w-0 flex-wrap items-center gap-2 text-sm text-slate-600 dark:text-zinc-300">
          <span v-if="snapshot">
            {{ t('admin.extensions.queryCatalog.sealed', { digest: queryCatalogDigestShort(snapshot.artifactDigest) }) }}
            <span class="mx-1 text-slate-300 dark:text-zinc-700">·</span>
          </span>
          <span>{{ t('admin.extensions.queryCatalog.totalQueries', { count: snapshot?.total ?? 0 }) }}</span>
          <UBadge color="primary" variant="subtle">
            {{ t('admin.extensions.queryCatalog.stableQueries', { count: stableCount }) }}
          </UBadge>
          <UBadge v-if="snapshot" color="neutral" variant="subtle">
            {{ t('admin.extensions.queryCatalog.costMaximum', { cost: snapshot.costMaximum }) }}
          </UBadge>
        </div>
      </template>
      <template #right>
        <UButton
          icon="i-lucide-rotate-cw"
          color="neutral"
          variant="subtle"
          size="sm"
          :loading="pending"
          @click="refresh"
        >
          {{ t('admin.extensions.refresh') }}
        </UButton>
      </template>
    </UDashboardToolbar>

    <SFAlert v-if="error" variant="danger" :title="error" class="mb-4" />

    <div v-if="pending && !snapshot" class="space-y-3" aria-busy="true">
      <USkeleton v-for="index in 3" :key="index" class="h-24 w-full rounded-lg" />
    </div>

    <SFEmptyState
      v-else-if="!error && snapshot && snapshot.queries.length === 0"
      icon-label="QRY"
      :title="t('admin.extensions.queryCatalog.emptyTitle')"
      :description="t('admin.extensions.queryCatalog.emptyDescription')"
    />

    <div v-else-if="!error && snapshot" class="space-y-3">
      <SFAdminQueryCatalogTable :queries="snapshot.queries" />
      <p class="px-1 text-xs text-slate-500 dark:text-zinc-400">
        {{ t('admin.extensions.queryCatalog.hint') }}
      </p>
    </div>
  </div>
</template>
