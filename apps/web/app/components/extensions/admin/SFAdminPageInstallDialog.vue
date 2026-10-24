<script setup lang="ts">
import type {
  AdminHostInstallRequest,
  AdminHostInstallResult,
  AdminPageInstallState
} from '~/utils/admin/extensions/hostActions'

defineOptions({ name: 'SFAdminPageInstallDialog' })

const props = defineProps<{
  request: AdminHostInstallRequest
  state: AdminPageInstallState
  result: AdminHostInstallResult | null
  /** Localized failure text for state=error; already mapped to a stable code. */
  error: string
  /** Permission key the declared type needs, e.g. extension.plugin.manage. */
  permission: string
  isSuperAdmin: boolean
}>()
const emit = defineEmits<{
  confirm: []
  cancel: []
  retry: []
  close: []
  openManagement: []
}>()
const open = defineModel<boolean>('open', { required: true })
const { t } = useI18n()

let restoreFocusTo: HTMLElement | null = null
const urlHost = computed(() => {
  try {
    return new URL(props.request.url).host
  } catch {
    return props.request.url
  }
})
const busy = computed(() => props.state === 'installing')
const typeLabelKey = computed(() => props.request.expectedType === 'theme'
  ? 'admin.extensionInstall.hostAction.typeTheme'
  : 'admin.extensionInstall.hostAction.typePlugin')
const resultTypeLabelKey = computed(() => props.result?.type === 'theme'
  ? 'admin.extensionInstall.hostAction.typeTheme'
  : 'admin.extensionInstall.hostAction.typePlugin')
const managementKey = computed(() => props.result?.type === 'theme'
  ? 'admin.extensionInstall.hostAction.openThemes'
  : 'admin.extensionInstall.hostAction.openPlugins')

// 焦点管理：打开时记住触发元素并把焦点送到主按钮，关闭后还回去。
// 宿主不得把键盘用户困在对话框里，也不得让焦点掉回 body。
// 焦点管理：宿主只负责“记住触发元素并在关闭后归还”。对话框内的初始焦点与
// 焦点陷阱由 UModal/FocusScope 负责——它落在关闭按钮上（后果性动作不是默认
// 焦点），Tab 一次即可到达“安装”。不要和 FocusScope 抢焦点。
function syncFocus(visible: boolean) {
  if (!import.meta.client) return
  if (visible) {
    restoreFocusTo = document.activeElement instanceof HTMLElement ? document.activeElement : null
    return
  }
  const previous = restoreFocusTo
  restoreFocusTo = null
  previous?.focus()
}

watch(open, syncFocus)

function cancel() {
  if (busy.value) return
  emit('cancel')
}
</script>

