<script setup lang="ts">
import type { AdminExtension } from '~/utils/adminExtensions'

const props = defineProps<{ extension: AdminExtension }>()
const emit = defineEmits<{ changed: [] }>()
const extension = computed(() => props.extension)
const { user } = useAuthSession()
const { t } = useI18n()
const { status, exactTrustManaged, error, busy, mutate, challenge } = useAdminFrontendTrust(extension)
const isSuperAdmin = computed(() => user.value?.roleKeys?.includes('super_admin') === true)
const confirmOpen = ref(false)
const confirmationPhrase = ref('')
const confirmationCode = ref('')
const currentChallenge = ref<Awaited<ReturnType<typeof challenge>>>()
const acknowledged = ref(false)
const isPrebuilt = computed(() => status.value?.kind === 'prebuilt_component')
const canConfirm = computed(() => confirmationPhrase.value === extension.value.id
  && confirmationCode.value === currentChallenge.value?.code
  && acknowledged.value)

async function requestGrant() {
  confirmationPhrase.value = ''
  confirmationCode.value = ''
  acknowledged.value = false
  try {
    currentChallenge.value = await challenge()
    confirmOpen.value = true
  } catch {
    // composable 已把错误显示在面板内。
  }
}

async function confirmGrant() {
  const component = status.value?.component
  const digest = status.value?.digest
  if (!component || !digest || !canConfirm.value || !currentChallenge.value) return
  await mutate('grant', {
    challengeId: currentChallenge.value.challengeId,
    code: confirmationCode.value,
    extensionId: extension.value.id,
    version: extension.value.version,
    digest,
    apiVersion: component.apiVersion,
    componentId: component.id,
    phrase: confirmationPhrase.value,
    acknowledged: true
  })
  emit('changed')
  if (status.value?.trustState === 'trusted') confirmOpen.value = false
}

function closeConfirmation() {
  confirmOpen.value = false
}
</script>

<template>
  <section v-if="status && status.kind !== 'none'" class="mt-4 border-t border-slate-200 pt-4 dark:border-zinc-800">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h4 class="text-sm font-semibold">{{ t('admin.extensions.frontend.title') }}</h4>
        <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ status?.trustState || t('admin.extensions.frontend.loading') }}</p>
      </div>
      <div v-if="isSuperAdmin && !exactTrustManaged" class="flex gap-2">
        <UButton v-if="['required', 'invalidated', 'revoked'].includes(status?.trustState || '')" size="xs" icon="i-lucide-shield-check" :loading="busy" @click="requestGrant">{{ t('admin.extensions.frontend.grant') }}</UButton>
        <UButton v-if="status?.trustState === 'trusted'" size="xs" color="error" variant="subtle" icon="i-lucide-shield-x" :loading="busy" @click="mutate('revoke')">{{ t('admin.extensions.frontend.revoke') }}</UButton>
      </div>
    </div>
    <UAlert v-if="error" class="mt-3" color="error" icon="i-lucide-triangle-alert" :description="error" />
    <UAlert
      v-if="exactTrustManaged && status?.trustState !== 'trusted'"
      class="mt-3"
      color="warning"
      variant="subtle"
      icon="i-lucide-shield-alert"
      :title="t('admin.extensions.frontend.exactTrustTitle')"
      :description="t('admin.extensions.frontend.exactTrustDescription')"
    />
    <UAlert
      v-if="isPrebuilt"
      class="mt-3"
      color="warning"
      variant="subtle"
      icon="i-lucide-shield-alert"
      :title="t('admin.extensions.frontend.prebuiltTitle')"
      :description="t('admin.extensions.frontend.prebuiltDescription')"
    />
    <dl v-if="status?.component" class="mt-3 grid gap-2 text-xs sm:grid-cols-2">
      <div><dt class="text-slate-500">Digest</dt><dd class="break-all font-mono">{{ status.digest }}</dd></div>
      <div><dt class="text-slate-500">API / Component</dt><dd>{{ status.component.apiVersion }} / {{ status.component.id }}</dd></div>
      <div class="sm:col-span-2"><dt class="text-slate-500">Entry</dt><dd class="break-all font-mono">{{ status.component.entry }}</dd></div>
    </dl>
  </section>

  <UModal v-model:open="confirmOpen" :title="t('admin.extensions.frontend.confirmTitle')">
    <template #body>
      <div class="space-y-4">
        <UAlert
          color="error"
          variant="subtle"
          icon="i-lucide-triangle-alert"
          :title="t('admin.extensions.frontend.fullTrustTitle')"
          :description="t('admin.extensions.frontend.fullTrustDescription', { name: extension.name, version: extension.version })"
        />
        <div>
          <label class="text-sm font-medium">{{ t('admin.extensions.frontend.confirmPhrase', { id: extension.id }) }}</label>
          <UInput v-model="confirmationPhrase" class="mt-2 w-full" :placeholder="extension.id" autocomplete="off" />
        </div>
        <div>
          <label class="text-sm font-medium">{{ t('admin.extensions.frontend.confirmCode') }}</label>
          <div class="mt-2 flex items-center gap-3">
            <code class="rounded bg-slate-100 px-3 py-2 font-mono text-base tracking-[0.3em] dark:bg-zinc-800">{{ currentChallenge?.code }}</code>
            <UInput v-model="confirmationCode" class="max-w-40" inputmode="numeric" maxlength="6" autocomplete="off" />
          </div>
        </div>
        <UCheckbox v-model="acknowledged" :label="t('admin.extensions.frontend.confirmAcknowledge')" />
      </div>
    </template>
    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton color="neutral" variant="ghost" @click="closeConfirmation">{{ t('common.cancel') }}</UButton>
        <UButton color="error" icon="i-lucide-shield-check" :loading="busy" :disabled="!canConfirm" @click="confirmGrant">
          {{ t('admin.extensions.frontend.confirmGrant') }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
