<script setup lang="ts">
import type { ApiEnvelope } from '~/composables/useApiClient'
import { useAdminTabs } from '~/composables/useAdminTabs'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminRoles'
})

const { t } = useI18n()
const { apiBaseUrl, apiHeaders } = useApiClient()
const search = ref('')
const adminTabs = useAdminTabs()

onMounted(() => {
  adminTabs.openTab('/roles', 'admin.nav.roles', 'i-lucide-shield-check', 'AdminRoles')
})

type Role = {
  id: number
  key: string
  alias: string
  description: string
  isSystem: boolean
  isDefault: boolean
  isDeletable: boolean
  isEnabled: boolean
}

const { data: rolesEnvelope, pending, error, refresh } = await useFetch<ApiEnvelope<Role[]>>(`${apiBaseUrl}/roles`, {
  credentials: 'include',
  headers: apiHeaders(),
  default: () => ({
    code: 200,
    message: 'OK',
    data: []
  })
})

const roles = computed(() => rolesEnvelope.value?.data ?? [])

const columns = computed(() => [
  {
    accessorKey: 'key',
    header: t('admin.roles.key')
  },
  {
    accessorKey: 'alias',
    header: t('admin.roles.alias')
  },
  {
    accessorKey: 'description',
    header: t('admin.roles.description')
  },
  {
    id: 'status',
    header: t('admin.roles.status')
  }
])

const filteredRoles = computed(() => {
  const keyword = search.value.trim().toLowerCase()

  if (!keyword) {
    return roles.value
  }

  return roles.value.filter((role) => {
    return [role.key, role.alias, role.description]
      .some((value) => value.toLowerCase().includes(keyword))
  })
})

useSeoMeta({
  title: t('admin.roles.metaTitle')
})
</script>

<template>
  <!-- 局部标题 -->
  <div class="mb-4">
    <h2 class="text-xl font-bold flex items-center gap-2 text-slate-900 dark:text-zinc-100">
      <UIcon name="i-lucide-shield-check" class="size-5 text-teal-600 dark:text-teal-400" />
      {{ t('admin.roles.title') }}
    </h2>
  </div>

  <!-- 整合刷新按钮与搜索栏的统一 Toolbar -->
  <UDashboardToolbar class="border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 rounded-lg px-4 py-2.5 mb-6 text-slate-500 dark:text-zinc-400">
    <template #left>
      <UInput
        v-model="search"
        icon="i-lucide-search"
        :placeholder="t('admin.roles.searchPlaceholder')"
        class="w-72 max-w-full"
      />
    </template>
    <template #right>
      <div class="flex items-center gap-3">
        <UButton
          color="neutral"
          variant="outline"
          leading-icon="i-lucide-refresh-cw"
          :loading="pending"
          class="border-slate-200 dark:border-zinc-700"
          @click="refresh()"
        >
          {{ t('admin.roles.refresh') }}
        </UButton>
        <UBadge color="neutral" variant="soft" class="border border-slate-200 dark:border-zinc-800 font-medium">
          {{ t('admin.roles.count', { count: filteredRoles.length }) }}
        </UBadge>
      </div>
    </template>
  </UDashboardToolbar>

  <div class="flex flex-col gap-4">
    <UAlert
      v-if="error"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="t('admin.roles.loadFailed')"
    />

    <UCard v-else class="border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100">
      <UTable
        :data="filteredRoles"
        :columns="columns"
        :loading="pending"
        :empty="t('admin.roles.empty')"
        :caption="t('admin.roles.caption')"
        sticky
        class="max-h-[calc(100vh-16rem)]"
      >
        <template #key-cell="{ row }">
          <code class="rounded bg-slate-50 dark:bg-zinc-800 border border-slate-200 dark:border-zinc-700 px-2 py-1 text-xs font-semibold text-slate-900 dark:text-white">
            {{ row.original.key }}
          </code>
        </template>

        <template #alias-cell="{ row }">
          <span class="font-semibold text-slate-900 dark:text-white text-sm">
            {{ row.original.alias }}
          </span>
        </template>

        <template #description-cell="{ row }">
          <span class="text-xs text-slate-500 dark:text-zinc-400">
            {{ row.original.description || t('admin.roles.noDescription') }}
          </span>
        </template>

        <template #status-cell="{ row }">
          <div class="flex flex-wrap gap-1.5">
            <UBadge v-if="row.original.isSystem" color="info" variant="soft">
              {{ t('admin.roles.system') }}
            </UBadge>
            <UBadge v-if="row.original.isDefault" color="success" variant="soft">
              {{ t('admin.roles.default') }}
            </UBadge>
            <UBadge v-if="row.original.isDeletable" color="neutral" variant="outline">
              {{ t('admin.roles.custom') }}
            </UBadge>
          </div>
        </template>
      </UTable>
    </UCard>
  </div>
</template>
