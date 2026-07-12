<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminExtensionPages'
})

const { t } = useI18n()
const toast = useToast()
const adminPage = useAdminPage('/extensions/pages')
const { request } = useApiClient()
const { user } = useAuthSession()
const isSuperAdmin = computed(() => user.value?.roleKeys?.includes('super_admin') === true)

type PageRow = {
  page: {
    id: string
    pathPattern: string
    access: string
    contractVersion: string
    replaceable: boolean
  }
  provider: string
  extensionId?: string
  contributionId?: string
  candidates?: Array<{ id: string, extensionId: string, version?: string, packageDigest?: string, template?: string }>
}

const pending = ref(true)
const error = ref('')
const rows = ref<PageRow[]>([])

async function load() {
  pending.value = true
  error.value = ''
  try {
    rows.value = await request<PageRow[]>('/admin/pages')
  } catch (e) {
    error.value = apiErrorMessage(e) || t('admin.extensions.pages.loadFailed')
    rows.value = []
  } finally {
    pending.value = false
  }
}

async function restoreCore(pageId: string) {
  try {
    await request(`/admin/pages/${encodeURIComponent(pageId)}/restore-core`, { method: 'POST' })
    toast.add({
      color: 'primary',
      icon: 'i-lucide-rotate-ccw',
      title: t('admin.extensions.pages.restored'),
      duration: 10000
    })
    await load()
  } catch (e) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-alert-circle',
      title: apiErrorMessage(e) || t('admin.extensions.pages.restoreFailed')
    })
  }
}

async function approve(pageId: string, candidate: { id: string, extensionId: string, version?: string, packageDigest?: string, template?: string }) {
  if (!isSuperAdmin.value) {
    return
  }
  try {
    await request(`/admin/pages/${encodeURIComponent(pageId)}/approve`, {
      method: 'POST',
      body: {
        extensionId: candidate.extensionId,
        contributionId: candidate.id,
        version: candidate.version || '',
        packageDigest: candidate.packageDigest || '',
        templatePath: candidate.template || ''
      }
    })
    toast.add({
      color: 'primary',
      icon: 'i-lucide-check',
      title: t('admin.extensions.pages.approved'),
      duration: 10000
    })
    await load()
  } catch (e) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-alert-circle',
      title: apiErrorMessage(e) || t('admin.extensions.pages.approveFailed')
    })
  }
}

useSeoMeta({
  title: t('admin.extensions.pages.metaTitle')
})

void load()
</script>

<template>
  <div data-testid="admin-extension-pages" class="min-w-0 shrink-0">
    <div class="mb-4 flex flex-col gap-1">
      <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
        <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
        {{ t('admin.extensions.pages.title') }}
      </h2>
      <p class="text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.extensions.pages.intro') }}
      </p>
    </div>

    <div class="mb-4 flex justify-end">
      <UButton
        color="neutral"
        variant="soft"
        icon="i-lucide-refresh-cw"
        :loading="pending"
        @click="load"
      >
        {{ t('admin.common.refresh') }}
      </UButton>
    </div>

    <SFAlert
      v-if="error"
      variant="danger"
      :title="error"
      class="mb-4"
    />

    <div class="overflow-x-auto rounded-xl border border-slate-200 dark:border-zinc-800">
      <table class="min-w-full text-left text-sm">
        <thead class="bg-slate-50 text-slate-600 dark:bg-zinc-900 dark:text-zinc-300">
          <tr>
            <th class="px-3 py-2 font-medium">{{ t('admin.extensions.pages.colId') }}</th>
            <th class="px-3 py-2 font-medium">{{ t('admin.extensions.pages.colPath') }}</th>
            <th class="px-3 py-2 font-medium">{{ t('admin.extensions.pages.colAccess') }}</th>
            <th class="px-3 py-2 font-medium">{{ t('admin.extensions.pages.colProvider') }}</th>
            <th class="px-3 py-2 font-medium">{{ t('admin.extensions.pages.colActions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="row in rows"
            :key="row.page.id"
            class="border-t border-slate-100 dark:border-zinc-800"
          >
            <td class="px-3 py-2 font-mono text-xs">{{ row.page.id }}</td>
            <td class="px-3 py-2 font-mono text-xs">{{ row.page.pathPattern || '—' }}</td>
            <td class="px-3 py-2">{{ row.page.access }}</td>
            <td class="px-3 py-2">
              <span class="font-medium">{{ row.provider }}</span>
              <span
                v-if="row.candidates?.length"
                class="ml-2 text-xs text-slate-500"
              >
                ({{ row.candidates.length }} {{ t('admin.extensions.pages.candidates') }})
              </span>
            </td>
            <td class="px-3 py-2">
              <div class="flex flex-wrap gap-2">
                <UButton
                  v-if="row.provider !== 'core' && row.page.replaceable"
                  size="xs"
                  color="neutral"
                  variant="soft"
                  icon="i-lucide-rotate-ccw"
                  @click="restoreCore(row.page.id)"
                >
                  {{ t('admin.extensions.pages.restoreCore') }}
                </UButton>
                <template v-if="isSuperAdmin && row.page.replaceable">
                  <UButton
                    v-for="c in row.candidates || []"
                    :key="c.id"
                    size="xs"
                    color="primary"
                    variant="soft"
                    icon="i-lucide-check"
                    @click="approve(row.page.id, c)"
                  >
                    {{ t('admin.extensions.pages.approve') }}: {{ c.extensionId }}
                  </UButton>
                </template>
              </div>
            </td>
          </tr>
          <tr v-if="!pending && rows.length === 0">
            <td
              colspan="5"
              class="px-3 py-8 text-center text-slate-500"
            >
              {{ t('admin.extensions.pages.empty') }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
