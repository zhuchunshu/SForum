<script setup lang="ts">
import { Editor, EditorContent } from '@tiptap/vue-3'
import type { AnyExtension } from '@tiptap/core'
import {
  createSFEditorExtensions,
  escapeHtml,
  normalizeImageUrl,
  normalizeUserUrl,
  sforumEditorEmojiItems,
  type SFEditorContentPayload,
  type SForumEmojiItem,
  type TiptapContentReader
} from '~/utils/sfEditor'

const props = withDefaults(defineProps<{
  modelValue?: string
  placeholder?: string
  rows?: number
  hint?: string
  disabled?: boolean
  error?: string
  maxCharacters?: number
  submitLabel?: string
  // Host-admitted trusted L2 Tiptap extensions (digest-verified before pass-in).
  trustedExtensions?: unknown[]
}>(), {
  modelValue: '',
  placeholder: '写下你的回复...',
  rows: 6,
  hint: undefined,
  disabled: false,
  error: undefined,
  maxCharacters: 12000,
  submitLabel: '发布回复',
  trustedExtensions: () => []
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'content-change': [payload: SFEditorContentPayload]
  submit: [payload: SFEditorContentPayload]
}>()

const editor = shallowRef<Editor | null>(null)
const editorForContent = computed(() => editor.value || undefined)
const viewMode = ref<'write' | 'preview' | 'markdown' | 'native'>('write')
const markdownDraft = ref(props.modelValue)
const showEmojiPanel = ref(false)
const editorStateTick = ref(0)
const lastEmittedMarkdown = ref('')

const editorMinHeight = computed(() => `${Math.max(props.rows, 4) * 1.55 + 1.5}rem`)

const editorClass = computed(() => [
  'sf-editor',
  props.disabled ? 'sf-editor--disabled' : '',
  props.error ? 'sf-editor--invalid' : '',
  viewMode.value !== 'write' ? 'sf-editor--inspection' : ''
].filter(Boolean).join(' '))

const modeItems = [
  { value: 'write', label: '撰写' },
  { value: 'preview', label: '预览' },
  { value: 'markdown', label: 'Markdown' },
  { value: 'native', label: 'JSON' }
] as const

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
    isEmpty: currentEditor.isEmpty
  }
})

const nativeContent = computed(() => JSON.stringify(currentPayload.value.native, null, 2))

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

