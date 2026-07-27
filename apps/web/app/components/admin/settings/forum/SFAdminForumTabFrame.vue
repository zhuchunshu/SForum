<script setup lang="ts">
const props = defineProps<{
  tab: string
  dirty: boolean
  validationKey: string | null
  saving: boolean
  restoring: boolean
  canSave: boolean
}>()

const emit = defineEmits<{
  save: []
  reset: []
  restore: []
}>()
const { t } = useI18n()
</script>

<template>
  <form class="flex flex-col" @submit.prevent="emit('save')">
    <UCard class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900" :ui="{ footer: 'sticky bottom-0 z-20 bg-white/95 dark:bg-zinc-900/95 backdrop-blur-sm border-t border-slate-200 dark:border-zinc-800 p-4 sm:px-6' }">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div>
            <h3 class="text-base font-bold">{{ t(`admin.forum.settings.sections.${props.tab}.title`) }}</h3>
            <p class="mt-1 text-xs text-muted">{{ t(`admin.forum.settings.sections.${props.tab}.description`) }}</p>
          </div>
          <UBadge color="neutral" variant="soft" class="font-mono">forum.*</UBadge>
        </div>
      </template>

      <div class="grid max-w-5xl gap-6">
        <slot />
      </div>

      <template #footer>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <UAlert v-if="validationKey" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="t(`admin.forum.settings.validation.${validationKey}`)" class="flex-1" />
          <UAlert v-else-if="dirty" color="warning" variant="soft" icon="i-lucide-pencil" :title="t('admin.forum.settings.unsaved')" class="flex-1" />
          <div class="ml-auto flex flex-wrap gap-2">
            <UButton type="button" color="neutral" variant="ghost" icon="i-lucide-undo-2" :disabled="!dirty" :aria-label="t('admin.forum.settings.discarded')" @click="emit('reset')" />
            <UButton type="button" color="neutral" variant="outline" leading-icon="i-lucide-rotate-ccw" :loading="restoring" :disabled="!canSave" @click="emit('restore')">
              {{ t('admin.forum.settings.restoreRecommended') }}
            </UButton>
            <UButton type="submit" leading-icon="i-lucide-save" :loading="saving" :disabled="!canSave || Boolean(validationKey)">
              {{ t('admin.common.save') }}
            </UButton>
          </div>
        </div>
      </template>
    </UCard>
  </form>
</template>
