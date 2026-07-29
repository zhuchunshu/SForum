<script setup lang="ts">
type PublicPageHeaderVariant = 'page' | 'section'

const props = withDefaults(defineProps<{
  titleId: string
  title: string
  subtitle?: string
  level?: 1 | 2
  variant?: PublicPageHeaderVariant
}>(), {
  subtitle: '',
  level: 1,
  variant: 'page'
})
</script>

<template>
  <header
    class="sf-public-page-header"
    :class="`sf-public-page-header--${props.variant}`"
  >
    <div class="sf-public-page-header__copy">
      <slot name="eyebrow" />
      <component
        :is="props.level === 1 ? 'h1' : 'h2'"
        :id="props.titleId"
        class="sf-public-page-header__title"
      >
        {{ props.title }}
      </component>
      <p
        v-if="props.subtitle || $slots.subtitle"
        class="sf-public-page-header__subtitle"
      >
        <slot name="subtitle">{{ props.subtitle }}</slot>
      </p>
      <slot name="meta" />
    </div>
    <slot name="aside" />
  </header>
</template>

<style scoped>
.sf-public-page-header__copy {
  min-width: 0;
}

.sf-public-page-header__title {
  margin: 0;
  overflow-wrap: anywhere;
  color: var(--sf-public-text);
  font-family: var(--sf-public-heading-font-family);
  letter-spacing: 0;
}

.sf-public-page-header--page .sf-public-page-header__title {
  font-size: var(--sf-public-page-title-size);
  font-weight: var(--sf-public-page-title-weight);
  line-height: var(--sf-public-page-title-line-height);
}

.sf-public-page-header--section .sf-public-page-header__title {
  font-size: var(--sf-public-section-title-size);
  font-weight: var(--sf-public-section-title-weight);
  line-height: var(--sf-public-section-title-line-height);
}

.sf-public-page-header__subtitle {
  max-width: 70ch;
  color: var(--sf-public-text-muted);
}

.sf-public-page-header--page .sf-public-page-header__subtitle {
  margin: 6px 0 0;
  font-size: var(--sf-public-page-subtitle-size);
  line-height: var(--sf-public-page-subtitle-line-height);
}

.sf-public-page-header--section .sf-public-page-header__subtitle {
  margin: 4px 0 0;
  font-size: var(--sf-public-section-subtitle-size);
  line-height: var(--sf-public-section-subtitle-line-height);
}

@media (max-width: 700px) {
  .sf-public-page-header--page .sf-public-page-header__title {
    font-size: var(--sf-public-page-title-mobile-size);
  }
}
</style>
