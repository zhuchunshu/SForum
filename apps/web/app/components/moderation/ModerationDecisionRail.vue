<script setup lang="ts">
import type { ModerationAction, ModerationReviewContext } from '~/composables/useModerationApi'
import { REVIEW_REQUIRED_ACTIONS, actionListForContext } from '~/utils/moderationWorkbench'

const props = defineProps<{
  context: ModerationReviewContext | null
  readonly?: boolean
  note: string
  submitting?: ModerationAction | null
  error?: string
  progressLabel?: string
  hasPrevious?: boolean
  hasNext?: boolean
  noteId?: string
}>()

const emit = defineEmits<{
  'update:note': [value: string]
  decide: [action: ModerationAction]
  previous: []
  next: []
}>()

const { t } = useI18n()
const actions = computed(() => actionListForContext(props.context, props.readonly))
</script>

<template>
  <aside class="sforum-moderation__right" :aria-label="t('moderation.workbench.decisionRail')">
    <div class="sforum-moderation-stepper">
      <button type="button" class="sforum-moderation-icon-button" :disabled="!hasPrevious" :aria-label="t('moderation.workbench.previousItem')" @click="$emit('previous')">
        <UIcon name="i-lucide-arrow-up" class="size-4" aria-hidden="true" />
      </button>
      <span>{{ progressLabel || t('moderation.workbench.noCurrentItem') }}</span>
      <button type="button" class="sforum-moderation-icon-button" :disabled="!hasNext" :aria-label="t('moderation.workbench.nextItem')" @click="$emit('next')">
        <UIcon name="i-lucide-arrow-down" class="size-4" aria-hidden="true" />
      </button>
    </div>

    <section class="sforum-moderation-rail-section">
      <header class="sforum-moderation-rail-section__head">
        <h2>{{ t('moderation.workbench.contentInfo') }}</h2>
        <span v-if="context">{{ t(`admin.moderation.type.${context.targetType}`) }} #{{ context.targetId }}</span>
      </header>
      <dl v-if="context" class="sforum-moderation-context-list">
        <div>
          <dt>{{ t('moderation.workbench.status') }}</dt>
          <dd>{{ context.status }}</dd>
        </div>
        <div>
          <dt>{{ t('moderation.workbench.category') }}</dt>
          <dd>{{ context.category }}</dd>
        </div>
        <div>
          <dt>{{ t('moderation.workbench.author') }}</dt>
          <dd>{{ context.authorName }}</dd>
        </div>
        <div v-if="context.triggers.length">
          <dt>{{ t('moderation.workbench.triggers') }}</dt>
          <dd>{{ context.triggers.map(trigger => t(`moderation.trigger.${trigger}`)).join(' / ') }}</dd>
        </div>
        <div v-if="context.ipAddress">
          <dt>{{ t('moderation.workbench.createIp') }}</dt>
          <dd class="font-mono">{{ context.ipAddress }}</dd>
        </div>
        <div v-if="context.lastEditIp && context.lastEditIp !== context.ipAddress">
          <dt>{{ t('moderation.workbench.lastEditIp') }}</dt>
          <dd class="font-mono">{{ context.lastEditIp }}</dd>
        </div>
      </dl>
      <p v-else class="sforum-moderation-rail-copy">{{ t('moderation.workbench.selectItemHint') }}</p>
    </section>

    <section class="sforum-moderation-rail-section">
      <header class="sforum-moderation-rail-section__head">
        <h3>{{ t('moderation.workbench.reviewNote') }}</h3>
        <span>{{ t('moderation.workbench.noteRuleShort') }}</span>
      </header>
      <label class="sforum-moderation-field-label" :for="noteId || 'moderation-review-note'">{{ t('moderation.workbench.reviewNoteLabel') }}</label>
      <textarea
        :id="noteId || 'moderation-review-note'"
        class="sforum-moderation-note"
        :value="note"
        :placeholder="t('moderation.workbench.reviewNotePlaceholder')"
        :disabled="readonly || Boolean(submitting)"
        @input="$emit('update:note', ($event.target as HTMLTextAreaElement).value)"
      />
      <p class="sforum-moderation-field-help">{{ t('moderation.workbench.noteRule') }}</p>
      <p v-if="error" class="sforum-moderation-field-error" role="alert">{{ error }}</p>
    </section>

    <section class="sforum-moderation-rail-section">
      <p v-if="readonly" class="sforum-moderation-rail-copy">
        {{ t('moderation.workbench.historyReadonly') }}
      </p>
      <div v-else class="sforum-moderation-actions">
        <button
          v-for="action in actions"
          :key="action"
          type="button"
          class="sforum-moderation-action"
          :class="{
            'sforum-moderation-action--primary': action === 'approve' || action === 'keep_and_close',
            'sforum-moderation-action--danger': action === 'delete_and_close',
            'sforum-moderation-action--warning': REVIEW_REQUIRED_ACTIONS.has(action) && action !== 'delete_and_close'
          }"
          :disabled="Boolean(submitting)"
          :aria-busy="submitting === action ? 'true' : undefined"
          @click="$emit('decide', action)"
        >
          <UIcon
            :name="action === 'approve' || action === 'keep_and_close' ? 'i-lucide-check' : action === 'delete_and_close' ? 'i-lucide-trash-2' : action === 'hide_and_close' ? 'i-lucide-eye-off' : 'i-lucide-x'"
            class="size-4"
            aria-hidden="true"
          />
          <span>{{ submitting === action ? t('moderation.workbench.submittingDecision') : t(`moderation.action.${action}`) }}</span>
        </button>
      </div>
    </section>
  </aside>
</template>
