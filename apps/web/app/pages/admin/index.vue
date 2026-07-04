<script setup lang="ts">
import { useAdminRoutes } from '~/composables/useAdminRoutes'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

const { t } = useI18n()
const adminRoutes = useAdminRoutes()

const overviewCards = computed(() => [
  {
    label: t('admin.home.cards.access.label'),
    value: t('admin.home.cards.access.value'),
    icon: 'i-lucide-shield-check',
    tone: 'text-primary'
  },
  {
    label: t('admin.home.cards.prefix.label'),
    value: adminRoutes.prefix,
    icon: 'i-lucide-route',
    tone: 'text-info'
  },
  {
    label: t('admin.home.cards.stack.label'),
    value: t('admin.home.cards.stack.value'),
    icon: 'i-lucide-box',
    tone: 'text-success'
  }
])

const nextSections = computed(() => [
  {
    title: t('admin.home.next.roles.title'),
    description: t('admin.home.next.roles.description'),
    icon: 'i-lucide-users-round',
    to: adminRoutes.path('/roles')
  },
  {
    title: t('admin.home.next.audit.title'),
    description: t('admin.home.next.audit.description'),
    icon: 'i-lucide-scroll-text'
  },
  {
    title: t('admin.home.next.settings.title'),
    description: t('admin.home.next.settings.description'),
    icon: 'i-lucide-settings-2',
    to: adminRoutes.path('/settings')
  }
])

useSeoMeta({
  title: t('admin.home.metaTitle')
})
</script>

<template>
  <UDashboardNavbar :title="t('admin.home.title')" icon="i-lucide-layout-dashboard">
    <template #right>
      <UButton
        :to="adminRoutes.path('/roles')"
        color="neutral"
        variant="outline"
        leading-icon="i-lucide-shield-check"
      >
        {{ t('admin.home.rolesLink') }}
      </UButton>
    </template>
  </UDashboardNavbar>

  <UDashboardToolbar>
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm text-muted">
        <UIcon name="i-lucide-lock-keyhole" class="size-4" />
        <span class="truncate">{{ t('admin.home.intro') }}</span>
      </div>
    </template>
    <template #right>
      <UBadge color="neutral" variant="soft">
        {{ adminRoutes.prefix }}
      </UBadge>
    </template>
  </UDashboardToolbar>

  <div class="flex flex-1 flex-col gap-6 p-4 sm:p-6">
    <div class="grid gap-4 lg:grid-cols-3">
      <UCard v-for="card in overviewCards" :key="card.label" variant="subtle">
        <div class="flex items-start justify-between gap-4">
          <div class="min-w-0">
            <p class="text-sm text-muted">
              {{ card.label }}
            </p>
            <p class="mt-2 truncate text-xl font-semibold text-highlighted">
              {{ card.value }}
            </p>
          </div>
          <span class="grid size-10 shrink-0 place-items-center rounded-md bg-elevated">
            <UIcon :name="card.icon" class="size-5" :class="card.tone" />
          </span>
        </div>
      </UCard>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div>
            <h2 class="text-base font-semibold text-highlighted">
              {{ t('admin.home.nextTitle') }}
            </h2>
            <p class="mt-1 text-sm text-muted">
              {{ t('admin.home.nextIntro') }}
            </p>
          </div>
          <UIcon name="i-lucide-list-checks" class="size-5 text-muted" />
        </div>
      </template>

      <div class="divide-y divide-default">
        <component
          :is="section.to ? 'NuxtLink' : 'div'"
          v-for="section in nextSections"
          :key="section.title"
          :to="section.to"
          class="flex items-center gap-4 py-4 first:pt-0 last:pb-0"
        >
          <span class="grid size-10 shrink-0 place-items-center rounded-md bg-muted">
            <UIcon :name="section.icon" class="size-5 text-muted" />
          </span>
          <span class="min-w-0 flex-1">
            <span class="block font-medium text-highlighted">
              {{ section.title }}
            </span>
            <span class="mt-1 block text-sm text-muted">
              {{ section.description }}
            </span>
          </span>
          <UIcon
            v-if="section.to"
            name="i-lucide-arrow-right"
            class="size-4 shrink-0 text-muted"
          />
        </component>
      </div>
    </UCard>
  </div>
</template>
