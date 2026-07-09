<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminUsers'
})

type UserStatus = 'active' | 'disabled' | 'banned'

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

type Permission = {
  key: string
  module: string
  description: string
}

type AdminUserSummary = {
  id: number
  username: string
  email: string
  displayName: string
  locale: string
  status: UserStatus
  isInitialSuperAdmin: boolean
  roleKeys: string[]
}

type PermissionOverrides = {
  allow: string[]
  deny: string[]
}

type AdminUserDetail = AdminUserSummary & {
  permissions: string[]
  permissionOverrides: PermissionOverrides
}

type AdminUserList = {
  items: AdminUserSummary[]
  total: number
  page: number
  perPage: number
}

const { t } = useI18n()
const { request } = useApiClient()
const { permissionLabel, permissionDescription, permissionModuleLabel } = usePermissionText()
const adminPage = useAdminPage('/users')

const search = ref('')
const status = ref('')
const roleKey = ref('')
const users = ref<AdminUserSummary[]>([])
const total = ref(0)
const page = ref(1)
const perPage = ref(20)
const roles = ref<Role[]>([])
const permissions = ref<Permission[]>([])
const selectedUser = ref<AdminUserDetail | null>(null)
const selectedRoleKeys = ref<string[]>([])
const allowOverrides = ref<string[]>([])
const denyOverrides = ref<string[]>([])
const pending = ref(false)
const detailPending = ref(false)
const savingRoles = ref(false)
const savingOverrides = ref(false)
const revokingSessions = ref(false)
const errorMessage = ref('')
const successMessage = ref('')
const overrideModes = ['inherit', 'allow', 'deny'] as const

onMounted(() => {
  void loadInitialData()
})

const columns = computed(() => [
  {
    accessorKey: 'username',
    header: t('admin.users.username')
  },
  {
    accessorKey: 'email',
    header: t('admin.users.email')
  },
  {
    id: 'roles',
    header: t('admin.users.roles')
  },
  {
    id: 'status',
    header: t('admin.users.status')
  },
  {
    id: 'actions',
    header: t('admin.users.actions')
  }
])

const groupedPermissions = computed(() => {
  const groups = new Map<string, Permission[]>()
  for (const permission of permissions.value) {
    const group = groups.get(permission.module) ?? []
    group.push(permission)
    groups.set(permission.module, group)
  }
  return Array.from(groups.entries()).map(([module, items]) => ({ module, items }))
})

const selectedUserIsSuperAdmin = computed(() => {
  return selectedUser.value?.roleKeys.includes('super_admin') ?? false
})

const roleOptions = computed(() => {
  return roles.value.map(role => ({
    key: role.key,
    label: role.alias || role.key
  }))
})

useSeoMeta({
  title: t('admin.users.metaTitle')
})

async function loadInitialData() {
  pending.value = true
  errorMessage.value = ''
  try {
    const [roleItems, permissionItems] = await Promise.all([
      request<Role[]>('/roles'),
      request<Permission[]>('/permissions')
    ])
    roles.value = roleItems
    permissions.value = permissionItems
    await loadUsers()
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.users.loadFailed')
  } finally {
    pending.value = false
  }
}

async function loadUsers() {
  const params = new URLSearchParams({
    page: String(page.value),
    perPage: String(perPage.value)
  })
  if (search.value.trim()) {
    params.set('query', search.value.trim())
  }
  if (status.value) {
    params.set('status', status.value)
  }
  if (roleKey.value) {
    params.set('roleKey', roleKey.value)
  }

  const result = await request<AdminUserList>(`/users?${params.toString()}`)
  users.value = result.items
  total.value = result.total
  page.value = result.page
  perPage.value = result.perPage
}

async function refreshUsers() {
  pending.value = true
  errorMessage.value = ''
  try {
    await loadUsers()
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.users.loadFailed')
  } finally {
    pending.value = false
  }
}

async function openUser(user: AdminUserSummary) {
  detailPending.value = true
  errorMessage.value = ''
  successMessage.value = ''
  try {
    const detail = await request<AdminUserDetail>(`/users/${user.id}`)
    selectedUser.value = detail
    selectedRoleKeys.value = [...detail.roleKeys]
    allowOverrides.value = [...detail.permissionOverrides.allow]
    denyOverrides.value = [...detail.permissionOverrides.deny]
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.users.detailLoadFailed')
  } finally {
    detailPending.value = false
  }
}

