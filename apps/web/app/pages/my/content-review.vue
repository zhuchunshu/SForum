<script setup lang="ts">
definePageMeta({ requiresAuth: true })
const { t } = useI18n()
const forumApi = useForumApi()
const { data, pending, error, refresh } = await useAsyncData('author-content-review', () => forumApi.listAuthorReviewItems(), { default: () => ({ items: [] }) })
</script>

<template>
  <SFPageOutlet page="forum.my.content_review">
  <main class="sf-public-page sf-public-page__container mx-auto w-full px-4 py-8 sm:px-6">
    <header class="flex flex-wrap items-start justify-between gap-3">
      <div><h1 class="text-2xl font-bold text-slate-950 dark:text-zinc-50">{{ t('moderation.authorStatus.title') }}</h1><p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">{{ t('moderation.authorStatus.description') }}</p></div>
      <UButton icon="i-lucide-rotate-cw" color="neutral" variant="subtle" :loading="pending" @click="refresh()">{{ t('admin.home.refresh') }}</UButton>
    </header>
    <SFAlert v-if="error" variant="danger" :title="t('moderation.authorStatus.loadFailed')" class="mt-5" />
    <div v-else-if="pending" class="mt-6"><SFSkeleton width="100%" height="140px" /></div>
    <AuthorContentReviewStatus v-else class="mt-6" :items="data.items" />
  </main>

  </SFPageOutlet>
</template>
