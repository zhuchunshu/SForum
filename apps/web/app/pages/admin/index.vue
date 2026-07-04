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
  <!-- 局部标题 -->
  <div class="mb-4">
    <h2 class="text-xl font-bold flex items-center gap-2 text-slate-900 dark:text-zinc-100">
      <UIcon name="i-lucide-layout-dashboard" class="size-5 text-teal-600 dark:text-teal-400" />
      {{ t('admin.home.title') }}
    </h2>
  </div>

  <!-- 整合原 navbar 按钮的统一 Toolbar -->
  <UDashboardToolbar class="border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 rounded-lg px-4 py-2.5 mb-6 text-slate-500 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm text-slate-500 dark:text-zinc-400">
        <UIcon name="i-lucide-lock-keyhole" class="size-4" />
        <span class="truncate">{{ t('admin.home.intro') }}</span>
      </div>
    </template>
    <template #right>
      <div class="flex items-center gap-3">
        <UButton
          :to="adminRoutes.path('/roles')"
          color="neutral"
          variant="outline"
          leading-icon="i-lucide-shield-check"
          class="border-slate-200 dark:border-zinc-700"
        >
          {{ t('admin.home.rolesLink') }}
        </UButton>
        <UBadge color="neutral" variant="soft" class="border border-slate-200 dark:border-zinc-800 font-mono">
          {{ adminRoutes.prefix }}
        </UBadge>
      </div>
    </template>
  </UDashboardToolbar>

  <div class="flex flex-col gap-6">
    <div class="grid gap-4 lg:grid-cols-3">
      <UCard v-for="card in overviewCards" :key="card.label" class="border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100">
        <div class="flex items-start justify-between gap-4">
          <div class="min-w-0">
            <p class="text-xs font-medium text-slate-500 dark:text-zinc-400">
              {{ card.label }}
            </p>
            <p class="mt-2 truncate text-xl font-bold text-slate-900 dark:text-white">
              {{ card.value }}
            </p>
          </div>
          <span class="grid size-10 shrink-0 place-items-center rounded-md bg-slate-50 dark:bg-zinc-800 border border-slate-200 dark:border-zinc-700">
            <UIcon :name="card.icon" class="size-5" :class="card.tone" />
          </span>
        </div>
      </UCard>
    </div>

    <UCard class="border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div>
            <h2 class="text-base font-bold text-slate-900 dark:text-white">
              {{ t('admin.home.nextTitle') }}
            </h2>
            <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.home.nextIntro') }}
            </p>
          </div>
          <UIcon name="i-lucide-list-checks" class="size-5 text-slate-400 dark:text-zinc-500" />
        </div>
      </template>

      <div class="divide-y divide-slate-100 dark:divide-zinc-800">
        <component
          :is="section.to ? 'NuxtLink' : 'div'"
          v-for="section in nextSections"
          :key="section.title"
          :to="section.to"
          class="flex items-center gap-4 py-4 first:pt-0 last:pb-0"
        >
          <span class="grid size-10 shrink-0 place-items-center rounded-md bg-slate-50 dark:bg-zinc-800 border border-slate-200 dark:border-zinc-700">
            <UIcon :name="section.icon" class="size-5 text-slate-500 dark:text-zinc-400" />
          </span>
          <span class="min-w-0 flex-1">
            <span class="block font-semibold text-slate-900 dark:text-white text-sm">
              {{ section.title }}
            </span>
            <span class="mt-1 block text-xs text-slate-500 dark:text-zinc-400">
              {{ section.description }}
            </span>
          </span>
          <UIcon
            v-if="section.to"
            name="i-lucide-arrow-right"
            class="size-4 shrink-0 text-slate-400 dark:text-zinc-500"
          />
        </component>
      </div>
    </UCard>
  </div>
</template>
