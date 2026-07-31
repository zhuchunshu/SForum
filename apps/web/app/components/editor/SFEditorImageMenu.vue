<script setup lang="ts">
import type { Editor } from '@tiptap/core'
import { BubbleMenu } from '@tiptap/vue-3/menus'
import {
  normalizeEditorImageDisplaySize,
  sfEditorImageDisplaySizes,
  type SFEditorImageDisplaySize
} from '~/utils/editor/editorImage'

const props = defineProps<{
  editor: Editor
  disabled?: boolean
}>()

const { t } = useI18n()
const activeSize = ref<SFEditorImageDisplaySize>('standard')

const items = computed(() => [
  { value: 'compact' as const, icon: 'i-lucide-minimize-2', label: t('composer.imageUpload.sizeCompact') },
  { value: 'standard' as const, icon: 'i-lucide-square', label: t('composer.imageUpload.sizeStandard') },
  { value: 'wide' as const, icon: 'i-lucide-maximize-2', label: t('composer.imageUpload.sizeWide') }
])

function syncActiveSize() {
  if (!props.editor.isActive('image')) return
  activeSize.value = normalizeEditorImageDisplaySize(props.editor.getAttributes('image').displaySize)
}

function setSize(size: SFEditorImageDisplaySize) {
  if (props.disabled || !sfEditorImageDisplaySizes.includes(size)) return
  props.editor.chain().focus().updateAttributes('image', { displaySize: size }).run()
  syncActiveSize()
}

function shouldShow() {
  return !props.disabled && props.editor.isActive('image')
}

onMounted(() => {
  props.editor.on('transaction', syncActiveSize)
  syncActiveSize()
})

onBeforeUnmount(() => {
  props.editor.off('transaction', syncActiveSize)
})
</script>

<template>
  <BubbleMenu
    :editor="editor"
    :should-show="shouldShow"
    :options="{ placement: 'top', offset: 8 }"
    class="sf-editor-image-menu"
  >
    <span class="sf-editor-image-menu__label">{{ t('composer.imageUpload.sizeLabel') }}</span>
    <div class="sf-editor-image-menu__options" role="group" :aria-label="t('composer.imageUpload.sizeLabel')">
      <button
        v-for="item in items"
        :key="item.value"
        type="button"
        class="sf-editor-image-menu__option"
        :class="{ 'sf-editor-image-menu__option--active': activeSize === item.value }"
        :aria-pressed="activeSize === item.value"
        :title="item.label"
        @click="setSize(item.value)"
      >
        <UIcon :name="item.icon" class="size-3.5" aria-hidden="true" />
        <span>{{ item.label }}</span>
      </button>
    </div>
  </BubbleMenu>
</template>
