<script setup lang="ts">
import { useModerationApi } from '~/composables/moderation/useModerationApi'
import { invalidateForumPublicData } from '~/composables/forum/useForumPublicDataInvalidation'
import { apiErrorMessage } from '~/composables/useApiClient'
import type { ModerationAction, ModerationReviewContext } from '~/composables/moderation/useModerationApi'
import { sanitizeHtml } from '~/utils/sfSanitize'

const props = defineProps<{ context: ModerationReviewContext; reportId?: number }>()
const emit = defineEmits<{ decided: []; close: [] }>()
const { t } = useI18n()
const toast = useToast()
const moderationApi = useModerationApi()
const reviewNote = ref('')
const submitting = ref<ModerationAction | null>(null)
const errorMessage = ref('')
const destructive = new Set<ModerationAction>(['reject', 'hide_and_close', 'delete_and_close'])

const actions = computed<ModerationAction[]>(() => props.context.source === 'pre_publish'
  ? ['approve', 'reject']
  : ['keep_and_close', 'hide_and_close', 'delete_and_close'])

async function decide(action: ModerationAction) {
  errorMessage.value = ''
  if (destructive.has(action) && !reviewNote.value.trim()) {
    errorMessage.value = t('moderation.workbench.noteRequired')
    return
  }
  submitting.value = action
  try {
    await moderationApi.submitDecision({
      source: props.context.source,
      targetType: props.context.targetType,
      targetId: props.context.targetId,
      reportId: props.reportId,
      action,
      reviewNote: reviewNote.value
    })
    if (action === 'approve') {
      invalidateForumPublicData()
    }
    toast.add({ color: 'primary', icon: 'i-lucide-check', title: t('moderation.workbench.decisionSaved'), duration: 10000 })
    emit('decided')
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('moderation.workbench.decisionFailed')
  } finally {
    submitting.value = null
  }
}
</script>

<template>
  <section class="border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-950 sm:p-5" aria-labelledby="review-context-title">
    <div class="flex items-start justify-between gap-3">
      <div><p class="text-xs font-semibold uppercase text-[var(--sf-accent)]">{{ t(`admin.moderation.type.${context.targetType}`) }} #{{ context.targetId }}</p><h2 id="review-context-title" class="mt-1 text-base font-semibold text-slate-900 dark:text-zinc-100">{{ context.title }}</h2></div>
      <UButton icon="i-lucide-x" color="neutral" variant="ghost" :aria-label="t('moderation.workbench.closeReview')" @click="$emit('close')" />
    </div>
    <div class="mt-4 grid min-w-0 gap-5 lg:grid-cols-[minmax(0,1fr)_15rem]">
      <div class="min-w-0">
        <div class="mb-3 flex flex-wrap gap-2 text-xs text-slate-500">
          <span>{{ context.authorName }}</span>
          <span>{{ context.category }}</span>
          <span v-if="context.parentTopic">{{ context.parentTopic }}</span>
          <span v-if="context.ipAddress" class="inline-flex items-center gap-1 font-mono text-slate-600 dark:text-zinc-300" :title="t('moderation.workbench.createIp')">
            <UIcon name="i-lucide-network" class="size-3.5 shrink-0" />
            {{ context.ipAddress }}
          </span>
          <span v-if="context.lastEditIp && context.lastEditIp !== context.ipAddress" class="inline-flex items-center gap-1 font-mono text-slate-500 dark:text-zinc-400" :title="t('moderation.workbench.lastEditIp')">
            <UIcon name="i-lucide-pencil" class="size-3.5 shrink-0" />
            {{ context.lastEditIp }}
          </span>
        </div>
        <div class="sf-prose max-w-none overflow-wrap-anywhere" v-highlight v-html="sanitizeHtml(context.html)" />
      </div>
      <aside class="border-t border-slate-200 pt-4 dark:border-zinc-800 lg:border-l lg:border-t-0 lg:pl-4 lg:pt-0">
        <label class="text-xs font-semibold text-slate-700 dark:text-zinc-300">{{ t('moderation.workbench.reviewNote') }}</label>
        <UTextarea v-model="reviewNote" :rows="5" class="mt-2 w-full" :placeholder="t('moderation.workbench.reviewNotePlaceholder')" />
        <SFAlert v-if="errorMessage" variant="danger" :title="errorMessage" closable class="mt-3" @close="errorMessage = ''" />
        <div class="mt-4 grid gap-2">
          <UButton v-for="action in actions" :key="action" block :color="action === 'approve' || action === 'keep_and_close' ? 'primary' : action === 'delete_and_close' ? 'error' : 'neutral'" :variant="action === 'approve' || action === 'keep_and_close' ? 'solid' : 'subtle'" :loading="submitting === action" :disabled="Boolean(submitting)" @click="decide(action)">{{ t(`moderation.action.${action}`) }}</UButton>
        </div>
      </aside>
    </div>
  </section>
</template>
