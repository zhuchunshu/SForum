<script setup lang="ts">
import {
  buildSelectionQuoteMarkdown,
  parseSelectionQuoteTarget,
  selectionQuoteToolbarPosition,
  type SelectionQuoteRequest,
  type SelectionQuoteTarget
} from '~/utils/forum/forumSelectionQuote'

const props = defineProps<{
  enabled: boolean
}>()

const emit = defineEmits<{
  quote: [request: SelectionQuoteRequest]
}>()

const { t } = useI18n()
const toolbar = ref<HTMLElement | null>(null)
const visible = ref(false)
const placement = ref<'above' | 'below'>('above')
const toolbarStyle = ref({ left: '0px', top: '0px' })
let selectedText = ''
let selectedTarget: SelectionQuoteTarget | null = null
let selectionFrame = 0

function elementFromNode(node: Node | null) {
  if (!node) return null
  return node.nodeType === Node.ELEMENT_NODE
    ? node as Element
    : node.parentElement
}

function selectionSource(node: Node | null) {
  return elementFromNode(node)?.closest<HTMLElement>('[data-selection-quote-source]') || null
}

function hide() {
  visible.value = false
  selectedText = ''
  selectedTarget = null
}

async function updateFromSelection() {
  selectionFrame = 0
  if (!props.enabled) {
    hide()
    return
  }

  const selection = window.getSelection()
  if (!selection || selection.isCollapsed || selection.rangeCount !== 1) {
    hide()
    return
  }

  const anchorSource = selectionSource(selection.anchorNode)
  const focusSource = selectionSource(selection.focusNode)
  if (!anchorSource || anchorSource !== focusSource) {
    hide()
    return
  }

  const range = selection.getRangeAt(0)
  if (!anchorSource.contains(range.commonAncestorContainer)) {
    hide()
    return
  }

  const target = parseSelectionQuoteTarget(
    anchorSource.dataset.selectionQuoteSource,
    anchorSource.dataset.selectionQuoteCommentId
  )
  const markdown = buildSelectionQuoteMarkdown(selection.toString())
  const selectionRect = range.getBoundingClientRect()
  if (!target || !markdown || selectionRect.width <= 0 || selectionRect.height <= 0) {
    hide()
    return
  }

  selectedText = selection.toString()
  selectedTarget = target
  visible.value = true
  await nextTick()

  const toolbarElement = toolbar.value
  const host = toolbarElement?.parentElement
  if (!toolbarElement || !host) {
    hide()
    return
  }

  const nextPosition = selectionQuoteToolbarPosition({
    selectionRect,
    hostRect: host.getBoundingClientRect(),
    hostScrollLeft: host.scrollLeft,
    hostScrollTop: host.scrollTop,
    hostClientWidth: host.clientWidth,
    toolbarWidth: toolbarElement.offsetWidth,
    toolbarHeight: toolbarElement.offsetHeight
  })
  placement.value = nextPosition.placement
  toolbarStyle.value = {
    left: `${nextPosition.left}px`,
    top: `${nextPosition.top}px`
  }
}

function scheduleSelectionUpdate() {
  if (selectionFrame) cancelAnimationFrame(selectionFrame)
  selectionFrame = requestAnimationFrame(() => {
    void updateFromSelection()
  })
}

function onDocumentPointerDown(event: PointerEvent) {
  if (toolbar.value?.contains(event.target as Node)) return
  hide()
}

function onDocumentKeyDown(event: KeyboardEvent) {
  if (event.key === 'Escape') hide()
}

function submitQuote() {
  if (!selectedTarget) return
  const markdown = buildSelectionQuoteMarkdown(selectedText)
  if (!markdown) return

  emit('quote', { target: selectedTarget, markdown })
  window.getSelection()?.removeAllRanges()
  hide()
}

watch(() => props.enabled, enabled => {
  if (!enabled) hide()
})

onMounted(() => {
  document.addEventListener('selectionchange', scheduleSelectionUpdate)
  document.addEventListener('pointerdown', onDocumentPointerDown)
  document.addEventListener('keyup', scheduleSelectionUpdate)
  window.addEventListener('resize', hide)
})

onBeforeUnmount(() => {
  if (selectionFrame) cancelAnimationFrame(selectionFrame)
  document.removeEventListener('selectionchange', scheduleSelectionUpdate)
  document.removeEventListener('pointerdown', onDocumentPointerDown)
  document.removeEventListener('keyup', scheduleSelectionUpdate)
  window.removeEventListener('resize', hide)
})
</script>

<template>
  <div
    v-show="visible"
    ref="toolbar"
    class="sf-selection-quote-action"
    :class="`sf-selection-quote-action--${placement}`"
    :style="toolbarStyle"
    role="toolbar"
    :aria-label="t('topicDetail.selectionQuote.toolbarLabel')"
  >
    <button type="button" @pointerdown.prevent @click="submitQuote">
      <UIcon name="i-lucide-quote" class="size-3.5" aria-hidden="true" />
      <span>{{ t('topicDetail.selectionQuote.quoteAndReply') }}</span>
    </button>
  </div>
</template>

<style scoped>
.sf-selection-quote-action {
  position: absolute;
  z-index: 75;
  display: flex;
  align-items: center;
  height: 36px;
  padding: 3px;
  border: 1px solid #202a3b;
  border-radius: 6px;
  background: #202a3b;
  color: #fff;
  box-shadow: 0 9px 24px rgb(15 25 45 / 28%);
}

.sf-selection-quote-action::after {
  position: absolute;
  left: 50%;
  width: 10px;
  height: 10px;
  background: #202a3b;
  content: '';
  transform: translateX(-50%) rotate(45deg);
}

.sf-selection-quote-action--above::after {
  bottom: -6px;
}

.sf-selection-quote-action--below::after {
  top: -6px;
}

.sf-selection-quote-action button {
  position: relative;
  z-index: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 28px;
  padding: 0 8px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: inherit;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0;
  cursor: pointer;
}

.sf-selection-quote-action button:hover {
  background: rgb(255 255 255 / 14%);
}

.sf-selection-quote-action button:focus-visible {
  outline: 2px solid #fff;
  outline-offset: 1px;
}

@media (max-width: 640px) {
  .sf-selection-quote-action {
    max-width: calc(100% - 24px);
  }
}
</style>
