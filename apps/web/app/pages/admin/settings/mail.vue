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
const { fetchAdminEnvelope } = useWebOptions()
const adminPage = useAdminPage('/settings/mail')
const adminRoutes = useAdminRoutes()
const toast = useToast()
const activeView = ref('overview')
const pending = ref(true)
const saving = ref(false)
const errorMessage = ref('')
const testRecipient = ref('')
const testRecipientError = ref('')
// 来自 site.admin_email 的预填默认值；仅作建议，测试邮件可不保存该字段。
const adminEmailDefault = ref('')
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
// 无可用提供商时不渲染空 Select，改为引导启用插件。
const hasProviders = computed(() => providers.value.length > 0)
// SMTP 设置入口仅在启用中的 sforum.smtp 提供方可选时展示，避免禁用后仍可进入插件页。
const smtpProviderAvailable = computed(() => providers.value.some(item => item.extensionId === 'sforum.smtp'))
const eventRows = computed(() => [
  { key: 'reply' as const, label: t('admin.mailSettings.reply') },
  { key: 'mention' as const, label: t('admin.mailSettings.mention') },
  { key: 'moderation' as const, label: t('admin.mailSettings.moderationResult') }
])

async function loadAdminEmailDefault() {
  try {
    const envelope = await fetchAdminEnvelope()
    const items = envelope.data || []
    const adminEmail = items.find(item => item.name === 'site.admin_email')?.value?.trim() || ''
    adminEmailDefault.value = adminEmail
    // 仅在输入框仍为空时预填，避免覆盖运营已改的测试收件人。
    if (!testRecipient.value.trim() && adminEmail) {
      testRecipient.value = adminEmail
    }
  } catch {
    adminEmailDefault.value = ''
  }
}

