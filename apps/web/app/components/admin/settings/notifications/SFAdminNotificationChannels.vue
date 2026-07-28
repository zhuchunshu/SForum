<script setup lang="ts">
import type { ProviderSlotItem } from '~/composables/admin/useAdminProviderSlots'
import { useAdminNotificationChannels, type NotificationChannelDelivery } from '~/composables/notifications/useAdminNotificationChannels'
import { apiErrorMessage } from '~/composables/useApiClient'

const props = defineProps<{ canManage: boolean }>()
const { t } = useI18n()
const toast = useToast()
const api = useAdminNotificationChannels()
const channels = ref<ProviderSlotItem[]>([])
const deliveries = ref<NotificationChannelDelivery[]>([])
const selectedCandidates = reactive<Record<string, string>>({})
const pending = ref(false)
const action = ref('')
const errorMessage = ref('')

function channelKey(item: ProviderSlotItem) {
  return item.contract.slot.replace('notification.channel.', '')
}

function selectedRevision(item: ProviderSlotItem) {
  return item.selection?.revision || 0
}

function statusColor(status: NotificationChannelDelivery['status']) {
  if (status === 'sent') return 'success'
  if (status === 'failed') return 'error'
  if (status === 'skipped') return 'warning'
  return 'neutral'
}

function candidateOptions(item: ProviderSlotItem) {
  return item.candidates.map(candidate => ({
    label: candidate.label || candidate.artifact.extensionId,
    value: candidate.id,
    disabled: candidate.availability !== 'available'
  }))
}

async function load() {
  pending.value = true
  errorMessage.value = ''
  try {
    const [catalog, history] = await Promise.all([api.inspect(), api.deliveries()])
    channels.value = catalog.items
    deliveries.value = history.items
    for (const item of channels.value) {
      selectedCandidates[item.contract.slot] = item.selection?.candidateId || item.candidates.find(candidate => candidate.availability === 'available')?.id || ''
    }
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.notificationSettings.channelsAdmin.loadFailed')
  } finally {
    pending.value = false
  }
}

async function selectProvider(item: ProviderSlotItem) {
  const candidateId = selectedCandidates[item.contract.slot]
  if (!props.canManage || !candidateId) return
  action.value = `${item.contract.slot}:select`
  errorMessage.value = ''
  try {
    await api.select(channelKey(item), candidateId, selectedRevision(item))
    await load()
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('admin.notificationSettings.channelsAdmin.selected'), duration: 10000 })
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.notificationSettings.channelsAdmin.selectFailed')
  } finally {
    action.value = ''
  }
}

async function resetProvider(item: ProviderSlotItem) {
  if (!props.canManage || !item.selection) return
  action.value = `${item.contract.slot}:reset`
  errorMessage.value = ''
  try {
    await api.reset(channelKey(item), item.selection.revision)
    await load()
    toast.add({ color: 'success', icon: 'i-lucide-rotate-ccw', title: t('admin.notificationSettings.channelsAdmin.reset'), description: t('admin.notificationSettings.secretsPreserved'), duration: 10000 })
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.notificationSettings.channelsAdmin.resetFailed')
  } finally {
    action.value = ''
  }
}

async function sendTest(item: ProviderSlotItem) {
  if (!props.canManage || item.availability !== 'available') return
  action.value = `${item.contract.slot}:test`
  errorMessage.value = ''
  try {
    await api.test(channelKey(item))
    const history = await api.deliveries()
    deliveries.value = history.items
    toast.add({ color: 'success', icon: 'i-lucide-send', title: t('admin.notificationSettings.channelsAdmin.testQueued'), duration: 10000 })
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('admin.notificationSettings.channelsAdmin.testFailed')
  } finally {
    action.value = ''
  }
}

defineExpose({ refresh: load, pending })
onMounted(load)
</script>

