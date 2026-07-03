<script setup lang="ts">
const { t } = useI18n()
const localePath = useLocalePath()

useSeoMeta({
  title: t('home.metaTitle'),
  description: t('home.metaDescription'),
  ogTitle: t('home.metaTitle'),
  ogDescription: t('home.metaDescription')
})
</script>

<template>
  <main class="page-shell">
    <section class="forum-board" aria-labelledby="home-title">
      <div class="board-header">
        <div>
          <p class="eyebrow">
            {{ t('home.eyebrow') }}
          </p>
          <h1 id="home-title">
            {{ t('home.title') }}
          </h1>
          <p class="intro">
            {{ t('home.intro') }}
          </p>
        </div>

        <nav class="language-switch" :aria-label="t('home.languageNav')">
          <NuxtLink :to="localePath('/', 'zh-CN')">
            简体中文
          </NuxtLink>
          <NuxtLink :to="localePath('/', 'en')">
            English
          </NuxtLink>
        </nav>
      </div>

      <div class="status-grid">
        <article>
          <span>{{ t('home.cards.frontend.label') }}</span>
          <strong>{{ t('home.cards.frontend.value') }}</strong>
        </article>
        <article>
          <span>{{ t('home.cards.backend.label') }}</span>
          <strong>{{ t('home.cards.backend.value') }}</strong>
        </article>
        <article>
          <span>{{ t('home.cards.locale.label') }}</span>
          <strong>{{ t('home.cards.locale.value') }}</strong>
        </article>
      </div>
    </section>
  </main>
</template>