onMounted(() => {
  editor.value = new Editor({
    content: props.modelValue,
    contentType: 'markdown',
    editable: !props.disabled,
    extensions: createSFEditorExtensions({
      placeholder: props.placeholder,
      maxCharacters: props.maxCharacters,
      trustedExtensions: props.trustedExtensions
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
        'aria-label': '正文编辑器'
      }
    }
  })
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

watch(() => props.modelValue, value => {
  const nextMarkdown = value || ''
  const currentEditor = editor.value

  if (!currentEditor || nextMarkdown === currentPayload.value.markdown) {
    return
  }

  markdownDraft.value = nextMarkdown
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
    isEmpty: markdown.trim().length === 0
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
    isEmpty: sourceEditor.isEmpty
  }

  markdownDraft.value = payload.markdown

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

function insertImage() {
  const currentEditor = editor.value

  if (!currentEditor || props.disabled) {
    return
  }

  const rawUrl = window.prompt('输入图片地址。正式发布时应使用附件上传返回的地址。')
  const src = normalizeImageUrl(rawUrl || '')

  if (!src) {
    return
  }

  currentEditor.chain().focus().setImage({ src, alt: '插入图片' }).run()
}

function insertEmoji(emoji: SForumEmojiItem) {
  runEditorCommand(currentEditor => currentEditor.chain().focus().insertSForumEmoji(emoji).run())
  showEmojiPanel.value = false
}

function onMarkdownInput(event: Event) {
  const value = (event.target as HTMLTextAreaElement).value
  markdownDraft.value = value

  editor.value?.commands.setContent(value, {
    contentType: 'markdown'
  })
}

function submitContent() {
  emit('submit', currentPayload.value)
}
</script>

<template>
  <div
    :class="editorClass"
    :style="{ '--sf-editor-min-height': editorMinHeight }"
  >
    <div class="sf-editor__topbar">
      <div class="sf-editor__toolbar" aria-label="编辑工具栏">
        <button
          type="button"
          class="sf-editor__tool"
          title="撤销"
          aria-label="撤销"
          :disabled="disabled || !canUndo()"
          @click="runEditorCommand(currentEditor => currentEditor.chain().focus().undo().run())"
        >
          <UIcon name="i-lucide-undo-2" class="sf-editor__tool-icon" />
        </button>
        <button
          type="button"
          class="sf-editor__tool"
          title="重做"
          aria-label="重做"
          :disabled="disabled || !canRedo()"
          @click="runEditorCommand(currentEditor => currentEditor.chain().focus().redo().run())"
        >
          <UIcon name="i-lucide-redo-2" class="sf-editor__tool-icon" />
        </button>
        <span class="sf-editor__separator" aria-hidden="true" />
        <button
          type="button"
          class="sf-editor__tool"
          :class="{ 'sf-editor__tool--active': isActive('bold') }"
          title="加粗"
          aria-label="加粗"
          :disabled="disabled"
          @click="runEditorCommand(currentEditor => currentEditor.chain().focus().toggleBold().run())"
        >
          <UIcon name="i-lucide-bold" class="sf-editor__tool-icon" />
        </button>
        <button
          type="button"
          class="sf-editor__tool"
          :class="{ 'sf-editor__tool--active': isActive('italic') }"
          title="斜体"
          aria-label="斜体"
          :disabled="disabled"
          @click="runEditorCommand(currentEditor => currentEditor.chain().focus().toggleItalic().run())"
        >
          <UIcon name="i-lucide-italic" class="sf-editor__tool-icon" />
        </button>
        <button
          type="button"
          class="sf-editor__tool"
          :class="{ 'sf-editor__tool--active': isActive('underline') }"
          title="下划线"
          aria-label="下划线"
          :disabled="disabled"
          @click="runEditorCommand(currentEditor => currentEditor.chain().focus().toggleUnderline().run())"
        >
          <UIcon name="i-lucide-underline" class="sf-editor__tool-icon" />
        </button>
        <button
          type="button"
          class="sf-editor__tool"
          :class="{ 'sf-editor__tool--active': isActive('code') }"
          title="行内代码"
          aria-label="行内代码"
          :disabled="disabled"
          @click="runEditorCommand(currentEditor => currentEditor.chain().focus().toggleCode().run())"
        >
          <UIcon name="i-lucide-code" class="sf-editor__tool-icon" />
        </button>
        <span class="sf-editor__separator" aria-hidden="true" />
        <button
          type="button"
          class="sf-editor__tool"
          :class="{ 'sf-editor__tool--active': isActive('heading', { level: 2 }) }"
          title="二级标题"
          aria-label="二级标题"
          :disabled="disabled"
          @click="runEditorCommand(currentEditor => currentEditor.chain().focus().toggleHeading({ level: 2 }).run())"
        >
          H2
        </button>
        <button
          type="button"
          class="sf-editor__tool"
          :class="{ 'sf-editor__tool--active': isActive('blockquote') }"
          title="引用"
          aria-label="引用"
          :disabled="disabled"
          @click="runEditorCommand(currentEditor => currentEditor.chain().focus().toggleBlockquote().run())"
        >
          <UIcon name="i-lucide-quote" class="sf-editor__tool-icon" />
        </button>
        <button
          type="button"
          class="sf-editor__tool"
          :class="{ 'sf-editor__tool--active': isActive('bulletList') }"
          title="无序列表"
          aria-label="无序列表"
          :disabled="disabled"
          @click="runEditorCommand(currentEditor => currentEditor.chain().focus().toggleBulletList().run())"
        >
          <UIcon name="i-lucide-list" class="sf-editor__tool-icon" />
        </button>
        <button
          type="button"
          class="sf-editor__tool"
          :class="{ 'sf-editor__tool--active': isActive('orderedList') }"
          title="有序列表"
          aria-label="有序列表"
          :disabled="disabled"
          @click="runEditorCommand(currentEditor => currentEditor.chain().focus().toggleOrderedList().run())"
        >
          <UIcon name="i-lucide-list-ordered" class="sf-editor__tool-icon" />
        </button>
        <button
          type="button"
          class="sf-editor__tool"
          :class="{ 'sf-editor__tool--active': isActive('codeBlock') }"
          title="代码块"
          aria-label="代码块"
          :disabled="disabled"
          @click="runEditorCommand(currentEditor => currentEditor.chain().focus().toggleCodeBlock().run())"
        >
          <UIcon name="i-lucide-square-code" class="sf-editor__tool-icon" />
        </button>
        <span class="sf-editor__separator" aria-hidden="true" />
        <button
          type="button"
          class="sf-editor__tool"
          :class="{ 'sf-editor__tool--active': isActive('link') }"
          title="链接"
          aria-label="链接"
          :disabled="disabled"
          @click="setLink"
        >
          <UIcon name="i-lucide-link" class="sf-editor__tool-icon" />
        </button>
        <button
          type="button"
          class="sf-editor__tool"
          title="插入图片"
          aria-label="插入图片"
          :disabled="disabled"
          @click="insertImage"
        >
          <UIcon name="i-lucide-image" class="sf-editor__tool-icon" />
        </button>
        <button
          type="button"
          class="sf-editor__tool"
          title="自定义表情"
          aria-label="自定义表情"
          :disabled="disabled"
          @click="showEmojiPanel = !showEmojiPanel"
        >
          <UIcon name="i-lucide-smile-plus" class="sf-editor__tool-icon" />
        </button>
      </div>

      <div class="sf-editor__modes" aria-label="内容视图">
        <button
          v-for="mode in modeItems"
          :key="mode.value"
          type="button"
          class="sf-editor__mode"
          :class="{ 'sf-editor__mode--active': viewMode === mode.value }"
          @click="viewMode = mode.value"
        >
          {{ mode.label }}
        </button>
      </div>
    </div>

    <div v-if="showEmojiPanel" class="sf-editor__emoji-panel">
      <button
        v-for="emoji in sforumEditorEmojiItems"
        :key="emoji.name"
        type="button"
        class="sf-editor__emoji-option"
        :title="emoji.label"
        @click="insertEmoji(emoji)"
      >
        <span class="sf-editor__emoji-native">{{ emoji.native }}</span>
        <span>{{ emoji.label }}</span>
      </button>
    </div>

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

      <textarea
        v-show="viewMode === 'markdown'"
        class="sf-editor__source"
        :value="markdownDraft"
        :disabled="disabled"
        spellcheck="false"
        aria-label="Markdown 源码"
        @input="onMarkdownInput"
      />

      <pre
        v-show="viewMode === 'native'"
        class="sf-editor__native"
      >{{ nativeContent }}</pre>
    </div>

    <div class="sf-editor__footer">
      <span
        class="sf-editor__status"
        :class="{ 'sf-editor__status--error': error }"
      >
        {{ footerText }}
      </span>
      <div class="sf-editor__meta">
        <span>{{ currentPayload.wordCount }} 词</span>
        <span>HTML / Markdown / JSON</span>
      </div>
      <SFButton
        size="sm"
        :disabled="disabled || currentPayload.isEmpty"
        @click="submitContent"
      >
        {{ submitLabel }}
      </SFButton>
    </div>
  </div>
</template>
