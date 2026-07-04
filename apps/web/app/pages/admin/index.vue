<script setup lang="ts">
import { useAdminRoutes } from '~/composables/useAdminRoutes'
import { useAdminTabs } from '~/composables/useAdminTabs'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

// 明确声明组件名用于 KeepAlive 匹配
defineOptions({
  name: 'AdminIndex'
})

const { t } = useI18n()
const adminRoutes = useAdminRoutes()
const adminTabs = useAdminTabs()

// 挂载当前页签
onMounted(() => {
  adminTabs.openTab('/', 'admin.nav.dashboard', 'i-lucide-layout-dashboard', 'AdminIndex')
})

const overviewCards = computed(() => [
  {
    label: t('admin.home.cards.access.label'),
    value: t('admin.home.cards.access.value'),
    icon: 'i-lucide-shield-check',
    tone: 'text-teal-600 dark:text-teal-400'
  },
  {
    label: t('admin.home.cards.prefix.label'),
    value: adminRoutes.prefix,
    icon: 'i-lucide-route',
    tone: 'text-blue-600 dark:text-blue-400'
  },
  {
    label: t('admin.home.cards.stack.label'),
    value: t('admin.home.cards.stack.value'),
    icon: 'i-lucide-box',
    tone: 'text-green-600 dark:text-green-400'
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
  <UDashboardNavbar :title="t('admin.home.title')" icon="i-lucide-layout-dashboard" class="border-b border-[var(--border-admin)] bg-[var(--bg-admin-card)] text-[var(--text-admin-main)]">
    <template #right>
      <UButton
        :to="adminRoutes.path('/roles')"
        color="neutral"
        variant="outline"
        leading-icon="i-lucide-shield-check"
        class="border-[var(--border-admin)]"
      >
        {{ t('admin.home.rolesLink') }}
      </UButton>
    </template>
  </UDashboardNavbar>

  <UDashboardToolbar class="border-b border-[var(--border-admin)] bg-[var(--bg-admin-app)] text-[var(--text-admin-muted)]">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm text-[var(--text-admin-muted)]">
        <UIcon name="i-lucide-lock-keyhole" class="size-4" />
        <span class="truncate">{{ t('admin.home.intro') }}</span>
      </div>
    </template>
    <template #right>
      <UBadge color="neutral" variant="soft" class="border border-[var(--border-admin)]">
        {{ adminRoutes.prefix }}
      </UBadge>
    </template>
  </UDashboardToolbar>

  <div class="flex flex-1 flex-col gap-6 p-4 sm:p-6 bg-[var(--bg-admin-app)]">
    <div class="grid gap-4 lg:grid-cols-3">
      <UCard v-for="card in overviewCards" :key="card.label" class="border-[var(--border-admin)] bg-[var(--bg-admin-card)] text-[var(--text-admin-main)]">
        <div class="flex items-start justify-between gap-4">
          <div class="min-w-0">
            <p class="text-sm text-[var(--text-admin-muted)]">
              {{ card.label }}
            </p>
            <p class="mt-2 truncate text-xl font-semibold text-[var(--text-admin-main)]">
              {{ card.value }}
            </p>
          </div>
          <span class="grid size-10 shrink-0 place-items-center rounded-md bg-[var(--bg-admin-app)] border border-[var(--border-admin)]">
            <UIcon :name="card.icon" class="size-5" :class="card.tone" />
          </span>
        </div>
      </UCard>
    </div>

    <UCard class="border-[var(--border-admin)] bg-[var(--bg-admin-card)] text-[var(--text-admin-main)]">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div>
            <h2 class="text-base font-semibold text-[var(--text-admin-main)]">
              {{ t('admin.home.nextTitle') }}
            </h2>
            <p class="mt-1 text-sm text-[var(--text-admin-muted)]">
              {{ t('admin.home.nextIntro') }}
            </p>
          </div>
          <UIcon name="i-lucide-list-checks" class="size-5 text-[var(--text-admin-muted)]" />
        </div>
      </template>

      <div class="divide-y divide-[var(--border-admin)]">
        <component
          :is="section.to ? 'NuxtLink' : 'div'"
          v-for="section in nextSections"
          :key="section.title"
          :to="section.to"
          class="flex items-center gap-4 py-4 first:pt-0 last:pb-0"
        >
          <span class="grid size-10 shrink-0 place-items-center rounded-md bg-[var(--bg-admin-app)] border border-[var(--border-admin)]">
            <UIcon :name="section.icon" class="size-5 text-[var(--text-admin-muted)]" />
          </span>
          <span class="min-w-0 flex-1">
            <span class="block font-medium text-[var(--text-admin-main)]">
              {{ section.title }}
            </span>
            <span class="mt-1 block text-sm text-[var(--text-admin-muted)]">
              {{ section.description }}
            </span>
          </span>
          <UIcon
            v-if="section.to"
            name="i-lucide-arrow-right"
            class="size-4 shrink-0 text-[var(--text-admin-muted)]"
          />
        </component>
      </div>
    </UCard>
  </div>
</template>
