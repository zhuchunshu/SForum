<script setup lang="ts">
/**
 * 法律文档 body 岛（terms / privacy / guidelines）。
 * 主题 L1 经 site.component.* 挂载；路由页仅 SEO + fail-closed 回退。
 */
const props = defineProps<{
  kind: 'terms' | 'privacy' | 'guidelines'
}>()

const { t, locale } = useI18n()
const { legalBody } = useWebOptions()

const titleKey = computed(() => {
  switch (props.kind) {
    case 'privacy':
      return 'legal.privacyTitle'
    case 'guidelines':
      return 'legal.guidelinesTitle'
    default:
      return 'legal.termsTitle'
  }
})

const body = computed(() => legalBody(props.kind, locale.value))
</script>

<template>
  <main class="legal-page" :data-sforum-island-body="`site.component.${kind}`">
    <article class="legal-page__card">
      <h1 class="legal-page__title">
        {{ t(titleKey) }}
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
