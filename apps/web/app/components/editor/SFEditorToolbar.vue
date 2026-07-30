<script setup lang="ts">
export type SFEditorToolbarAction =
  | 'undo'
  | 'redo'
  | 'bold'
  | 'italic'
  | 'strike'
  | 'code'
  | 'bulletList'
  | 'orderedList'
  | 'blockquote'
  | 'codeBlock'
  | 'link'
  | 'image'

export type SFEditorBlockFormat = 'paragraph' | 'heading-2' | 'heading-3'
export type SFEditorViewMode = 'write' | 'preview'

const props = defineProps<{
  preset: 'full' | 'basic-field'
  disabled: boolean
  canUndo: boolean
  canRedo: boolean
  active: Partial<Record<SFEditorToolbarAction, boolean>>
  blockFormat: SFEditorBlockFormat
  viewMode: SFEditorViewMode
}>()

const emit = defineEmits<{
  action: [action: SFEditorToolbarAction]
  'block-format': [format: SFEditorBlockFormat]
  'view-mode': [mode: SFEditorViewMode]
}>()

const full = computed(() => props.preset === 'full')

const markActions = computed(() => [
  { action: 'bold' as const, label: '加粗', icon: 'i-lucide-bold' },
  { action: 'italic' as const, label: '斜体', icon: 'i-lucide-italic' },
  ...(full.value ? [
    { action: 'strike' as const, label: '删除线', icon: 'i-lucide-strikethrough' },
    { action: 'code' as const, label: '行内代码', icon: 'i-lucide-code-2' }
  ] : [])
])

const blockActions = computed(() => [
  { action: 'bulletList' as const, label: '无序列表', icon: 'i-lucide-list' },
  { action: 'orderedList' as const, label: '有序列表', icon: 'i-lucide-list-ordered' },
  ...(full.value ? [
    { action: 'blockquote' as const, label: '引用', icon: 'i-lucide-quote' },
    { action: 'codeBlock' as const, label: '代码块', icon: 'i-lucide-square-code' }
  ] : [])
])

const insertActions = computed(() => [
  { action: 'link' as const, label: '链接', icon: 'i-lucide-link-2' },
  ...(full.value ? [
    { action: 'image' as const, label: '插入图片', icon: 'i-lucide-image' }
  ] : [])
])

const modeItems: Array<{ value: SFEditorViewMode, label: string }> = [
  { value: 'write', label: '撰写' },
  { value: 'preview', label: '预览' }
]

function selectBlockFormat(event: Event) {
  emit('block-format', (event.target as HTMLSelectElement).value as SFEditorBlockFormat)
}
</script>

<template>
  <div class="sf-editor__topbar">
    <div class="sf-editor__toolbar" aria-label="编辑工具栏">
      <div class="sf-editor__tool-group">
        <button
          type="button"
          class="sf-editor__tool"
          title="撤销"
          aria-label="撤销"
          :disabled="disabled || !canUndo"
          @pointerdown.prevent
          @click="emit('action', 'undo')"
        >
          <UIcon name="i-lucide-undo-2" class="sf-editor__tool-icon" />
        </button>
        <button
          type="button"
          class="sf-editor__tool"
          title="重做"
          aria-label="重做"
          :disabled="disabled || !canRedo"
          @pointerdown.prevent
          @click="emit('action', 'redo')"
        >
          <UIcon name="i-lucide-redo-2" class="sf-editor__tool-icon" />
        </button>
      </div>

      <span class="sf-editor__separator" aria-hidden="true" />

      <select
        v-if="full"
        class="sf-editor__format"
        :value="blockFormat"
        :disabled="disabled"
        aria-label="段落格式"
        @change="selectBlockFormat"
      >
        <option value="paragraph">正文</option>
        <option value="heading-2">标题 2</option>
        <option value="heading-3">标题 3</option>
      </select>

      <div class="sf-editor__tool-group">
        <button
          v-for="item in markActions"
          :key="item.action"
          type="button"
          class="sf-editor__tool"
          :class="{ 'sf-editor__tool--active': active[item.action] }"
          :title="item.label"
          :aria-label="item.label"
          :aria-pressed="Boolean(active[item.action])"
          :disabled="disabled"
          @pointerdown.prevent
          @click="emit('action', item.action)"
        >
          <UIcon :name="item.icon" class="sf-editor__tool-icon" />
        </button>
      </div>

      <span class="sf-editor__separator" aria-hidden="true" />

      <div class="sf-editor__tool-group">
        <button
          v-for="item in blockActions"
          :key="item.action"
          type="button"
          class="sf-editor__tool"
          :class="{ 'sf-editor__tool--active': active[item.action] }"
          :title="item.label"
          :aria-label="item.label"
          :aria-pressed="Boolean(active[item.action])"
          :disabled="disabled"
          @pointerdown.prevent
          @click="emit('action', item.action)"
        >
          <UIcon :name="item.icon" class="sf-editor__tool-icon" />
        </button>
      </div>

      <span class="sf-editor__separator" aria-hidden="true" />

      <div class="sf-editor__tool-group">
        <button
          v-for="item in insertActions"
          :key="item.action"
          type="button"
          class="sf-editor__tool"
          :class="{ 'sf-editor__tool--active': active[item.action] }"
          :title="item.label"
          :aria-label="item.label"
          :aria-pressed="Boolean(active[item.action])"
          :disabled="disabled"
          @pointerdown.prevent
          @click="emit('action', item.action)"
        >
          <UIcon :name="item.icon" class="sf-editor__tool-icon" />
        </button>
      </div>
    </div>

    <div v-if="full" class="sf-editor__modes" aria-label="内容视图">
      <button
        v-for="mode in modeItems"
        :key="mode.value"
        type="button"
        class="sf-editor__mode"
        :class="{ 'sf-editor__mode--active': viewMode === mode.value }"
        :aria-pressed="viewMode === mode.value"
        @click="emit('view-mode', mode.value)"
      >
        {{ mode.label }}
      </button>
    </div>
  </div>
</template>
