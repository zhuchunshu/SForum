<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminMailSettings' })

type Provider = { extensionId: string, label: string, healthy: boolean }
type Delivery = { id: number, recipient: string, templateKey: string, status: string, reason?: string, errorSummary?: string, createdAt: string }
type ChannelPolicy = { inAppEnabled: boolean, emailEnabled: boolean }
type Policy = { reply: ChannelPolicy, mention: ChannelPolicy, moderation: ChannelPolicy }

const { t } = useI18n()
const { request } = useApiClient()
const adminPage = useAdminPage('/settings/mail')
const adminRoutes = useAdminRoutes()
const toast = useToast()
const activeView = ref('overview')
const pending = ref(true)
const saving = ref(false)
const errorMessage = ref('')
const testRecipient = ref('')
const testRecipientError = ref('')
const providers = ref<Provider[]>([])
const selected = ref('')
const configured = ref(false)
const deliveries = ref<Delivery[]>([])
const policy = reactive<Policy>({
  reply: { inAppEnabled: true, emailEnabled: true },
  mention: { inAppEnabled: true, emailEnabled: true },
  moderation: { inAppEnabled: true, emailEnabled: true }
})

const tabs = computed(() => [
  { label: t('admin.mailSettings.overview'), value: 'overview', icon: 'i-lucide-layout-dashboard' },
  { label: t('admin.mailSettings.mail'), value: 'mail', icon: 'i-lucide-mail' },
  { label: t('admin.mailSettings.inApp'), value: 'notifications', icon: 'i-lucide-bell' },
  { label: t('admin.mailSettings.deliveries'), value: 'deliveries', icon: 'i-lucide-list-checks' }
])
const providerItems = computed(() => providers.value.map(item => ({ label: `${item.label}${item.healthy ? '' : ` (${t('admin.mailSettings.unhealthy')})`}`, value: item.extensionId })))
const eventRows = computed(() => [
  { key: 'reply' as const, label: t('admin.mailSettings.reply') },
  { key: 'mention' as const, label: t('admin.mailSettings.mention') },
  { key: 'moderation' as const, label: t('admin.mailSettings.moderationResult') }
])

async function load() {
  pending.value = true
  errorMessage.value = ''
  try {
    const [state, deliveryState, policyState] = await Promise.all([
      request<{ items: Provider[], selected: { extensionId?: string }, configured: boolean }>('/admin/mail/providers'),
      request<{ items: Delivery[] }>('/admin/mail/deliveries'),
      request<Policy>('/admin/mail/policy')
    ])
    providers.value = state.items
    selected.value = state.selected?.extensionId || ''
    configured.value = state.configured
    deliveries.value = deliveryState.items
    Object.assign(policy, policyState)
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.mailSettings.loadFailed')
  } finally {
    pending.value = false
  }
}

await load()

async function chooseProvider() {
  await runAction(async () => {
    await request('/admin/mail/provider', { method: 'PUT', body: { extensionId: selected.value } })
    configured.value = true
  }, 'admin.mailSettings.saved')
}

async function resetProvider() {
  await runAction(async () => {
    await request('/admin/mail/provider/reset', { method: 'POST' })
    selected.value = ''
    configured.value = false
  }, 'admin.mailSettings.resetDone', 'admin.mailSettings.secretsPreserved')
}

async function savePolicy() {
  await runAction(() => request('/admin/mail/policy', { method: 'PUT', body: policy }), 'admin.mailSettings.policySaved')
}

async function restorePolicy() {
  await runAction(async () => Object.assign(policy, await request<Policy>('/admin/mail/policy/restore', { method: 'POST' })), 'admin.mailSettings.policyRestored')
}

async function testMail() {
  testRecipientError.value = ''
  if (!/^\S+@\S+\.\S+$/.test(testRecipient.value.trim())) {
    testRecipientError.value = t('admin.mailSettings.invalidRecipient')
    return
  }
  await runAction(async () => {
    await request('/admin/mail/test', { method: 'POST', body: { recipient: testRecipient.value } })
    await load()
  }, 'admin.mailSettings.testQueued')
}

async function testNotification() {
  await runAction(() => request('/admin/notifications/test', { method: 'POST' }), 'admin.mailSettings.notificationCreated')
}

