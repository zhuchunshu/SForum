<script setup lang="ts">
type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger'
type ButtonSize = 'sm' | 'md' | 'lg'
type ButtonType = 'button' | 'submit' | 'reset'

const props = withDefaults(defineProps<{
  variant?: ButtonVariant
  size?: ButtonSize
  type?: ButtonType
  disabled?: boolean
  loading?: boolean
  block?: boolean
  iconOnly?: boolean
  ariaLabel?: string
}>(), {
  variant: 'primary',
  size: 'md',
  type: 'button',
  disabled: false,
  loading: false,
  block: false,
  iconOnly: false,
  ariaLabel: undefined
})

const emit = defineEmits<{
  click: [event: MouseEvent]
}>()

const buttonClass = computed(() => [
  'sf-button',
  `sf-button--${props.variant}`,
  `sf-button--${props.size}`,
  props.block ? 'sf-button--block' : '',
  props.iconOnly ? 'sf-button--icon-only' : ''
].filter(Boolean).join(' '))
</script>

<template>
  <button
    :type="type"
    :class="buttonClass"
    :disabled="disabled || loading"
    :aria-busy="loading ? 'true' : undefined"
    :aria-label="ariaLabel"
    @click="emit('click', $event)"
  >
    <span v-if="$slots.leading" class="sf-button__icon" aria-hidden="true">
      <slot name="leading" />
    </span>
    <span v-if="!iconOnly" class="sf-button__label">
      <slot />
    </span>
    <span v-else class="sf-button__icon" aria-hidden="true">
      <slot />
    </span>
    <span v-if="$slots.trailing" class="sf-button__icon" aria-hidden="true">
      <slot name="trailing" />
    </span>
  </button>
</template>
