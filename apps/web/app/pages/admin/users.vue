<script setup lang="ts">
import { useAdminTabs } from '~/composables/useAdminTabs'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminUsers'
})

const { t } = useI18n()
const adminTabs = useAdminTabs()

onMounted(() => {
  adminTabs.openTab('/users', 'admin.nav.userManagement', 'i-lucide-contact', 'AdminUsers')
})

useSeoMeta({
  title: t('admin.nav.userManagement')
})
</script>

<template>
  <div class="mb-4">
    <h2 class="text-xl font-bold flex items-center gap-2 text-slate-900 dark:text-zinc-100">
      <UIcon name="i-lucide-contact" class="size-5 text-teal-600 dark:text-teal-400" />
      {{ t('admin.nav.userManagement') }}
    </h2>
  </div>

  <UCard class="border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100">
    <div class="flex flex-col items-center justify-center py-16 text-slate-400 dark:text-zinc-500">
      <UIcon name="i-lucide-contact" class="size-12 mb-4" />
      <p class="text-sm">{{ t('admin.nav.userManagement') }} - 即将上线</p>
    </div>
  </UCard>
</template>
