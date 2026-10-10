<script setup lang="ts">
import type { AdminAIProfile } from '~/composables/admin/useAdminAI'

export type AIProviderOption = {
  id: string
  label: string
  hint: string
}

const { t } = useI18n()

defineProps<{
  options: AIProviderOption[]
  selectedId: string
  credentialStatus: Record<string, boolean>
  disabled: boolean
}>()

const emit = defineEmits<{ select: [id: string] }>()
</script>

<template>
  <section class="min-w-0 rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
    <div class="border-b border-slate-200 px-4 py-3 dark:border-zinc-800">
      <p class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.ai.providerTitle') }}</p>
      <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.ai.providerIntro') }}</p>
    </div>
    <div class="divide-y divide-slate-200 dark:divide-zinc-800">
      <label
        v-for="option in options"
        :key="option.id || 'none'"
        class="flex cursor-pointer items-start gap-3 px-4 py-3"
      >
        <input
          type="radio"
          class="mt-1 size-4 accent-[var(--sf-accent)]"
          :value="option.id"
          :checked="selectedId === option.id"
          :disabled="disabled"
          @change="emit('select', option.id)"
        >
        <span class="min-w-0 flex-1">
          <span class="flex flex-wrap items-center gap-2">
            <span class="text-sm font-medium text-slate-900 dark:text-zinc-100">{{ option.label }}</span>
            <UBadge v-if="option.id && credentialStatus[option.id]" color="success" variant="soft" size="sm">
              {{ t('admin.ai.credentialConfigured') }}
            </UBadge>
          </span>
          <span class="mt-0.5 block text-xs text-slate-500 dark:text-zinc-400">{{ option.hint }}</span>
        </span>
      </label>
    </div>
  </section>
</template>
