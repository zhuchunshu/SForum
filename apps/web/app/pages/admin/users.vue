<script setup lang="ts">
import { usePermissionText } from '~/composables/identity/usePermissionText'
import { useAdminListSurfaces } from '~/composables/admin/useAdminListSurfaces'
import SFAdminSurfaceOutlet from '~/components/admin/SFAdminSurfaceOutlet.vue'
import SFAdminUserListToolbar from '~/components/admin/identity/users/SFAdminUserListToolbar.vue'
import SFAdminUserEmailVerificationControl from '~/components/admin/identity/users/SFAdminUserEmailVerificationControl.vue'
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/admin/useAdminPage'
import { paginateItems } from '~/utils/admin/adminExtensions'
import type {
  AdminUserDetail, AdminUserList,
  AdminUserSortField, AdminUserSortOrder,
  AdminUserSummary,
  Permission, Role,
  UserStatus
} from '~/utils/admin/adminUsers'
definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})
defineOptions({
  name: 'AdminUsers'
})

const DEFAULT_PER_PAGE = 20
// 预览弹层内列表分页：会话卡片较占高，权限 chip 可更密。
const PREVIEW_SESSION_PAGE_SIZE = 5
const PREVIEW_AUTH_EVENT_PAGE_SIZE = 10
const PREVIEW_PERMISSION_PAGE_SIZE = 24

const { t } = useI18n()
const { request } = useApiClient()
const { permissionLabel, permissionDescription, permissionModuleLabel } = usePermissionText()
const toast = useToast()
const adminPage = useAdminPage('/users')

const search = ref('')
const status = ref('')
const roleKey = ref('')
const sortBy = ref<AdminUserSortField>('createdAt')
const sortOrder = ref<AdminUserSortOrder>('desc')
const users = ref<AdminUserSummary[]>([])
const total = ref(0)
const page = ref(1)
const perPage = ref(DEFAULT_PER_PAGE)
const roles = ref<Role[]>([])
const permissions = ref<Permission[]>([])
const selectedUser = ref<AdminUserDetail | null>(null)
const previewUser = ref<AdminUserDetail | null>(null)
const previewTargetId = ref<number | null>(null)
const previewOpen = ref(false)
const previewPending = ref(false)
const previewSessionsPage = ref(1)
const previewAuthEventsPage = ref(1)
const previewPermissionsPage = ref(1)
const selectedRoleKeys = ref<string[]>([])
const allowOverrides = ref<string[]>([])
const denyOverrides = ref<string[]>([])

// 账户/资料编辑草稿
const editUsername = ref('')
const editEmail = ref('')
const editDisplayName = ref('')
const editLocale = ref('zh-CN')
const editStatus = ref<UserStatus>('active')
const editBio = ref('')
const editSignature = ref('')
const editLocation = ref('')
const editWebsiteUrl = ref('')

const pending = ref(false)
const detailPending = ref(false)
const savingAccount = ref(false)
const savingProfile = ref(false)
const savingRoles = ref(false)
const savingOverrides = ref(false)
const revokingSessions = ref(false)
const clearingClientIPs = ref(false)
const resettingPassword = ref(false)
const newPassword = ref('')
const confirmPassword = ref('')
const errorMessage = ref('')
const successMessage = ref('')
const overrideModes = ['inherit', 'allow', 'deny'] as const

onMounted(() => {
  void loadInitialData()
})

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / Math.max(perPage.value, 1))))

const previewSessionsPageInfo = computed(() =>
  paginateItems(previewUser.value?.sessions || [], previewSessionsPage.value, PREVIEW_SESSION_PAGE_SIZE)
)
const previewAuthEventsPageInfo = computed(() =>
  paginateItems(previewUser.value?.recentAuthEvents || [], previewAuthEventsPage.value, PREVIEW_AUTH_EVENT_PAGE_SIZE)
)
const previewPermissionsPageInfo = computed(() =>
  paginateItems(previewUser.value?.permissions || [], previewPermissionsPage.value, PREVIEW_PERMISSION_PAGE_SIZE)
)

// 分页计算会夹紧页码；同步回 ref，避免删减数据后落在空页。
watch(() => previewSessionsPageInfo.value.page, (next) => {
  previewSessionsPage.value = next
})
watch(() => previewAuthEventsPageInfo.value.page, (next) => {
  previewAuthEventsPage.value = next
})
watch(() => previewPermissionsPageInfo.value.page, (next) => {
  previewPermissionsPage.value = next
})

const userSurfaceResources = computed(() => users.value.map(user => ({
  id: String(user.id),
  attributes: {
    username: user.username,
    email: user.email,
    displayName: user.displayName,
    locale: user.locale,
    status: user.status,
    isInitialSuperAdmin: user.isInitialSuperAdmin,
    roleKeys: [...user.roleKeys],
    createdAt: user.createdAt,
    updatedAt: user.updatedAt
  }
})))
const userListSurfaces = useAdminListSurfaces({
  pageId: '/users',
  resources: userSurfaceResources,
  context: computed(() => ({
    resourceType: 'user',
    page: page.value,
    perPage: perPage.value,
    total: total.value,
    coreFilters: {
      query: search.value.trim(),
      status: status.value,
      roleKey: roleKey.value,
      sortBy: sortBy.value,
      sortOrder: sortOrder.value
    }
  })),
  refreshHost: refreshUsers
})
const visibleUsers = computed(() => users.value.filter(user => userListSurfaces.isResourceVisible(String(user.id))))

