<script setup lang="ts">
import { nodeViewProps, NodeViewWrapper } from '@tiptap/vue-3'
import {
  sforumReferenceKind,
  sforumReferenceNumericID,
  sforumReferenceSelection,
  type SFShortcodeNodeViewHost,
  type SFShortcodeReferencePreview
} from '~/utils/editor/shortcodes'
import type { EditorReferenceKind } from '~/runtime/editor-extensions/types'

const props = defineProps(nodeViewProps)
const { t } = useI18n()

const state = ref<'loading' | 'ready' | 'empty' | 'error'>('loading')
const reference = ref<SFShortcodeReferencePreview | null>(null)

const host = computed(() => props.extension.options.nodeViewHost as SFShortcodeNodeViewHost | undefined)
const kind = computed(() => sforumReferenceKind(props.node.attrs as Record<string, unknown>))
const numericID = computed(() => sforumReferenceNumericID(props.node.attrs as Record<string, unknown>))
const disabled = computed(() => host.value?.disabled() ?? true)
const kindLabel = computed(() => kind.value ? t(`composer.shortcodes.kinds.${kind.value}`) : t('composer.shortcodes.unavailable'))
const fallbackIcon = computed(() => iconFor(kind.value))
const iconName = computed(() => {
  const candidate = reference.value?.icon?.trim() || ''
  return kind.value === 'category' && candidate.startsWith('i-') ? candidate : fallbackIcon.value
})
const iconColor = computed(() => {
  const candidate = reference.value?.iconColor?.trim() || ''
  return kind.value === 'category' && /^#[0-9a-f]{6}$/i.test(candidate) ? candidate : undefined
})
const contractLabel = computed(() => {
  const id = String(props.node.attrs?.id || '').trim()
  const version = String(props.node.attrs?.contractVersion || '').trim()
  if (!id) return 'shortcode'
  const name = id.split('.').pop() || id
  return version ? `${name} · ${version.split('@').pop() || '1'}` : name
})

watch([kind, numericID, host], () => void loadReference(), { immediate: true })

async function loadReference() {
  const currentKind = kind.value
  reference.value = null
  if (!currentKind || !host.value) {
    state.value = 'error'
    return
  }
  if (currentKind === 'friend-links') {
    reference.value = { label: kindLabel.value }
    state.value = 'ready'
    return
  }
  if (numericID.value <= 0) {
    state.value = 'empty'
    return
  }
  state.value = 'loading'
  try {
    const item = await host.value.resolveReference(currentKind, numericID.value)
    if (!item) {
      state.value = 'empty'
      return
    }
    reference.value = item
    state.value = 'ready'
  } catch {
    state.value = 'error'
  }
}

function currentSelection() {
  const position = typeof props.getPos === 'function' ? props.getPos() : undefined
  return typeof position === 'number'
    ? sforumReferenceSelection(props.node.attrs as Record<string, unknown>, position)
    : undefined
}

function edit() {
  const selection = currentSelection()
  if (!selection || disabled.value) return
  host.value?.editReference(selection)
}

function iconFor(referenceKind: EditorReferenceKind | null) {
  switch (referenceKind) {
    case 'user': return 'i-lucide-user-round'
    case 'topic': return 'i-lucide-message-square-text'
    case 'comment': return 'i-lucide-message-circle'
    case 'category': return 'i-lucide-folder'
    case 'friend-links': return 'i-lucide-link-2'
    default: return 'i-lucide-shield-alert'
  }
}
</script>

<template>
  <NodeViewWrapper
    as="div"
    class="sf-shortcode-node"
    :class="{
      'sf-shortcode-node--selected': selected,
      'sf-shortcode-node--disabled': disabled,
      'sf-shortcode-node--user': kind === 'user',
      'sf-shortcode-node--category': kind === 'category'
    }"
    :data-shortcode-kind="kind || 'unavailable'"
    role="group"
    tabindex="0"
    :aria-label="t('composer.shortcodes.nodeLabel', { kind: kindLabel })"
    contenteditable="false"
    @dblclick.stop="edit"
    @keydown.enter.stop.prevent="edit"
  >
    <div class="sf-shortcode-node__label">
      <span class="sf-shortcode-node__label-mark" :style="iconColor ? { color: iconColor, borderColor: iconColor } : undefined" aria-hidden="true">
        <UIcon :name="iconName" class="size-3.5" />
      </span>
      <span class="sf-shortcode-node__kind">{{ kindLabel }}</span>
      <code class="sf-shortcode-node__contract">{{ contractLabel }}</code>
    </div>

    <div class="sf-shortcode-node__preview">
      <SFAvatar
        v-if="kind === 'user' && state === 'ready'"
        :name="reference?.label || ''"
        :avatar="reference?.avatar"
        size="sm"
        class="sf-shortcode-node__avatar"
      />
      <span
        v-else
        class="sf-shortcode-node__icon"
        :style="iconColor ? { color: iconColor, backgroundColor: `color-mix(in srgb, ${iconColor} 12%, transparent)` } : undefined"
        aria-hidden="true"
      >
        <UIcon :name="iconName" class="size-4" />
      </span>
      <span class="sf-shortcode-node__content">
        <span v-if="state === 'loading'" class="sf-shortcode-node__status" aria-live="polite">{{ t('composer.shortcodes.loading') }}</span>
        <template v-else-if="state === 'ready'">
          <span class="sf-shortcode-node__label-text">{{ reference?.label || kindLabel }}</span>
          <span v-if="reference?.secondaryLabel" class="sf-shortcode-node__secondary">{{ reference.secondaryLabel }}</span>
          <span v-if="kind === 'category' && iconColor" class="sf-shortcode-node__visual-meta" :style="{ color: iconColor }">{{ reference?.icon || 'i-lucide-folder' }}</span>
        </template>
        <span v-else-if="state === 'empty'" class="sf-shortcode-node__status" role="status">{{ t('composer.shortcodes.referenceUnavailable') }}</span>
        <span v-else class="sf-shortcode-node__status sf-shortcode-node__status--error" role="alert">{{ t('composer.shortcodes.loadError') }}</span>
      </span>
    </div>
  </NodeViewWrapper>
</template>
