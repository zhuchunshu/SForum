<script setup lang="ts">
import type { AppliedRolePermission } from '~/components/admin/identity/roles/model'
import { useRoleSuggestions } from '~/composables/identity/useRoleSuggestions'

const emit = defineEmits<{
  permissionApplied: [permission: Omit<AppliedRolePermission, 'sequence'>]
}>()

const { t } = useI18n()
const {
  selectedFilter,
  suggestions,
  nextCursor,
  loading,
  loadingMore,
  loadError,
  decisionOpen,
  pendingDecision,
  deciding,
  decisionError,
  filterItems,
  confirmationTitle,
  confirmationDescription,
  confirmationAction,
  loadSuggestions,
  openDecision,
  closeDecision,
  submitDecision,
  statusColor,
  statusLabel,
  shortDigest,
  formatDate
} = useRoleSuggestions((suggestion) => {
  emit('permissionApplied', {
    roleKey: suggestion.roleKey,
    permissionKey: suggestion.permissionKey
  })
})

const pending = computed(() => loading.value || loadingMore.value || deciding.value)

function setFilter(value: unknown) {
  if (value === 'pending' || value === 'approved' || value === 'rejected' || value === 'all') {
    selectedFilter.value = value
  }
}

defineExpose({
  refresh: () => loadSuggestions(true),
  pending,
  filterItems,
  selectedFilter,
  setFilter
})
</script>

<template>
  <section class="min-w-0">
    <div class="mb-4 min-w-0">
      <div class="flex items-center gap-2">
        <UIcon name="i-lucide-shield-check" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
        <h3 class="text-base font-semibold text-slate-950 dark:text-zinc-50">
          {{ t('admin.roles.suggestions.title') }}
        </h3>
        <UBadge color="neutral" variant="soft">{{ suggestions.length }}</UBadge>
      </div>
      <p class="mt-1 max-w-3xl text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.roles.suggestions.intro') }}
      </p>
    </div>

    <UAlert
      v-if="loadError"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="loadError"
      class="mb-4"
    />

    <div
      v-if="loading"
      class="flex min-h-28 items-center justify-center border border-slate-200 bg-slate-50 text-sm text-slate-500 dark:border-zinc-800 dark:bg-zinc-950/40 dark:text-zinc-400"
    >
      <UIcon name="i-lucide-loader-circle" class="mr-2 size-4 animate-spin" />
      {{ t('admin.roles.suggestions.loading') }}
    </div>
    <div
      v-else-if="!suggestions.length"
      class="flex min-h-28 flex-col items-center justify-center border border-dashed border-slate-300 px-4 text-center dark:border-zinc-700"
    >
      <UIcon name="i-lucide-inbox" class="mb-2 size-5 text-slate-400 dark:text-zinc-500" />
      <p class="text-sm text-slate-500 dark:text-zinc-400">{{ t('admin.roles.suggestions.empty') }}</p>
    </div>
    <div v-else class="border border-slate-200 dark:border-zinc-800">
      <article
        v-for="suggestion in suggestions"
        :key="suggestion.id"
        class="grid gap-3 border-b border-slate-200 p-4 last:border-b-0 dark:border-zinc-800 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center"
      >
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <code class="break-all text-sm font-semibold text-slate-950 dark:text-zinc-50">{{ suggestion.permissionKey }}</code>
            <UIcon name="i-lucide-arrow-right" class="size-4 shrink-0 text-slate-400" />
            <UBadge color="info" variant="soft" class="font-mono">{{ suggestion.roleKey }}</UBadge>
            <UBadge :color="statusColor(suggestion)" variant="soft">{{ statusLabel(suggestion) }}</UBadge>
          </div>
          <div class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500 dark:text-zinc-400">
            <span class="inline-flex min-w-0 items-center gap-1">
              <UIcon name="i-lucide-blocks" class="size-3.5 shrink-0" />
              <span class="break-all">{{ suggestion.ownerExtensionId }}@{{ suggestion.extensionVersion }}</span>
            </span>
            <span class="inline-flex items-center gap-1 font-mono" :title="suggestion.packageDigest">
              <UIcon name="i-lucide-fingerprint" class="size-3.5" />
              {{ shortDigest(suggestion.packageDigest) }}
            </span>
            <span class="inline-flex items-center gap-1">
              <UIcon name="i-lucide-clock-3" class="size-3.5" />
              {{ formatDate(suggestion.createdAt) }}
            </span>
          </div>
        </div>
        <div class="flex flex-wrap items-center gap-2 lg:justify-end">
          <template v-if="suggestion.approvalState === 'pending'">
            <UButton color="neutral" variant="outline" leading-icon="i-lucide-x" :disabled="deciding" @click="openDecision(suggestion, 'rejected')">
              {{ t('admin.roles.suggestions.reject') }}
            </UButton>
            <UButton color="primary" leading-icon="i-lucide-shield-check" :disabled="deciding" @click="openDecision(suggestion, 'approved')">
              {{ t('admin.roles.suggestions.approve') }}
            </UButton>
          </template>
          <UButton
            v-else-if="suggestion.approvalState === 'approved' && !suggestion.applied"
            color="primary"
            leading-icon="i-lucide-shield-plus"
            :disabled="deciding"
            @click="openDecision(suggestion, 'approved')"
          >
            {{ t('admin.roles.suggestions.apply') }}
          </UButton>
        </div>
      </article>
    </div>

    <div v-if="nextCursor" class="mt-4 flex justify-center">
      <UButton color="neutral" variant="outline" leading-icon="i-lucide-chevrons-down" :loading="loadingMore" @click="loadSuggestions(false)">
        {{ t('admin.roles.suggestions.loadMore') }}
      </UButton>
    </div>
  </section>

  <UModal
    v-model:open="decisionOpen"
    :ui="{ content: 'sm:max-w-lg' }"
    @update:open="(open) => { if (!open) closeDecision() }"
  >
    <template #content>
      <div class="p-5 sm:p-6">
        <div class="flex items-start gap-3">
          <UIcon
            :name="pendingDecision?.state === 'rejected' ? 'i-lucide-circle-x' : 'i-lucide-shield-alert'"
            class="mt-0.5 size-5 shrink-0 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]"
          />
          <div class="min-w-0">
            <h3 class="text-base font-semibold text-slate-950 dark:text-zinc-50">{{ confirmationTitle }}</h3>
            <p class="mt-1 text-sm text-slate-600 dark:text-zinc-300">{{ confirmationDescription }}</p>
          </div>
        </div>
        <UAlert v-if="decisionError" class="mt-4" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="decisionError" />
        <div class="mt-6 flex justify-end gap-2">
          <UButton color="neutral" variant="ghost" :disabled="deciding" @click="closeDecision">
            {{ t('admin.roles.suggestions.cancel') }}
          </UButton>
          <UButton
            :color="pendingDecision?.state === 'rejected' ? 'error' : 'primary'"
            :leading-icon="pendingDecision?.state === 'rejected' ? 'i-lucide-x' : 'i-lucide-shield-check'"
            :loading="deciding"
            @click="submitDecision"
          >
            {{ confirmationAction }}
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
