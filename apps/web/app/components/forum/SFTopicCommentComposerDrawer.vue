<script setup lang="ts">
import type { SFEditorContentPayload } from '~/utils/sfEditor'
import type {
  TopicCommentComposerContext,
  TopicCommentComposerMode
} from '~/composables/forum/useTopicCommentComposerDrawer'

type AvatarView = {
  kind: 'uploaded' | 'initials' | 'gravatar' | 'static'
  url: string
  attachmentId?: number | null
  alt: string
}

const props = withDefaults(defineProps<{
  open: boolean
  mode: TopicCommentComposerMode
  editorKey: string
  topicTitle: string
  topicExcerpt?: string
  actorName?: string
  avatar?: AvatarView | null
  context?: TopicCommentComposerContext | null
  modelValue: string
  initialContent?: string | Record<string, unknown>
  submitting?: boolean
  submitDisabled?: boolean
  error?: string
  errorClosable?: boolean
  reason?: string
  requireReason?: boolean
  reasonError?: string
}>(), {
  topicExcerpt: '',
  actorName: '',
  avatar: null,
  context: null,
  initialContent: undefined,
  submitting: false,
  submitDisabled: false,
  error: '',
  errorClosable: true,
  reason: '',
  requireReason: false,
  reasonError: ''
})

const emit = defineEmits<{
  'update:open': [value: boolean]
  'update:modelValue': [value: string]
  'update:reason': [value: string]
  'submit': [payload: SFEditorContentPayload]
  'cancel': []
  'dismiss-error': []
}>()

const { t } = useI18n()
const discardPromptOpen = ref(false)
const baselineMarkdown = ref('')
const baselineCaptured = ref(false)
const currentPayload = shallowRef<SFEditorContentPayload | null>(null)

const title = computed(() => {
  if (props.mode === 'edit') return t('topicDetail.composerDrawer.editTitle')
  if (props.mode === 'reply' && props.context) {
    return t('topicDetail.composerDrawer.replyTitle', { name: props.context.author })
  }
  return t('topicDetail.advancedReply')
})

const subtitle = computed(() => {
  if (props.mode === 'edit') {
    return t('topicDetail.composerDrawer.editSubtitle', {
      floor: props.context?.floorLabel || t('topicDetail.composerDrawer.thisReply')
    })
  }
  if (props.mode === 'reply') {
    return t('topicDetail.composerDrawer.replySubtitle', {
      floor: props.context?.floorLabel || t('topicDetail.composerDrawer.thisReply')
    })
  }
  return t('topicDetail.advancedReplySubtitle', { title: props.topicTitle })
})

const icon = computed(() => props.mode === 'edit'
  ? 'i-lucide-pencil'
  : props.mode === 'reply'
    ? 'i-lucide-corner-up-left'
    : 'i-lucide-message-square-text')

const contextLabel = computed(() => {
  if (props.mode === 'edit') return t('topicDetail.composerDrawer.originalReply')
  if (props.mode === 'reply') return t('topicDetail.composerDrawer.replyingTo')
  return t('topicDetail.composerDrawer.topicContext')
})

const contextAuthor = computed(() => props.context?.author || t('topicDetail.authorLabel'))
const contextExcerpt = computed(() => props.context?.excerpt || props.topicExcerpt)
const submitLabel = computed(() => {
  if (props.submitting) {
    return props.mode === 'edit'
      ? t('topicDetail.composerDrawer.saving')
      : t('topicDetail.submitting')
  }
  return props.mode === 'edit' ? t('topicDetail.saveEdit') : t('topicDetail.submitReply')
})
const submitIcon = computed(() => props.mode === 'edit' ? 'i-lucide-save' : 'i-lucide-send')
const dirty = computed(() => baselineCaptured.value
  && currentPayload.value != null
  && currentPayload.value.markdown !== baselineMarkdown.value)
const reasonMissing = computed(() => props.requireReason && !props.reason.trim())
const canSubmit = computed(() => Boolean(
  currentPayload.value
  && !currentPayload.value.isEmpty
  && !props.submitting
  && !props.submitDisabled
  && !reasonMissing.value
))

watch(
  () => [props.open, props.editorKey] as const,
  ([open]) => {
    if (!open) return
    discardPromptOpen.value = false
    baselineMarkdown.value = ''
    baselineCaptured.value = false
    currentPayload.value = null
  }
)

