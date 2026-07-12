<script setup lang="ts">
const { t, locale } = useI18n()
const { legalBody, siteName } = useWebOptions()

const body = computed(() => legalBody('guidelines', locale.value))

useSeoMeta({
  title: () => `${t('legal.guidelinesTitle')} · ${siteName.value}`
})
</script>

<template>
  <main class="legal-page">
    <article class="legal-page__card">
      <h1 class="legal-page__title">
        {{ t('legal.guidelinesTitle') }}
      </h1>
      <div class="legal-page__body">
        {{ body }}
      </div>
    </article>
  </main>
</template>

<style scoped>
.legal-page {
  max-width: var(--sf-public-container, 1100px);
  margin: 0 auto;
  padding: 2rem 1rem 3rem;
}

.legal-page__card {
  border: 1px solid var(--sf-public-border, #e4e8ef);
  border-radius: 10px;
  background: var(--sf-public-surface, #fff);
  padding: 1.5rem 1.25rem;
}

.legal-page__title {
  margin: 0 0 1rem;
  font-size: 1.5rem;
  font-weight: 750;
  color: var(--sf-public-text, #0f172a);
}

.legal-page__body {
  white-space: pre-wrap;
  font-size: 0.9375rem;
  line-height: 1.7;
  color: var(--sf-public-text-muted, #334155);
}

.dark .legal-page__card {
  border-color: #27272a;
  background: #09090b;
}

.dark .legal-page__title {
  color: #f4f4f5;
}

.dark .legal-page__body {
  color: #a1a1aa;
}
</style>
