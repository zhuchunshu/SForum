<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router'
import type { AdminSystemUpdateStatus } from '~/composables/admin/useAdminSystemUpdates'
import {
  isSystemUpdatePromptSuppressed,
  systemUpdatePromptRecord
} from '~/utils/admin/systemUpdatePrompt'

const props = defineProps<{
  status: AdminSystemUpdateStatus | null
  settingsRoute: RouteLocationRaw
}>()

const STORAGE_KEY = 'sforum.admin.system-update-prompt'
const { t } = useI18n()
const open = ref(false)
const mounted = ref(false)

onMounted(() => {
  mounted.value = true
  showWhenAvailable(props.status)
})

watch(() => props.status, (status) => {
  if (mounted.value) showWhenAvailable(status)
})

function showWhenAvailable(status: AdminSystemUpdateStatus | null) {
  if (!status?.updateAvailable || status.state !== 'update_available') return

  try {
    if (isSystemUpdatePromptSuppressed(localStorage.getItem(STORAGE_KEY))) return
    localStorage.setItem(STORAGE_KEY, systemUpdatePromptRecord())
  } catch {
    // Storage may be unavailable in hardened browsers; the prompt still works for this page load.
  }
  open.value = true
}

function close() {
  open.value = false
}
</script>

<template>
  <UModal v-model:open="open" :ui="{ content: 'sm:max-w-lg' }">
    <template #content>
      <div class="flex flex-col">
        <header class="flex items-start justify-between gap-4 border-b border-slate-200 px-5 py-4 dark:border-zinc-800">
          <div class="flex min-w-0 items-start gap-3">
            <div class="grid size-10 shrink-0 place-items-center rounded-md bg-amber-100 text-amber-700 dark:bg-amber-400/15 dark:text-amber-300">
              <UIcon name="i-lucide-arrow-up-circle" class="size-5" />
            </div>
            <div class="min-w-0">
              <h2 class="text-base font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.shell.updatePrompt.title') }}
              </h2>
              <p class="mt-1 text-sm leading-6 text-slate-600 dark:text-zinc-300">
                {{ t('admin.shell.updatePrompt.description', { version: status?.latestVersion }) }}
              </p>
            </div>
          </div>
          <UButton
            icon="i-lucide-x"
            color="neutral"
            variant="ghost"
            :aria-label="t('admin.shell.updatePrompt.later')"
            :title="t('admin.shell.updatePrompt.later')"
            @click="close"
          />
        </header>

        <dl class="grid grid-cols-2 gap-4 px-5 py-5 text-sm">
          <div class="min-w-0">
            <dt class="text-xs font-medium text-muted">{{ t('admin.settings.updates.currentVersion') }}</dt>
            <dd class="mt-1 truncate font-mono font-semibold">{{ status?.currentVersion }}</dd>
          </div>
          <div class="min-w-0">
            <dt class="text-xs font-medium text-muted">{{ t('admin.settings.updates.latestVersion') }}</dt>
            <dd class="mt-1 truncate font-mono font-semibold text-amber-700 dark:text-amber-300">{{ status?.latestVersion }}</dd>
          </div>
        </dl>

        <footer class="flex flex-col-reverse gap-2 border-t border-slate-200 px-5 py-4 sm:flex-row sm:justify-end dark:border-zinc-800">
          <UButton color="neutral" variant="soft" @click="close">
            {{ t('admin.shell.updatePrompt.later') }}
          </UButton>
          <UButton
            :to="status?.releaseUrl || settingsRoute"
            :target="status?.releaseUrl ? '_blank' : undefined"
            color="warning"
            trailing-icon="i-lucide-external-link"
            @click="close"
          >
            {{ status?.releaseUrl ? t('admin.settings.updates.viewRelease') : t('admin.shell.updatePrompt.openSettings') }}
          </UButton>
        </footer>
      </div>
    </template>
  </UModal>
</template>
