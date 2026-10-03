<script setup lang="ts">
import { nodeViewProps, NodeViewContent, NodeViewWrapper } from '@tiptap/vue-3'
import {
  sforumProtectedKind,
  type SFShortcodeNodeViewHost
} from '~/utils/editor/shortcodes'
import type { EditorProtectedKind } from '~/runtime/editor-extensions/types'

const props = defineProps(nodeViewProps)
const { t } = useI18n()

const host = computed(() => props.extension.options.nodeViewHost as SFShortcodeNodeViewHost | undefined)
const kind = computed(() => sforumProtectedKind(props.node.attrs as Record<string, unknown>))
const disabled = computed(() => host.value?.disabled() ?? true)
const kindLabel = computed(() => kind.value ? t(`composer.shortcodes.kinds.${kind.value}`) : t('composer.shortcodes.unavailable'))
const contractLabel = computed(() => {
  const id = String(props.node.attrs?.id || '').trim()
  const version = String(props.node.attrs?.contractVersion || '').trim()
  const name = id.split('.').pop() || 'protected'
  return version ? `${name} · ${version.split('@').pop() || '1'}` : name
})

function iconFor(value: EditorProtectedKind | null) {
  switch (value) {
    case 'login': return 'i-lucide-log-in'
    case 'reply': return 'i-lucide-message-square-reply'
    case 'only-author': return 'i-lucide-user-round-check'
    default: return 'i-lucide-shield-alert'
  }
}
</script>

<template>
  <NodeViewWrapper
    as="section"
    class="sf-protected-node"
    :class="{
      'sf-protected-node--selected': selected,
      'sf-protected-node--disabled': disabled,
      'sf-protected-node--author-only': kind === 'only-author'
    }"
    :data-shortcode-kind="kind || 'unavailable'"
    role="group"
    :aria-label="t('composer.shortcodes.protectedNodeLabel', { kind: kindLabel })"
  >
    <div class="sf-protected-node__label" contenteditable="false">
      <span class="sf-protected-node__label-mark" aria-hidden="true"><UIcon :name="iconFor(kind)" class="size-3.5" /></span>
      <span>{{ kindLabel }}</span>
      <code>{{ contractLabel }}</code>
    </div>
    <NodeViewContent class="sf-protected-node__content" />
  </NodeViewWrapper>
</template>