function onOpenUpdate(value: boolean) {
  if (value) {
    emit('update:open', true)
    return
  }
  requestClose()
}

function onContentChange(payload: SFEditorContentPayload) {
  currentPayload.value = payload
  if (!baselineCaptured.value) {
    baselineMarkdown.value = payload.markdown
    baselineCaptured.value = true
  }
}

function requestClose() {
  if (props.submitting) return
  if (dirty.value) {
    discardPromptOpen.value = true
    return
  }
  close()
}

function close() {
  discardPromptOpen.value = false
  emit('cancel')
  emit('update:open', false)
}

function submit() {
  if (!canSubmit.value || !currentPayload.value) return
  emit('submit', currentPayload.value)
}
</script>

<template>
  <USlideover
    :open="open"
    side="right"
    :dismissible="false"
    :close="false"
    :title="title"
    :description="subtitle"
    :ui="{
      overlay: 'sf-topic-composer-drawer__overlay',
      content: 'sf-topic-composer-drawer',
      body: 'sf-topic-composer-drawer__body',
      footer: 'sf-topic-composer-drawer__footer'
    }"
    @update:open="onOpenUpdate"
    @close:prevent="requestClose"
  >
    <template #title>
      <span class="sf-topic-composer-drawer__header-title">
        <span class="sf-topic-composer-drawer__icon" aria-hidden="true">
          <UIcon :name="icon" />
        </span>
        <span>{{ title }}</span>
      </span>
    </template>

    <template #description>
      {{ subtitle }}
    </template>

    <template #close>
      <button
        type="button"
        class="sf-topic-composer-drawer__close"
        :aria-label="t('topicDetail.composerDrawer.close')"
        :title="t('topicDetail.composerDrawer.close')"
        :disabled="submitting"
        @click="requestClose"
      >
        <UIcon name="i-lucide-x" aria-hidden="true" />
      </button>
    </template>

    <template #body>
      <div class="sf-topic-composer-drawer__context">
        <div class="sf-topic-composer-drawer__context-meta">
          <strong>{{ contextLabel }}</strong>
          <span>{{ contextAuthor }}</span>
          <a v-if="context?.href" :href="context.href">{{ t('topicDetail.composerDrawer.backToContent') }}</a>
        </div>
        <p>{{ contextExcerpt }}</p>
      </div>

      <SFAlert
        v-if="error"
        variant="danger"
        :title="error"
        :closable="errorClosable"
        @close="emit('dismiss-error')"
      />

      <LazySFEditor
        :key="editorKey"
        :model-value="modelValue"
        :initial-content="initialContent"
        :rows="12"
        :placeholder="mode === 'edit' ? t('topicDetail.editPlaceholder') : t('topicDetail.replyPlaceholder')"
        :submit-label="submitLabel"
        :submit-visible="false"
        :disabled="submitting"
        :submit-disabled="submitDisabled"
        :error="error"
        @update:model-value="emit('update:modelValue', $event)"
        @content-change="onContentChange"
      />

      <label v-if="requireReason" class="sf-topic-composer-drawer__reason">
        <span>{{ t('topicDetail.composerDrawer.editReason') }}</span>
        <textarea
          :value="reason"
          :placeholder="t('topicDetail.composerDrawer.editReasonPlaceholder')"
          :disabled="submitting"
          maxlength="500"
          rows="2"
          @input="emit('update:reason', ($event.target as HTMLTextAreaElement).value)"
        />
        <small v-if="reasonError" role="alert">{{ reasonError }}</small>
        <small v-else>{{ t('topicDetail.composerDrawer.editReasonHint') }}</small>
      </label>

      <div v-if="discardPromptOpen" class="sf-topic-composer-discard" role="alert">
        <div>
          <strong>{{ t('topicDetail.composerDrawer.discardTitle') }}</strong>
          <p>{{ t('topicDetail.composerDrawer.discardDescription') }}</p>
        </div>
        <div class="sf-topic-composer-discard__actions">
          <SFButton variant="ghost" @click="discardPromptOpen = false">
            {{ t('topicDetail.composerDrawer.keepEditing') }}
          </SFButton>
          <SFButton variant="danger" @click="close">
            {{ t('topicDetail.composerDrawer.discard') }}
          </SFButton>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="sf-topic-composer-drawer__actor">
        <SFAvatar v-if="actorName" :name="actorName" :avatar="avatar" size="sm" />
        <span v-if="actorName">{{ t('topicDetail.replyAs', { name: actorName }) }}</span>
      </div>
      <div class="sf-topic-composer-drawer__actions">
        <SFButton variant="ghost" :disabled="submitting" @click="requestClose">
          {{ t('topicDetail.cancel') }}
        </SFButton>
        <SFButton :loading="submitting" :disabled="!canSubmit" @click="submit">
          <template #leading>
            <UIcon :name="submitIcon" class="size-4" aria-hidden="true" />
          </template>
          {{ submitLabel }}
        </SFButton>
      </div>
    </template>

  </USlideover>
