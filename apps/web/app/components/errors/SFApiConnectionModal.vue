<script setup lang="ts">
const { t } = useI18n()
const { state, close, reload } = useApiConnectionError()

const open = computed({
  get: () => state.value.open,
  set: (value: boolean) => {
    if (!value) {
      close()
    }
  }
})

const statusText = computed(() => state.value.statusCode
  ? t('errors.apiConnection.status', { statusCode: state.value.statusCode })
  : ''
)
</script>

<template>
  <UModal v-model:open="open">
    <template #content>
      <div class="space-y-5 p-6" role="alert" aria-live="assertive">
        <div class="flex items-start gap-3">
          <div class="rounded-full bg-red-50 p-2 text-red-600 dark:bg-red-950/40 dark:text-red-300">
            <UIcon name="i-lucide-wifi-off" class="size-5" />
          </div>
          <div class="min-w-0 flex-1">
            <h2 class="text-base font-semibold text-slate-900 dark:text-slate-50">
              {{ t('errors.apiConnection.title') }}
            </h2>
            <p class="mt-1 text-sm leading-6 text-slate-600 dark:text-slate-300">
              {{ t('errors.apiConnection.description') }}
            </p>
            <p v-if="statusText" class="mt-2 text-xs font-medium text-slate-500 dark:text-slate-400">
              {{ statusText }}
            </p>
          </div>
        </div>

        <div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <SFButton variant="ghost" @click="close">
            {{ t('errors.apiConnection.dismiss') }}
          </SFButton>
          <SFButton variant="primary" @click="reload">
            <template #leading>
              <UIcon name="i-lucide-refresh-cw" class="size-4" />
            </template>
            {{ t('errors.apiConnection.retry') }}
          </SFButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
