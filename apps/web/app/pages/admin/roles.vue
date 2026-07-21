<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'
import { ROLE_TEMPLATE_DEFINITIONS, type RoleTemplateDefinition } from '~/config/roleTemplates'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminRoles'
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
  permissionKeys: string[]
}

type Permission = {
  key: string
  module: string
  description: string
}

const { t } = useI18n()
const toast = useToast()
const { request } = useApiClient()
const { permissionLabel, permissionDescription, permissionModuleLabel } = usePermissionText()
const search = ref('')
const adminPage = useAdminPage('/roles')

const roles = ref<Role[]>([])
const permissions = ref<Permission[]>([])
const selectedRole = ref<Role | null>(null)
const editingNew = ref(false)
const roleModalOpen = ref(false)
const formKey = ref('')
const formAlias = ref('')
const formDescription = ref('')
const formPermissionKeys = ref<string[]>([])
/** 创建流程中选中的内置模板 key；空字符串表示不套用模板。 */
const selectedTemplateKey = ref('')
const pending = ref(false)
const saving = ref(false)
const deleting = ref(false)
const errorMessage = ref('')
const successMessage = ref('')
const roleFormSubmitted = ref(false)

onMounted(() => {
  void loadData()
})

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
    id: 'permissions',
    header: t('admin.roles.permissions')
  },
  {
    id: 'status',
    header: t('admin.roles.status')
  },
  {
    id: 'actions',
    header: t('admin.roles.actions')
  }
])

const filteredRoles = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  if (!keyword) {
    return roles.value
  }

  return roles.value.filter((role) => {
    return [role.key, role.alias, role.description]
      .some(value => value.toLowerCase().includes(keyword))
  })
})

const groupedPermissions = computed(() => {
  const groups = new Map<string, Permission[]>()
  for (const permission of permissions.value) {
    const group = groups.get(permission.module) ?? []
    group.push(permission)
    groups.set(permission.module, group)
  }
  return Array.from(groups.entries()).map(([module, items]) => ({ module, items }))
})

const permissionEditingLocked = computed(() => selectedRole.value?.key === 'super_admin')
const ROLE_TEMPLATE_NONE_VALUE = '__none__'

function selectedRoleFormEdits() {
  const role = selectedRole.value
  if (!role || editingNew.value) return null
  const currentPermissions = [...formPermissionKeys.value].sort()
  const savedPermissions = [...role.permissionKeys].sort()
  return {
    alias: formAlias.value !== role.alias,
    description: formDescription.value !== role.description,
    permissions: currentPermissions.length !== savedPermissions.length ||
      currentPermissions.some((key, index) => key !== savedPermissions[index])
  }
}

const selectedTitle = computed(() => {
  if (editingNew.value) {
    return t('admin.roles.createTitle')
  }
  return selectedRole.value?.alias || t('admin.roles.selectTitle')
})

const roleTemplates = computed(() => ROLE_TEMPLATE_DEFINITIONS)

const templateSelectItems = computed(() => [
  {
    label: t('admin.roles.templateNone'),
    value: ROLE_TEMPLATE_NONE_VALUE
  },
  ...roleTemplates.value.map(template => ({
    label: t(template.aliasKey),
    value: template.key
  }))
])

const roleKeyError = computed(() => {
  if (!roleFormSubmitted.value || !editingNew.value || formKey.value.trim()) {
    return undefined
  }
  return t('admin.roles.keyRequired')
})

const roleAliasError = computed(() => {
  if (!roleFormSubmitted.value || formAlias.value.trim()) {
    return undefined
  }
  return t('admin.roles.aliasRequired')
})

useSeoMeta({
  title: t('admin.roles.metaTitle')
})

