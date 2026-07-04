<script setup lang="ts">
const props = withDefaults(defineProps<{
  modelValue?: string
  placeholder?: string
  rows?: number
  hint?: string
}>(), {
  modelValue: '',
  placeholder: '写下你的回复...',
  rows: 6,
  hint: undefined
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  submit: []
}>()

function onInput(event: Event) {
  emit('update:modelValue', (event.target as HTMLTextAreaElement).value)
}
</script>

<template>
  <div class="sf-editor">
    <div class="sf-editor__toolbar">
      <slot name="toolbar">
        <SFBadge variant="neutral">B</SFBadge>
        <SFBadge variant="neutral">I</SFBadge>
        <SFBadge variant="neutral">Link</SFBadge>
      </slot>
    </div>
    <textarea
      class="sf-editor__control"
      :value="modelValue"
      :rows="rows"
      :placeholder="placeholder"
      @input="onInput"
    />
    <div class="sf-editor__footer">
      <span>{{ hint || `${modelValue.length} 字` }}</span>
      <SFButton size="sm" @click="emit('submit')">
        发布回复
      </SFButton>
    </div>
  </div>
</template>
