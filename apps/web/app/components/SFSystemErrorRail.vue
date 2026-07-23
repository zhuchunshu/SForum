<script setup lang="ts">
const { t, locale } = useI18n()
const localePath = useLocalePath()
const {
  hotTopics,
  activeTags,
  totalTopics,
  totalReplies,
  categories,
  topicTo
} = useSystemErrorRecoveryData()

const commonLinks = computed(() => [
  { key: 'home', label: t('errors.page.rail.home'), to: localePath('/'), icon: 'i-lucide-house' },
  { key: 'categories', label: t('errors.page.rail.categories'), to: localePath('/categories'), icon: 'i-lucide-folder-tree' },
  { key: 'tags', label: t('errors.page.rail.tags'), to: localePath('/tags'), icon: 'i-lucide-tags' }
])
const helpLinks = computed(() => [
  { key: 'guidelines', label: t('footer.guidelines'), to: localePath('/guidelines') },
  { key: 'terms', label: t('footer.terms'), to: localePath('/terms') },
  { key: 'privacy', label: t('footer.privacy'), to: localePath('/privacy') }
])

function formatCount(value: number) {
  const n = Math.max(0, Number.isFinite(value) ? Math.floor(value) : 0)
  return new Intl.NumberFormat(locale.value).format(n)
}
</script>

<template>
  <aside class="sforum-system-error__rail" :aria-label="t('errors.page.rail.ariaLabel')" data-system-error-region="rail">
    <section class="sforum-system-error__rail-card">
      <h2 class="sforum-system-error__rail-title">{{ t('errors.page.rail.common') }}</h2>
      <div class="sforum-system-error__rail-links">
        <NuxtLink v-for="link in commonLinks" :key="link.key" :to="link.to">
          <UIcon :name="link.icon" aria-hidden="true" />
          <span>{{ link.label }}</span>
        </NuxtLink>
      </div>
    </section>

    <section class="sforum-system-error__rail-card">
      <h2 class="sforum-system-error__rail-title">{{ t('home.sidebar.hotThreads') }}</h2>
      <ol v-if="hotTopics.length" class="sforum-system-error__hot-list">
        <li v-for="(topic, index) in hotTopics" :key="topic.id">
          <span class="sforum-system-error__rank">{{ String(index + 1).padStart(2, '0') }}</span>
          <NuxtLink :to="localePath(topicTo(topic))" :prefetch="false">
            {{ topic.title }}
          </NuxtLink>
          <span>{{ formatCount(topic.commentCount) }}</span>
        </li>
      </ol>
      <p v-else class="sforum-system-error__rail-empty">{{ t('home.rightRail.hotEmpty') }}</p>
    </section>

    <section class="sforum-system-error__rail-card">
      <h2 class="sforum-system-error__rail-title">{{ t('errors.page.rail.needHelp') }}</h2>
      <div class="sforum-system-error__help-links">
        <NuxtLink v-for="link in helpLinks" :key="link.key" :to="link.to">
          {{ link.label }}
        </NuxtLink>
      </div>
      <p class="sforum-system-error__rail-meta">
        {{ t('errors.page.rail.stats', {
          topics: formatCount(totalTopics),
          replies: formatCount(totalReplies),
          categories: formatCount(categories.length),
          tags: formatCount(activeTags.length)
        }) }}
      </p>
    </section>
  </aside>
</template>