const columns = computed(() => [
  ...(userListSurfaces.hasBulkActions.value ? [{ id: 'selection', header: '' }] : []),
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
  ...userListSurfaces.columns.value.map(item => ({
    id: `extension_${item.surface.id.replace(/[^a-zA-Z0-9_]/g, '_')}`,
    header: item.view.title || item.surface.label,
    accessorFn: (user: AdminUserSummary) => userListSurfaces.columnValue(item, String(user.id))
  })),
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

const localeOptions = computed(() => [
  { value: 'zh-CN', label: t('admin.users.localeZh') },
  { value: 'en-US', label: t('admin.users.localeEn') }
])

useSeoMeta({
  title: t('admin.users.metaTitle')
})

function formatDateTime(value?: string) {
  if (!value) {
    return '—'
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleString()
}

function showSuccess(message: string) {
  successMessage.value = message
  toast.add({
    title: message,
    color: 'success',
    icon: 'i-lucide-check-circle',
    duration: 10000
  })
}

function showError(message: string) {
  errorMessage.value = message
  toast.add({
    title: message,
    color: 'error',
    icon: 'i-lucide-triangle-alert'
  })
}

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
    showError(apiErrorMessage(error) || t('admin.users.loadFailed'))
  } finally {
    pending.value = false
  }
}

async function loadUsers() {
  const params = new URLSearchParams({
    page: String(page.value),
    perPage: String(DEFAULT_PER_PAGE)
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
  params.set('sortBy', sortBy.value)
  params.set('sortOrder', sortOrder.value)

  const result = await request<AdminUserList>(`/users?${params.toString()}`)
  users.value = result.items
  total.value = result.total
  page.value = result.page || page.value
  perPage.value = result.perPage || DEFAULT_PER_PAGE
}

async function refreshUsers() {
  pending.value = true
  errorMessage.value = ''
  try {
    await loadUsers()
  } catch (error) {
    showError(apiErrorMessage(error) || t('admin.users.loadFailed'))
  } finally {
    pending.value = false
  }
}

async function goToPage(nextPage: number) {
  if (nextPage < 1 || nextPage > totalPages.value || nextPage === page.value) {
    return
  }
  page.value = nextPage
  await refreshUsers()
}

async function applyFilters() {
  page.value = 1
  await refreshUsers()
}

function applyDetailToForm(detail: AdminUserDetail) {
  selectedUser.value = detail
  selectedRoleKeys.value = [...detail.roleKeys]
  allowOverrides.value = [...detail.permissionOverrides.allow]
  denyOverrides.value = [...detail.permissionOverrides.deny]
  editUsername.value = detail.username
  editEmail.value = detail.email
  editDisplayName.value = detail.displayName
  editLocale.value = detail.locale || 'zh-CN'
  editStatus.value = detail.status
  editBio.value = detail.profile?.bio ?? ''
  editSignature.value = detail.profile?.signature ?? ''
  editLocation.value = detail.profile?.location ?? ''
  editWebsiteUrl.value = detail.profile?.websiteUrl ?? ''
  // 切换用户时清空密码表单，避免误提交到下一位。
  newPassword.value = ''
  confirmPassword.value = ''
}

async function openUser(user: AdminUserSummary) {
  detailPending.value = true
  errorMessage.value = ''
  successMessage.value = ''
  try {
    const detail = await request<AdminUserDetail>(`/users/${user.id}`)
    applyDetailToForm(detail)
  } catch (error) {
    showError(apiErrorMessage(error) || t('admin.users.detailLoadFailed'))
  } finally {
    detailPending.value = false
  }
}

function resetPreviewListPages() {
  previewSessionsPage.value = 1
  previewAuthEventsPage.value = 1
  previewPermissionsPage.value = 1
}

async function openUserPreview(user: AdminUserSummary) {
  previewPending.value = true
  previewTargetId.value = user.id
  previewOpen.value = true
  previewUser.value = null
  resetPreviewListPages()
  errorMessage.value = ''
  try {
    previewUser.value = await request<AdminUserDetail>(`/users/${user.id}`)
    // 加载完成后按实际条数夹紧页码（通常仍是第 1 页）。
    resetPreviewListPages()
  } catch (error) {
    previewOpen.value = false
    previewTargetId.value = null
    resetPreviewListPages()
    showError(apiErrorMessage(error) || t('admin.users.previewLoadFailed'))
  } finally {
    previewPending.value = false
  }
}

function closeUserPreview() {
  previewOpen.value = false
  previewUser.value = null
  previewTargetId.value = null
  resetPreviewListPages()
}

async function manageFromPreview() {
  const user = previewUser.value
  closeUserPreview()
  if (user) {
    await openUser(user)
  }
}

function displayOrDash(value?: string | null) {
  const text = (value || '').trim()
  return text || t('admin.users.previewEmptyValue')
}

function authActionLabel(action: string) {
  if (action === 'auth.login.success') {
    return t('admin.users.previewAuthLogin')
  }
  if (action === 'auth.register.success') {
    return t('admin.users.previewAuthRegister')
  }
  return action
}

async function refreshSelectedUser() {
  if (!selectedUser.value) return
  await openUser(selectedUser.value)
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

async function saveAccount() {
  if (!selectedUser.value) {
    return
  }
  savingAccount.value = true
  errorMessage.value = ''
  successMessage.value = ''
  try {
    const detail = await request<AdminUserDetail>(`/users/${selectedUser.value.id}`, {
      method: 'PATCH',
      body: {
        username: editUsername.value.trim(),
        email: editEmail.value.trim(),
        displayName: editDisplayName.value.trim(),
        locale: editLocale.value,
        status: editStatus.value
      }
    })
    applyDetailToForm(detail)
    await loadUsers()
    showSuccess(t('admin.users.accountSaved'))
  } catch (error) {
    showError(apiErrorMessage(error) || t('admin.users.saveFailed'))
  } finally {
    savingAccount.value = false
  }
}

async function saveProfile() {
  if (!selectedUser.value) {
    return
  }
  savingProfile.value = true
  errorMessage.value = ''
  successMessage.value = ''
  try {
    const detail = await request<AdminUserDetail>(`/users/${selectedUser.value.id}`, {
      method: 'PATCH',
      body: {
        bio: editBio.value,
        signature: editSignature.value,
        location: editLocation.value,
        websiteUrl: editWebsiteUrl.value.trim()
      }
    })
    applyDetailToForm(detail)
    showSuccess(t('admin.users.profileSaved'))
  } catch (error) {
    showError(apiErrorMessage(error) || t('admin.users.saveFailed'))
  } finally {
    savingProfile.value = false
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
    const detail = await request<AdminUserDetail>(`/users/${selectedUser.value.id}/roles`, {
      method: 'PUT',
      body: {
        roleKeys: selectedRoleKeys.value
      }
    })
    applyDetailToForm(detail)
    await loadUsers()
    showSuccess(t('admin.users.rolesSaved'))
  } catch (error) {
    showError(apiErrorMessage(error) || t('admin.users.saveFailed'))
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
    const detail = await request<AdminUserDetail>(`/users/${selectedUser.value.id}/permission-overrides`, {
      method: 'PUT',
      body: {
        allow: allowOverrides.value,
        deny: denyOverrides.value
      }
    })
    applyDetailToForm(detail)
    await loadUsers()
    showSuccess(t('admin.users.permissionsSaved'))
  } catch (error) {
    showError(apiErrorMessage(error) || t('admin.users.saveFailed'))
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
    showSuccess(t('admin.users.sessionsRevoked', { count: result.revoked }))
  } catch (error) {
    showError(apiErrorMessage(error) || t('admin.users.revokeSessionsFailed'))
  } finally {
    revokingSessions.value = false
  }
}

// 隐私合规：清空该用户相关真实客户端 IP。
async function clearUserClientIPs() {
  if (!selectedUser.value || clearingClientIPs.value) {
    return
  }
  clearingClientIPs.value = true
  errorMessage.value = ''
  successMessage.value = ''
  try {
    await request(`/users/${selectedUser.value.id}/client-ips/clear`, {
      method: 'POST'
    })
    showSuccess(t('admin.users.clientIPsCleared'))
  } catch (error) {
    showError(apiErrorMessage(error) || t('admin.users.clearClientIPsFailed'))
  } finally {
    clearingClientIPs.value = false
  }
}

// 管理员直接设置目标用户密码（user.manage）；成功后目标全部设备会被下线。
async function resetUserPassword() {
  if (!selectedUser.value || resettingPassword.value) {
    return
  }
  const password = newPassword.value
  if (!password) {
    showError(t('admin.users.passwordRequired'))
    return
  }
  if (password !== confirmPassword.value) {
    showError(t('admin.users.passwordMismatch'))
    return
  }
  resettingPassword.value = true
  errorMessage.value = ''
  successMessage.value = ''
  try {
    const result = await request<{ revokedSessions: number }>(`/users/${selectedUser.value.id}/password`, {
      method: 'POST',
      body: { password }
    })
    newPassword.value = ''
    confirmPassword.value = ''
    showSuccess(t('admin.users.passwordResetSuccess', { count: result.revokedSessions ?? 0 }))
  } catch (error) {
    showError(apiErrorMessage(error) || t('admin.users.passwordResetFailed'))
  } finally {
    resettingPassword.value = false
  }
}

watch([status, roleKey, sortBy, sortOrder], () => {
  void applyFilters()
})
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

  <SFAdminUserListToolbar
    v-model:search="search"
    v-model:status="status"
    v-model:role-key="roleKey"
    v-model:sort-by="sortBy"
    v-model:sort-order="sortOrder"
    :role-options="roleOptions"
    :pending="pending"
    :total="total"
    :per-page="perPage"
    @apply="applyFilters"
    @refresh="refreshUsers"
  >
    <label v-for="item in userListSurfaces.filters.value" :key="item.surface.id" class="flex items-center gap-2">
      <span class="sr-only">{{ item.view.title || item.surface.label }}</span>
      <select
        :value="userListSurfaces.filterValue(item.surface.id)"
        :disabled="userListSurfaces.pending.value"
        class="h-9 rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-700 outline-none transition-colors hover:border-slate-300 focus:border-[var(--sf-accent)] dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200"
        @change="userListSurfaces.setFilter(item.surface.id, ($event.target as HTMLSelectElement).value)"
      >
        <option value="">{{ item.view.title || item.surface.label }}: {{ t('admin.surfaces.filterAll') }}</option>
        <option v-for="option in item.view.options" :key="option.value" :value="option.value">
          {{ option.label }}
        </option>
      </select>
    </label>
  </SFAdminUserListToolbar>

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
    <UAlert
      v-if="userListSurfaces.failureMessage.value"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="t('admin.surfaces.loadFailed')"
      :description="userListSurfaces.failureMessage.value"
    />

    <UCard class="border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100">
      <div
        v-if="userListSurfaces.hasBulkActions.value"
        class="mb-3 flex min-h-9 flex-wrap items-center gap-2 border-b border-slate-200 pb-3 dark:border-zinc-800"
      >
        <UCheckbox
          :model-value="userListSurfaces.allVisibleSelected.value"
          :label="t('admin.surfaces.selectVisible')"
          @update:model-value="userListSurfaces.selectAllVisible($event === true)"
        />
        <UBadge v-if="userListSurfaces.selectedResourceIds.value.length" color="neutral" variant="soft">
          {{ t('admin.surfaces.selectedCount', { count: userListSurfaces.selectedResourceIds.value.length }) }}
        </UBadge>
        <UButton
          v-for="action in userListSurfaces.bulkActions.value"
          :key="action.surface.id"
          size="sm"
          :color="action.tone"
          variant="soft"
          :icon="action.icon"
          :loading="userListSurfaces.busySurfaceId.value === action.surface.id"
          :disabled="userListSurfaces.selectedResourceIds.value.length === 0"
          @click="userListSurfaces.executeAction(action, userListSurfaces.selectedResourceIds.value)"
        >
          {{ action.label }}
        </UButton>
      </div>
      <UTable
        :data="visibleUsers"
        :columns="columns"
        :loading="pending"
        :empty="t('admin.users.empty')"
        :caption="t('admin.users.caption')"
        sticky
        class="max-h-[calc(100vh-20rem)]"
      >
        <template #selection-cell="{ row }">
          <UCheckbox
            :model-value="userListSurfaces.selectedResourceIds.value.includes(String(row.original.id))"
            :aria-label="t('admin.surfaces.selectResource', { label: row.original.displayName })"
            @update:model-value="userListSurfaces.toggleResource(String(row.original.id), $event === true)"
          />
        </template>

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
          <div class="flex flex-wrap items-center gap-1">
            <UButton
              color="neutral"
              variant="ghost"
              leading-icon="i-lucide-eye"
              :loading="previewPending && previewTargetId === row.original.id"
              @click="openUserPreview(row.original)"
            >
              {{ t('admin.users.preview') }}
            </UButton>
            <UButton
              color="primary"
              variant="ghost"
              leading-icon="i-lucide-panel-right-open"
              :loading="detailPending && selectedUser?.id === row.original.id"
              @click="openUser(row.original)"
            >
              {{ t('admin.users.manage') }}
            </UButton>
            <UButton
              v-for="action in userListSurfaces.rowActionsFor(String(row.original.id))"
              :key="action.surface.id"
              size="sm"
              :color="action.tone"
              variant="ghost"
              :icon="action.icon"
              :loading="userListSurfaces.busySurfaceId.value === action.surface.id"
              @click="userListSurfaces.executeAction(action, [String(row.original.id)])"
            >
              {{ action.label }}
            </UButton>
          </div>
        </template>
      </UTable>

      <div
        v-if="total > 0"
        class="mt-4 flex flex-col gap-3 border-t border-slate-200 pt-4 sm:flex-row sm:items-center sm:justify-between dark:border-zinc-800"
      >
        <p class="text-xs text-slate-500 dark:text-zinc-400">
          {{ t('admin.users.pagination', { page, pages: totalPages }) }}
          · {{ t('admin.users.count', { count: total }) }}
        </p>
        <UPagination
          v-if="totalPages > 1"
          :page="page"
          :total="total"
          :items-per-page="perPage"
          class="justify-end"
          @update:page="goToPage"
        />
      </div>
    </UCard>
  </div>

  <!-- 用户信息预览（只读，含完整 IP / UA） -->
  <UModal
    v-model:open="previewOpen"
    :ui="{ content: 'sm:max-w-4xl' }"
    @update:open="(open) => { if (!open) closeUserPreview() }"
  >
    <template #content>
      <div class="flex max-h-[90vh] flex-col">
        <div class="flex items-start justify-between gap-3 border-b border-slate-200 px-5 py-4 dark:border-zinc-800">
          <div class="min-w-0">
            <h3 class="text-base font-bold text-slate-900 dark:text-white">
              {{ t('admin.users.previewTitle') }}
            </h3>
            <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.users.previewIntro') }}
            </p>
          </div>
          <UButton
            type="button"
            color="neutral"
            variant="ghost"
            icon="i-lucide-x"
            :aria-label="t('admin.users.close')"
            @click="closeUserPreview"
          />
        </div>

        <div class="flex-1 space-y-5 overflow-y-auto px-5 py-4">
          <div v-if="previewPending" class="flex min-h-40 items-center justify-center text-sm text-slate-500">
            <UIcon name="i-lucide-loader-circle" class="mr-2 size-4 animate-spin" />
            {{ t('admin.common.loading') }}
          </div>

          <template v-else-if="previewUser">
            <section class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-zinc-800 dark:bg-zinc-900/60">
              <div class="mb-3 flex flex-wrap items-center gap-2">
                <h4 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                  {{ t('admin.users.previewSectionAccount') }}
                </h4>
                <UBadge :color="previewUser.status === 'active' ? 'success' : previewUser.status === 'banned' ? 'error' : 'neutral'" variant="soft">
                  {{ t(`admin.users.statusMap.${previewUser.status}`) }}
                </UBadge>
                <UBadge v-if="previewUser.isInitialSuperAdmin" color="warning" variant="soft">
                  {{ t('admin.users.initialSuperAdmin') }}
                </UBadge>
              </div>
              <dl class="grid gap-3 text-sm sm:grid-cols-2">
                <div>
                  <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.users.userId') }}</dt>
                  <dd class="mt-0.5 font-mono text-slate-900 dark:text-zinc-100">{{ previewUser.id }}</dd>
                </div>
                <div>
                  <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.users.username') }}</dt>
                  <dd class="mt-0.5 font-semibold text-slate-900 dark:text-zinc-100">{{ previewUser.username }}</dd>
                </div>
                <div>
                  <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.users.displayName') }}</dt>
                  <dd class="mt-0.5 text-slate-900 dark:text-zinc-100">{{ displayOrDash(previewUser.displayName) }}</dd>
                </div>
                <div>
                  <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.users.email') }}</dt>
                  <dd class="mt-0.5 break-all text-slate-900 dark:text-zinc-100">{{ previewUser.email }}</dd>
                </div>
                <div>
                  <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.users.locale') }}</dt>
                  <dd class="mt-0.5 text-slate-900 dark:text-zinc-100">{{ previewUser.locale }}</dd>
                </div>
                <div>
                  <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.users.createdAt') }}</dt>
                  <dd class="mt-0.5 text-slate-900 dark:text-zinc-100">{{ formatDateTime(previewUser.createdAt) }}</dd>
                </div>
                <div>
                  <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.users.updatedAt') }}</dt>
                  <dd class="mt-0.5 text-slate-900 dark:text-zinc-100">{{ formatDateTime(previewUser.updatedAt) }}</dd>
                </div>
                <div>
                  <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.users.previewPasswordChangedAt') }}</dt>
                  <dd class="mt-0.5 text-slate-900 dark:text-zinc-100">{{ formatDateTime(previewUser.passwordChangedAt || undefined) }}</dd>
                </div>
              </dl>
            </section>

            <section class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-zinc-800 dark:bg-zinc-900/60">
              <h4 class="mb-3 text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.users.previewSectionProfile') }}
              </h4>
              <dl class="grid gap-3 text-sm sm:grid-cols-2">
                <div class="sm:col-span-2">
                  <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.users.bio') }}</dt>
                  <dd class="mt-0.5 whitespace-pre-wrap text-slate-900 dark:text-zinc-100">{{ displayOrDash(previewUser.profile?.bio) }}</dd>
                </div>
                <div class="sm:col-span-2">
                  <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.users.signature') }}</dt>
                  <dd class="mt-0.5 whitespace-pre-wrap text-slate-900 dark:text-zinc-100">{{ displayOrDash(previewUser.profile?.signature) }}</dd>
                </div>
                <div>
                  <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.users.location') }}</dt>
                  <dd class="mt-0.5 text-slate-900 dark:text-zinc-100">{{ displayOrDash(previewUser.profile?.location) }}</dd>
                </div>
                <div>
                  <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.users.websiteUrl') }}</dt>
                  <dd class="mt-0.5 break-all text-slate-900 dark:text-zinc-100">{{ displayOrDash(previewUser.profile?.websiteUrl) }}</dd>
                </div>
              </dl>
            </section>

            <section class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-zinc-800 dark:bg-zinc-900/60">
              <h4 class="mb-3 text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.users.previewSectionActivity') }}
              </h4>
              <dl class="grid gap-3 text-sm sm:grid-cols-2 lg:grid-cols-3">
                <div>
                  <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.users.previewTopicCount') }}</dt>
                  <dd class="mt-0.5 tabular-nums text-slate-900 dark:text-zinc-100">{{ previewUser.activity?.topicCount ?? 0 }}</dd>
                </div>
                <div>
                  <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.users.previewCommentCount') }}</dt>
                  <dd class="mt-0.5 tabular-nums text-slate-900 dark:text-zinc-100">{{ previewUser.activity?.commentCount ?? 0 }}</dd>
                </div>
                <div>
                  <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.users.previewActiveSessions') }}</dt>
                  <dd class="mt-0.5 tabular-nums text-slate-900 dark:text-zinc-100">{{ previewUser.activity?.activeSessionCount ?? 0 }}</dd>
                </div>
                <div>
                  <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.users.previewTotalSessions') }}</dt>
                  <dd class="mt-0.5 tabular-nums text-slate-900 dark:text-zinc-100">{{ previewUser.activity?.totalSessionCount ?? 0 }}</dd>
                </div>
                <div>
                  <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.users.previewLastLoginAt') }}</dt>
                  <dd class="mt-0.5 text-slate-900 dark:text-zinc-100">{{ formatDateTime(previewUser.activity?.lastLoginAt || undefined) }}</dd>
                </div>
                <div>
                  <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.users.previewLastSeenAt') }}</dt>
                  <dd class="mt-0.5 text-slate-900 dark:text-zinc-100">{{ formatDateTime(previewUser.activity?.lastSeenAt || undefined) }}</dd>
                </div>
                <div>
                  <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.users.previewLastLoginIP') }}</dt>
                  <dd class="mt-0.5 font-mono text-slate-900 dark:text-zinc-100">{{ displayOrDash(previewUser.activity?.lastLoginIP) }}</dd>
                </div>
                <div class="sm:col-span-2 lg:col-span-3">
                  <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.users.previewLastLoginUA') }}</dt>
                  <dd class="mt-0.5 break-all font-mono text-xs text-slate-900 dark:text-zinc-100">{{ displayOrDash(previewUser.activity?.lastLoginUserAgent) }}</dd>
                </div>
              </dl>
            </section>

            <section class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-zinc-800 dark:bg-zinc-900/60">
              <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
                <h4 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                  {{ t('admin.users.previewSectionSessions') }}
                </h4>
                <div class="flex flex-wrap items-center gap-2">
                  <UBadge color="neutral" variant="soft">
                    {{ previewSessionsPageInfo.total }}
                  </UBadge>
                  <p v-if="previewSessionsPageInfo.total > 0" class="text-xs text-slate-500 dark:text-zinc-400">
                    {{ t('admin.users.previewListRange', {
                      start: previewSessionsPageInfo.start,
                      end: previewSessionsPageInfo.end,
                      total: previewSessionsPageInfo.total
                    }) }}
                  </p>
                </div>
              </div>
              <p v-if="!previewSessionsPageInfo.total" class="text-sm text-slate-500 dark:text-zinc-400">
                {{ t('admin.users.previewNoSessions') }}
              </p>
              <div v-else class="space-y-3">
                <article
                  v-for="session in previewSessionsPageInfo.items"
                  :key="session.id"
                  class="rounded-md border border-slate-200 bg-white p-3 text-sm dark:border-zinc-800 dark:bg-zinc-950"
                >
                  <div class="flex flex-wrap items-center gap-2">
                    <span class="font-semibold text-slate-900 dark:text-zinc-100">
                      {{ displayOrDash(session.deviceName || `${session.browser} / ${session.os}`) }}
                    </span>
                    <UBadge :color="session.isActive ? 'success' : 'neutral'" variant="soft">
                      {{ session.isActive ? t('admin.users.previewSessionActive') : t('admin.users.previewSessionRevoked') }}
                    </UBadge>
                    <code class="text-[11px] text-slate-400">{{ session.id }}</code>
                  </div>
                  <dl class="mt-2 grid gap-2 sm:grid-cols-2">
                    <div>
                      <dt class="text-xs text-slate-500">{{ t('admin.users.previewSessionIP') }}</dt>
                      <dd class="font-mono text-xs">{{ displayOrDash(session.ipAddress || session.ipPrefix) }}</dd>
                    </div>
                    <div>
                      <dt class="text-xs text-slate-500">{{ t('admin.users.previewSessionIPPrefix') }}</dt>
                      <dd class="font-mono text-xs">{{ displayOrDash(session.ipPrefix) }}</dd>
                    </div>
                    <div>
                      <dt class="text-xs text-slate-500">{{ t('admin.users.previewSessionCreated') }}</dt>
                      <dd class="text-xs">{{ formatDateTime(session.createdAt) }}</dd>
                    </div>
                    <div>
                      <dt class="text-xs text-slate-500">{{ t('admin.users.previewSessionLastSeen') }}</dt>
                      <dd class="text-xs">{{ formatDateTime(session.lastSeenAt) }}</dd>
                    </div>
                    <div v-if="!session.isActive && session.revokeReason" class="sm:col-span-2">
                      <dt class="text-xs text-slate-500">{{ t('admin.users.previewSessionRevokeReason') }}</dt>
                      <dd class="font-mono text-xs">{{ session.revokeReason }} · {{ formatDateTime(session.revokedAt || undefined) }}</dd>
                    </div>
                    <div class="sm:col-span-2">
                      <dt class="text-xs text-slate-500">{{ t('admin.users.previewSessionUA') }}</dt>
                      <dd class="break-all font-mono text-[11px] text-slate-700 dark:text-zinc-300">{{ displayOrDash(session.userAgent) }}</dd>
                    </div>
                  </dl>
                </article>
                <div
                  v-if="previewSessionsPageInfo.totalPages > 1"
                  class="flex flex-col gap-2 border-t border-slate-200 pt-3 sm:flex-row sm:items-center sm:justify-between dark:border-zinc-800"
                >
                  <p class="text-xs text-slate-500 dark:text-zinc-400">
                    {{ t('admin.users.pagination', {
                      page: previewSessionsPageInfo.page,
                      pages: previewSessionsPageInfo.totalPages
                    }) }}
                  </p>
                  <UPagination
                    v-model:page="previewSessionsPage"
                    :total="previewSessionsPageInfo.total"
                    :items-per-page="PREVIEW_SESSION_PAGE_SIZE"
                    size="sm"
                    class="justify-end"
                  />
                </div>
              </div>
            </section>

            <section class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-zinc-800 dark:bg-zinc-900/60">
              <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
                <h4 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                  {{ t('admin.users.previewSectionAuthEvents') }}
                </h4>
                <div class="flex flex-wrap items-center gap-2">
                  <UBadge color="neutral" variant="soft">
                    {{ previewAuthEventsPageInfo.total }}
                  </UBadge>
                  <p v-if="previewAuthEventsPageInfo.total > 0" class="text-xs text-slate-500 dark:text-zinc-400">
                    {{ t('admin.users.previewListRange', {
                      start: previewAuthEventsPageInfo.start,
                      end: previewAuthEventsPageInfo.end,
                      total: previewAuthEventsPageInfo.total
                    }) }}
                  </p>
                </div>
              </div>
              <p v-if="!previewAuthEventsPageInfo.total" class="text-sm text-slate-500 dark:text-zinc-400">
                {{ t('admin.users.previewNoAuthEvents') }}
              </p>
              <div v-else class="space-y-3">
                <div class="overflow-x-auto">
                  <table class="min-w-full text-left text-sm">
                    <thead class="text-xs text-slate-500">
                      <tr>
                        <th class="px-2 py-2">{{ t('admin.users.previewAuthAction') }}</th>
                        <th class="px-2 py-2">{{ t('admin.users.previewAuthIP') }}</th>
                        <th class="px-2 py-2">{{ t('admin.users.previewAuthTime') }}</th>
                        <th class="px-2 py-2">{{ t('admin.users.previewAuthUA') }}</th>
                      </tr>
                    </thead>
                    <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
                      <tr v-for="event in previewAuthEventsPageInfo.items" :key="event.id">
                        <td class="px-2 py-2 align-top">
                          <UBadge color="neutral" variant="soft">{{ authActionLabel(event.action) }}</UBadge>
                          <p v-if="event.sessionHash" class="mt-1 font-mono text-[10px] text-slate-400">
                            {{ t('admin.users.previewAuthSessionHash') }}: {{ event.sessionHash.slice(0, 16) }}…
                          </p>
                        </td>
                        <td class="px-2 py-2 align-top font-mono text-xs">{{ displayOrDash(event.ipAddress) }}</td>
                        <td class="px-2 py-2 align-top text-xs tabular-nums">{{ formatDateTime(event.createdAt) }}</td>
                        <td class="px-2 py-2 align-top break-all font-mono text-[11px]">{{ displayOrDash(event.userAgent) }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
                <div
                  v-if="previewAuthEventsPageInfo.totalPages > 1"
                  class="flex flex-col gap-2 border-t border-slate-200 pt-3 sm:flex-row sm:items-center sm:justify-between dark:border-zinc-800"
                >
                  <p class="text-xs text-slate-500 dark:text-zinc-400">
                    {{ t('admin.users.pagination', {
                      page: previewAuthEventsPageInfo.page,
                      pages: previewAuthEventsPageInfo.totalPages
                    }) }}
                  </p>
                  <UPagination
                    v-model:page="previewAuthEventsPage"
                    :total="previewAuthEventsPageInfo.total"
                    :items-per-page="PREVIEW_AUTH_EVENT_PAGE_SIZE"
                    size="sm"
                    class="justify-end"
                  />
                </div>
              </div>
            </section>

            <section class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-zinc-800 dark:bg-zinc-900/60">
              <h4 class="mb-3 text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.users.previewSectionRoles') }}
              </h4>
              <div class="flex flex-wrap gap-1.5">
                <UBadge
                  v-for="key in previewUser.roleKeys"
                  :key="key"
                  color="neutral"
                  variant="outline"
                >
                  {{ roles.find(role => role.key === key)?.alias || key }}
                </UBadge>
              </div>
            </section>

            <section class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-zinc-800 dark:bg-zinc-900/60">
              <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
                <h4 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                  {{ t('admin.users.previewSectionPermissions') }}
                </h4>
                <div class="flex flex-wrap items-center gap-2">
                  <UBadge color="neutral" variant="soft">
                    {{ t('admin.users.previewPermissionCount', { count: previewPermissionsPageInfo.total }) }}
                  </UBadge>
                  <p v-if="previewPermissionsPageInfo.total > 0" class="text-xs text-slate-500 dark:text-zinc-400">
                    {{ t('admin.users.previewListRange', {
                      start: previewPermissionsPageInfo.start,
                      end: previewPermissionsPageInfo.end,
                      total: previewPermissionsPageInfo.total
                    }) }}
                  </p>
                </div>
              </div>
              <div v-if="!previewPermissionsPageInfo.total" class="text-sm text-slate-500 dark:text-zinc-400">
                {{ t('admin.users.previewEmptyValue') }}
              </div>
              <div v-else class="space-y-3">
                <div class="flex flex-wrap gap-1.5">
                  <code
                    v-for="key in previewPermissionsPageInfo.items"
                    :key="key"
                    class="rounded bg-white px-1.5 py-0.5 text-[11px] text-slate-700 dark:bg-zinc-950 dark:text-zinc-300"
                  >
                    {{ key }}
                  </code>
                </div>
                <div
                  v-if="previewPermissionsPageInfo.totalPages > 1"
                  class="flex flex-col gap-2 border-t border-slate-200 pt-3 sm:flex-row sm:items-center sm:justify-between dark:border-zinc-800"
                >
                  <p class="text-xs text-slate-500 dark:text-zinc-400">
                    {{ t('admin.users.pagination', {
                      page: previewPermissionsPageInfo.page,
                      pages: previewPermissionsPageInfo.totalPages
                    }) }}
                  </p>
                  <UPagination
                    v-model:page="previewPermissionsPage"
                    :total="previewPermissionsPageInfo.total"
                    :items-per-page="PREVIEW_PERMISSION_PAGE_SIZE"
                    size="sm"
                    class="justify-end"
                  />
                </div>
              </div>
            </section>

            <section class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-zinc-800 dark:bg-zinc-900/60">
              <h4 class="mb-3 text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.users.previewSectionOverrides') }}
              </h4>
              <div
                v-if="!(previewUser.permissionOverrides?.allow?.length || previewUser.permissionOverrides?.deny?.length)"
                class="text-sm text-slate-500 dark:text-zinc-400"
              >
                {{ t('admin.users.previewNoOverrides') }}
              </div>
              <div v-else class="grid gap-3 sm:grid-cols-2">
                <div>
                  <p class="mb-1 text-xs font-medium text-emerald-700 dark:text-emerald-300">
                    {{ t('admin.users.previewAllowOverrides') }}
                  </p>
                  <div class="flex flex-wrap gap-1">
                    <code
                      v-for="key in previewUser.permissionOverrides.allow"
                      :key="`allow-${key}`"
                      class="rounded bg-white px-1.5 py-0.5 text-[11px] dark:bg-zinc-950"
                    >{{ key }}</code>
                  </div>
                </div>
                <div>
                  <p class="mb-1 text-xs font-medium text-rose-700 dark:text-rose-300">
                    {{ t('admin.users.previewDenyOverrides') }}
                  </p>
                  <div class="flex flex-wrap gap-1">
                    <code
                      v-for="key in previewUser.permissionOverrides.deny"
                      :key="`deny-${key}`"
                      class="rounded bg-white px-1.5 py-0.5 text-[11px] dark:bg-zinc-950"
                    >{{ key }}</code>
                  </div>
                </div>
              </div>
            </section>
          </template>
        </div>

        <div class="flex flex-wrap justify-end gap-2 border-t border-slate-200 px-5 py-4 dark:border-zinc-800">
          <UButton
            v-if="previewUser"
            color="primary"
            variant="soft"
            leading-icon="i-lucide-panel-right-open"
            @click="manageFromPreview"
          >
            {{ t('admin.users.manage') }}
          </UButton>
          <UButton color="neutral" variant="ghost" @click="closeUserPreview">
            {{ t('admin.users.close') }}
          </UButton>
        </div>
      </div>
    </template>
  </UModal>

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
          <p class="mt-1 text-xs text-slate-400 dark:text-zinc-500">
            {{ t('admin.users.userId') }}: {{ selectedUser.id }}
            · {{ t('admin.users.createdAt') }}: {{ formatDateTime(selectedUser.createdAt) }}
            · {{ t('admin.users.updatedAt') }}: {{ formatDateTime(selectedUser.updatedAt) }}
          </p>
        </div>
        <UButton color="neutral" variant="ghost" icon="i-lucide-x" :aria-label="t('admin.users.close')" @click="closeUser" />
      </header>

      <div class="space-y-5 px-5 py-5">
        <SFAdminSurfaceOutlet
          page-id="/users"
          :kinds="['detail_region', 'editor_panel']"
          :context="{ resourceType: 'user', resource: selectedUser }"
          @refresh="refreshSelectedUser"
        />

        <!-- 账户信息 -->
        <section class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-zinc-800 dark:bg-zinc-900/60">
          <div class="mb-4 flex items-center justify-between gap-3">
            <div>
              <h4 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.users.accountSection') }}
              </h4>
              <p class="text-xs text-slate-500 dark:text-zinc-400">
                {{ t('admin.users.accountSectionHelp') }}
              </p>
            </div>
            <UButton
              color="primary"
              leading-icon="i-lucide-save"
              :loading="savingAccount"
              @click="saveAccount"
            >
              {{ t('admin.users.saveAccount') }}
            </UButton>
          </div>

          <div class="grid gap-3 sm:grid-cols-2">
            <label class="block space-y-1.5 text-sm">
              <span class="font-medium text-slate-700 dark:text-zinc-300">{{ t('admin.users.username') }}</span>
              <UInput v-model="editUsername" class="w-full" />
            </label>
            <label class="block space-y-1.5 text-sm">
              <span class="font-medium text-slate-700 dark:text-zinc-300">{{ t('admin.users.email') }}</span>
              <UInput v-model="editEmail" type="email" class="w-full" />
            </label>
            <label class="block space-y-1.5 text-sm">
              <span class="font-medium text-slate-700 dark:text-zinc-300">{{ t('admin.users.displayName') }}</span>
              <UInput v-model="editDisplayName" class="w-full" />
            </label>
            <label class="block space-y-1.5 text-sm">
              <span class="font-medium text-slate-700 dark:text-zinc-300">{{ t('admin.users.locale') }}</span>
              <select
                v-model="editLocale"
                class="h-9 w-full rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-700 outline-none dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-200"
              >
                <option v-for="opt in localeOptions" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </option>
              </select>
            </label>
            <label class="block space-y-1.5 text-sm sm:col-span-2">
              <span class="font-medium text-slate-700 dark:text-zinc-300">{{ t('admin.users.status') }}</span>
              <select
                v-model="editStatus"
                class="h-9 w-full max-w-xs rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-700 outline-none dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-200"
              >
                <option value="active">{{ t('admin.users.statusActive') }}</option>
                <option value="disabled">{{ t('admin.users.statusDisabled') }}</option>
                <option value="banned">{{ t('admin.users.statusBanned') }}</option>
              </select>
              <span
                v-if="selectedUser.isInitialSuperAdmin"
                class="mt-1 block text-xs text-amber-600 dark:text-amber-400"
              >
                {{ t('admin.users.initialSuperAdmin') }}
              </span>
            </label>
          </div>
          <SFAdminUserEmailVerificationControl :user="selectedUser" @updated="applyDetailToForm" />
        </section>

        <!-- 公开资料 -->
        <section class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-zinc-800 dark:bg-zinc-900/60">
          <div class="mb-4 flex items-center justify-between gap-3">
            <div>
              <h4 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.users.profileSection') }}
              </h4>
              <p class="text-xs text-slate-500 dark:text-zinc-400">
                {{ t('admin.users.profileSectionHelp') }}
              </p>
            </div>
            <UButton
              color="primary"
              leading-icon="i-lucide-save"
              :loading="savingProfile"
              @click="saveProfile"
            >
              {{ t('admin.users.saveProfile') }}
            </UButton>
          </div>

          <div class="grid gap-3">
            <label class="block space-y-1.5 text-sm">
              <span class="font-medium text-slate-700 dark:text-zinc-300">{{ t('admin.users.bio') }}</span>
              <UTextarea v-model="editBio" :rows="3" class="w-full" />
            </label>
            <label class="block space-y-1.5 text-sm">
              <span class="font-medium text-slate-700 dark:text-zinc-300">{{ t('admin.users.signature') }}</span>
              <UTextarea v-model="editSignature" :rows="2" class="w-full" />
            </label>
            <div class="grid gap-3 sm:grid-cols-2">
              <label class="block space-y-1.5 text-sm">
                <span class="font-medium text-slate-700 dark:text-zinc-300">{{ t('admin.users.location') }}</span>
                <UInput v-model="editLocation" class="w-full" />
              </label>
              <label class="block space-y-1.5 text-sm">
                <span class="font-medium text-slate-700 dark:text-zinc-300">{{ t('admin.users.websiteUrl') }}</span>
                <UInput v-model="editWebsiteUrl" type="url" placeholder="https://" class="w-full" />
              </label>
            </div>
          </div>
        </section>

        <!-- 账号安全 -->
        <section class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-zinc-800 dark:bg-zinc-900/60">
          <div class="space-y-4">
            <div>
              <h4 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.users.passwordSection') }}
              </h4>
              <p class="text-xs text-slate-500 dark:text-zinc-400">
                {{ t('admin.users.passwordSectionHelp') }}
              </p>
              <div class="mt-3 grid gap-3 sm:grid-cols-2">
                <label class="block space-y-1.5 text-sm">
                  <span class="font-medium text-slate-700 dark:text-zinc-300">{{ t('admin.users.newPassword') }}</span>
                  <UInput
                    v-model="newPassword"
                    type="password"
                    autocomplete="new-password"
                    class="w-full"
                    :placeholder="t('admin.users.newPasswordPlaceholder')"
                  />
                </label>
                <label class="block space-y-1.5 text-sm">
                  <span class="font-medium text-slate-700 dark:text-zinc-300">{{ t('admin.users.confirmPassword') }}</span>
                  <UInput
                    v-model="confirmPassword"
                    type="password"
                    autocomplete="new-password"
                    class="w-full"
                    :placeholder="t('admin.users.confirmPasswordPlaceholder')"
                  />
                </label>
              </div>
              <div class="mt-3 flex flex-wrap items-center gap-2">
                <UButton
                  color="primary"
                  leading-icon="i-lucide-key-round"
                  :loading="resettingPassword"
                  :disabled="!newPassword || !confirmPassword"
                  @click="resetUserPassword"
                >
                  {{ t('admin.users.resetPassword') }}
                </UButton>
                <p class="text-xs text-slate-500 dark:text-zinc-400">
                  {{ t('admin.users.passwordResetHint') }}
                </p>
              </div>
            </div>
            <div class="flex flex-wrap items-center justify-between gap-3 border-t border-slate-200 pt-4 dark:border-zinc-800">
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
            <div class="flex flex-wrap items-center justify-between gap-3 border-t border-slate-200 pt-4 dark:border-zinc-800">
              <div>
                <h4 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                  {{ t('admin.users.clearClientIPs') }}
                </h4>
                <p class="text-xs text-slate-500 dark:text-zinc-400">
                  {{ t('admin.users.clearClientIPsHelp') }}
                </p>
              </div>
              <UButton
                color="neutral"
                variant="soft"
                leading-icon="i-lucide-shield-off"
                :loading="clearingClientIPs"
                @click="clearUserClientIPs"
              >
                {{ t('admin.users.clearClientIPs') }}
              </UButton>
            </div>
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
