<script setup lang="ts">
import { useTrustedEditorCatalog } from '~/composables/editor/useTrustedEditorCatalog'
import SFEditorToolbar, {
  type SFEditorBlockFormat,
  type SFEditorToolbarAction,
  type SFEditorViewMode
} from '~/components/editor/SFEditorToolbar.vue'
import SFEditorImageUploadModal from '~/components/editor/SFEditorImageUploadModal.vue'
import SFEditorImageMenu from '~/components/editor/SFEditorImageMenu.vue'
import { Editor, EditorContent } from '@tiptap/vue-3'
import type { AnyExtension } from '@tiptap/core'
import {
  collectEditorAttachmentIds,
  createSFEditorExtensions,
  escapeHtml,
  normalizeUserUrl,
  type SFEditorContentPayload,
  type TiptapContentReader
} from '~/utils/sfEditor'
import { imageFilesFromList, useEditorImageUpload } from '~/composables/editor/useEditorImageUpload'

const props = withDefaults(defineProps<{
  modelValue?: string
  /** 首次挂载使用的 Markdown 或原生 Tiptap JSON；后续编辑仍通过 Markdown v-model 同步。 */
  initialContent?: string | Record<string, unknown>
  placeholder?: string
  rows?: number
  hint?: string
  disabled?: boolean
  submitDisabled?: boolean
  error?: string
  maxCharacters?: number
  submitLabel?: string
  /** 由抽屉等宿主提供统一底部操作区时，可隐藏编辑器内建提交按钮。 */
  submitVisible?: boolean
  compact?: boolean
  preset?: 'full' | 'basic-field'
  imageSurface?: 'topic' | 'comment'
  ariaLabel?: string
  cancelLabel?: string
  supportLabel?: string
  // Host-admitted trusted L2 Tiptap extensions (digest-verified before pass-in).
  trustedExtensions?: unknown[]
  // 默认从 Host editor-catalog 拉取并 digest-verify 准入 L2；失败 fail-closed。
  loadTrustedCatalog?: boolean
}>(), {
  modelValue: '',
  initialContent: undefined,
  placeholder: '写下你的回复...',
  rows: 6,
  hint: undefined,
  disabled: false,
  submitDisabled: false,
  error: undefined,
  maxCharacters: 12000,
  submitLabel: '发布回复',
  submitVisible: true,
  compact: false,
  preset: 'full',
  imageSurface: 'topic',
  ariaLabel: '正文编辑器',
  cancelLabel: '',
  supportLabel: '支持 Markdown',
  trustedExtensions: () => [],
  loadTrustedCatalog: true
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'content-change': [payload: SFEditorContentPayload]
  submit: [payload: SFEditorContentPayload]
  cancel: []
}>()

const editor = shallowRef<Editor | null>(null)
const editorForContent = computed(() => editor.value || undefined)
const viewMode = ref<SFEditorViewMode>('write')
// modelValue 只承载 Markdown；原生 JSON 必须走 initialContent，避免把 rawContent 当正文。
const editorStateTick = ref(0)
const lastEmittedMarkdown = ref('')
const imageDialogOpen = ref(false)
const imageInsertPosition = ref<number | null>(null)
const { t } = useI18n()
const { pendingUploadCount, uploadImages } = useEditorImageUpload({
  uploading: t('composer.imageUpload.uploading'),
  invalidType: t('composer.imageUpload.invalidType'),
  notAllowed: t('composer.imageUpload.notAllowed'),
  tooLarge: (fileName, maxSize) => t('composer.imageUpload.tooLarge', { fileName, maxSize }),
  uploaded: count => t('composer.imageUpload.uploaded', { count }),
  partiallyUploaded: (uploaded, failed) => t('composer.imageUpload.partiallyUploaded', { uploaded, failed }),
  failed: t('composer.imageUpload.failed'),
  positionLost: t('composer.imageUpload.positionLost')
})

const editorMinHeight = computed(() => `${Math.max(props.rows, 4) * 1.55 + 1.5}rem`)

const editorClass = computed(() => [
  'sf-editor',
  props.disabled ? 'sf-editor--disabled' : '',
  props.error ? 'sf-editor--invalid' : '',
  props.compact ? 'sf-editor--compact' : '',
  props.preset === 'basic-field' ? 'sf-editor--basic-field' : '',
  props.imageSurface === 'comment' ? 'sf-editor--image-comment' : ''
].filter(Boolean).join(' '))

const blockFormat = computed<SFEditorBlockFormat>(() => {
  if (isActive('heading', { level: 2 })) return 'heading-2'
  if (isActive('heading', { level: 3 })) return 'heading-3'
  return 'paragraph'
})

const toolbarActive = computed<Partial<Record<SFEditorToolbarAction, boolean>>>(() => ({
  bold: isActive('bold'),
  italic: isActive('italic'),
  strike: isActive('strike'),
  code: isActive('code'),
  bulletList: isActive('bulletList'),
  orderedList: isActive('orderedList'),
  blockquote: isActive('blockquote'),
  codeBlock: isActive('codeBlock'),
  link: isActive('link')
}))

const currentPayload = computed<SFEditorContentPayload>(() => {
  void editorStateTick.value

  if (!editor.value) {
    return emptyPayload(props.modelValue)
  }

  const currentEditor = editor.value

  return {
    html: currentEditor.getHTML(),
    markdown: currentEditor.getMarkdown(),
    native: currentEditor.getJSON(),
    text: currentEditor.getText(),
    characterCount: currentEditor.storage.characterCount.characters(),
    wordCount: currentEditor.storage.characterCount.words(),
    isEmpty: currentEditor.isEmpty,
    attachmentIds: collectEditorAttachmentIds(currentEditor.getJSON()),
    pendingUploadCount: pendingUploadCount.value
  }
})

const footerText = computed(() => {
  if (props.error) {
    return props.error
  }

  if (props.hint) {
    return props.hint
  }

  const count = currentPayload.value.characterCount
  const suffix = props.maxCharacters ? ` / ${props.maxCharacters}` : ''
  return `${count}${suffix} 字`
})

const shouldLoadTrustedCatalog = props.loadTrustedCatalog && props.preset === 'full'
const catalogReady = ref(!shouldLoadTrustedCatalog)
const admittedExtensions = shallowRef<unknown[]>(props.trustedExtensions || [])

onMounted(async () => {
  let trusted = props.trustedExtensions || []
  if (shouldLoadTrustedCatalog) {
    try {
      const { loadAdmittedExtensions } = useTrustedEditorCatalog()
      const admitted = await loadAdmittedExtensions()
      // 父组件显式传入的扩展优先于 catalog 准入结果。
      trusted = [...admitted.extensions, ...trusted]
    } catch {
      // fail-closed：catalog 失败时仅核心扩展
    }
  }
  admittedExtensions.value = trusted
  catalogReady.value = true
  // object → Tiptap JSON 文档；string → Markdown。禁止把 editor-document 的 raw JSON 字符串当 Markdown。
  const initialContent = props.initialContent !== undefined && props.initialContent !== null
    ? props.initialContent
    : (props.modelValue || '')
  let nextEditor: Editor
  nextEditor = new Editor({
    content: initialContent,
    ...(typeof initialContent === 'string' ? { contentType: 'markdown' as const } : {}),
    editable: !props.disabled,
    extensions: createSFEditorExtensions({
      placeholder: props.placeholder,
      maxCharacters: props.maxCharacters,
      preset: props.preset,
      trustedExtensions: trusted,
      onImageDrop: (dropEditor, files, pos) => {
        void uploadImages(dropEditor, files, pos)
      }
    }) as AnyExtension[],
    onCreate: ({ editor: createdEditor }) => {
      syncFromEditor(createdEditor)
    },
    onUpdate: ({ editor: updatedEditor }) => {
      syncFromEditor(updatedEditor)
    },
    onSelectionUpdate: () => {
      editorStateTick.value += 1
    },
    editorProps: {
      attributes: {
        class: 'sf-editor__content',
        'aria-label': props.ariaLabel
      },
      handlePaste: (_view, event) => {
        const files = imageFilesFromList(event.clipboardData?.files)
        if (files.length === 0) return false

        event.preventDefault()
        void uploadImages(nextEditor, files, nextEditor.state.selection.from)
        return true
      }
    }
  })
  editor.value = nextEditor
})

onBeforeUnmount(() => {
  editor.value?.destroy()
})

watch(() => props.disabled, disabled => {
  editor.value?.setEditable(!disabled)
})

watch(() => props.placeholder, placeholder => {
  const placeholderExtension = editor.value?.extensionManager.extensions
    .find(extension => extension.name === 'placeholder')

  if (placeholderExtension) {
    placeholderExtension.options.placeholder = placeholder
  }
})

watch(pendingUploadCount, () => {
  if (editor.value) syncFromEditor(editor.value)
})

// 仅接受外部 Markdown 同步；跳过与自身 emit 相同的回写，以及与当前文档一致的值。
watch(() => props.modelValue, value => {
  const nextMarkdown = value || ''
  const currentEditor = editor.value

  if (!currentEditor) {
    return
  }
  if (nextMarkdown === lastEmittedMarkdown.value || nextMarkdown === currentPayload.value.markdown) {
    return
  }

  lastEmittedMarkdown.value = nextMarkdown
  currentEditor.commands.setContent(nextMarkdown, {
    contentType: 'markdown',
    emitUpdate: false
  })
  editorStateTick.value += 1
})

function emptyPayload(markdown: string): SFEditorContentPayload {
  return {
    html: markdown ? `<p>${escapeHtml(markdown)}</p>` : '<p></p>',
    markdown,
    native: {
      type: 'doc',
      content: markdown
        ? [{ type: 'paragraph', content: [{ type: 'text', text: markdown }] }]
        : [{ type: 'paragraph' }]
    },
    text: markdown,
    characterCount: markdown.length,
    wordCount: markdown.trim() ? markdown.trim().split(/\s+/).length : 0,
    isEmpty: markdown.trim().length === 0,
    attachmentIds: [],
    pendingUploadCount: pendingUploadCount.value
  }
}

function syncFromEditor(sourceEditor: TiptapContentReader) {
  editorStateTick.value += 1

  const payload: SFEditorContentPayload = {
    html: sourceEditor.getHTML(),
    markdown: sourceEditor.getMarkdown(),
    native: sourceEditor.getJSON(),
    text: sourceEditor.getText(),
    characterCount: sourceEditor.storage.characterCount.characters(),
    wordCount: sourceEditor.storage.characterCount.words(),
    isEmpty: sourceEditor.isEmpty,
    attachmentIds: collectEditorAttachmentIds(sourceEditor.getJSON()),
    pendingUploadCount: pendingUploadCount.value
  }

  if (payload.markdown !== lastEmittedMarkdown.value) {
    lastEmittedMarkdown.value = payload.markdown
    emit('update:modelValue', payload.markdown)
  }

  emit('content-change', payload)
}

function runEditorCommand(command: (currentEditor: Editor) => boolean) {
  const currentEditor = editor.value

  if (!currentEditor || props.disabled) {
    return
  }

  command(currentEditor)
}

function isActive(name: string, attrs?: Record<string, unknown>) {
  void editorStateTick.value
  return editor.value?.isActive(name, attrs) || false
}

function canUndo() {
  void editorStateTick.value
  return editor.value?.can().undo() || false
}

function canRedo() {
  void editorStateTick.value
  return editor.value?.can().redo() || false
}

function setLink() {
  const currentEditor = editor.value

  if (!currentEditor || props.disabled) {
    return
  }

  const previousUrl = currentEditor.getAttributes('link').href || ''
  const rawUrl = window.prompt('输入链接地址', previousUrl)

  if (rawUrl === null) {
    return
  }

  const href = normalizeUserUrl(rawUrl)

  if (!href) {
    currentEditor.chain().focus().unsetLink().run()
    return
  }

  currentEditor.chain().focus().extendMarkRange('link').setLink({ href }).run()
}

function openImageDialog() {
  const currentEditor = editor.value

  if (!currentEditor || props.disabled) {
    return
  }
  imageInsertPosition.value = currentEditor.state.selection.from
  imageDialogOpen.value = true
}

function onImageDialogOpenChange(open: boolean) {
  imageDialogOpen.value = open
  if (!open) imageInsertPosition.value = null
}

function onImageFilesSelected(files: File[]) {
  const currentEditor = editor.value
  const position = imageInsertPosition.value
  imageInsertPosition.value = null

  if (!currentEditor || position == null || files.length === 0) return
  void uploadImages(currentEditor, files, position)
}

function runToolbarAction(action: SFEditorToolbarAction) {
  if (action === 'link') {
    setLink()
    return
  }
  if (action === 'image') {
    openImageDialog()
    return
  }

  const commands: Record<Exclude<SFEditorToolbarAction, 'link' | 'image'>, (currentEditor: Editor) => boolean> = {
    undo: currentEditor => currentEditor.chain().focus().undo().run(),
    redo: currentEditor => currentEditor.chain().focus().redo().run(),
    bold: currentEditor => currentEditor.chain().focus().toggleBold().run(),
    italic: currentEditor => currentEditor.chain().focus().toggleItalic().run(),
    strike: currentEditor => currentEditor.chain().focus().toggleStrike().run(),
    code: currentEditor => currentEditor.chain().focus().toggleCode().run(),
    bulletList: currentEditor => currentEditor.chain().focus().toggleBulletList().run(),
    orderedList: currentEditor => currentEditor.chain().focus().toggleOrderedList().run(),
    blockquote: currentEditor => currentEditor.chain().focus().toggleBlockquote().run(),
    codeBlock: currentEditor => currentEditor.chain().focus().toggleCodeBlock().run()
  }
  runEditorCommand(commands[action])
}

function setBlockFormat(format: SFEditorBlockFormat) {
  runEditorCommand(currentEditor => {
    if (format === 'heading-2') return currentEditor.chain().focus().setHeading({ level: 2 }).run()
    if (format === 'heading-3') return currentEditor.chain().focus().setHeading({ level: 3 }).run()
    return currentEditor.chain().focus().setParagraph().run()
  })
}

function submitContent() {
  if (props.disabled || props.submitDisabled || pendingUploadCount.value > 0) {
    return
  }
  emit('submit', currentPayload.value)
}
</script>

<template>
  <div
    :class="editorClass"
    :style="{ '--sf-editor-min-height': editorMinHeight }"
  >
    <SFEditorToolbar
      v-if="!compact"
      :preset="preset"
      :disabled="disabled"
      :can-undo="canUndo()"
      :can-redo="canRedo()"
      :active="toolbarActive"
      :block-format="blockFormat"
      :view-mode="viewMode"
      @action="runToolbarAction"
      @block-format="setBlockFormat"
      @view-mode="viewMode = $event"
    />

    <SFEditorImageUploadModal
      :open="imageDialogOpen"
      :disabled="disabled"
      @update:open="onImageDialogOpenChange"
      @select="onImageFilesSelected"
    />

    <SFEditorImageMenu
      v-if="editor && preset === 'full'"
      :editor="editor"
      :disabled="disabled"
    />

    <div class="sf-editor__body">
      <ClientOnly>
        <EditorContent
          v-show="viewMode === 'write'"
          :editor="editorForContent"
        />
        <template #fallback>
          <div class="sf-editor__loading">
            编辑器加载中
          </div>
        </template>
      </ClientOnly>

      <div
        v-show="viewMode === 'preview'"
        class="sf-editor__preview"
        v-highlight
        v-html="sanitizeHtml(currentPayload.html)"
      />
    </div>

    <div class="sf-editor__footer">
      <template v-if="compact">
        <span class="sf-editor__compact-support">
          <UIcon name="i-lucide-file-code-2" class="size-3.5" aria-hidden="true" />
          {{ supportLabel }}
        </span>
        <div class="sf-editor__compact-actions">
          <SFButton
            v-if="cancelLabel"
            type="button"
            variant="ghost"
            size="sm"
            :disabled="disabled"
            @click="emit('cancel')"
          >
            {{ cancelLabel }}
          </SFButton>
          <SFButton
            size="sm"
            :disabled="disabled || submitDisabled || currentPayload.isEmpty || currentPayload.pendingUploadCount > 0"
            @click="submitContent"
          >
            <template #leading>
              <UIcon name="i-lucide-send" class="size-3.5" aria-hidden="true" />
            </template>
            {{ submitLabel }}
          </SFButton>
        </div>
      </template>
      <template v-else-if="preset === 'basic-field'">
        <span
          class="sf-editor__status"
          :class="{ 'sf-editor__status--error': error }"
        >
          {{ footerText }}
        </span>
      </template>
      <template v-else>
        <span
          class="sf-editor__status"
          :class="{ 'sf-editor__status--error': error }"
        >
          {{ footerText }}
        </span>
        <div class="sf-editor__meta">
          <span>{{ currentPayload.wordCount }} 词</span>
          <span>结构化文档</span>
        </div>
        <SFButton
          v-if="submitVisible"
          size="sm"
          :disabled="disabled || submitDisabled || currentPayload.isEmpty || currentPayload.pendingUploadCount > 0"
          @click="submitContent"
        >
          {{ submitLabel }}
        </SFButton>
      </template>
    </div>
  </div>
</template>
