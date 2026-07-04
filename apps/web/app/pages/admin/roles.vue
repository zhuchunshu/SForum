<script setup lang="ts">
import type { ApiEnvelope } from '~/composables/useApiClient'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
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

const { t } = useI18n()
const { apiBaseUrl, apiHeaders } = useApiClient()
const search = ref('')

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
  <UDashboardNavbar :title="t('admin.roles.title')" icon="i-lucide-shield-check">
    <template #right>
      <UButton
        color="neutral"
        variant="outline"
        leading-icon="i-lucide-refresh-cw"
        :loading="pending"
        @click="refresh()"
      >
        {{ t('admin.roles.refresh') }}
      </UButton>
    </template>
  </UDashboardNavbar>

  <UDashboardToolbar>
    <template #left>
      <UInput
        v-model="search"
        icon="i-lucide-search"
        :placeholder="t('admin.roles.searchPlaceholder')"
        class="w-72 max-w-full"
      />
    </template>
    <template #right>
      <UBadge color="neutral" variant="soft">
        {{ t('admin.roles.count', { count: filteredRoles.length }) }}
      </UBadge>
    </template>
  </UDashboardToolbar>

  <div class="flex flex-1 flex-col gap-4 p-4 sm:p-6">
    <UAlert
      v-if="error"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="t('admin.roles.loadFailed')"
    />

    <UCard v-else>
      <UTable
        :data="filteredRoles"
        :columns="columns"
        :loading="pending"
        :empty="t('admin.roles.empty')"
        :caption="t('admin.roles.caption')"
        sticky
        class="max-h-[calc(100vh-13rem)]"
      >
        <template #key-cell="{ row }">
          <code class="rounded bg-muted px-2 py-1 text-xs font-medium text-highlighted">
            {{ row.original.key }}
          </code>
        </template>

        <template #alias-cell="{ row }">
          <span class="font-medium text-highlighted">
            {{ row.original.alias }}
          </span>
        </template>

        <template #description-cell="{ row }">
          <span class="text-muted">
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