<template>
  <UCard
    class="border-slate-200 bg-white text-slate-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100"
    data-testid="admin-notification-channels"
  >
    <template #header>
      <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 class="text-base font-bold text-slate-900 dark:text-white">{{ t('admin.notificationSettings.channelsAdmin.title') }}</h2>
          <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.notificationSettings.channelsAdmin.description') }}</p>
        </div>
        <UButton color="neutral" variant="outline" size="sm" icon="i-lucide-refresh-cw" :loading="pending" @click="load">{{ t('admin.common.refresh') }}</UButton>
      </div>
    </template>

    <div class="space-y-5">
      <UAlert v-if="errorMessage" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="errorMessage" />
      <div v-if="pending && channels.length === 0" class="space-y-3" aria-busy="true">
        <SFSkeleton class="h-32 w-full" />
      </div>
      <SFEmptyState v-else-if="channels.length === 0" icon-label="EXT" :title="t('admin.notificationSettings.channelsAdmin.emptyTitle')" :description="t('admin.notificationSettings.channelsAdmin.emptyDescription')" />

      <div v-else class="divide-y divide-slate-200 border-y border-slate-200 dark:divide-zinc-800 dark:border-zinc-800">
        <section v-for="item in channels" :key="item.contract.slot" class="py-4">
          <header class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <div class="flex flex-wrap items-center gap-2">
                <h3 class="text-sm font-semibold">{{ t(`admin.notificationSettings.channels.${channelKey(item)}`) }}</h3>
                <UBadge :color="item.availability === 'available' ? 'success' : 'warning'" variant="soft">{{ t(`admin.notificationSettings.channelsAdmin.availability.${item.availability}`) }}</UBadge>
                <UBadge color="neutral" variant="soft">{{ t(`admin.notificationSettings.channelsAdmin.selection.${item.selectionStatus}`) }}</UBadge>
              </div>
              <p class="mt-1 text-xs text-muted">{{ item.contract.contractVersion }} · {{ t('admin.notificationSettings.channelsAdmin.timeout', { value: item.contract.timeoutMs }) }}</p>
            </div>
            <UButton icon="i-lucide-send" size="sm" :disabled="!canManage || item.availability !== 'available'" :loading="action === `${item.contract.slot}:test`" @click="sendTest(item)">{{ t('admin.notificationSettings.channelsAdmin.sendTest') }}</UButton>
          </header>
          <div class="mt-4 grid gap-4 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
            <div>
              <label :for="`channel-${channelKey(item)}`" class="text-sm font-medium">{{ t('admin.notificationSettings.channelsAdmin.provider') }}</label>
              <USelect
                :id="`channel-${channelKey(item)}`"
                v-model="selectedCandidates[item.contract.slot]"
                class="mt-2 w-full"
                :items="candidateOptions(item)"
                value-key="value"
                label-key="label"
                :disabled="!canManage || item.candidates.length === 0"
              />
              <p v-if="item.selection" class="mt-2 break-all text-xs text-muted">{{ item.selection.providerArtifact.extensionId }} @ {{ item.selection.providerArtifact.extensionVersion }} · {{ item.selection.providerArtifact.packageDigest }}</p>
              <p v-else class="mt-2 text-xs text-muted">{{ t('admin.notificationSettings.channelsAdmin.defaultSelection') }}</p>
            </div>
            <div class="flex flex-wrap gap-2">
              <UButton color="neutral" variant="outline" icon="i-lucide-rotate-ccw" :disabled="!canManage || !item.selection" :loading="action === `${item.contract.slot}:reset`" @click="resetProvider(item)">{{ t('admin.notificationSettings.channelsAdmin.restoreDefault') }}</UButton>
              <UButton icon="i-lucide-save" :disabled="!canManage || !selectedCandidates[item.contract.slot] || selectedCandidates[item.contract.slot] === item.selection?.candidateId" :loading="action === `${item.contract.slot}:select`" @click="selectProvider(item)">{{ t('admin.notificationSettings.channelsAdmin.select') }}</UButton>
            </div>
          </div>
        </section>
      </div>

      <section class="border-t border-slate-200 pt-5 dark:border-zinc-800">
        <div>
          <h3 class="text-sm font-semibold">{{ t('admin.notificationSettings.channelsAdmin.deliveryTitle') }}</h3>
          <p class="mt-1 text-sm text-muted">{{ t('admin.notificationSettings.channelsAdmin.deliveryDescription') }}</p>
        </div>
        <p v-if="deliveries.length === 0" class="mt-4 text-sm text-muted">{{ t('admin.notificationSettings.channelsAdmin.deliveryEmpty') }}</p>
        <div v-else class="mt-4 overflow-x-auto">
          <table class="w-full min-w-[44rem] text-left text-sm">
            <thead class="text-xs text-muted"><tr><th class="pb-2 font-medium">{{ t('admin.notificationSettings.channelsAdmin.deliveryId') }}</th><th class="pb-2 font-medium">{{ t('admin.notificationSettings.channelsAdmin.deliveryStatus') }}</th><th class="pb-2 font-medium">{{ t('admin.notificationSettings.channelsAdmin.deliveryProvider') }}</th><th class="pb-2 font-medium">{{ t('admin.notificationSettings.channelsAdmin.deliveryAttempts') }}</th><th class="pb-2 font-medium">{{ t('admin.notificationSettings.channelsAdmin.deliveryResult') }}</th><th class="pb-2 font-medium">{{ t('admin.notificationSettings.channelsAdmin.deliveryUpdated') }}</th></tr></thead>
            <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
              <tr v-for="delivery in deliveries" :key="delivery.id">
                <td class="py-3 font-mono">#{{ delivery.id }}</td>
                <td class="py-3"><UBadge :color="statusColor(delivery.status)" variant="soft">{{ t(`admin.notificationSettings.channelsAdmin.status.${delivery.status}`) }}</UBadge></td>
                <td class="max-w-52 truncate py-3" :title="delivery.providerArtifactDigest">{{ delivery.providerExtensionId || t('admin.notificationSettings.channelsAdmin.noProvider') }}</td>
                <td class="py-3">{{ delivery.attemptCount }}</td>
                <td class="max-w-64 truncate py-3" :title="delivery.errorSummary || delivery.reason">{{ delivery.errorSummary || delivery.reason || '—' }}</td>
                <td class="whitespace-nowrap py-3">{{ new Date(delivery.updatedAt).toLocaleString() }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </UCard>
</template>
