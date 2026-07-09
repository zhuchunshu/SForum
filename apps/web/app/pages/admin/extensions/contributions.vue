<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'
import {
  extensionContributionLabel,
  extensionContributionPage,
  extensionContributionPayloadSummary,
  type AdminContributionPointDefinition
} from '~/utils/adminExtensions'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminExtensionContributions'
})

const { t, locale } = useI18n()
const adminPage = useAdminPage('/extensions/contributions')
const contributionPage = ref(1)
const {
  pending,
  error,
  refresh,
  contributionPoints,
  contributions,
  loadingContributionPoints,
  loadingContributions,
  loadContributionPoints,
  loadContributions
} = await useAdminExtensionsManager()

const contributionPageInfo = computed(() => extensionContributionPage(contributions.value, contributionPage.value))

useSeoMeta({
  title: t('admin.extensions.contributions.metaTitle')
})

watch(() => contributionPageInfo.value.page, (page) => {
  contributionPage.value = page
})

void loadContributionPoints()
void loadContributions()

async function refreshContributions() {
  await refresh()
  await loadContributionPoints()
  await loadContributions()
}

function pointDefinition(point: string): AdminContributionPointDefinition | undefined {
  return contributionPoints.value.find(item => item.id === point)
}
</script>

<template>
  <div data-testid="admin-extension-contributions-page" class="min-w-0 shrink-0">
    <div class="mb-4 flex flex-col gap-1">
      <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
        <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
        {{ t('admin.extensions.contributions.title') }}
      </h2>
      <p class="text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.extensions.contributions.intro') }}
      </p>
    </div>

    <UDashboardToolbar class="mb-6 rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-500 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400">
      <template #left>
        <div class="flex min-w-0 items-center gap-2 text-sm">
          <UIcon name="i-lucide-blocks" class="size-4" />
          <span class="truncate">{{ t('admin.extensions.contributions.pointsCount', { count: contributionPoints.length }) }}</span>
          <span class="text-slate-300 dark:text-zinc-700">/</span>
          <span class="truncate">{{ t('admin.extensions.contributions.activeCount', { count: contributions.length }) }}</span>
        </div>
      </template>
      <template #right>
        <UButton icon="i-lucide-rotate-cw" color="neutral" variant="subtle" :loading="pending || loadingContributionPoints || loadingContributions" @click="refreshContributions">
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
          {{ t('admin.extensions.contributions.pointsTitle') }}
        </h3>
        <UBadge color="neutral" variant="outline">
          {{ t('admin.extensions.contributions.pointsCount', { count: contributionPoints.length }) }}
        </UBadge>
      </div>
      <div v-if="contributionPoints.length === 0 && !loadingContributionPoints" class="p-10">
        <SFEmptyState icon-label="EXT" :title="t('admin.extensions.contributions.emptyPointsTitle')" :description="t('admin.extensions.contributions.emptyPointsDescription')" />
      </div>
      <div v-else class="divide-y divide-slate-200 dark:divide-zinc-800">
        <div
          v-for="point in contributionPoints"
          :key="point.id"
          class="grid gap-3 px-4 py-4 md:grid-cols-[minmax(0,1fr)_180px]"
        >
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <UIcon name="i-lucide-radio" class="size-4 text-[var(--sf-accent)]" />
              <h3 class="min-w-0 break-all text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ point.id }}
              </h3>
              <UBadge color="neutral" variant="subtle">
                {{ point.kind }}
              </UBadge>
            </div>
            <p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">
              {{ point.description }}
            </p>
          </div>
          <div class="flex flex-col gap-1 text-xs text-slate-500 md:items-end dark:text-zinc-400">
            <span>{{ point.owner }}</span>
            <span>{{ point.payloadType }}</span>
          </div>
        </div>
      </div>
    </section>

    <section class="overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <div class="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-zinc-800">
        <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
          {{ t('admin.extensions.contributions.activeTitle') }}
        </h3>
        <UBadge color="neutral" variant="outline">
          {{ t('admin.extensions.contributions.activeCount', { count: contributions.length }) }}
        </UBadge>
      </div>
      <div v-if="contributions.length === 0 && !loadingContributions" class="p-10">
        <SFEmptyState icon-label="ACT" :title="t('admin.extensions.contributions.emptyActiveTitle')" :description="t('admin.extensions.contributions.emptyActiveDescription')" />
      </div>
      <div v-else class="divide-y divide-slate-200 dark:divide-zinc-800">
        <div
          v-for="contribution in contributionPageInfo.items"
          :key="`${contribution.extensionId}:${contribution.point}:${contribution.id}`"
          class="grid gap-3 px-4 py-4 md:grid-cols-[minmax(0,1fr)_220px]"
        >
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <UIcon :name="contribution.icon || 'i-lucide-plug'" class="size-4 text-[var(--sf-accent)]" />
              <h3 class="min-w-0 break-all text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ extensionContributionLabel(contribution, locale) }}
              </h3>
              <UBadge color="neutral" variant="subtle">
                {{ contribution.point }}
              </UBadge>
            </div>
            <p class="mt-1 break-all text-xs text-slate-500 dark:text-zinc-400">
              {{ contribution.extensionName }} · {{ contribution.id }}
            </p>
            <p class="mt-1 break-all text-xs text-slate-500 dark:text-zinc-400">
              {{ pointDefinition(contribution.point)?.description || contribution.point }}
            </p>
          </div>
          <div class="flex flex-col gap-1 text-xs text-slate-500 md:items-end dark:text-zinc-400">
            <span>{{ t('admin.extensions.contributions.order', { order: contribution.order }) }}</span>
            <span class="break-all text-right">{{ extensionContributionPayloadSummary(contribution) }}</span>
          </div>
        </div>
        <div
          v-if="contributionPageInfo.totalPages > 1"
          class="flex flex-col gap-3 px-4 py-4 text-xs text-slate-500 sm:flex-row sm:items-center sm:justify-between dark:text-zinc-400"
        >
          <span>
            {{ t('admin.extensions.eventPageSummary', { start: contributionPageInfo.start, end: contributionPageInfo.end, count: contributionPageInfo.total }) }}
          </span>
          <SFPagination v-model:page="contributionPage" :total-pages="contributionPageInfo.totalPages" />
        </div>
      </div>
    </section>
  </div>
</template>