function closeUser() {
  selectedUser.value = null
  selectedRoleKeys.value = []
  allowOverrides.value = []
  denyOverrides.value = []
}

function toggleRole(key: string) {
  if (selectedRoleKeys.value.includes(key)) {
    selectedRoleKeys.value = selectedRoleKeys.value.filter(item => item !== key)
    return
  }
  selectedRoleKeys.value = [...selectedRoleKeys.value, key].sort()
}

function permissionMode(key: string) {
  if (allowOverrides.value.includes(key)) {
    return 'allow'
  }
  if (denyOverrides.value.includes(key)) {
    return 'deny'
  }
  return 'inherit'
}

function setPermissionMode(key: string, mode: 'inherit' | 'allow' | 'deny') {
  allowOverrides.value = allowOverrides.value.filter(item => item !== key)
  denyOverrides.value = denyOverrides.value.filter(item => item !== key)
  if (mode === 'allow') {
    allowOverrides.value = [...allowOverrides.value, key].sort()
  }
  if (mode === 'deny') {
    denyOverrides.value = [...denyOverrides.value, key].sort()
  }
}

async function saveRoles() {
  if (!selectedUser.value) {
    return
  }
  savingRoles.value = true
  errorMessage.value = ''
  successMessage.value = ''
  try {
    selectedUser.value = await request<AdminUserDetail>(`/users/${selectedUser.value.id}/roles`, {
      method: 'PUT',
      body: {
        roleKeys: selectedRoleKeys.value
      }
    })
    selectedRoleKeys.value = [...selectedUser.value.roleKeys]
    allowOverrides.value = [...selectedUser.value.permissionOverrides.allow]
    denyOverrides.value = [...selectedUser.value.permissionOverrides.deny]
    await loadUsers()
    successMessage.value = t('admin.users.rolesSaved')
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.users.saveFailed')
  } finally {
    savingRoles.value = false
  }
}

async function savePermissionOverrides() {
  if (!selectedUser.value) {
    return
  }
  savingOverrides.value = true
  errorMessage.value = ''
  successMessage.value = ''
  try {
    selectedUser.value = await request<AdminUserDetail>(`/users/${selectedUser.value.id}/permission-overrides`, {
      method: 'PUT',
      body: {
        allow: allowOverrides.value,
        deny: denyOverrides.value
      }
    })
    allowOverrides.value = [...selectedUser.value.permissionOverrides.allow]
    denyOverrides.value = [...selectedUser.value.permissionOverrides.deny]
    await loadUsers()
    successMessage.value = t('admin.users.permissionsSaved')
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.users.saveFailed')
  } finally {
    savingOverrides.value = false
  }
}

// 管理员强制下线目标用户的全部设备（user.manage；禁止对自己操作）。
async function revokeUserSessions() {
  if (!selectedUser.value || revokingSessions.value) {
    return
  }
  revokingSessions.value = true
  errorMessage.value = ''
  successMessage.value = ''
  try {
    const result = await request<{ revoked: number }>(`/users/${selectedUser.value.id}/sessions/revoke`, {
      method: 'POST'
    })
    successMessage.value = t('admin.users.sessionsRevoked', { count: result.revoked })
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.users.revokeSessionsFailed')
  } finally {
    revokingSessions.value = false
  }
}
</script>

