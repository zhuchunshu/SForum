<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminTabs } from '~/composables/useAdminTabs'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminPermissions'
})

type Permission = {
  key: string
  module: string
  description: string
}

type Role = {
  id: number
  key: string
  alias: string
  description: string
  isSystem: boolean
  isDefault: boolean
  isDeletable: boolean
  isEnabled: boolean
  permissionKeys: string[]
}

type RolePermissionSet = {
  roleKey: string
  permissionKeys: string[]
}

type PermissionMatrix = {
  permissions: Permission[]
  roles: RolePermissionSet[]
}

const { t } = useI18n()
const { request } = useApiClient()
const adminTabs = useAdminTabs()

const roles = ref<Role[]>([])
const matrix = ref<PermissionMatrix>({ permissions: [], roles: [] })
const pending = ref(false)
const errorMessage = ref('')

onMounted(() => {
  adminTabs.openTab('/permissions', 'admin.nav.permissionManagement', 'i-lucide-shield-check', 'AdminPermissions')
  void loadData()
})

const groupedPermissions = computed(() => {
  const groups = new Map<string, Permission[]>()
  for (const permission of matrix.value.permissions) {
    const group = groups.get(permission.module) ?? []
    group.push(permission)
    groups.set(permission.module, group)
  }
  return Array.from(groups.entries()).map(([module, items]) => ({ module, items }))
})

const rolePermissionMap = computed(() => {
  const map = new Map<string, Set<string>>()
  for (const role of matrix.value.roles) {
    map.set(role.roleKey, new Set(role.permissionKeys))
  }
  return map
})

useSeoMeta({
  title: t('admin.permissions.metaTitle')
})

async function loadData() {
  pending.value = true
  errorMessage.value = ''
  try {
    const [roleItems, matrixData] = await Promise.all([
      request<Role[]>('/roles'),
      request<PermissionMatrix>('/permissions/matrix')
    ])
    roles.value = roleItems
    matrix.value = matrixData
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.permissions.loadFailed')
  } finally {
    pending.value = false
  }
}

function roleHasPermission(roleKey: string, permissionKey: string) {
  return rolePermissionMap.value.get(roleKey)?.has(permissionKey) ?? false
}
</script>

<template>
  <div class="mb-4 flex flex-col gap-1">
    <h2 class="text-xl font-bold flex items-center gap-2 text-slate-900 dark:text-zinc-100">
      <UIcon name="i-lucide-shield-check" class="size-5 text-teal-600 dark:text-teal-400" />
      {{ t('admin.permissions.title') }}
    </h2>
    <p class="text-sm text-slate-500 dark:text-zinc-400">
      {{ t('admin.permissions.intro') }}
    </p>
  </div>

  <UDashboardToolbar class="border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 rounded-lg px-4 py-2.5 mb-6 text-slate-500 dark:text-zinc-400">
    <template #left>
      <div class="flex items-center gap-2 text-sm">
        <UIcon name="i-lucide-list-checks" class="size-4 text-teal-600 dark:text-teal-400" />
        <span>{{ t('admin.permissions.matrix') }}</span>
      </div>
    </template>
    <template #right>
      <div class="flex items-center gap-3">
        <UButton
          color="neutral"
          variant="outline"
          leading-icon="i-lucide-refresh-cw"
          :loading="pending"
          class="border-slate-200 dark:border-zinc-700"
          @click="loadData"
        >
          {{ t('admin.permissions.refresh') }}
        </UButton>
        <UBadge color="neutral" variant="soft" class="border border-slate-200 dark:border-zinc-800 font-medium">
          {{ t('admin.permissions.count', { count: matrix.permissions.length }) }}
        </UBadge>
      </div>
    </template>
  </UDashboardToolbar>

  <div class="flex flex-col gap-4">
    <UAlert
      v-if="errorMessage"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="errorMessage"
    />

    <UCard class="border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100">
      <div v-if="pending" class="flex items-center justify-center py-16 text-slate-400 dark:text-zinc-500">
        <UIcon name="i-lucide-loader-circle" class="mr-2 size-5 animate-spin" />
        <span class="text-sm">{{ t('admin.permissions.loading') }}</span>
      </div>

      <div v-else-if="matrix.permissions.length === 0" class="flex flex-col items-center justify-center py-16 text-slate-400 dark:text-zinc-500">
        <UIcon name="i-lucide-shield-check" class="size-12 mb-4" />
        <p class="text-sm">{{ t('admin.permissions.empty') }}</p>
      </div>

      <div v-else class="space-y-6">
        <section v-for="group in groupedPermissions" :key="group.module" class="space-y-3">
          <div class="flex items-center justify-between gap-3">
            <h3 class="text-sm font-semibold uppercase tracking-normal text-slate-600 dark:text-zinc-300">
              {{ group.module }}
            </h3>
            <UBadge color="neutral" variant="soft">
              {{ t('admin.permissions.moduleCount', { count: group.items.length }) }}
            </UBadge>
          </div>

          <div class="overflow-x-auto rounded-lg border border-slate-200 dark:border-zinc-800">
            <table class="min-w-full divide-y divide-slate-200 text-sm dark:divide-zinc-800">
              <thead class="bg-slate-50 dark:bg-zinc-950">
                <tr>
                  <th class="sticky left-0 z-10 min-w-72 bg-slate-50 px-3 py-2 text-left text-xs font-semibold text-slate-500 dark:bg-zinc-950 dark:text-zinc-400">
                    {{ t('admin.permissions.permission') }}
                  </th>
                  <th
                    v-for="role in roles"
                    :key="role.key"
                    class="min-w-36 px-3 py-2 text-center text-xs font-semibold text-slate-500 dark:text-zinc-400"
                  >
                    <span class="block truncate text-slate-700 dark:text-zinc-200">{{ role.alias }}</span>
                    <code class="text-[11px] font-normal text-slate-400 dark:text-zinc-500">{{ role.key }}</code>
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-200 bg-white dark:divide-zinc-800 dark:bg-zinc-900">
                <tr v-for="permission in group.items" :key="permission.key">
                  <td class="sticky left-0 z-10 min-w-72 bg-white px-3 py-2 dark:bg-zinc-900">
                    <code class="font-semibold text-slate-900 dark:text-zinc-100">{{ permission.key }}</code>
                    <p class="mt-0.5 text-xs text-slate-500 dark:text-zinc-400">
                      {{ permission.description || t('admin.permissions.noDescription') }}
                    </p>
                  </td>
                  <td
                    v-for="role in roles"
                    :key="`${permission.key}:${role.key}`"
                    class="px-3 py-2 text-center"
                  >
                    <span
                      class="inline-flex size-7 items-center justify-center rounded-full"
                      :class="roleHasPermission(role.key, permission.key)
                        ? 'bg-teal-50 text-teal-700 dark:bg-teal-500/15 dark:text-teal-300'
                        : 'bg-slate-50 text-slate-300 dark:bg-zinc-950 dark:text-zinc-700'"
                    >
                      <UIcon :name="roleHasPermission(role.key, permission.key) ? 'i-lucide-check' : 'i-lucide-minus'" class="size-4" />
                    </span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </UCard>
  </div>
</template>