</template>

<style>
.sf-topic-composer-drawer__overlay { background: rgb(15 23 42 / 0.34); }

.sf-topic-composer-drawer {
  width: min(640px, calc(100vw - 48px));
  max-width: 640px;
  border-left: 1px solid var(--sf-public-border, var(--sf-border));
  background: var(--sf-public-surface, var(--sf-card));
  color: var(--sf-public-text, var(--sf-fg));
}

.sf-topic-composer-drawer > [data-slot="header"] {
  min-height: 76px;
  justify-content: space-between;
  border-bottom: 1px solid var(--sf-public-border, var(--sf-border));
  padding: 14px 20px;
}

.sf-topic-composer-drawer__header-title { display: flex; min-width: 0; align-items: center; gap: 12px; font-size: 16px; font-weight: 750; line-height: 1.35; }
.sf-topic-composer-drawer [data-slot="description"] { max-width: 470px; margin: 3px 48px 0; overflow: hidden; color: var(--sf-public-text-muted, var(--sf-fg-tertiary)); font-size: 12px; text-overflow: ellipsis; white-space: nowrap; }
.sf-topic-composer-drawer__icon { display: grid; width: 36px; height: 36px; flex: 0 0 36px; place-items: center; border-radius: 6px; background: var(--sf-accent-soft); color: var(--sf-accent); }
.sf-topic-composer-drawer__icon svg { width: 18px; height: 18px; }
.sf-topic-composer-drawer__close { display: grid; width: 36px; height: 36px; flex: 0 0 36px; place-items: center; border: 0; border-radius: 5px; background: transparent; color: var(--sf-public-text-muted, var(--sf-fg-tertiary)); cursor: pointer; }
.sf-topic-composer-drawer__close:hover { background: var(--sf-public-surface-muted, var(--sf-muted)); color: var(--sf-public-text, var(--sf-fg)); }

