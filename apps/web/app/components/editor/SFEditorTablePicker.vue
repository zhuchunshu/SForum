<script setup lang="ts">
import {
  SF_EDITOR_TABLE_LIMITS,
  type SFEditorTableSize
} from '~/utils/editor/table'

const props = withDefaults(defineProps<{
  disabled?: boolean
  rows?: number
  cols?: number
}>(), {
  disabled: false,
  rows: SF_EDITOR_TABLE_LIMITS.pickerRows,
  cols: SF_EDITOR_TABLE_LIMITS.pickerCols
})

const emit = defineEmits<{
  select: [size: SFEditorTableSize]
}>()

const { t } = useI18n()
const open = ref(false)
const hoverRows = ref(0)
const hoverCols = ref(0)
const focusRow = ref(1)
const focusCol = ref(1)
const cellRefs = ref<HTMLButtonElement[]>([])

const cells = computed(() => Array.from({ length: props.rows }, (_, rowIndex) => (
  Array.from({ length: props.cols }, (_, colIndex) => ({
    rows: rowIndex + 1,
    cols: colIndex + 1
  }))
)).flat())

function cellLabel(rows: number, cols: number) {
  return `${t('composer.table.insert')} ${rows} × ${cols}`
}

function resetPreview() {
  hoverRows.value = 0
  hoverCols.value = 0
}

function preview(rows: number, cols: number) {
  hoverRows.value = rows
  hoverCols.value = cols
}

function selectCell(rows: number, cols: number) {
  if (props.disabled) return
  emit('select', { rows, cols })
  open.value = false
  resetPreview()
}

function cellIndex(rows: number, cols: number) {
  return (rows - 1) * props.cols + (cols - 1)
}

function focusCell(rows: number, cols: number) {
  const nextRow = Math.min(props.rows, Math.max(1, rows))
  const nextCol = Math.min(props.cols, Math.max(1, cols))
  focusRow.value = nextRow
  focusCol.value = nextCol
  cellRefs.value[cellIndex(nextRow, nextCol)]?.focus()
  preview(nextRow, nextCol)
}

function onCellKeydown(event: KeyboardEvent, rows: number, cols: number) {
  const moves: Record<string, [number, number]> = {
    ArrowUp: [-1, 0],
    ArrowDown: [1, 0],
    ArrowLeft: [0, -1],
    ArrowRight: [0, 1]
  }
  const move = moves[event.key]

  if (move) {
    event.preventDefault()
    focusCell(rows + move[0], cols + move[1])
    return
  }

  if (event.key === 'Home') {
    event.preventDefault()
    focusCell(rows, 1)
    return
  }

  if (event.key === 'End') {
    event.preventDefault()
    focusCell(rows, props.cols)
  }
}

function setCellRef(element: unknown, rows: number, cols: number) {
  const index = cellIndex(rows, cols)
  if (element instanceof HTMLButtonElement) {
    cellRefs.value[index] = element
  }
}

watch(open, isOpen => {
  if (isOpen) {
    focusRow.value = 1
    focusCol.value = 1
    return
  }
  resetPreview()
})
</script>

<template>
  <UPopover
    v-model:open="open"
    :content="{ align: 'start', side: 'bottom', collisionPadding: 8 }"
  >
    <button
      type="button"
      class="sf-editor__tool"
      :title="t('composer.table.insert')"
      :aria-label="t('composer.table.insert')"
      :aria-expanded="open"
      :disabled="disabled"
      @pointerdown.prevent
    >
      <UIcon name="i-lucide-table" class="sf-editor__tool-icon" />
    </button>

    <template #content>
      <div
        class="sf-editor-table-picker"
        role="group"
        :aria-label="t('composer.table.pickerLabel')"
      >
        <p class="sf-editor-table-picker__status" aria-live="polite">
          {{ hoverRows && hoverCols
            ? `${hoverRows} × ${hoverCols}`
            : t('composer.table.pickerLabel') }}
        </p>
        <div
          class="sf-editor-table-picker__grid"
          :style="{ '--sf-table-picker-cols': cols }"
          @mouseleave="resetPreview"
        >
          <button
            v-for="cell in cells"
            :key="`${cell.rows}-${cell.cols}`"
            :ref="element => setCellRef(element, cell.rows, cell.cols)"
            type="button"
            class="sf-editor-table-picker__cell"
            :class="{ 'sf-editor-table-picker__cell--active': cell.rows <= hoverRows && cell.cols <= hoverCols }"
            :tabindex="cell.rows === focusRow && cell.cols === focusCol ? 0 : -1"
            :aria-label="cellLabel(cell.rows, cell.cols)"
            :title="cellLabel(cell.rows, cell.cols)"
            @mouseenter="preview(cell.rows, cell.cols)"
            @focus="preview(cell.rows, cell.cols); focusRow = cell.rows; focusCol = cell.cols"
            @keydown="onCellKeydown($event, cell.rows, cell.cols)"
            @click="selectCell(cell.rows, cell.cols)"
          />
        </div>
      </div>
    </template>
  </UPopover>
</template>