<template>
  <UModal
    v-model:open="open"
    :ui="{ content: 'sm:max-w-lg' }"
    :close="false"
    :dismissible="false"
  >
    <template #content>
      <div
        data-testid="admin-page-install-dialog"
        class="flex max-h-[min(88vh,760px)] min-h-0 min-w-0 flex-col"
      >
        <header class="shrink-0 border-b border-slate-200 px-5 py-4 dark:border-zinc-800">
          <div class="flex items-start justify-between gap-4">
            <div class="min-w-0">
              <h2 class="text-base font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.extensionInstall.hostAction.title') }}
              </h2>
              <p class="mt-1 break-all text-sm text-slate-600 dark:text-zinc-300">
                {{ t('admin.extensionInstall.hostAction.subtitle', { host: urlHost }) }}
              </p>
            </div>
            <UButton
              icon="i-lucide-x"
              color="neutral"
              variant="ghost"
              :disabled="busy"
              :aria-label="t('admin.extensionInstall.hostAction.close')"
              :title="t('admin.extensionInstall.hostAction.close')"
              @click="emit('close')"
            />
          </div>
        </header>

        <div class="min-h-0 flex-1 space-y-4 overflow-y-auto px-5 py-4">
          <template v-if="state === 'success' && result">
            <div class="flex min-w-0 items-start gap-3">
              <span class="flex size-9 shrink-0 items-center justify-center rounded-full bg-emerald-500/15 text-emerald-700 dark:text-emerald-300">
                <UIcon name="i-lucide-package-check" class="size-5" />
              </span>
              <div class="min-w-0">
                <p class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                  {{ result.upgraded
                    ? t('admin.extensionInstall.result.upgradeStaged')
                    : t('admin.extensionInstall.result.installed') }}
                </p>
                <p class="mt-1 break-all text-xs text-slate-500 dark:text-zinc-400">
                  {{ result.extensionId }} · {{ result.version }}
                </p>
              </div>
            </div>
            <dl class="grid gap-3 text-sm">
              <div class="min-w-0">
                <dt class="text-xs font-medium text-slate-500 dark:text-zinc-400">
                  {{ t('admin.extensionInstall.hostAction.packageDigest') }}
                </dt>
                <dd class="mt-1 break-all font-mono text-xs text-slate-700 dark:text-zinc-200">
                  {{ result.packageDigest || '—' }}
                </dd>
              </div>
            </dl>
            <UAlert
              color="warning"
              variant="subtle"
              icon="i-lucide-shield-alert"
              :title="t('admin.extensionInstall.hostAction.stagedOnlyTitle')"
              :description="t('admin.extensionInstall.hostAction.stagedOnlyBody')"
            />
          </template>

          <template v-else-if="state === 'error'">
            <UAlert
              color="error"
              variant="subtle"
              icon="i-lucide-triangle-alert"
              role="alert"
              :title="error || t('admin.extensionInstall.hostAction.failedTitle')"
            />
            <p class="text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.extensionInstall.hostAction.failedHint') }}
            </p>
          </template>

          <template v-else>
            <UAlert
              v-if="state === 'installing'"
              color="info"
              variant="subtle"
              icon="i-lucide-loader-circle"
              :title="t('admin.extensionInstall.hostAction.installingTitle')"
              :description="t('admin.extensionInstall.hostAction.installingHint')"
            />
            <dl class="grid gap-3 text-sm">
              <div class="min-w-0">
                <dt class="text-xs font-medium text-slate-500 dark:text-zinc-400">
                  {{ t('admin.extensionInstall.hostAction.url') }}
                </dt>
                <dd class="mt-1 min-w-0 break-all text-slate-900 dark:text-zinc-100">
                  {{ urlHost }}
                  <details class="mt-1">
                    <summary class="cursor-pointer text-xs text-[var(--sf-accent)]">
                      {{ t('admin.extensionInstall.hostAction.expandUrl') }}
                    </summary>
                    <code class="mt-1 block break-all font-mono text-xs text-slate-600 dark:text-zinc-300">
                      {{ request.url }}
                    </code>
                  </details>
                </dd>
              </div>
              <div class="min-w-0">
                <dt class="text-xs font-medium text-slate-500 dark:text-zinc-400">
                  {{ t('admin.extensionInstall.hostAction.sha256') }}
                </dt>
                <dd class="mt-1 break-all font-mono text-xs text-slate-700 dark:text-zinc-200">
                  {{ request.sha256 }}
                </dd>
              </div>
              <div class="grid min-w-0 gap-3 sm:grid-cols-3">
                <div class="min-w-0">
                  <dt class="text-xs font-medium text-slate-500 dark:text-zinc-400">
                    {{ t('admin.extensionInstall.hostAction.extensionId') }}
                  </dt>
                  <dd class="mt-1 break-all font-mono text-xs text-slate-900 dark:text-zinc-100">
                    {{ request.expectedExtensionId }}
                  </dd>
                </div>
                <div class="min-w-0">
                  <dt class="text-xs font-medium text-slate-500 dark:text-zinc-400">
                    {{ t('admin.extensionInstall.hostAction.version') }}
                  </dt>
                  <dd class="mt-1 break-all font-mono text-xs text-slate-900 dark:text-zinc-100">
                    {{ request.expectedVersion || '—' }}
                  </dd>
                </div>
                <div class="min-w-0">
                  <dt class="text-xs font-medium text-slate-500 dark:text-zinc-400">
                    {{ t('admin.extensionInstall.hostAction.type') }}
                  </dt>
                  <dd class="mt-1 break-all text-xs text-slate-900 dark:text-zinc-100">
                    {{ t(typeLabelKey) }}
                  </dd>
                </div>
              </div>
              <div class="min-w-0">
                <dt class="text-xs font-medium text-slate-500 dark:text-zinc-400">
                  {{ t('admin.extensionInstall.hostAction.permission') }}
                </dt>
                <dd class="mt-1 break-all font-mono text-xs text-slate-700 dark:text-zinc-200">
                  {{ permission }}
                </dd>
              </div>
            </dl>

            <UAlert
              v-if="!isSuperAdmin"
              color="warning"
              variant="subtle"
              icon="i-lucide-lock-keyhole"
              :title="t('admin.extensionInstall.hostAction.superAdminTitle')"
              :description="t('admin.extensionInstall.hostAction.superAdminBody')"
            />
            <UAlert
              color="neutral"
              variant="subtle"
              icon="i-lucide-shield-check"
              :title="t('admin.extensionInstall.hostAction.verifyTitle')"
              :description="t('admin.extensionInstall.hostAction.verifyBody')"
            />
          </template>
        </div>

        <footer class="flex shrink-0 flex-wrap justify-end gap-2 border-t border-slate-200 px-5 py-4 dark:border-zinc-800">
          <template v-if="state === 'success'">
            <UButton color="neutral" variant="ghost" @click="emit('close')">
              {{ t('admin.extensionInstall.hostAction.close') }}
            </UButton>
            <UButton icon="i-lucide-external-link" @click="emit('openManagement')">
              {{ t(managementKey) }}
            </UButton>
          </template>
          <template v-else-if="state === 'error'">
            <UButton color="neutral" variant="ghost" @click="emit('close')">
              {{ t('admin.extensionInstall.hostAction.close') }}
            </UButton>
            <UButton icon="i-lucide-rotate-ccw" @click="emit('retry')">
              {{ t('admin.extensionInstall.hostAction.retry') }}
            </UButton>
          </template>
          <template v-else>
            <UButton color="neutral" variant="ghost" :disabled="busy" @click="cancel">
              {{ t('admin.extensionInstall.hostAction.cancel') }}
            </UButton>
            <UButton
              icon="i-lucide-download"
              :loading="busy"
              :disabled="busy"
              @click="emit('confirm')"
            >
              {{ busy
                ? t('admin.extensionInstall.hostAction.installing')
                : t('admin.extensionInstall.hostAction.confirm') }}
            </UButton>
          </template>
        </footer>
      </div>
    </template>
  </UModal>
</template>