const {
  selectedFilter: roleSuggestionFilter,
  suggestions: roleSuggestions,
  nextCursor: roleSuggestionNextCursor,
  loading: roleSuggestionsLoading,
  loadingMore: roleSuggestionsLoadingMore,
  loadError: roleSuggestionsLoadError,
  decisionOpen: roleSuggestionDecisionOpen,
  pendingDecision: pendingRoleSuggestionDecision,
  deciding: roleSuggestionDeciding,
  decisionError: roleSuggestionDecisionError,
  filterItems: roleSuggestionFilterItems,
  confirmationTitle: roleSuggestionConfirmationTitle,
  confirmationDescription: roleSuggestionConfirmationDescription,
  confirmationAction: roleSuggestionConfirmationAction,
  loadSuggestions,
  openDecision: openRoleSuggestionDecision,
  closeDecision: closeRoleSuggestionDecision,
  submitDecision: submitRoleSuggestionDecision,
  statusColor: roleSuggestionStatusColor,
  statusLabel: roleSuggestionStatusLabel,
  shortDigest: shortRoleSuggestionDigest,
  formatDate: formatRoleSuggestionDate
} = useRoleSuggestions(async (suggestion) => {
  await loadData({
    preserveUnsavedForm: true,
    appliedPermission: { roleKey: suggestion.roleKey, permissionKey: suggestion.permissionKey }
  })
})

async function loadData(options: {
  preserveUnsavedForm?: boolean
  appliedPermission?: { roleKey: string, permissionKey: string }
} = {}) {
  pending.value = true
  errorMessage.value = ''
  try {
    const [roleItems, permissionItems] = await Promise.all([
      request<Role[]>('/roles'),
      request<Permission[]>('/permissions')
    ])
    const edits = options.preserveUnsavedForm ? selectedRoleFormEdits() : null
    roles.value = roleItems
    permissions.value = permissionItems
    if (selectedRole.value) {
      const refreshed = roleItems.find(role => role.key === selectedRole.value?.key)
      if (refreshed) {
        if (edits && (edits.alias || edits.description || edits.permissions)) {
          // 逐字段采纳 Host 权威数据，避免覆盖操作员真正修改过的输入。
          selectedRole.value = refreshed
          formKey.value = refreshed.key
          if (!edits.alias) formAlias.value = refreshed.alias
          if (!edits.description) formDescription.value = refreshed.description
          if (!edits.permissions) {
            formPermissionKeys.value = [...refreshed.permissionKeys]
            selectedTemplateKey.value = ''
          } else if (
            options.appliedPermission?.roleKey === refreshed.key &&
            refreshed.permissionKeys.includes(options.appliedPermission.permissionKey) &&
            !formPermissionKeys.value.includes(options.appliedPermission.permissionKey)
          ) {
            // 刚批准的 Host 授权必须并入旧草稿，避免后续保存把它无意撤销。
            formPermissionKeys.value = [...formPermissionKeys.value, options.appliedPermission.permissionKey]
          }
        } else {
          fillRoleForm(refreshed)
        }
      }
    }
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.roles.loadFailed')
  } finally {
    pending.value = false
  }
}

/** 仅填充表单，不打开弹窗（供 loadData 刷新已打开编辑态使用）。 */
function fillRoleForm(role: Role) {
  selectedRole.value = role
  editingNew.value = false
  formKey.value = role.key
  formAlias.value = role.alias
  formDescription.value = role.description
  formPermissionKeys.value = [...role.permissionKeys]
  selectedTemplateKey.value = ''
  roleFormSubmitted.value = false
  errorMessage.value = ''
  successMessage.value = ''
}

function selectRole(role: Role) {
  fillRoleForm(role)
  roleModalOpen.value = true
}

function startCreateRole() {
  selectedRole.value = null
  editingNew.value = true
  formKey.value = ''
  formAlias.value = ''
  formDescription.value = ''
  formPermissionKeys.value = []
  selectedTemplateKey.value = ''
  roleFormSubmitted.value = false
  errorMessage.value = ''
  successMessage.value = ''
  roleModalOpen.value = true
}

function closeRoleModal() {
  roleModalOpen.value = false
  selectedRole.value = null
  editingNew.value = false
  formKey.value = ''
  formAlias.value = ''
  formDescription.value = ''
  formPermissionKeys.value = []
  selectedTemplateKey.value = ''
  roleFormSubmitted.value = false
}

