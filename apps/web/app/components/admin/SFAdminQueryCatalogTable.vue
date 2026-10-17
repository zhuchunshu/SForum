<script setup lang="ts">
import {
  isQueryCatalogHostInternal,
  queryCatalogDigestShort,
  type QueryCatalogEntry
} from '~/composables/admin/useAdminQueryCatalog'

const props = defineProps<{
  queries: QueryCatalogEntry[]
}>()

const { t } = useI18n()

function listSummary(values: string[]) {
  return values.length > 0 ? values.join(', ') : '—'
}
</script>

<template>
  <div
    :data-testid="'query-catalog-table'"
    class="overflow-x-auto rounded-lg border border-(--ui-border) bg-(--ui-bg)"
  >
    <table class="w-full min-w-[880px] text-left text-sm">
      <thead>
        <tr class="border-b border-(--ui-border) text-xs uppercase tracking-wide text-muted">
          <th class="px-4 py-2">{{ t('admin.extensions.queryCatalog.columns.query') }}</th>
          <th class="px-3 py-2">{{ t('admin.extensions.queryCatalog.columns.host') }}</th>
          <th class="px-3 py-2">{{ t('admin.extensions.queryCatalog.columns.shape') }}</th>
          <th class="px-3 py-2">{{ t('admin.extensions.queryCatalog.columns.pagination') }}</th>
          <th class="px-3 py-2">{{ t('admin.extensions.queryCatalog.columns.policy') }}</th>
          <th class="px-4 py-2">{{ t('admin.extensions.queryCatalog.columns.schema') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="entry in props.queries"
          :key="entry.queryId"
          class="border-b border-(--ui-border) last:border-b-0 align-top"
        >
          <td class="px-4 py-3">
            <p class="font-mono text-xs font-semibold break-all">{{ entry.queryId }}</p>
            <p class="mt-1 text-xs text-muted">{{ entry.entity }}</p>
            <UBadge
              v-if="isQueryCatalogHostInternal(entry)"
              class="mt-1"
              color="neutral"
              variant="subtle"
              size="sm"
            >
              {{ t('admin.extensions.queryCatalog.hostInternal') }}
            </UBadge>
          </td>
          <td class="px-3 py-3">
            <p class="font-mono text-xs break-all">{{ entry.hostQueryId }}</p>
            <p class="mt-1 font-mono text-xs text-muted">plan@{{ entry.hostPlanVersion }}</p>
          </td>
          <td class="px-3 py-3">
            <p class="text-xs">
              <span class="font-medium">{{ t('admin.extensions.queryCatalog.fields') }}:</span>
              {{ entry.fields.length }} — {{ listSummary(entry.fields) }}
            </p>
            <p class="mt-1 text-xs">
              <span class="font-medium">{{ t('admin.extensions.queryCatalog.filters') }}:</span>
              {{ entry.filters.length > 0 ? listSummary(entry.filters) : '—' }}
            </p>
            <p class="mt-1 text-xs">
              <span class="font-medium">{{ t('admin.extensions.queryCatalog.sorts') }}:</span>
              {{ entry.sort.length > 0 ? listSummary(entry.sort) : '—' }}
            </p>
          </td>
          <td class="px-3 py-3">
            <UBadge
              :color="entry.pagination === 'none' ? 'neutral' : 'info'"
              variant="subtle"
              size="sm"
            >
              {{ entry.pagination }}
            </UBadge>
          </td>
          <td class="px-3 py-3">
            <p class="font-mono text-xs break-all">{{ entry.permissionPolicy }}</p>
          </td>
          <td class="px-4 py-3">
            <p class="font-mono text-xs break-all">{{ entry.resultSchema }}</p>
            <p class="mt-1 font-mono text-xs text-muted">
              {{ queryCatalogDigestShort(entry.resultSchemaDigest) }}
            </p>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
