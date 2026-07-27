<script setup lang="ts">
export type ReportReasonOption = {
  label: string
  value: string
}

defineProps<{
  open: boolean
  reasons: ReportReasonOption[]
  reason: string
  body: string
  submitting: boolean
  error?: string
  success?: boolean
}>()

const emit = defineEmits<{
  close: []
  submit: []
  'update:reason': [value: string]
  'update:body': [value: string]
  'dismiss-error': []
}>()

const { t } = useI18n()

function updateBody(event: Event) {
  emit('update:body', (event.target as HTMLTextAreaElement).value)
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="sf-report-dialog__overlay" @click.self="emit('close')">
      <section
        class="sf-report-dialog"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="'sf-report-dialog-title'"
      >
        <header class="sf-report-dialog__header">
          <h2 id="sf-report-dialog-title">{{ t('moderation.reportTitle') }}</h2>
          <button
            type="button"
            class="sf-report-dialog__close"
            :aria-label="t('moderation.close')"
            @click="emit('close')"
          >
            <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
          </button>
        </header>

        <div v-if="success" class="sf-report-dialog__body">
          <SFAlert variant="success" :title="t('moderation.reportSubmitted')" />
        </div>
        <div v-else class="sf-report-dialog__body">
          <SFAlert
            v-if="error"
            variant="danger"
            :title="error"
            closable
            @close="emit('dismiss-error')"
          />

          <fieldset class="sf-report-dialog__field">
            <legend>{{ t('moderation.reasonLabel') }}</legend>
            <div class="sf-report-dialog__reasons">
              <button
                v-for="option in reasons"
                :key="option.value"
                type="button"
                class="sf-report-dialog__reason"
                :class="{ 'is-active': reason === option.value }"
                :aria-pressed="reason === option.value"
                @click="emit('update:reason', option.value)"
              >
                {{ option.label }}
              </button>
            </div>
          </fieldset>

          <label class="sf-report-dialog__field">
            <span>{{ t('moderation.bodyLabel') }}</span>
            <textarea
              :value="body"
              rows="3"
              maxlength="2000"
              class="sf-report-dialog__textarea"
              :placeholder="t('moderation.bodyPlaceholder')"
              @input="updateBody"
            />
          </label>
        </div>

        <footer v-if="!success" class="sf-report-dialog__footer">
          <SFButton variant="ghost" size="sm" :disabled="submitting" @click="emit('close')">
            {{ t('moderation.cancel') }}
          </SFButton>
          <SFButton
            variant="primary"
            size="sm"
            :disabled="!reason || submitting"
            @click="emit('submit')"
          >
            {{ submitting ? t('moderation.submitting') : t('moderation.submit') }}
          </SFButton>
        </footer>
      </section>
    </div>
  </Teleport>
</template>
