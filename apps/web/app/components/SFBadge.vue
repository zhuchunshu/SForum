<script setup lang="ts">
type BadgeVariant = 'neutral' | 'primary' | 'info' | 'success' | 'warning' | 'danger'

const props = withDefaults(defineProps<{
  label?: string
  variant?: BadgeVariant
  dot?: boolean
}>(), {
  label: undefined,
  variant: 'neutral',
  dot: false
})

const badgeClass = computed(() => [
  'sf-badge',
  `sf-badge--${props.variant}`
].join(' '))
</script>

<template>
  <span :class="badgeClass">
    <span v-if="dot" class="sf-badge__dot" aria-hidden="true" />
    <slot>{{ label }}</slot>
  </span>
</template>
