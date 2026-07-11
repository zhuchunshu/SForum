<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'
import { extensionDefinitionPage, extensionDeliveryPage, extensionDisplayName, extensionEventPage, type AdminExtensionDeliveryStatus, type AdminExtensionEventKind } from '~/utils/adminExtensions'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminExtensionEvents'
})

const { t, locale } = useI18n()
const { format: formatSiteDateTime } = useSiteDateTime()
const adminPage = useAdminPage('/extensions/events')
const definitionPage = ref(1)
const eventPage = ref(1)
const deliveryPage = ref(1)
const {
  extensions,
  pending,
  error,
  refresh,
  eventDefinitions,
  aggregatedDeliveries,
  aggregatedEvents,
  loadingEventDefinitions,
  loadingEventDeliveries,
  loadingAllEvents,
  loadAllEvents,
  loadEventDefinitions,
  loadEventDeliveries
} = await useAdminExtensionsManager()
const definitionPageInfo = computed(() => extensionDefinitionPage(eventDefinitions.value, definitionPage.value))
const eventPageInfo = computed(() => extensionEventPage(aggregatedEvents.value, eventPage.value))
const deliveryPageInfo = computed(() => extensionDeliveryPage(aggregatedDeliveries.value, deliveryPage.value))

useSeoMeta({
  title: t('admin.extensions.eventLog.metaTitle')
})

watch(() => extensions.value.map(item => item.id).join('|'), () => {
  definitionPage.value = 1
  eventPage.value = 1
  deliveryPage.value = 1
  void loadAllEvents()
  void loadEventDeliveries()
}, { immediate: true })

void loadEventDefinitions()

watch(() => definitionPageInfo.value.page, (page) => {
  definitionPage.value = page
})

watch(() => eventPageInfo.value.page, (page) => {
  eventPage.value = page
})

watch(() => deliveryPageInfo.value.page, (page) => {
  deliveryPage.value = page
})

async function refreshEvents() {
  await refresh()
  await loadEventDefinitions()
  await loadEventDeliveries()
  await loadAllEvents()
}

function extensionName(id: string) {
  const item = extensions.value.find(entry => entry.id === id)
  // 直接读 locale.value，保证语言切换时事件日志里的扩展名同步更新。
  return item ? extensionDisplayName(item, locale.value) : id
}

function deliveryStatusColor(status: AdminExtensionDeliveryStatus) {
  if (status === 'succeeded') {
    return 'success'
  }
  if (status === 'failed') {
    return 'error'
  }
  if (status === 'running' || status === 'queued') {
    return 'warning'
  }
  return 'neutral'
}

function eventKindColor(kind: AdminExtensionEventKind) {
  if (kind === 'filter') {
    return 'primary'
  }
  if (kind === 'validate') {
    return 'warning'
  }
  return 'neutral'
}

function fieldList(fields?: string[]) {
  return fields?.length ? fields.join(', ') : '-'
}
</script>

