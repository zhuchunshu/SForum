<script setup lang="ts">
const { t } = useI18n()
const pending = ref(false)

defineExpose({ pending })
</script>

<template>
  <section class="border border-slate-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900 sm:p-6">
    <div class="flex items-start gap-3">
      <div class="grid size-10 shrink-0 place-items-center rounded-md bg-slate-100 text-slate-600 dark:bg-zinc-800 dark:text-zinc-300">
        <UIcon name="i-lucide-file-archive" class="size-5" />
      </div>
      <div class="min-w-0">
        <h3 class="text-base font-semibold text-slate-900 dark:text-zinc-100">
          {{ t('admin.attachments.tabs.compression') }}
        </h3>
        <p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">
          {{ t('admin.attachments.compression.description') }}
        </p>
      </div>
    </div>

    <UAlert
      class="mt-5"
      color="neutral"
      variant="soft"
      icon="i-lucide-info"
      :title="t('admin.attachments.compression.emptyTitle')"
      :description="t('admin.attachments.compression.emptyDescription')"
    />
  </section>
</template>
