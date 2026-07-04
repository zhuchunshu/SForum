<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'

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
const { request } = useApiClient()
const { permissionLabel, permissionDescription, permissionModuleLabel } = usePermissionText()
const search = ref('')
const adminPage = useAdminPage('/roles')

const roles = ref<Role[]>([])
const permissions = ref<Permission[]>([])
const selectedRole = ref<Role | null>(null)
const editingNew = ref(false)
const formKey = ref('')
const formAlias = ref('')
const formDescription = ref('')
const formPermissionKeys = ref<string[]>([])
const pending = ref(false)
const saving = ref(false)
const deleting = ref(false)
const errorMessage = ref('')
const successMessage = ref('')

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

const selectedTitle = computed(() => {
  if (editingNew.value) {
    return t('admin.roles.createTitle')
  }
  return selectedRole.value?.alias || t('admin.roles.selectTitle')
})

useSeoMeta({
  title: t('admin.roles.metaTitle')
})

async function loadData() {
  pending.value = true
  errorMessage.value = ''
  try {
    const [roleItems, permissionItems] = await Promise.all([
      request<Role[]>('/roles'),
      request<Permission[]>('/permissions')
    ])
    roles.value = roleItems
    permissions.value = permissionItems
    if (selectedRole.value) {
      const refreshed = roleItems.find(role => role.key === selectedRole.value?.key)
      if (refreshed) {
        selectRole(refreshed)
      }
    }
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.roles.loadFailed')
  } finally {
    pending.value = false
  }
}

function selectRole(role: Role) {
  selectedRole.value = role
  editingNew.value = false
  formKey.value = role.key
  formAlias.value = role.alias
  formDescription.value = role.description
  formPermissionKeys.value = [...role.permissionKeys]
  successMessage.value = ''
}

function startCreateRole() {
  selectedRole.value = null
  editingNew.value = true
  formKey.value = ''
  formAlias.value = ''
  formDescription.value = ''
  formPermissionKeys.value = []
  successMessage.value = ''
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

async function saveRole() {
  saving.value = true
  errorMessage.value = ''
  successMessage.value = ''
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
    await loadData()
    successMessage.value = t('admin.roles.saved')
    editingNew.value = false
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
    await request<null>(`/roles/${selectedRole.value.key}`, {
      method: 'DELETE'
    })
    selectedRole.value = null
    await loadData()
    startCreateRole()
    successMessage.value = t('admin.roles.deleted')
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
          @click="loadData"
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

  <div class="grid gap-4 xl:grid-cols-[minmax(0,1fr)_420px]">
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

    <UCard class="border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100">
      <template #header>
        <div class="flex items-start justify-between gap-3">
          <div>
            <h3 class="text-sm font-semibold text-slate-950 dark:text-zinc-50">
              {{ selectedTitle }}
            </h3>
            <p class="text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.roles.editorIntro') }}
            </p>
          </div>
          <UBadge v-if="selectedRole?.isSystem" color="info" variant="soft">
            {{ t('admin.roles.system') }}
          </UBadge>
        </div>
      </template>

      <div class="space-y-4">
        <UInput
          v-model="formKey"
          icon="i-lucide-key-round"
          :placeholder="t('admin.roles.keyPlaceholder')"
          :disabled="!editingNew"
        />
        <UInput
          v-model="formAlias"
          icon="i-lucide-tag"
          :placeholder="t('admin.roles.aliasPlaceholder')"
        />
        <textarea
          v-model="formDescription"
          class="min-h-24 w-full resize-y rounded-md border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700 outline-none transition-colors placeholder:text-slate-400 hover:border-slate-300 focus:border-[var(--sf-accent)] dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200"
          :placeholder="t('admin.roles.descriptionPlaceholder')"
        />

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

          <div class="max-h-[42vh] space-y-4 overflow-y-auto pr-1">
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

        <div class="flex flex-wrap items-center justify-between gap-3 border-t border-slate-200 pt-4 dark:border-zinc-800">
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
    </UCard>
  </div>
</template>