async function runAction(action: () => Promise<unknown>, titleKey: string, descriptionKey?: string) {
  saving.value = true
  errorMessage.value = ''
  try {
    await action()
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t(titleKey), description: descriptionKey ? t(descriptionKey) : undefined, duration: 10000 })
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.mailSettings.saveFailed')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-5">
    <header>
      <h1 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-50"><UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)]" />{{ t('admin.mailSettings.title') }}</h1>
      <p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">{{ t('admin.mailSettings.description') }}</p>
    </header>
    <UDashboardToolbar class="border-y border-slate-200 px-1 py-2 dark:border-zinc-800">
      <template #left><UTabs v-model="activeView" :items="tabs" size="sm" /></template>
      <template #right><UButton icon="i-lucide-rotate-cw" color="neutral" variant="ghost" :loading="pending" @click="load">{{ t('admin.home.refresh') }}</UButton></template>
    </UDashboardToolbar>
    <SFAlert v-if="errorMessage" variant="danger" :title="errorMessage" closable @close="errorMessage = ''" />

    <section v-if="activeView === 'overview'" class="grid gap-4 md:grid-cols-2">
      <div class="border-l-2 border-[var(--sf-accent)] px-4 py-2"><h2 class="font-semibold">{{ t('admin.mailSettings.inApp') }}</h2><p class="mt-1 text-sm text-slate-500">{{ t('admin.mailSettings.inAppReady') }}</p></div>
      <div class="border-l-2 border-slate-300 px-4 py-2"><h2 class="font-semibold">{{ t('admin.mailSettings.mail') }}</h2><p class="mt-1 text-sm text-slate-500">{{ configured ? t('admin.mailSettings.providerReady') : t('admin.mailSettings.inAppContinues') }}</p></div>
    </section>

    <section v-else-if="activeView === 'mail'" class="max-w-3xl space-y-6">
      <SFAlert v-if="!configured" variant="info" :title="t('admin.mailSettings.unconfigured')" :message="t('admin.mailSettings.inAppContinues')" />
      <div class="grid gap-3 sm:grid-cols-[1fr_auto_auto] sm:items-end">
        <UFormField :label="t('admin.mailSettings.provider')"><USelect v-model="selected" :items="providerItems" value-key="value" class="w-full" /></UFormField>
        <UButton icon="i-lucide-save" :disabled="!selected" :loading="saving" @click="chooseProvider">{{ t('admin.mailSettings.save') }}</UButton>
        <UButton icon="i-lucide-rotate-ccw" color="neutral" variant="subtle" :loading="saving" @click="resetProvider">{{ t('admin.mailSettings.reset') }}</UButton>
      </div>
      <UButton :to="adminRoutes.path('/extensions/sforum.smtp/pages/settings')" icon="i-lucide-settings" color="neutral" variant="outline">{{ t('admin.mailSettings.smtpSettings') }}</UButton>
      <div class="border-t border-slate-200 pt-5 dark:border-zinc-800">
        <h2 class="font-semibold">{{ t('admin.mailSettings.testTitle') }}</h2><p class="mt-1 text-sm text-slate-500">{{ t('admin.mailSettings.testDesc') }}</p>
        <div class="mt-3 flex flex-col gap-2 sm:flex-row"><UInput v-model="testRecipient" type="email" class="flex-1" :placeholder="t('admin.mailSettings.testRecipientPlaceholder')" /><UButton icon="i-lucide-send" :loading="saving" @click="testMail">{{ t('admin.mailSettings.sendTest') }}</UButton></div>
        <p v-if="testRecipientError" class="mt-1 text-sm text-red-600">{{ testRecipientError }}</p>
      </div>
    </section>

    <section v-else-if="activeView === 'notifications'" class="max-w-3xl space-y-5">
      <div class="grid grid-cols-[1fr_90px_90px] gap-3 border-b border-slate-200 pb-2 text-xs font-semibold text-slate-500 dark:border-zinc-800"><span>{{ t('admin.mailSettings.event') }}</span><span>{{ t('admin.mailSettings.inApp') }}</span><span>{{ t('admin.mailSettings.mail') }}</span></div>
      <div v-for="row in eventRows" :key="row.key" class="grid grid-cols-[1fr_90px_90px] items-center gap-3"><span class="text-sm font-medium">{{ row.label }}</span><USwitch v-model="policy[row.key].inAppEnabled" /><USwitch v-model="policy[row.key].emailEnabled" /></div>
      <div class="flex flex-wrap gap-2 border-t border-slate-200 pt-4 dark:border-zinc-800"><UButton icon="i-lucide-save" :loading="saving" @click="savePolicy">{{ t('admin.mailSettings.save') }}</UButton><UButton icon="i-lucide-rotate-ccw" color="neutral" variant="subtle" @click="restorePolicy">{{ t('admin.mailSettings.reset') }}</UButton><UButton icon="i-lucide-bell-ring" color="neutral" variant="outline" @click="testNotification">{{ t('admin.mailSettings.testNotification') }}</UButton><UButton to="/notifications" icon="i-lucide-external-link" color="neutral" variant="ghost">{{ t('admin.mailSettings.openInbox') }}</UButton></div>
    </section>

    <section v-else class="overflow-x-auto">
      <div class="min-w-[680px] divide-y divide-slate-200 text-sm dark:divide-zinc-800"><div class="grid grid-cols-[1.2fr_1fr_100px_1fr] gap-3 pb-2 text-xs font-semibold text-slate-500"><span>{{ t('admin.mailSettings.recipient') }}</span><span>{{ t('admin.mailSettings.template') }}</span><span>{{ t('admin.mailSettings.status') }}</span><span>{{ t('admin.mailSettings.reason') }}</span></div><div v-for="item in deliveries" :key="item.id" class="grid grid-cols-[1.2fr_1fr_100px_1fr] gap-3 py-3"><span>{{ item.recipient }}</span><span>{{ item.templateKey }}</span><SFBadge>{{ item.status }}</SFBadge><span class="text-slate-500">{{ item.reason || item.errorSummary || '-' }}</span></div><p v-if="!deliveries.length" class="py-6 text-slate-500">{{ t('admin.mailSettings.noDeliveries') }}</p></div>
    </section>
  </div>
</template>
