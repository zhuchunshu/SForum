<script setup lang="ts">
const props = withDefaults(defineProps<{
  open: boolean
  disabled?: boolean
}>(), {
  disabled: false
})

const emit = defineEmits<{
  'update:open': [open: boolean]
  select: [files: File[]]
}>()

const { t } = useI18n()
const input = ref<HTMLInputElement | null>(null)
const dragDepth = ref(0)

watch(() => props.open, open => {
  if (!open) {
    dragDepth.value = 0
  }
})

function close() {
  emit('update:open', false)
}

function openFilePicker() {
  if (props.disabled) return
  if (input.value) input.value.value = ''
  input.value?.click()
}

function selectFiles(files: File[]) {
  dragDepth.value = 0
  if (files.length === 0) return
  emit('select', files)
  close()
}

function onInputChange(event: Event) {
  const target = event.target as HTMLInputElement
  selectFiles(Array.from(target.files || []))
  target.value = ''
}

function onDragEnter() {
  if (!props.disabled) dragDepth.value += 1
}

function onDragLeave() {
  dragDepth.value = Math.max(0, dragDepth.value - 1)
}

function onDrop(event: DragEvent) {
  if (props.disabled) return
  selectFiles(Array.from(event.dataTransfer?.files || []))
}

function onDropzoneKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter' && event.key !== ' ') return
  event.preventDefault()
  openFilePicker()
}
</script>

<template>
  <UModal
    :open="open"
    :ui="{ content: 'sm:max-w-lg' }"
    @update:open="emit('update:open', $event)"
  >
    <template #content>
      <section class="sf-editor-image-upload-dialog" aria-labelledby="sf-editor-image-upload-dialog-title">
        <div class="sf-editor-image-upload-dialog__header">
          <div>
            <h2 id="sf-editor-image-upload-dialog-title" class="sf-editor-image-upload-dialog__title">
              {{ t('composer.imageUpload.dialogTitle') }}
            </h2>
            <p class="sf-editor-image-upload-dialog__description">
              {{ t('composer.imageUpload.dialogDescription') }}
            </p>
          </div>
          <UButton
            type="button"
            color="neutral"
            variant="ghost"
            icon="i-lucide-x"
            :aria-label="t('common.close')"
            :title="t('common.close')"
            @click="close"
          />
        </div>

        <input
          ref="input"
          class="sr-only"
          type="file"
          accept="image/*"
          multiple
          tabindex="-1"
          aria-hidden="true"
          @change="onInputChange"
        >

        <div
          class="sf-editor-image-upload-dialog__dropzone"
          :class="{ 'sf-editor-image-upload-dialog__dropzone--active': dragDepth > 0, 'sf-editor-image-upload-dialog__dropzone--disabled': disabled }"
          role="button"
          :aria-disabled="disabled"
          tabindex="0"
          @click="openFilePicker"
          @keydown="onDropzoneKeydown"
          @dragenter.prevent="onDragEnter"
          @dragover.prevent
          @dragleave.prevent="onDragLeave"
          @drop.prevent="onDrop"
        >
          <UIcon name="i-lucide-image-up" class="sf-editor-image-upload-dialog__icon" aria-hidden="true" />
          <span class="sf-editor-image-upload-dialog__prompt">{{ t('composer.imageUpload.dialogPrompt') }}</span>
          <span class="sf-editor-image-upload-dialog__hint">{{ t('composer.imageUpload.dialogHint') }}</span>
        </div>
      </section>
    </template>
  </UModal>
</template>