.sf-topic-composer-drawer__body { display: flex; min-height: 0; flex-direction: column; gap: 12px; overflow-y: auto; padding: 14px 20px 12px; }
.sf-topic-composer-drawer__context { flex: 0 0 auto; border-left: 3px solid var(--sf-accent); border-radius: 0 6px 6px 0; padding: 10px 12px; background: var(--sf-public-surface-muted, var(--sf-muted)); }
.sf-topic-composer-drawer__context-meta { display: flex; align-items: center; gap: 6px; color: var(--sf-public-text-muted, var(--sf-fg-tertiary)); font-size: 11px; }
.sf-topic-composer-drawer__context-meta strong { color: var(--sf-public-text-secondary, var(--sf-fg-secondary)); }
.sf-topic-composer-drawer__context-meta a { margin-left: auto; color: var(--sf-accent); font-weight: 650; }
.sf-topic-composer-drawer__context p { display: -webkit-box; margin: 6px 0 0; overflow: hidden; color: var(--sf-public-text-secondary, var(--sf-fg-secondary)); font-size: 12px; line-height: 1.55; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.sf-topic-composer-drawer .sf-editor { min-height: 0; flex: 1 1 auto; border-radius: 6px; }
.sf-topic-composer-drawer .sf-editor__body { min-height: min(44vh, 390px); }
.sf-topic-composer-drawer__reason { display: grid; flex: 0 0 auto; gap: 6px; }
.sf-topic-composer-drawer__reason > span { font-size: 12px; font-weight: 700; }
.sf-topic-composer-drawer__reason textarea { width: 100%; resize: vertical; border: 1px solid var(--sf-public-border, var(--sf-border)); border-radius: 6px; padding: 9px 10px; outline: 0; background: var(--sf-public-surface, var(--sf-card)); color: var(--sf-public-text, var(--sf-fg)); font-size: 12px; line-height: 1.5; }
.sf-topic-composer-drawer__reason textarea:focus { border-color: var(--sf-accent); box-shadow: 0 0 0 3px color-mix(in srgb, var(--sf-accent) 14%, transparent); }
.sf-topic-composer-drawer__reason small { color: var(--sf-public-text-muted, var(--sf-fg-tertiary)); font-size: 11px; }
.sf-topic-composer-drawer__reason small[role="alert"] { color: var(--sf-danger, #dc2626); }

.sf-topic-composer-drawer__footer { min-height: 68px; justify-content: space-between; border-top: 1px solid var(--sf-public-border, var(--sf-border)); padding: 11px 20px; }
.sf-topic-composer-drawer__actor, .sf-topic-composer-drawer__actions { display: flex; align-items: center; gap: 8px; }
.sf-topic-composer-drawer__actor { min-width: 0; color: var(--sf-public-text-muted, var(--sf-fg-tertiary)); font-size: 11px; }

.sf-topic-composer-discard { position: fixed; right: 18px; bottom: 78px; z-index: 90; display: flex; width: min(604px, calc(100vw - 84px)); align-items: center; justify-content: space-between; gap: 18px; border: 1px solid color-mix(in srgb, var(--sf-danger, #dc2626) 35%, var(--sf-border)); border-radius: 6px; padding: 13px 14px; background: var(--sf-public-surface, var(--sf-card)); color: var(--sf-public-text, var(--sf-fg)); box-shadow: 0 14px 36px rgb(15 23 42 / 0.2); }
.sf-topic-composer-discard strong { font-size: 12px; }
.sf-topic-composer-discard p { margin: 3px 0 0; color: var(--sf-public-text-muted, var(--sf-fg-tertiary)); font-size: 11px; }
.sf-topic-composer-discard__actions { display: flex; flex: 0 0 auto; gap: 7px; }

@media (max-width: 640px) {
  .sf-topic-composer-drawer[data-side="right"] { top: auto; right: 0; bottom: 0; left: 0; width: 100%; height: min(90dvh, 780px); max-width: none; border-top: 1px solid var(--sf-public-border, var(--sf-border)); border-left: 0; border-radius: 8px 8px 0 0; }
  .sf-topic-composer-drawer[data-side="right"][data-state="open"] { animation-name: slide-in-from-bottom; }
  .sf-topic-composer-drawer[data-side="right"][data-state="closed"] { animation-name: slide-out-to-bottom; }
  .sf-topic-composer-drawer > [data-slot="header"] { min-height: 62px; padding: 9px 14px; }
  .sf-topic-composer-drawer__icon { width: 32px; height: 32px; flex-basis: 32px; }
  .sf-topic-composer-drawer__header-title { font-size: 15px; }
  .sf-topic-composer-drawer [data-slot="description"] { max-width: calc(100vw - 128px); margin-left: 44px; }
  .sf-topic-composer-drawer__body { gap: 10px; padding: 10px 14px; }
  .sf-topic-composer-drawer__context { padding: 8px 10px; }
  .sf-topic-composer-drawer__context p { -webkit-line-clamp: 1; }
  .sf-topic-composer-drawer .sf-editor__body { min-height: 190px; }
  .sf-topic-composer-drawer__footer { min-height: 64px; padding: 9px 14px; }
  .sf-topic-composer-drawer__actor { display: none; }
  .sf-topic-composer-drawer__actions { width: 100%; }
  .sf-topic-composer-drawer__actions .sf-button { flex: 1; }
  .sf-topic-composer-discard { right: 10px; bottom: 72px; left: 10px; width: auto; align-items: flex-start; flex-direction: column; }
  .sf-topic-composer-discard__actions { width: 100%; }
  .sf-topic-composer-discard__actions .sf-button { flex: 1; }
}

@media (max-width: 640px) and (max-height: 680px) {
  .sf-topic-composer-drawer[data-side="right"] { height: 96dvh; }
  .sf-topic-composer-drawer__context { display: none; }
}
</style>
