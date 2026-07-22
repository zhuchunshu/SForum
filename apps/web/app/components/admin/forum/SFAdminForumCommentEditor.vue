<script setup lang="ts">
import type { AdminForumCommentDetail } from '~/utils/adminForumContent'
import { forumContentFromEditorPayload } from '~/utils/forumTaxonomy'
import type { SFEditorContentPayload } from '~/utils/sfEditor'

const props = defineProps<{
  comment: AdminForumCommentDetail
  requireReason: boolean
  reason: string
}>()

const emit = defineEmits<{
  'update:reason': [value: string]
  saved: []
  conflict: []
}>()

const { t } = useI18n()
const toast = useToast()
const forumApi = useForumApi()
const saving = ref(false)
const errorMessage = ref('')
const fieldErrors = ref<Record<string, string[]>>({})
const body = ref(props.comment.content.rawContent)

watch(() => props.comment, (comment) => {
  body.value = comment.content.rawContent
  errorMessage.value = ''
  fieldErrors.value = {}
})

async function save(payload: SFEditorContentPayload) {
  const reason = props.reason.trim()
  if (props.requireReason && !reason) {
    fieldErrors.value = { reason: [t('admin.forum.content.reasonRequired')] }
    return
  }

  saving.value = true
  errorMessage.value = ''
  fieldErrors.value = {}
  try {
    await forumApi.updateComment(
      props.comment.id,
      forumContentFromEditorPayload(payload),
      props.comment.currentRevision,
      reason || undefined
    )
    toast.add({ color: 'primary', icon: 'i-lucide-check', title: t('admin.forum.content.saved'), duration: 10000 })
    emit('saved')
  } catch (cause) {
    if (apiErrorReason(cause) === 'forum.revision_conflict') {
      emit('conflict')
      return
    }
    errorMessage.value = apiErrorMessage(cause) || t('admin.forum.content.saveFailed')
    fieldErrors.value = apiErrorFields(cause)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="space-y-4">
    <UAlert v-if="errorMessage" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="errorMessage" />
    <UFormField v-if="requireReason" :label="t('admin.forum.content.reason')" :error="fieldErrors.reason?.join(', ')">
      <UTextarea
        :model-value="reason"
        :placeholder="t('admin.forum.content.reasonPlaceholder')"
        :disabled="saving"
        :rows="3"
        class="w-full"
        @update:model-value="emit('update:reason', String($event))"
      />
    </UFormField>
    <LazySFEditor
      v-model="body"
      :placeholder="t('composer.bodyPlaceholder')"
      :submit-label="saving ? t('composer.submitting') : t('composer.save')"
      :disabled="saving"
      :error="fieldErrors.content?.join(', ')"
      @submit="save"
    />
  </div>
</template>
