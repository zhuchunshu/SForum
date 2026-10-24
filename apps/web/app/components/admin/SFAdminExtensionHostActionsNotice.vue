<script setup lang="ts">
import type { AdminExtension } from '~/utils/admin/adminExtensions'
import { extensionCanRequestInstall } from '~/utils/admin/extensions/hostActions'

defineOptions({ name: 'SFAdminExtensionHostActionsNotice' })

const props = defineProps<{
  extension: AdminExtension | null
}>()

const { t } = useI18n()
const declaresInstall = computed(() => extensionCanRequestInstall(props.extension))
</script>

<template>
  <!-- 单独一条披露，不并入 capability 风险等级：它是宿主动作，不是进程能力。 -->
  <div
    v-if="declaresInstall"
    data-testid="extension-host-actions-notice"
    class="mt-2 flex min-w-0 items-start gap-2 rounded-md border border-slate-200 px-3 py-2 text-xs dark:border-zinc-700"
  >
    <UIcon name="i-lucide-download" class="mt-0.5 size-3.5 shrink-0 text-slate-500 dark:text-zinc-400" />
    <div class="min-w-0">
      <p class="font-medium text-slate-700 dark:text-zinc-200">
        {{ t('admin.extensions.hostActions.title') }}
      </p>
      <p class="mt-0.5 break-words text-slate-500 dark:text-zinc-400">
        {{ t('admin.extensions.hostActions.body') }}
      </p>
    </div>
  </div>
</template>
