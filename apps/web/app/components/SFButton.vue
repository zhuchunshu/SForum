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
  /**
   * 传入后渲染成 `<a>`：无 JS（入口 chunk 被拒载）时仍可导航。
   * 链接形态不接受 type/disabled，禁用语义降级为 aria-disabled + 拦截点击。
   */
  href?: string
}>(), {
  variant: 'primary',
  size: 'md',
  type: 'button',
  disabled: false,
  loading: false,
  block: false,
  iconOnly: false,
  ariaLabel: undefined,
  href: undefined
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

const inert = computed(() => props.disabled || props.loading)

function onClick(event: MouseEvent) {
  if (props.href && inert.value) {
    event.preventDefault()
    return
  }
  emit('click', event)
}
</script>

<template>
  <component
    :is="href ? 'a' : 'button'"
    :href="href"
    :type="href ? undefined : type"
    :class="buttonClass"
    :disabled="href ? undefined : inert"
    :aria-busy="!href && loading ? 'true' : undefined"
    :aria-disabled="href && inert ? 'true' : undefined"
    :aria-label="ariaLabel"
    @click="onClick"
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
  </component>
</template>