<template>
  <div data-testid="admin-extension-events-page" class="min-w-0 shrink-0">
    <div class="mb-4 flex flex-col gap-1">
      <h2 class="text-xl font-bold flex items-center gap-2 text-slate-900 dark:text-zinc-100">
        <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
        {{ t('admin.extensions.eventLog.title') }}
      </h2>
      <p class="text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.extensions.eventLog.intro') }}
      </p>
    </div>

    <UDashboardToolbar class="border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 rounded-lg px-4 py-2.5 mb-6 text-slate-500 dark:text-zinc-400">
      <template #left>
        <div class="flex min-w-0 items-center gap-2 text-sm">
          <UIcon name="i-lucide-scroll-text" class="size-4" />
          <span class="truncate">{{ t('admin.extensions.eventLog.definitionsCount', { count: eventDefinitions.length }) }}</span>
          <span class="text-slate-300 dark:text-zinc-700">/</span>
          <span class="truncate">{{ t('admin.extensions.eventLog.deliveriesCount', { count: aggregatedDeliveries.length }) }}</span>
          <span class="text-slate-300 dark:text-zinc-700">/</span>
          <span class="truncate">{{ t('admin.extensions.eventLog.count', { count: aggregatedEvents.length }) }}</span>
        </div>
      </template>
      <template #right>
        <UButton icon="i-lucide-rotate-cw" color="neutral" variant="subtle" :loading="pending || loadingAllEvents || loadingEventDefinitions || loadingEventDeliveries" @click="refreshEvents">
          {{ t('admin.extensions.refresh') }}
        </UButton>
      </template>
    </UDashboardToolbar>

    <UAlert
      v-if="error"
      color="error"
      icon="i-lucide-triangle-alert"
      variant="subtle"
      :title="apiErrorMessage(error) || t('admin.extensions.loadFailed')"
      class="mb-6"
    />

    <section class="mb-6 overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <div class="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-zinc-800">
        <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
          {{ t('admin.extensions.eventLog.definitionsTitle') }}
        </h3>
        <UBadge color="neutral" variant="outline">
          {{ t('admin.extensions.eventLog.definitionsCount', { count: eventDefinitions.length }) }}
        </UBadge>
      </div>
      <div v-if="eventDefinitions.length === 0 && !loadingEventDefinitions" class="p-10">
        <SFEmptyState icon-label="EVT" :title="t('admin.extensions.eventLog.emptyDefinitionsTitle')" :description="t('admin.extensions.eventLog.emptyDefinitionsDescription')" />
      </div>
      <div v-else class="divide-y divide-slate-200 dark:divide-zinc-800">
        <div
          v-for="definition in definitionPageInfo.items"
          :key="definition.name"
          class="grid gap-3 px-4 py-4 md:grid-cols-[minmax(0,1fr)_140px]"
        >
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <UIcon name="i-lucide-radio" class="size-4 text-[var(--sf-accent)]" />
              <h3 class="min-w-0 break-all text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ definition.name }}
              </h3>
              <UBadge :color="eventKindColor(definition.kind)" variant="subtle">
                {{ t(`admin.extensions.eventLog.kind.${definition.kind}`) }}
              </UBadge>
            </div>
            <p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">
              {{ definition.description }}
            </p>
            <p class="mt-1 break-all text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.extensions.eventLog.payloadFields', { fields: fieldList(definition.payloadFields) }) }}
            </p>
            <p class="mt-1 break-all text-xs text-slate-500 dark:text-zinc-400">
              {{ definition.patchFields?.length ? t('admin.extensions.eventLog.patchFields', { fields: fieldList(definition.patchFields) }) : t('admin.extensions.eventLog.noPatchFields') }}
            </p>
          </div>
          <div class="flex items-center text-xs text-slate-500 md:justify-end dark:text-zinc-400">
            {{ definition.timeoutMs }}ms
          </div>
        </div>
        <div
          v-if="definitionPageInfo.totalPages > 1"
          class="flex flex-col gap-3 px-4 py-4 text-xs text-slate-500 sm:flex-row sm:items-center sm:justify-between dark:text-zinc-400"
        >
          <span>
            {{ t('admin.extensions.eventPageSummary', { start: definitionPageInfo.start, end: definitionPageInfo.end, count: definitionPageInfo.total }) }}
          </span>
          <SFPagination v-model:page="definitionPage" :total-pages="definitionPageInfo.totalPages" />
        </div>
      </div>
    </section>

    <section class="mb-6 overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <div class="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-zinc-800">
        <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
          {{ t('admin.extensions.eventLog.deliveriesTitle') }}
        </h3>
        <UBadge color="neutral" variant="outline">
          {{ t('admin.extensions.eventLog.deliveriesCount', { count: aggregatedDeliveries.length }) }}
        </UBadge>
      </div>
      <div v-if="aggregatedDeliveries.length === 0 && !loadingEventDeliveries" class="p-10">
        <SFEmptyState icon-label="RUN" :title="t('admin.extensions.eventLog.emptyDeliveriesTitle')" :description="t('admin.extensions.eventLog.emptyDeliveriesDescription')" />
      </div>
      <div v-else class="divide-y divide-slate-200 dark:divide-zinc-800">
        <div
          v-for="delivery in deliveryPageInfo.items"
          :key="delivery.id"
          class="grid gap-3 px-4 py-4 md:grid-cols-[minmax(0,1fr)_220px]"
        >
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <UIcon name="i-lucide-send" class="size-4 text-[var(--sf-accent)]" />
              <h3 class="min-w-0 break-all text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ delivery.eventName }}
              </h3>
              <UBadge :color="eventKindColor(delivery.eventKind)" variant="subtle">
                {{ t(`admin.extensions.eventLog.kind.${delivery.eventKind}`) }}
              </UBadge>
              <UBadge :color="deliveryStatusColor(delivery.status)" variant="subtle">
                {{ t(`admin.extensions.eventLog.deliveryStatus.${delivery.status}`) }}
              </UBadge>
            </div>
            <p class="mt-1 break-all text-xs text-slate-500 dark:text-zinc-400">
              {{ extensionName(delivery.extensionId) }} · {{ delivery.correlationId }}
            </p>
            <p v-if="delivery.reason || delivery.message" class="mt-1 break-all text-xs text-red-600 dark:text-red-400">
              {{ delivery.reason || delivery.message }}
            </p>
          </div>
          <div class="flex items-center text-xs text-slate-500 md:justify-end dark:text-zinc-400">
            {{ formatSiteDateTime(delivery.createdAt) }}
          </div>
        </div>
        <div
          v-if="deliveryPageInfo.totalPages > 1"
          class="flex flex-col gap-3 px-4 py-4 text-xs text-slate-500 sm:flex-row sm:items-center sm:justify-between dark:text-zinc-400"
        >
          <span>
            {{ t('admin.extensions.eventPageSummary', { start: deliveryPageInfo.start, end: deliveryPageInfo.end, count: deliveryPageInfo.total }) }}
          </span>
          <SFPagination v-model:page="deliveryPage" :total-pages="deliveryPageInfo.totalPages" />
        </div>
      </div>
    </section>

    <section class="overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <div class="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-zinc-800">
        <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
          {{ t('admin.extensions.eventLog.auditTitle') }}
        </h3>
        <UBadge color="neutral" variant="outline">
          {{ t('admin.extensions.eventLog.count', { count: aggregatedEvents.length }) }}
        </UBadge>
      </div>
      <div v-if="aggregatedEvents.length === 0 && !pending && !loadingAllEvents" class="p-10">
        <SFEmptyState icon-label="LOG" :title="t('admin.extensions.eventLog.emptyTitle')" :description="t('admin.extensions.eventLog.emptyDescription')" />
      </div>
      <div v-else class="divide-y divide-slate-200 dark:divide-zinc-800">
        <div
          v-for="event in eventPageInfo.items"
          :key="`${event.extensionId}:${event.id}`"
          class="grid gap-3 px-4 py-4 md:grid-cols-[minmax(0,1fr)_220px]"
        >
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <UIcon name="i-lucide-activity" class="size-4 text-[var(--sf-accent)]" />
              <h3 class="min-w-0 break-all text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ event.action }}
              </h3>
              <UBadge color="neutral" variant="outline">
                {{ extensionName(event.extensionId) }}
              </UBadge>
            </div>
            <p v-if="event.message" class="mt-1 text-sm text-slate-500 dark:text-zinc-400">
              {{ event.message }}
            </p>
            <p class="mt-1 break-all text-xs text-slate-500 dark:text-zinc-400">
              {{ event.extensionId }}
            </p>
          </div>
          <div class="flex items-center text-xs text-slate-500 md:justify-end dark:text-zinc-400">
            {{ formatSiteDateTime(event.createdAt) }}
          </div>
        </div>
        <div
          v-if="eventPageInfo.totalPages > 1"
          class="flex flex-col gap-3 px-4 py-4 text-xs text-slate-500 sm:flex-row sm:items-center sm:justify-between dark:text-zinc-400"
        >
          <span>
            {{ t('admin.extensions.eventPageSummary', { start: eventPageInfo.start, end: eventPageInfo.end, count: eventPageInfo.total }) }}
          </span>
          <SFPagination v-model:page="eventPage" :total-pages="eventPageInfo.totalPages" />
        </div>
      </div>
    </section>
  </div>
</template>
