<script setup lang="ts">
import type { AdminForumTaxonomyDeleteTarget } from '~/utils/admin/adminForum'

const props = defineProps<{
  target: AdminForumTaxonomyDeleteTarget | null
  busy?: boolean
}>()

const emit = defineEmits<{
  cancel: []
  confirm: []
}>()

const open = defineModel<boolean>('open', { required: true })
const { t } = useI18n()

const kindKey = computed(() => props.target?.kind === 'category' ? 'category' : 'group')
const blocked = computed(() => Boolean(props.target?.blockedReason))
</script>

<template>
  <UModal
    v-model:open="open"
    :ui="{ content: 'sm:max-w-lg' }"
    @update:open="(value) => { if (!value) emit('cancel') }"
  >
    <template #content>
      <div class="p-5 sm:p-6">
        <div class="flex items-start gap-3">
          <div class="flex size-9 shrink-0 items-center justify-center rounded-md bg-red-50 text-red-600 dark:bg-red-950/40 dark:text-red-300">
            <UIcon name="i-lucide-trash-2" class="size-4" aria-hidden="true" />
          </div>
          <div class="min-w-0">
            <h3 class="text-base font-bold text-slate-900 dark:text-white">
              {{ t(`admin.forum.categories.delete.${kindKey}Title`) }}
            </h3>
            <p class="mt-1 break-words text-sm leading-6 text-slate-600 dark:text-zinc-300">
              {{ t(`admin.forum.categories.delete.${kindKey}Body`, { name: target?.name || '' }) }}
            </p>
          </div>
        </div>

        <UAlert
          v-if="blocked"
          class="mt-4"
          color="warning"
          variant="soft"
          icon="i-lucide-triangle-alert"
          :title="t(`admin.forum.categories.delete.${kindKey}BlockedTitle`)"
          :description="target?.blockedReason"
        />

        <div class="mt-5 flex flex-wrap justify-end gap-2 border-t border-slate-200 pt-4 dark:border-zinc-800">
          <UButton type="button" color="neutral" variant="ghost" @click="emit('cancel')">
            {{ t('admin.common.cancel') }}
          </UButton>
          <UButton
            type="button"
            color="error"
            leading-icon="i-lucide-trash-2"
            :loading="busy"
            :disabled="blocked"
            @click="emit('confirm')"
          >
            {{ t('admin.forum.categories.delete.confirm') }}
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
