<script setup lang="ts">
import { recommendedMailPolicy, type MailPolicy } from '../model'
const { t } = useI18n()
const toast = useToast()
const { request } = useApiClient()
const pending = ref(true)
const saving = ref(false)
const policy = reactive(recommendedMailPolicy())
const errorMessage = ref('')
const rows = computed(() => [{ key: 'reply' as const, label: t('admin.mailSettings.reply') }, { key: 'mention' as const, label: t('admin.mailSettings.mention') }, { key: 'moderation' as const, label: t('admin.mailSettings.moderationResult') }])
onMounted(load)
defineExpose({ refresh: load, pending })
async function load() { pending.value = true; try { Object.assign(policy, await request<MailPolicy>('/admin/mail/policy')) } catch (error) { errorMessage.value = apiErrorMessage(error) || t('admin.mailSettings.loadFailed') } finally { pending.value = false } }
async function act(path: string, body?: MailPolicy) { saving.value = true; try { const result = await request<MailPolicy>(path, { method: path.endsWith('restore') ? 'POST' : 'PUT', body }); if (result) Object.assign(policy, result); toast.add({ color: 'success', icon: 'i-lucide-check', title: t(path.endsWith('restore') ? 'admin.mailSettings.policyRestored' : 'admin.mailSettings.policySaved'), duration: 10000 }) } catch (error) { errorMessage.value = apiErrorMessage(error) || t('admin.mailSettings.saveFailed') } finally { saving.value = false } }
async function testNotification() {
  saving.value = true
  try {
    await request('/admin/notifications/test', { method: 'POST' })
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('admin.mailSettings.notificationCreated'), duration: 10000 })
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.mailSettings.saveFailed')
  } finally {
    saving.value = false
  }
}
</script>
<template>
  <UCard class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
    <template #header><h2 class="text-base font-bold">{{ t('admin.mailSettings.policyTitle') }}</h2></template>
    <SFAlert v-if="errorMessage" variant="danger" :title="errorMessage" />
    <div class="divide-y divide-slate-200 dark:divide-zinc-800"><div v-for="row in rows" :key="row.key" class="grid grid-cols-[1fr_auto_auto] items-center gap-4 py-3"><strong>{{ row.label }}</strong><UCheckbox v-model="policy[row.key].inAppEnabled" :label="t('admin.mailSettings.inApp')" /><UCheckbox v-model="policy[row.key].emailEnabled" :label="t('admin.mailSettings.mail')" /></div></div>
    <template #footer><div class="flex flex-wrap gap-2"><UButton :loading="saving" icon="i-lucide-save" @click="act('/admin/mail/policy', policy)">{{ t('admin.common.save') }}</UButton><UButton color="neutral" variant="outline" icon="i-lucide-rotate-ccw" @click="act('/admin/mail/policy/restore')">{{ t('admin.mailSettings.restorePolicy') }}</UButton><UButton color="neutral" variant="ghost" icon="i-lucide-bell" @click="testNotification">{{ t('admin.mailSettings.testNotification') }}</UButton><UButton to="/notifications" color="neutral" variant="ghost" icon="i-lucide-external-link">{{ t('admin.mailSettings.openInbox') }}</UButton></div></template>
  </UCard>
</template>