function applyRoleTemplate(template: RoleTemplateDefinition, options?: { fillIdentity?: boolean }) {
  if (permissionEditingLocked.value) {
    return
  }
  formPermissionKeys.value = [...template.permissionKeys].sort()
  // 仅创建流程预填别名/说明；不预填 key，避免与已 seed 的系统模板角色键冲突。
  if (options?.fillIdentity && editingNew.value) {
    formAlias.value = t(template.aliasKey)
    formDescription.value = t(template.descriptionKey)
  }
  selectedTemplateKey.value = template.key
  successMessage.value = t('admin.roles.templateApplied', { name: t(template.aliasKey) })
  toast.add({
    title: t('admin.roles.templateApplied', { name: t(template.aliasKey) }),
    color: 'success',
    icon: 'i-lucide-copy-check'
  })
}

function onTemplateSelect(value: string | undefined) {
  const key = value === ROLE_TEMPLATE_NONE_VALUE ? '' : `${value || ''}`.trim()
  selectedTemplateKey.value = key
  if (!key) {
    return
  }
  const template = ROLE_TEMPLATE_DEFINITIONS.find(item => item.key === key)
  if (!template) {
    return
  }
  applyRoleTemplate(template, { fillIdentity: editingNew.value })
}

function togglePermission(key: string) {
  if (permissionEditingLocked.value) {
    return
  }
  if (formPermissionKeys.value.includes(key)) {
    formPermissionKeys.value = formPermissionKeys.value.filter(item => item !== key)
    return
  }
  formPermissionKeys.value = [...formPermissionKeys.value, key].sort()
}

function validateRoleForm() {
  roleFormSubmitted.value = true
  if ((editingNew.value && !formKey.value.trim()) || !formAlias.value.trim()) {
    errorMessage.value = t('admin.roles.validationFailed')
    return false
  }
  return true
}

async function saveRole() {
  errorMessage.value = ''
  successMessage.value = ''
  if (!validateRoleForm()) {
    return
  }

  saving.value = true
  try {
    if (editingNew.value) {
      const role = await request<Role>('/roles', {
        method: 'POST',
        body: {
          key: formKey.value.trim(),
          alias: formAlias.value.trim(),
          description: formDescription.value.trim()
        }
      })
      if (formPermissionKeys.value.length > 0) {
        await request<null>(`/roles/${role.key}/permissions`, {
          method: 'PUT',
          body: {
            permissions: formPermissionKeys.value
          }
        })
      }
      selectedRole.value = role
    } else if (selectedRole.value) {
      await request<Role>(`/roles/${selectedRole.value.key}`, {
        method: 'PATCH',
        body: {
          alias: formAlias.value.trim(),
          description: formDescription.value.trim()
        }
      })
      if (!permissionEditingLocked.value) {
        await request<null>(`/roles/${selectedRole.value.key}/permissions`, {
          method: 'PUT',
          body: {
            permissions: formPermissionKeys.value
          }
        })
      }
    }
    closeRoleModal()
    await loadData()
    successMessage.value = t('admin.roles.saved')
    toast.add({
      title: t('admin.roles.saved'),
      color: 'success',
      icon: 'i-lucide-check',
      duration: 10000
    })
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.roles.saveFailed')
  } finally {
    saving.value = false
  }
}

