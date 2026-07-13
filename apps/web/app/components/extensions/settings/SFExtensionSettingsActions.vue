<script setup lang="ts">
import type { AdminExtensionSettingsAction, AdminExtensionSettingsActionResult } from '~/utils/adminExtensions'

defineProps<{
  actions: AdminExtensionSettingsAction[]
  loading: Record<string, boolean>
  results: Record<string, AdminExtensionSettingsActionResult | undefined>
}>()

const emit = defineEmits<{ execute: [action: AdminExtensionSettingsAction] }>()
</script>

<template>
  <section v-if="actions.length" class="space-y-3 rounded-lg border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
    <div v-for="action in actions" :key="action.id" class="space-y-2">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="min-w-0">
          <h4 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ action.label }}</h4>
          <p v-if="action.description" class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ action.description }}</p>
        </div>
        <UButton
          icon="i-lucide-activity"
          color="neutral"
          variant="subtle"
          :loading="loading[action.id]"
          :disabled="!action.available || loading[action.id]"
          @click="emit('execute', action)"
        >
          {{ action.label }}
        </UButton>
      </div>
      <p v-if="!action.available && action.unavailableReason" class="text-xs text-amber-700 dark:text-amber-300">
        {{ action.unavailableReason }}
      </p>
      <UAlert
        v-if="results[action.id]"
        :color="results[action.id]?.success ? 'success' : 'error'"
        variant="subtle"
        :icon="results[action.id]?.success ? 'i-lucide-circle-check' : 'i-lucide-triangle-alert'"
        :title="results[action.id]?.message || results[action.id]?.reason"
        :description="results[action.id]?.suggestions?.join(' ')"
      />
      <dl v-if="results[action.id]?.details" class="grid gap-2 text-xs sm:grid-cols-2">
        <div v-for="(value, key) in results[action.id]?.details" :key="key" class="rounded border border-slate-200 px-3 py-2 dark:border-zinc-800">
          <dt class="font-medium text-slate-600 dark:text-zinc-300">{{ key }}</dt>
          <dd class="mt-1 break-words text-slate-500 dark:text-zinc-400">{{ value }}</dd>
        </div>
      </dl>
    </div>
  </section>
</template>
