<script setup lang="ts">
import { renderLegalMarkdown } from '~/utils/legal/renderLegalMarkdown'

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

const title = computed(() => t(titleKey.value))
const body = computed(() => renderLegalMarkdown(
  stripLeadingMarkdownHeading(legalBody(props.kind, locale.value), title.value)
))

function stripLeadingMarkdownHeading(markdown: string, headingTitle: string) {
  const source = markdown.trimStart()
  const match = source.match(/^(#{1,6})\s+(.+?)(?:\r?\n+|$)/)
  if (!match) return markdown
  if (normalizeHeadingText(match[2] || '') !== normalizeHeadingText(headingTitle)) {
    return markdown
  }
  return source.slice(match[0].length).replace(/^\s+/, '')
}

function normalizeHeadingText(value: string) {
  return value.replace(/\s+/g, ' ').trim()
}
</script>

<template>
  <main class="sf-public-page legal-page" :data-sforum-island-body="`site.component.${kind}`">
    <div class="sf-public-page__container sf-public-page__container--narrow legal-page__container">
      <article class="sf-card legal-page__card">
        <div class="sf-card__body legal-page__body">
          <h1 class="legal-page__title">
            {{ title }}
          </h1>
          <div class="sf-prose legal-page__content" v-html="body" />
        </div>
      </article>
    </div>
  </main>
</template>

<style scoped>
.legal-page {
  padding: 2rem 1rem 3rem;
}

.legal-page__container {
  min-width: 0;
  margin: 0 auto;
}

.legal-page__card {
  overflow: hidden;
}

.legal-page__body {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.legal-page__title {
  margin: 0;
  color: var(--sf-public-text, var(--sf-fg));
  font-family: var(--sf-public-heading-font-family);
  font-size: var(--sf-public-page-title-size);
  font-weight: var(--sf-public-page-title-weight);
  line-height: var(--sf-public-page-title-line-height);
}

.legal-page__content {
  overflow-wrap: anywhere;
}

@media (max-width: 700px) {
  .legal-page__title {
    font-size: var(--sf-public-page-title-mobile-size);
  }
}
</style>
