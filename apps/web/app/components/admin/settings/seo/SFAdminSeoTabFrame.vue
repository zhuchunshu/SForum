<script setup lang="ts">
import SFAdminFormFooter from '~/components/admin/SFAdminFormFooter.vue'

defineProps<{
  tab: string
  dirty?: boolean
  saving?: boolean
  readonly?: boolean
}>()
const emit = defineEmits<{ save: [], reset: [] }>()
const { t } = useI18n()
</script>

<template>
  <form class="flex flex-col" @submit.prevent="emit('save')">
    <UCard class="border-slate-200 bg-white text-slate-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100" :ui="{ footer: 'sticky bottom-0 z-20 bg-white/95 dark:bg-zinc-900/95 backdrop-blur-sm border-t border-slate-200 dark:border-zinc-800 p-4 sm:px-6' }">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div><h2 class="text-base font-bold">{{ t(`admin.seo.sections.${tab}.title`) }}</h2><p class="mt-1 text-xs text-muted">{{ t(`admin.seo.sections.${tab}.description`) }}</p></div>
          <UBadge color="neutral" variant="soft" class="font-mono">seo.*</UBadge>
        </div>
      </template>
      <slot />
      <template v-if="!readonly" #footer>
        <SFAdminFormFooter :saving="saving" :show-unsaved-alert="dirty" :submit-text="t('admin.seo.save')" @reset="emit('reset')" />
      </template>
    </UCard>
  </form>
</template>
