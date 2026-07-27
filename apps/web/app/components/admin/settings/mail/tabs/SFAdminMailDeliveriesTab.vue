<script setup lang="ts">
import { mailDeliveryCodeKey, type MailDelivery } from '../model'
const { t, te } = useI18n()
const { request } = useApiClient()
const pending = ref(true)
const items = ref<MailDelivery[]>([])
const errorMessage = ref('')
onMounted(load)
defineExpose({ refresh: load, pending })
async function load() { pending.value = true; try { items.value = (await request<{ items: MailDelivery[] }>('/admin/mail/deliveries')).items } catch (error) { errorMessage.value = apiErrorMessage(error) || t('admin.mailSettings.loadFailed') } finally { pending.value = false } }
function label(group: 'deliveryStatus' | 'templates' | 'reasons', code: string) { const key = mailDeliveryCodeKey(group, code); return te(key) ? t(key) : code }
function result(item: MailDelivery) {
  const reason = item.reason?.trim() || ''
  const summary = item.errorSummary?.trim() || ''
  const reasonLabel = reason ? label('reasons', reason) : ''
  return reasonLabel && summary && summary !== reason && summary !== reasonLabel ? `${reasonLabel} (${summary})` : reasonLabel || summary || '-'
}
function color(status: string): 'success' | 'error' | 'warning' | 'neutral' | 'info' { return status === 'sent' ? 'success' : status === 'failed' ? 'error' : ['queued', 'sending'].includes(status) ? 'warning' : status === 'skipped' ? 'neutral' : 'info' }
</script>
<template>
  <UCard class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
    <template #header><div><h2 class="text-base font-bold">{{ t('admin.mailSettings.recentDeliveries') }}</h2><p class="mt-1 text-xs text-muted">{{ t('admin.mailSettings.deliveriesHelp') }}</p></div></template>
    <SFAlert v-if="errorMessage" variant="danger" :title="errorMessage" />
    <div class="overflow-x-auto"><table class="min-w-full text-sm"><thead class="text-left text-xs text-muted"><tr><th class="p-3">{{ t('admin.mailSettings.recipient') }}</th><th class="p-3">{{ t('admin.mailSettings.template') }}</th><th class="p-3">{{ t('admin.mailSettings.status') }}</th><th class="p-3">{{ t('admin.mailSettings.reason') }}</th></tr></thead><tbody><tr v-for="item in items" :key="item.id" class="border-b border-slate-200 dark:border-zinc-800"><td class="p-3">{{ item.recipient }}</td><td class="p-3">{{ label('templates', item.templateKey) }}</td><td class="p-3"><UBadge :color="color(item.status)" variant="soft">{{ label('deliveryStatus', item.status) }}</UBadge></td><td class="p-3 text-xs">{{ result(item) }}</td></tr></tbody></table></div>
    <SFEmptyState v-if="!pending && items.length === 0" :title="t('admin.mailSettings.noDeliveries')" />
  </UCard>
</template>
