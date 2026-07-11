<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminMailSettings' })
const { t } = useI18n()
const { request } = useApiClient()
const adminPage = useAdminPage('/settings/mail')
const adminRoutes = useAdminRoutes()
const toast = useToast()
type Provider = { extensionId: string, label: string, healthy: boolean }
type Delivery = { id: number, recipient: string, templateKey: string, status: string, reason?: string, createdAt: string }
const providers = ref<Provider[]>([])
const selected = ref('')
const configured = ref(false)
const deliveries = ref<Delivery[]>([])
const recipient = ref('')
const pending = ref(true)
const errorMessage = ref('')
async function load() { pending.value = true; errorMessage.value = ''; try { const state = await request<{ items: Provider[], selected: { extensionId?: string }, configured: boolean }>('/admin/mail/providers'); providers.value = state.items; selected.value = state.selected?.extensionId || ''; configured.value = state.configured; deliveries.value = (await request<{ items: Delivery[] }>('/admin/mail/deliveries')).items } catch (error) { errorMessage.value = apiErrorMessage(error) || t('admin.mailSettings.loadFailed') } finally { pending.value = false } }
await load()
async function choose() { try { await request('/admin/mail/provider', { method: 'PUT', body: { extensionId: selected.value } }); configured.value = true; toast.add({ color: 'success', icon: 'i-lucide-check', title: t('admin.mailSettings.saved'), duration: 10000 }) } catch (error) { errorMessage.value = apiErrorMessage(error) || t('admin.mailSettings.saveFailed') } }
async function reset() { try { await request('/admin/mail/provider/reset', { method: 'POST' }); selected.value = ''; configured.value = false; toast.add({ color: 'success', icon: 'i-lucide-rotate-ccw', title: t('admin.mailSettings.resetDone'), description: t('admin.mailSettings.secretsPreserved'), duration: 10000 }) } catch (error) { errorMessage.value = apiErrorMessage(error) || t('admin.mailSettings.saveFailed') } }
async function testMail() { try { await request('/admin/mail/test', { method: 'POST', body: { recipient: recipient.value } }); toast.add({ color: 'success', icon: 'i-lucide-send', title: t('admin.mailSettings.testQueued'), duration: 10000 }); await load() } catch (error) { errorMessage.value = apiErrorMessage(error) || t('admin.mailSettings.testFailed') } }
</script>

<template>
  <div class="space-y-5">
    <header><h1 class="text-xl font-bold text-slate-900 dark:text-zinc-50"><UIcon :name="adminPage.icon" class="mr-2 inline-block size-5 text-[var(--sf-accent)]" />{{ t('admin.mailSettings.title') }}</h1><p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">{{ t('admin.mailSettings.description') }}</p></header>
    <SFAlert v-if="errorMessage" variant="danger" :title="errorMessage" closable @close="errorMessage = ''" />
    <SFAlert v-if="!configured && !pending" variant="info" :title="t('admin.mailSettings.unconfigured')" :message="t('admin.mailSettings.inAppContinues')" />
    <SFCard class="space-y-4 p-6">
      <div class="flex flex-wrap items-end gap-3"><label class="min-w-64 flex-1 text-sm font-semibold text-slate-700 dark:text-zinc-300">{{ t('admin.mailSettings.provider') }}<select v-model="selected" class="sf-input mt-2 w-full"><option value="">{{ t('admin.mailSettings.noProvider') }}</option><option v-for="item in providers" :key="item.extensionId" :value="item.extensionId">{{ item.label }}{{ item.healthy ? '' : ` (${t('admin.mailSettings.unhealthy')})` }}</option></select></label><UButton icon="i-lucide-save" :disabled="!selected" @click="choose">{{ t('admin.mailSettings.save') }}</UButton><UButton icon="i-lucide-rotate-ccw" color="neutral" variant="subtle" @click="reset">{{ t('admin.mailSettings.reset') }}</UButton></div>
      <div class="flex justify-end"><UButton :to="adminRoutes.path('/extensions/sforum.smtp/pages/settings')" icon="i-lucide-settings" color="neutral" variant="outline">{{ t('admin.mailSettings.smtpSettings') }}</UButton></div>
    </SFCard>
    <SFCard class="space-y-3 p-6"><h2 class="font-semibold">{{ t('admin.mailSettings.testTitle') }}</h2><div class="flex gap-2"><input v-model="recipient" type="email" class="sf-input flex-1" :placeholder="t('admin.mailSettings.testRecipientPlaceholder')"><UButton icon="i-lucide-send" :disabled="!recipient" @click="testMail">{{ t('admin.mailSettings.sendTest') }}</UButton></div></SFCard>
    <SFCard class="p-6"><h2 class="mb-3 font-semibold">{{ t('admin.mailSettings.recentDeliveries') }}</h2><div class="divide-y divide-slate-200 text-sm dark:divide-zinc-800"><div v-for="item in deliveries" :key="item.id" class="grid gap-2 py-3 sm:grid-cols-[1fr_1fr_auto]"><span>{{ item.recipient }}</span><span class="text-slate-500">{{ item.templateKey }}</span><SFBadge>{{ item.status }}</SFBadge></div><p v-if="!deliveries.length" class="py-4 text-slate-500">{{ t('admin.mailSettings.noDeliveries') }}</p></div></SFCard>
  </div>
</template>

<style scoped>.sf-input { border: 1px solid #d1d5db; border-radius: 6px; padding: .5rem .75rem; background: transparent; }</style>