async function deleteRole() {
  if (!selectedRole.value || !selectedRole.value.isDeletable) {
    return
  }
  deleting.value = true
  errorMessage.value = ''
  successMessage.value = ''
  try {
    await request<null>(`/roles/${encodeURIComponent(selectedRole.value.key)}`, {
      method: 'DELETE'
    })
    closeRoleModal()
    await loadData()
    successMessage.value = t('admin.roles.deleted')
    toast.add({
      title: t('admin.roles.deleted'),
      color: 'success',
      icon: 'i-lucide-check',
      duration: 10000
    })
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.roles.deleteFailed')
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <div class="mb-4 flex flex-col gap-1">
    <h2 class="text-xl font-bold flex items-center gap-2 text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ t('admin.roles.title') }}
    </h2>
    <p class="text-sm text-slate-500 dark:text-zinc-400">
      {{ t('admin.roles.intro') }}
    </p>
  </div>

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
          @click="loadData()"
        >
          {{ t('admin.roles.refresh') }}
        </UButton>
        <UButton
          color="primary"
          leading-icon="i-lucide-plus"
          @click="startCreateRole"
        >
          {{ t('admin.roles.create') }}
        </UButton>
        <UBadge color="neutral" variant="soft" class="border border-slate-200 dark:border-zinc-800 font-medium">
          {{ t('admin.roles.count', { count: filteredRoles.length }) }}
        </UBadge>
      </div>
    </template>
  </UDashboardToolbar>

  <section class="mb-6 border-y border-slate-200 py-5 dark:border-zinc-800">
    <div class="mb-4 flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
      <div class="min-w-0">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-shield-check" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
          <h3 class="text-base font-semibold text-slate-950 dark:text-zinc-50">
            {{ t('admin.roles.suggestions.title') }}
          </h3>
          <UBadge color="neutral" variant="soft">
            {{ roleSuggestions.length }}
          </UBadge>
        </div>
        <p class="mt-1 max-w-3xl text-sm text-slate-500 dark:text-zinc-400">
          {{ t('admin.roles.suggestions.intro') }}
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <USelect
          v-model="roleSuggestionFilter"
          :items="roleSuggestionFilterItems"
          value-key="value"
          class="w-44"
          :aria-label="t('admin.roles.suggestions.filterLabel')"
        />
        <UButton
          color="neutral"
          variant="outline"
          icon="i-lucide-refresh-cw"
          :loading="roleSuggestionsLoading"
          :aria-label="t('admin.roles.suggestions.refresh')"
          :title="t('admin.roles.suggestions.refresh')"
          @click="loadSuggestions(true)"
        />
      </div>
    </div>

    <UAlert
      v-if="roleSuggestionsLoadError"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="roleSuggestionsLoadError"
      class="mb-4"
    />

    <div
      v-if="roleSuggestionsLoading"
      class="flex min-h-28 items-center justify-center border border-slate-200 bg-slate-50 text-sm text-slate-500 dark:border-zinc-800 dark:bg-zinc-950/40 dark:text-zinc-400"
    >
      <UIcon name="i-lucide-loader-circle" class="mr-2 size-4 animate-spin" />
      {{ t('admin.roles.suggestions.loading') }}
    </div>
    <div
      v-else-if="!roleSuggestions.length"
      class="flex min-h-28 flex-col items-center justify-center border border-dashed border-slate-300 px-4 text-center dark:border-zinc-700"
    >
      <UIcon name="i-lucide-inbox" class="mb-2 size-5 text-slate-400 dark:text-zinc-500" />
      <p class="text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.roles.suggestions.empty') }}
      </p>
    </div>
    <div v-else class="border border-slate-200 dark:border-zinc-800">
      <article
        v-for="suggestion in roleSuggestions"
        :key="suggestion.id"
        class="grid gap-3 border-b border-slate-200 p-4 last:border-b-0 dark:border-zinc-800 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center"
      >
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <code class="break-all text-sm font-semibold text-slate-950 dark:text-zinc-50">{{ suggestion.permissionKey }}</code>
            <UIcon name="i-lucide-arrow-right" class="size-4 shrink-0 text-slate-400" />
            <UBadge color="info" variant="soft" class="font-mono">
              {{ suggestion.roleKey }}
            </UBadge>
            <UBadge :color="roleSuggestionStatusColor(suggestion)" variant="soft">
              {{ roleSuggestionStatusLabel(suggestion) }}
            </UBadge>
          </div>
          <div class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500 dark:text-zinc-400">
            <span class="inline-flex min-w-0 items-center gap-1">
              <UIcon name="i-lucide-blocks" class="size-3.5 shrink-0" />
              <span class="break-all">{{ suggestion.ownerExtensionId }}@{{ suggestion.extensionVersion }}</span>
            </span>
            <span class="inline-flex items-center gap-1 font-mono" :title="suggestion.packageDigest">
              <UIcon name="i-lucide-fingerprint" class="size-3.5" />
              {{ shortRoleSuggestionDigest(suggestion.packageDigest) }}
            </span>
            <span class="inline-flex items-center gap-1">
              <UIcon name="i-lucide-clock-3" class="size-3.5" />
              {{ formatRoleSuggestionDate(suggestion.createdAt) }}
            </span>
          </div>
        </div>
        <div class="flex flex-wrap items-center gap-2 lg:justify-end">
          <template v-if="suggestion.approvalState === 'pending'">
            <UButton
              color="neutral"
              variant="outline"
              leading-icon="i-lucide-x"
              :disabled="roleSuggestionDeciding"
              @click="openRoleSuggestionDecision(suggestion, 'rejected')"
            >
              {{ t('admin.roles.suggestions.reject') }}
            </UButton>
            <UButton
              color="primary"
              leading-icon="i-lucide-shield-check"
              :disabled="roleSuggestionDeciding"
              @click="openRoleSuggestionDecision(suggestion, 'approved')"
            >
              {{ t('admin.roles.suggestions.approve') }}
            </UButton>
          </template>
          <UButton
            v-else-if="suggestion.approvalState === 'approved' && !suggestion.applied"
            color="primary"
            leading-icon="i-lucide-shield-plus"
            :disabled="roleSuggestionDeciding"
            @click="openRoleSuggestionDecision(suggestion, 'approved')"
          >
            {{ t('admin.roles.suggestions.apply') }}
          </UButton>
        </div>
      </article>
    </div>

    <div v-if="roleSuggestionNextCursor" class="mt-4 flex justify-center">
      <UButton
        color="neutral"
        variant="outline"
        leading-icon="i-lucide-chevrons-down"
        :loading="roleSuggestionsLoadingMore"
        @click="loadSuggestions(false)"
      >
        {{ t('admin.roles.suggestions.loadMore') }}
      </UButton>
    </div>

  </section>

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
        :data="filteredRoles"
        :columns="columns"
        :loading="pending"
        :empty="t('admin.roles.empty')"
        :caption="t('admin.roles.caption')"
        sticky
        class="max-h-[calc(100vh-17rem)]"
      >
        <template #key-cell="{ row }">
          <code class="rounded bg-slate-50 dark:bg-zinc-800 border border-slate-200 dark:border-zinc-700 px-2 py-1 text-xs font-semibold text-slate-900 dark:text-white">
            {{ row.original.key }}
          </code>
        </template>

        <template #alias-cell="{ row }">
          <div>
            <span class="font-semibold text-slate-900 dark:text-white text-sm">
              {{ row.original.alias }}
            </span>
            <p class="mt-0.5 text-xs text-slate-500 dark:text-zinc-400">
              {{ row.original.description || t('admin.roles.noDescription') }}
            </p>
          </div>
        </template>

        <template #permissions-cell="{ row }">
          <UBadge color="neutral" variant="soft">
            {{ t('admin.roles.permissionCount', { count: row.original.permissionKeys.length }) }}
          </UBadge>
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

        <template #actions-cell="{ row }">
          <UButton
            color="primary"
            variant="ghost"
            leading-icon="i-lucide-square-pen"
            @click="selectRole(row.original)"
          >
            {{ t('admin.roles.edit') }}
          </UButton>
        </template>
      </UTable>
    </UCard>
  </div>

  <!-- 用户组：新建 / 编辑 -->
  <UModal
    v-model:open="roleModalOpen"
    :ui="{ content: 'sm:max-w-3xl' }"
    @update:open="(open) => { if (!open) closeRoleModal() }"
  >
    <template #content>
      <div class="flex max-h-[90vh] flex-col">
        <div class="flex items-start justify-between gap-3 border-b border-slate-200 px-5 py-4 dark:border-zinc-800">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <h3 class="text-base font-bold text-slate-900 dark:text-white">
                {{ selectedTitle }}
              </h3>
              <UBadge v-if="selectedRole?.isSystem" color="info" variant="soft">
                {{ t('admin.roles.system') }}
              </UBadge>
            </div>
            <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.roles.editorIntro') }}
            </p>
          </div>
          <UButton
            type="button"
            color="neutral"
            variant="ghost"
            icon="i-lucide-x"
            :aria-label="t('admin.common.cancel')"
            @click="closeRoleModal"
          />
        </div>

        <div class="grid flex-1 gap-4 overflow-y-auto px-5 py-4">
          <UAlert
            v-if="errorMessage"
            color="error"
            variant="soft"
            icon="i-lucide-triangle-alert"
            :title="errorMessage"
          />

          <UFormField :label="t('admin.roles.key')" name="role-key" :error="roleKeyError">
            <UInput
              v-model="formKey"
              icon="i-lucide-key-round"
              :placeholder="t('admin.roles.keyPlaceholder')"
              :disabled="!editingNew"
              required
              class="w-full"
            />
          </UFormField>
          <UFormField :label="t('admin.roles.alias')" name="role-alias" :error="roleAliasError">
            <UInput
              v-model="formAlias"
              icon="i-lucide-tag"
              :placeholder="t('admin.roles.aliasPlaceholder')"
              required
              class="w-full"
            />
          </UFormField>
          <UFormField :label="t('admin.roles.description')" name="role-description">
            <textarea
              v-model="formDescription"
              class="min-h-24 w-full resize-y rounded-md border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700 outline-none transition-colors placeholder:text-slate-400 hover:border-slate-300 focus:border-[var(--sf-accent)] dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200"
              :placeholder="t('admin.roles.descriptionPlaceholder')"
            />
          </UFormField>

          <div
            v-if="!permissionEditingLocked"
            class="rounded-lg border border-slate-200 bg-white p-3 dark:border-zinc-800 dark:bg-zinc-950/40"
          >
            <div class="mb-2">
              <h4 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.roles.applyTemplate') }}
              </h4>
              <p class="text-xs text-slate-500 dark:text-zinc-400">
                {{ editingNew ? t('admin.roles.applyTemplateCreateHelp') : t('admin.roles.applyTemplateEditHelp') }}
              </p>
            </div>
            <UFormField :label="t('admin.roles.templateSelect')" name="role-template">
              <USelect
                :model-value="selectedTemplateKey || ROLE_TEMPLATE_NONE_VALUE"
                :items="templateSelectItems"
                value-key="value"
                class="w-full"
                :placeholder="t('admin.roles.templateSelectPlaceholder')"
                @update:model-value="onTemplateSelect"
              />
            </UFormField>
            <div class="mt-3 flex flex-wrap gap-2">
              <UButton
                v-for="template in roleTemplates"
                :key="template.key"
                color="neutral"
                variant="outline"
                size="sm"
                leading-icon="i-lucide-layout-template"
                class="border-slate-200 dark:border-zinc-700"
                @click="applyRoleTemplate(template, { fillIdentity: Boolean(editingNew) })"
              >
                {{ t(template.aliasKey) }}
              </UButton>
            </div>
          </div>

          <div class="rounded-lg border border-slate-200 bg-slate-50 p-3 dark:border-zinc-800 dark:bg-zinc-950/60">
            <div class="mb-3 flex items-start justify-between gap-3">
              <div>
                <h4 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                  {{ t('admin.roles.permissionEditor') }}
                </h4>
                <p class="text-xs text-slate-500 dark:text-zinc-400">
                  {{ permissionEditingLocked ? t('admin.roles.superAdminPermissionLocked') : t('admin.roles.permissionEditorHelp') }}
                </p>
              </div>
              <UBadge color="neutral" variant="soft">
                {{ formPermissionKeys.length }}
              </UBadge>
            </div>

            <div class="max-h-[36vh] space-y-4 overflow-y-auto pr-1">
              <div v-for="group in groupedPermissions" :key="group.module" class="space-y-2">
                <div class="text-xs font-semibold uppercase text-slate-500 dark:text-zinc-400">
                  {{ permissionModuleLabel(group.module) }}
                </div>
                <label
                  v-for="permission in group.items"
                  :key="permission.key"
                  class="flex cursor-pointer gap-3 rounded-md border border-slate-200 bg-white px-3 py-2 text-sm transition hover:border-[var(--sf-accent-soft-border)] dark:border-zinc-800 dark:bg-zinc-900"
                >
                  <input
                    type="checkbox"
                    class="mt-1 size-4 accent-[var(--sf-accent)]"
                    :checked="formPermissionKeys.includes(permission.key)"
                    :disabled="permissionEditingLocked"
                    @change="togglePermission(permission.key)"
                  >
                  <span class="min-w-0">
                    <span class="block font-semibold text-slate-900 dark:text-zinc-100">
                      {{ permissionLabel(permission) }}
                    </span>
                    <code class="mt-0.5 block text-xs text-slate-500 dark:text-zinc-400">{{ permission.key }}</code>
                    <span class="mt-0.5 block text-xs text-slate-500 dark:text-zinc-400">
                      {{ permissionDescription(permission) }}
                    </span>
                  </span>
                </label>
              </div>
            </div>
          </div>
        </div>

        <div class="flex flex-wrap items-center justify-between gap-3 border-t border-slate-200 px-5 py-4 dark:border-zinc-800">
          <UButton
            color="error"
            variant="outline"
            leading-icon="i-lucide-trash-2"
            :loading="deleting"
            :disabled="!selectedRole?.isDeletable || editingNew"
            @click="deleteRole"
          >
            {{ t('admin.roles.delete') }}
          </UButton>
          <div class="flex flex-wrap gap-2">
            <UButton type="button" color="neutral" variant="ghost" @click="closeRoleModal">
              {{ t('admin.common.cancel') }}
            </UButton>
            <UButton
              color="primary"
              leading-icon="i-lucide-save"
              :loading="saving"
              @click="saveRole"
            >
              {{ t('admin.roles.save') }}
            </UButton>
          </div>
        </div>
      </div>
    </template>
  </UModal>

  <!-- 角色权限建议：确认审批 -->
  <UModal
    v-model:open="roleSuggestionDecisionOpen"
    :ui="{ content: 'sm:max-w-lg' }"
    @update:open="(open) => { if (!open) closeRoleSuggestionDecision() }"
  >
    <template #content>
      <div class="p-5 sm:p-6">
        <div class="flex items-start gap-3">
          <UIcon
            :name="pendingRoleSuggestionDecision?.state === 'rejected' ? 'i-lucide-circle-x' : 'i-lucide-shield-alert'"
            class="mt-0.5 size-5 shrink-0 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]"
          />
          <div class="min-w-0">
            <h3 class="text-base font-semibold text-slate-950 dark:text-zinc-50">
              {{ roleSuggestionConfirmationTitle }}
            </h3>
            <p class="mt-1 text-sm text-slate-600 dark:text-zinc-300">
              {{ roleSuggestionConfirmationDescription }}
            </p>
          </div>
        </div>
        <UAlert
          v-if="roleSuggestionDecisionError"
          class="mt-4"
          color="error"
          variant="soft"
          icon="i-lucide-triangle-alert"
          :title="roleSuggestionDecisionError"
        />
        <div class="mt-6 flex justify-end gap-2">
          <UButton color="neutral" variant="ghost" :disabled="roleSuggestionDeciding" @click="closeRoleSuggestionDecision">
            {{ t('admin.roles.suggestions.cancel') }}
          </UButton>
          <UButton
            :color="pendingRoleSuggestionDecision?.state === 'rejected' ? 'error' : 'primary'"
            :leading-icon="pendingRoleSuggestionDecision?.state === 'rejected' ? 'i-lucide-x' : 'i-lucide-shield-check'"
            :loading="roleSuggestionDeciding"
            @click="submitRoleSuggestionDecision"
          >
            {{ roleSuggestionConfirmationAction }}
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
