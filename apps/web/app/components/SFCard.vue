<script setup lang="ts">
const props = withDefaults(defineProps<{
  title?: string
  subtitle?: string
  interactive?: boolean
  flush?: boolean
}>(), {
  title: undefined,
  subtitle: undefined,
  interactive: false,
  flush: false
})

const cardClass = computed(() => [
  'sf-card',
  props.interactive ? 'sf-card--interactive' : '',
  props.flush ? 'sf-card--flush' : ''
].filter(Boolean).join(' '))

const hasHeader = computed(() => Boolean(props.title || props.subtitle))
</script>

<template>
  <article :class="cardClass">
    <header v-if="hasHeader || $slots.header" class="sf-card__header">
      <slot name="header">
        <h3 v-if="title" class="sf-card__title">
          {{ title }}
        </h3>
        <p v-if="subtitle" class="sf-card__subtitle">
          {{ subtitle }}
        </p>
      </slot>
    </header>
    <div class="sf-card__body">
      <slot />
    </div>
    <footer v-if="$slots.footer" class="sf-card__footer">
      <slot name="footer" />
    </footer>
  </article>
</template>