<template>
  <div class="mb-4 flex flex-col gap-1">
    <h2 class="text-xl font-bold flex items-center gap-2 text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ t('admin.users.title') }}
    </h2>
    <p class="text-sm text-slate-500 dark:text-zinc-400">
      {{ t('admin.users.intro') }}
    </p>
  </div>

  <UDashboardToolbar class="border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 rounded-lg px-4 py-2.5 mb-6 text-slate-500 dark:text-zinc-400">
    <template #left>
      <div class="flex flex-wrap items-center gap-2">
        <UInput
          v-model="search"
          icon="i-lucide-search"
          :placeholder="t('admin.users.searchPlaceholder')"
          class="w-72 max-w-full"
          @keyup.enter="refreshUsers"
        />
        <select
          v-model="status"
          class="h-9 rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-700 outline-none transition-colors hover:border-slate-300 focus:border-[var(--sf-accent)] dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200"
        >
          <option value="">{{ t('admin.users.allStatuses') }}</option>
          <option value="active">{{ t('admin.users.statusActive') }}</option>
          <option value="disabled">{{ t('admin.users.statusDisabled') }}</option>
          <option value="banned">{{ t('admin.users.statusBanned') }}</option>
        </select>
        <select
          v-model="roleKey"
          class="h-9 rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-700 outline-none transition-colors hover:border-slate-300 focus:border-[var(--sf-accent)] dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200"
        >
          <option value="">{{ t('admin.users.allRoles') }}</option>
          <option v-for="role in roleOptions" :key="role.key" :value="role.key">
            {{ role.label }}
          </option>
        </select>
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
          @click="refreshUsers"
        >
          {{ t('admin.users.refresh') }}
        </UButton>
        <UBadge color="neutral" variant="soft" class="border border-slate-200 dark:border-zinc-800 font-medium">
          {{ t('admin.users.count', { count: total }) }}
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
    <UAlert
      v-if="successMessage"
      color="success"
      variant="soft"
      icon="i-lucide-check-circle"
      :title="successMessage"
    />

    <UCard class="border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100">
      <UTable
        :data="users"
        :columns="columns"
        :loading="pending"
        :empty="t('admin.users.empty')"
        :caption="t('admin.users.caption')"
        sticky
        class="max-h-[calc(100vh-17rem)]"
      >
        <template #username-cell="{ row }">
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <span class="font-semibold text-slate-900 dark:text-white">{{ row.original.displayName }}</span>
              <UBadge v-if="row.original.isInitialSuperAdmin" color="warning" variant="soft">
                {{ t('admin.users.initialSuperAdmin') }}
              </UBadge>
            </div>
            <code class="text-xs text-slate-500 dark:text-zinc-400">{{ row.original.username }}</code>
          </div>
        </template>

        <template #email-cell="{ row }">
          <span class="text-sm text-slate-600 dark:text-zinc-300">{{ row.original.email }}</span>
        </template>

        <template #roles-cell="{ row }">
          <div class="flex flex-wrap gap-1.5">
            <UBadge
              v-for="key in row.original.roleKeys"
              :key="key"
              color="neutral"
              variant="outline"
              class="font-medium"
            >
              {{ roles.find(role => role.key === key)?.alias || key }}
            </UBadge>
          </div>
        </template>

        <template #status-cell="{ row }">
          <UBadge :color="row.original.status === 'active' ? 'success' : row.original.status === 'banned' ? 'error' : 'neutral'" variant="soft">
            {{ t(`admin.users.statusMap.${row.original.status}`) }}
          </UBadge>
        </template>

        <template #actions-cell="{ row }">
          <UButton
            color="primary"
            variant="ghost"
            leading-icon="i-lucide-panel-right-open"
            :loading="detailPending && selectedUser?.id === row.original.id"
            @click="openUser(row.original)"
          >
            {{ t('admin.users.manage') }}
          </UButton>
        </template>
      </UTable>
    </UCard>
  </div>

  <div
    v-if="selectedUser"
    class="fixed inset-0 z-40 flex justify-end bg-slate-950/30 backdrop-blur-[1px]"
    @click.self="closeUser"
  >
    <aside class="h-full w-full max-w-3xl overflow-y-auto border-l border-slate-200 bg-white shadow-xl dark:border-zinc-800 dark:bg-zinc-950">
      <header class="sticky top-0 z-10 flex items-start justify-between gap-4 border-b border-slate-200 bg-white px-5 py-4 dark:border-zinc-800 dark:bg-zinc-950">
        <div>
          <h3 class="text-base font-semibold text-slate-950 dark:text-zinc-50">
            {{ selectedUser.displayName }}
          </h3>
          <p class="text-xs text-slate-500 dark:text-zinc-400">
            {{ selectedUser.username }} · {{ selectedUser.email }}
          </p>
        </div>
        <UButton color="neutral" variant="ghost" icon="i-lucide-x" :aria-label="t('admin.users.close')" @click="closeUser" />
      </header>

      <div class="space-y-5 px-5 py-5">
        <!-- 账号安全：强制下线该用户全部设备 -->
        <section class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-zinc-800 dark:bg-zinc-900/60">
          <div class="flex items-center justify-between gap-3">
            <div>
              <h4 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.users.sessionSection') }}
              </h4>
              <p class="text-xs text-slate-500 dark:text-zinc-400">
                {{ t('admin.users.sessionSectionHelp') }}
              </p>
            </div>
            <UButton
              color="error"
              variant="soft"
              leading-icon="i-lucide-log-out"
              :loading="revokingSessions"
              @click="revokeUserSessions"
            >
              {{ t('admin.users.revokeSessions') }}
            </UButton>
          </div>
        </section>

        <section class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-zinc-800 dark:bg-zinc-900/60">
          <div class="mb-4 flex items-center justify-between gap-3">
            <div>
              <h4 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.users.roleSection') }}
              </h4>
              <p class="text-xs text-slate-500 dark:text-zinc-400">
                {{ t('admin.users.roleSectionHelp') }}
              </p>
            </div>
            <UButton
              color="primary"
              leading-icon="i-lucide-save"
              :loading="savingRoles"
              @click="saveRoles"
            >
              {{ t('admin.users.saveRoles') }}
            </UButton>
          </div>

          <div class="grid gap-2 sm:grid-cols-2">
            <label
              v-for="role in roles"
              :key="role.key"
              class="flex min-h-14 cursor-pointer items-start gap-3 rounded-md border border-slate-200 bg-white px-3 py-2 text-sm transition hover:border-[var(--sf-accent-soft-border)] dark:border-zinc-800 dark:bg-zinc-950"
            >
              <input
                type="checkbox"
                class="mt-1 size-4 accent-[var(--sf-accent)]"
                :checked="selectedRoleKeys.includes(role.key)"
                @change="toggleRole(role.key)"
              >
              <span class="min-w-0">
                <span class="block font-semibold text-slate-900 dark:text-zinc-100">{{ role.alias }}</span>
                <span class="block truncate text-xs text-slate-500 dark:text-zinc-400">{{ role.key }}</span>
              </span>
            </label>
          </div>
        </section>

        <section class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-zinc-800 dark:bg-zinc-900/60">
          <div class="mb-4 flex items-center justify-between gap-3">
            <div>
              <h4 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.users.permissionSection') }}
              </h4>
              <p class="text-xs text-slate-500 dark:text-zinc-400">
                {{ selectedUserIsSuperAdmin ? t('admin.users.superAdminOverrideLocked') : t('admin.users.permissionSectionHelp') }}
              </p>
            </div>
            <UButton
              color="primary"
              leading-icon="i-lucide-save"
              :loading="savingOverrides"
              :disabled="selectedUserIsSuperAdmin"
              @click="savePermissionOverrides"
            >
              {{ t('admin.users.savePermissions') }}
            </UButton>
          </div>

          <div class="space-y-4">
            <div v-for="group in groupedPermissions" :key="group.module" class="space-y-2">
              <div class="text-xs font-semibold uppercase text-slate-500 dark:text-zinc-400">
                {{ permissionModuleLabel(group.module) }}
              </div>
              <div class="divide-y divide-slate-200 overflow-hidden rounded-md border border-slate-200 bg-white dark:divide-zinc-800 dark:border-zinc-800 dark:bg-zinc-950">
                <div
                  v-for="permission in group.items"
                  :key="permission.key"
                  class="grid gap-3 px-3 py-2 text-sm sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"
                >
                  <div class="min-w-0">
                    <span class="block font-semibold text-slate-900 dark:text-zinc-100">
                      {{ permissionLabel(permission) }}
                    </span>
                    <code class="mt-0.5 block text-xs text-slate-500 dark:text-zinc-400">{{ permission.key }}</code>
                    <p class="mt-0.5 text-xs text-slate-500 dark:text-zinc-400">
                      {{ permissionDescription(permission) }}
                    </p>
                  </div>
                  <div class="inline-flex rounded-md border border-slate-200 bg-slate-50 p-0.5 text-xs dark:border-zinc-800 dark:bg-zinc-900">
                    <button
                      v-for="mode in overrideModes"
                      :key="mode"
                      type="button"
                      class="h-7 px-2.5 font-medium transition"
                      :class="permissionMode(permission.key) === mode
                        ? 'rounded bg-white text-[var(--sf-accent)] shadow-sm dark:bg-zinc-800 dark:text-[var(--sf-accent-dark)]'
                        : 'text-slate-500 hover:text-slate-900 dark:text-zinc-400 dark:hover:text-zinc-100'"
                      :disabled="selectedUserIsSuperAdmin"
                      @click="setPermissionMode(permission.key, mode)"
                    >
                      {{ t(`admin.users.overrideMode.${mode}`) }}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>
    </aside>
  </div>
</template>