async function load() {
  pending.value = true
  errorMessage.value = ''
  try {
    const [state, deliveryState, policyState] = await Promise.all([
      request<{ items: Provider[], selected: { extensionId?: string }, configured: boolean }>('/admin/mail/providers'),
      request<{ items: Delivery[] }>('/admin/mail/deliveries'),
      request<Policy>('/admin/mail/policy'),
      loadAdminEmailDefault()
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
  const explicit = testRecipient.value.trim()
  // 输入框可空：后端会回落到 site.admin_email；两者皆空才提示。
  if (explicit && !/^\S+@\S+\.\S+$/.test(explicit)) {
    testRecipientError.value = t('admin.mailSettings.invalidRecipient')
    return
  }
  if (!explicit && !adminEmailDefault.value) {
    testRecipientError.value = t('admin.mailSettings.recipientOrAdminEmailRequired')
    return
  }
  await runAction(async () => {
    const body = explicit ? { recipient: explicit } : {}
    await request('/admin/mail/test', { method: 'POST', body })
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
      <h1 class="mail-admin-title flex items-center gap-2 font-bold text-slate-900 dark:text-zinc-50"><UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)]" />{{ t('admin.mailSettings.title') }}</h1>
      <p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">{{ t('admin.mailSettings.description') }}</p>
    </header>
    <UDashboardToolbar class="rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-500 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400">
      <template #left><UTabs v-model="activeView" :items="tabs" size="sm" /></template>
      <template #right><UButton icon="i-lucide-rotate-cw" color="neutral" variant="ghost" :loading="pending" @click="load">{{ t('admin.home.refresh') }}</UButton></template>
    </UDashboardToolbar>
    <SFAlert v-if="errorMessage" variant="danger" :title="errorMessage" closable @close="errorMessage = ''" />

    <section class="rounded-lg border border-teal-200 bg-teal-50/80 p-4 dark:border-teal-900/60 dark:bg-teal-950/30">
      <div class="flex gap-3">
        <div class="grid size-10 shrink-0 place-items-center rounded-lg bg-white text-teal-700 shadow-sm dark:bg-teal-900/60 dark:text-teal-200"><UIcon name="i-lucide-sparkles" class="size-5" /></div>
        <div><h2 class="text-base font-bold text-teal-950 dark:text-teal-100">{{ t('admin.mailSettings.recommendedTitle') }}</h2><p class="mt-1 max-w-4xl text-sm leading-6 text-teal-800 dark:text-teal-200">{{ t('admin.mailSettings.recommendedDescription') }}</p></div>
      </div>
    </section>

    <UCard v-if="activeView === 'overview'" class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <template #header><div><h2 class="text-base font-bold">{{ t('admin.mailSettings.gettingStarted') }}</h2><p class="mt-1 text-xs text-slate-500">{{ t('admin.mailSettings.description') }}</p></div></template>
      <div class="grid gap-4 md:grid-cols-2">
        <div class="rounded-lg border border-emerald-200 bg-emerald-50/60 p-4 dark:border-emerald-900 dark:bg-emerald-950/20"><div class="flex items-center justify-between gap-3"><span class="text-sm font-semibold">{{ t('admin.mailSettings.inAppStatus') }}</span><UBadge color="success" variant="soft">{{ t('admin.mailSettings.ready') }}</UBadge></div><p class="mt-2 text-sm text-slate-600 dark:text-zinc-300">{{ t('admin.mailSettings.inAppReady') }}</p></div>
        <div class="rounded-lg border p-4" :class="configured ? 'border-emerald-200 bg-emerald-50/60 dark:border-emerald-900 dark:bg-emerald-950/20' : 'border-amber-200 bg-amber-50/70 dark:border-amber-900 dark:bg-amber-950/20'"><div class="flex items-center justify-between gap-3"><span class="text-sm font-semibold">{{ t('admin.mailSettings.mailStatus') }}</span><UBadge :color="configured ? 'success' : 'warning'" variant="soft">{{ configured ? t('admin.mailSettings.ready') : t('admin.mailSettings.needsSetup') }}</UBadge></div><p class="mt-2 text-sm text-slate-600 dark:text-zinc-300">{{ configured ? t('admin.mailSettings.providerReady') : t('admin.mailSettings.inAppContinues') }}</p></div>
      </div>
      <ol class="mt-6 grid gap-3 lg:grid-cols-3">
        <li v-for="(step, index) in [{ title: t('admin.mailSettings.stepProvider'), help: t('admin.mailSettings.stepProviderHelp') }, { title: t('admin.mailSettings.stepConfigure'), help: t('admin.mailSettings.stepConfigureHelp') }, { title: t('admin.mailSettings.stepTest'), help: t('admin.mailSettings.stepTestHelp') }]" :key="step.title" class="flex gap-3 rounded-lg border border-slate-200 p-4 dark:border-zinc-800"><span class="grid size-7 shrink-0 place-items-center rounded-full bg-[var(--sf-accent)] text-xs font-bold text-white">{{ index + 1 }}</span><span><strong class="block text-sm">{{ step.title }}</strong><span class="mt-1 block text-xs leading-5 text-slate-500">{{ step.help }}</span></span></li>
      </ol>
    </UCard>

    <UCard v-else-if="activeView === 'mail'" class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <template #header><div><h2 class="text-base font-bold">{{ t('admin.mailSettings.providerConfigTitle') }}</h2><p class="mt-1 text-xs text-slate-500">{{ t('admin.mailSettings.providerConfigDescription') }}</p></div></template>
      <div class="max-w-4xl space-y-6">
        <!-- 无提供商：空 Select 会显示成空白选中，对运营不友好，改为明确引导 -->
        <div v-if="!pending && !hasProviders" class="rounded-lg border border-dashed border-slate-200 bg-slate-50/80 p-6 dark:border-zinc-700 dark:bg-zinc-950/40">
          <SFEmptyState
            icon-label="MAIL"
            :title="t('admin.mailSettings.noProvidersTitle')"
            :description="t('admin.mailSettings.noProvidersDescription')"
          />
          <div class="mt-5 flex flex-wrap justify-center gap-2">
            <UButton icon="i-lucide-plug" color="primary" :to="adminRoutes.path('/extensions/plugins')">
              {{ t('admin.mailSettings.openPlugins') }}
            </UButton>
            <UButton icon="i-lucide-rotate-cw" color="neutral" variant="subtle" :loading="pending" @click="load">
              {{ t('admin.home.refresh') }}
            </UButton>
          </div>
        </div>

        <template v-else>
          <SFAlert v-if="!configured" variant="info" :title="t('admin.mailSettings.unconfigured')" :description="t('admin.mailSettings.inAppContinues')" />
          <div>
            <div class="flex flex-col gap-3 lg:flex-row lg:items-end">
              <UFormField class="min-w-0 flex-1" :label="t('admin.mailSettings.provider')" :description="t('admin.mailSettings.providerHelp')">
                <USelect
                  v-model="selected"
                  :items="providerItems"
                  value-key="value"
                  class="w-full"
                  :placeholder="t('admin.mailSettings.providerPlaceholder')"
                />
              </UFormField>
              <div class="flex flex-wrap gap-2">
                <UButton icon="i-lucide-save" :disabled="!selected" :loading="saving" @click="chooseProvider">
                  {{ t('admin.mailSettings.save') }}
                </UButton>
                <UButton icon="i-lucide-rotate-ccw" color="neutral" variant="subtle" :loading="saving" @click="resetProvider">
                  {{ t('admin.mailSettings.reset') }}
                </UButton>
              </div>
            </div>
            <UButton
              v-if="smtpProviderAvailable"
              class="mt-3"
              :to="adminRoutes.path('/extensions/sforum.smtp/pages/settings')"
              icon="i-lucide-settings"
              color="neutral"
              variant="outline"
            >
              {{ t('admin.mailSettings.smtpSettings') }}
            </UButton>
          </div>
          <div class="border-t border-slate-200 pt-5 dark:border-zinc-800">
            <h3 class="font-semibold">{{ t('admin.mailSettings.testTitle') }}</h3>
            <p class="mt-1 text-sm text-slate-500">{{ adminEmailDefault ? t('admin.mailSettings.testHelpWithAdminEmail') : t('admin.mailSettings.testHelp') }}</p>
            <div class="mt-3 flex flex-col gap-2 sm:flex-row">
              <UInput
                v-model="testRecipient"
                type="email"
                class="flex-1"
                :placeholder="adminEmailDefault || t('admin.mailSettings.testRecipientPlaceholder')"
              />
              <UButton icon="i-lucide-send" :disabled="!configured" :loading="saving" @click="testMail">
                {{ t('admin.mailSettings.sendTest') }}
              </UButton>
            </div>
            <p v-if="testRecipientError" class="mt-1 text-sm text-red-600">{{ testRecipientError }}</p>
          </div>
        </template>
      </div>
    </UCard>

    <UCard v-else-if="activeView === 'notifications'" class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900" :ui="{ footer: 'border-t border-slate-200 dark:border-zinc-800 p-4 sm:px-6' }">
      <template #header><div><h2 class="text-base font-bold">{{ t('admin.mailSettings.policyTitle') }}</h2><p class="mt-1 text-xs text-slate-500">{{ t('admin.mailSettings.policyHelp') }}</p></div></template>
      <div class="max-w-4xl space-y-5">
      <div class="grid grid-cols-[1fr_90px_90px] gap-3 border-b border-slate-200 pb-2 text-xs font-semibold text-slate-500 dark:border-zinc-800"><span>{{ t('admin.mailSettings.event') }}</span><span>{{ t('admin.mailSettings.inApp') }}</span><span>{{ t('admin.mailSettings.mail') }}</span></div>
      <div v-for="row in eventRows" :key="row.key" class="grid grid-cols-[1fr_90px_90px] items-center gap-3"><span class="text-sm font-medium">{{ row.label }}</span><USwitch v-model="policy[row.key].inAppEnabled" /><USwitch v-model="policy[row.key].emailEnabled" /></div>
      </div>
      <template #footer><div class="flex flex-wrap gap-2"><UButton icon="i-lucide-save" :loading="saving" @click="savePolicy">{{ t('admin.mailSettings.save') }}</UButton><UButton icon="i-lucide-rotate-ccw" color="neutral" variant="subtle" @click="restorePolicy">{{ t('admin.mailSettings.reset') }}</UButton><UButton icon="i-lucide-bell-ring" color="neutral" variant="outline" @click="testNotification">{{ t('admin.mailSettings.testNotification') }}</UButton><UButton to="/notifications" icon="i-lucide-external-link" color="neutral" variant="ghost">{{ t('admin.mailSettings.openInbox') }}</UButton></div></template>
    </UCard>

    <UCard v-else class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <template #header><div><h2 class="text-base font-bold">{{ t('admin.mailSettings.recentDeliveries') }}</h2><p class="mt-1 text-xs text-slate-500">{{ t('admin.mailSettings.deliveriesHelp') }}</p></div></template>
      <div class="overflow-x-auto">
      <div class="min-w-[680px] divide-y divide-slate-200 text-sm dark:divide-zinc-800"><div class="grid grid-cols-[1.2fr_1fr_100px_1fr] gap-3 pb-2 text-xs font-semibold text-slate-500"><span>{{ t('admin.mailSettings.recipient') }}</span><span>{{ t('admin.mailSettings.template') }}</span><span>{{ t('admin.mailSettings.status') }}</span><span>{{ t('admin.mailSettings.reason') }}</span></div><div v-for="item in deliveries" :key="item.id" class="grid grid-cols-[1.2fr_1fr_100px_1fr] gap-3 py-3"><span>{{ item.recipient }}</span><span>{{ item.templateKey }}</span><SFBadge>{{ item.status }}</SFBadge><span class="text-slate-500">{{ item.reason || item.errorSummary || '-' }}</span></div><p v-if="!deliveries.length" class="py-6 text-slate-500">{{ t('admin.mailSettings.noDeliveries') }}</p></div>
      </div>
    </UCard>
  </div>
</template>

<style scoped>
.mail-admin-title {
  font-size: 1.25rem;
  line-height: 1.75rem;
}
</style>
