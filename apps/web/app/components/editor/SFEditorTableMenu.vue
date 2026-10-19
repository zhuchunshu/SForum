<script setup lang="ts">
import type { Editor } from '@tiptap/core'
import { BubbleMenu } from '@tiptap/vue-3/menus'
import {
  getSForumTableAlign,
  setSForumTableAlign,
  type SFTableCellAlign
} from '~/utils/editor/table'

const props = defineProps<{
  editor: Editor
  disabled?: boolean
  /** 预览模式隐藏：编辑器只是 v-show，选区仍在表格里。 */
  enabled?: boolean
}>()

const { t } = useI18n()
const activeAlign = ref<SFTableCellAlign | null>(null)

const rowActions = computed(() => [
  { command: 'addRowBefore' as const, icon: 'i-lucide-arrow-up-to-line', label: t('composer.table.addRowBefore') },
  { command: 'addRowAfter' as const, icon: 'i-lucide-arrow-down-to-line', label: t('composer.table.addRowAfter') },
  { command: 'deleteRow' as const, icon: 'i-lucide-table-rows-split', label: t('composer.table.deleteRow') }
])

const columnActions = computed(() => [
  { command: 'addColumnBefore' as const, icon: 'i-lucide-arrow-left-to-line', label: t('composer.table.addColumnBefore') },
  { command: 'addColumnAfter' as const, icon: 'i-lucide-arrow-right-to-line', label: t('composer.table.addColumnAfter') },
  { command: 'deleteColumn' as const, icon: 'i-lucide-table-columns-split', label: t('composer.table.deleteColumn') }
])

const alignActions = computed(() => [
  { value: 'left' as const, icon: 'i-lucide-align-left', label: t('composer.table.alignLeft') },
  { value: 'center' as const, icon: 'i-lucide-align-center', label: t('composer.table.alignCenter') },
  { value: 'right' as const, icon: 'i-lucide-align-right', label: t('composer.table.alignRight') }
])

const commands = {
  addRowBefore: () => props.editor.chain().focus().addRowBefore().run(),
  addRowAfter: () => props.editor.chain().focus().addRowAfter().run(),
  deleteRow: () => props.editor.chain().focus().deleteRow().run(),
  addColumnBefore: () => props.editor.chain().focus().addColumnBefore().run(),
  addColumnAfter: () => props.editor.chain().focus().addColumnAfter().run(),
  deleteColumn: () => props.editor.chain().focus().deleteColumn().run(),
  toggleHeaderRow: () => props.editor.chain().focus().toggleHeaderRow().run(),
  deleteTable: () => props.editor.chain().focus().deleteTable().run()
}

type SFEditorTableCommand = keyof typeof commands

function run(command: SFEditorTableCommand) {
  if (props.disabled) return
  commands[command]()
  syncActiveAlign()
}

function alignCells(align: SFTableCellAlign) {
  if (props.disabled) return
  setSForumTableAlign(props.editor, activeAlign.value === align ? null : align)
  syncActiveAlign()
}

function syncActiveAlign() {
  activeAlign.value = getSForumTableAlign(props.editor)
}

function shouldShow() {
  return props.enabled !== false && !props.disabled && props.editor.isActive('table')
}

// 编辑器容器是 overflow:hidden；表格靠近左/右边缘时，挂在容器内的气泡菜单会被裁掉，
// 因此把菜单挂到 body 上（tip 仍跟随当前单元格）。
function appendToBody() {
  return document.body
}

onMounted(() => {
  props.editor.on('transaction', syncActiveAlign)
  syncActiveAlign()
})

onBeforeUnmount(() => {
  props.editor.off('transaction', syncActiveAlign)
})
</script>

<template>
  <BubbleMenu
    :editor="editor"
    :should-show="shouldShow"
    :append-to="appendToBody"
    :options="{ placement: 'top', offset: 8 }"
    class="sf-editor-table-menu"
  >
    <div class="sf-editor-table-menu__group" role="group">
      <button
        v-for="item in rowActions"
        :key="item.command"
        type="button"
        class="sf-editor-table-menu__button"
        :title="item.label"
        :aria-label="item.label"
        @click="run(item.command)"
      >
        <UIcon :name="item.icon" class="size-3.5" aria-hidden="true" />
      </button>
    </div>

    <span class="sf-editor-table-menu__divider" aria-hidden="true" />

    <div class="sf-editor-table-menu__group" role="group">
      <button
        v-for="item in columnActions"
        :key="item.command"
        type="button"
        class="sf-editor-table-menu__button"
        :title="item.label"
        :aria-label="item.label"
        @click="run(item.command)"
      >
        <UIcon :name="item.icon" class="size-3.5" aria-hidden="true" />
      </button>
    </div>

    <span class="sf-editor-table-menu__divider" aria-hidden="true" />

    <div class="sf-editor-table-menu__group" role="group">
      <button
        type="button"
        class="sf-editor-table-menu__button"
        :title="t('composer.table.toggleHeaderRow')"
        :aria-label="t('composer.table.toggleHeaderRow')"
        @click="run('toggleHeaderRow')"
      >
        <UIcon name="i-lucide-panel-top" class="size-3.5" aria-hidden="true" />
      </button>
    </div>

    <span class="sf-editor-table-menu__divider" aria-hidden="true" />

    <div class="sf-editor-table-menu__group" role="group">
      <button
        v-for="item in alignActions"
        :key="item.value"
        type="button"
        class="sf-editor-table-menu__button"
        :class="{ 'sf-editor-table-menu__button--active': activeAlign === item.value }"
        :title="item.label"
        :aria-label="item.label"
        :aria-pressed="activeAlign === item.value"
        @click="alignCells(item.value)"
      >
        <UIcon :name="item.icon" class="size-3.5" aria-hidden="true" />
      </button>
    </div>

    <span class="sf-editor-table-menu__divider" aria-hidden="true" />

    <div class="sf-editor-table-menu__group" role="group">
      <button
        type="button"
        class="sf-editor-table-menu__button sf-editor-table-menu__button--danger"
        :title="t('composer.table.delete')"
        :aria-label="t('composer.table.delete')"
        @click="run('deleteTable')"
      >
        <UIcon name="i-lucide-trash-2" class="size-3.5" aria-hidden="true" />
      </button>
    </div>
  </BubbleMenu>
</template>
