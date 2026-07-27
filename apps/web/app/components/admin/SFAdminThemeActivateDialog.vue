<script setup lang="ts">
import type { AdminExtension } from '~/utils/admin/adminExtensions'

export type ThemeActivateImpact = {
  contribution: { id: string, action: string, target?: string, path?: string }
  conflicts?: Array<{ id: string, extensionId: string }>
}

const props = defineProps<{
  extension: AdminExtension | null
  impacts: ThemeActivateImpact[]
  addCount: number
  replaceCount: number
  busy?: boolean
  // 重新激活当前主题时用更贴切的标题/按钮文案。
  reactivating?: boolean
}>()

const emit = defineEmits<{
  cancel: []
  confirm: []
}>()

const open = defineModel<boolean>('open', { required: true })
const { t } = useI18n()

const impactRows = computed(() => props.impacts.map((impact) => {
  const destination = impact.contribution.target || impact.contribution.path || impact.contribution.id
  const conflicts = (impact.conflicts || [])
    .map(conflict => `${conflict.extensionId}:${conflict.id}`)
    .join(', ')
  return {
    key: `${impact.contribution.action}:${destination}`,
    action: impact.contribution.action,
    destination,
    conflicts
  }
}))

function cancel() {
  emit('cancel')
}

function confirm() {
  emit('confirm')
}
</script>

<template>
  <UModal v-model:open="open" :ui="{ content: 'sm:max-w-2xl' }">
    <template #content>
      <div class="flex max-h-[min(88vh,720px)] flex-col">
        <header class="shrink-0 border-b border-slate-200 px-5 py-4 sm:px-6 dark:border-zinc-800">
          <div class="flex items-start justify-between gap-4">
            <div class="min-w-0">
              <h2 class="text-base font-semibold text-slate-900 dark:text-zinc-100">
                {{ reactivating
                  ? t('admin.extensions.confirmThemeReactivationTitle')
                  : t('admin.extensions.confirmThemeActivationTitle') }}
              </h2>
              <p class="mt-1 text-sm leading-6 text-slate-600 dark:text-zinc-300">
                {{ t('admin.extensions.confirmThemeActivation', {
                  name: extension?.name || extension?.id || '',
                  addCount,
                  replaceCount
                }) }}
              </p>
            </div>
            <UButton
              icon="i-lucide-x"
              color="neutral"
              variant="ghost"
              :aria-label="t('admin.extensions.confirmThemeActivationCancel')"
              :disabled="busy"
              @click="cancel"
            />
          </div>
        </header>

        <div class="min-h-0 flex-1 space-y-4 overflow-y-auto px-5 py-4 sm:px-6">
          <UAlert
            v-if="replaceCount > 0"
            color="warning"
            variant="subtle"
            icon="i-lucide-shield-alert"
            :title="t('admin.extensions.confirmThemeActivationCoreWarning')"
          />

          <div v-if="impactRows.length" class="space-y-2">
            <h3 class="text-sm font-medium text-slate-900 dark:text-zinc-100">
              {{ t('admin.extensions.confirmThemeActivationImpacts', { count: impactRows.length }) }}
            </h3>
            <ul class="divide-y divide-slate-200 overflow-hidden rounded-lg border border-slate-200 dark:divide-zinc-800 dark:border-zinc-800">
              <li
                v-for="row in impactRows"
                :key="row.key"
                class="flex flex-wrap items-start gap-2 px-3 py-2.5 text-sm"
              >
                <UBadge
                  :color="row.action === 'replace' ? 'warning' : 'primary'"
                  variant="subtle"
                  class="shrink-0 uppercase"
                >
                  {{ row.action }}
                </UBadge>
                <span class="min-w-0 break-all text-slate-700 dark:text-zinc-200">
                  {{ row.destination }}
                </span>
                <span
                  v-if="row.conflicts"
                  class="w-full text-xs text-slate-500 dark:text-zinc-400"
                >
                  {{ t('admin.extensions.confirmThemeActivationConflicts', { conflicts: row.conflicts }) }}
                </span>
              </li>
            </ul>
          </div>
          <p v-else class="text-sm text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.confirmThemeActivationNoImpacts') }}
          </p>
        </div>

        <footer class="shrink-0 border-t border-slate-200 px-5 py-4 sm:px-6 dark:border-zinc-800">
          <div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
            <UButton color="neutral" variant="ghost" :disabled="busy" @click="cancel">
              {{ t('admin.extensions.confirmThemeActivationCancel') }}
            </UButton>
            <UButton
              color="primary"
              :icon="reactivating ? 'i-lucide-refresh-cw' : 'i-lucide-play'"
              :loading="busy"
              @click="confirm"
            >
              {{ reactivating
                ? t('admin.extensions.confirmThemeReactivationAction')
                : t('admin.extensions.confirmThemeActivationAction') }}
            </UButton>
          </div>
        </footer>
      </div>
    </template>
  </UModal>
</template>
