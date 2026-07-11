<script setup lang="ts">
definePageMeta({ requiresAuth: true })
const { t } = useI18n()
const localePath = useLocalePath()
const toast = useToast()
const notifications = useNotifications()
const { data, pending, error, refresh } = await useAsyncData('notification-inbox', () => notifications.list(), { default: () => ({ items: [], hasMore: false }) })
function itemLabel(type: string) { return t(`notifications.types.${type}`) }
function itemLink(item: { targetType: string, targetId: number, payload: Record<string, unknown> }) { const topicId = Number(item.payload.topicId || 0); return topicId > 0 ? localePath(`/t/${topicId}`) : item.targetType === 'topic' ? localePath(`/t/${item.targetId}`) : localePath('/my/content-review') }
async function markRead(id: number) { await notifications.markRead(id); const item = data.value.items.find(entry => entry.id === id); if (item) item.readAt = new Date().toISOString() }
async function markAllRead() { await notifications.markAllRead(); data.value.items.forEach(item => { item.readAt ||= new Date().toISOString() }); toast.add({ color: 'success', icon: 'i-lucide-check', title: t('notifications.allRead'), duration: 10000 }) }
</script>

<template>
  <main class="mx-auto w-full max-w-3xl px-4 py-8 sm:px-6">
    <header class="flex items-center justify-between gap-3"><div><h1 class="text-2xl font-bold text-slate-950 dark:text-zinc-50">{{ t('notifications.title') }}</h1><p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">{{ t('notifications.description') }}</p></div><UButton icon="i-lucide-check-check" color="neutral" variant="subtle" @click="markAllRead">{{ t('notifications.markAllRead') }}</UButton></header>
    <SFAlert v-if="error" class="mt-5" variant="danger" :title="t('notifications.loadFailed')" />
    <div v-else-if="pending" class="mt-6"><SFSkeleton width="100%" height="160px" /></div>
    <SFEmptyState v-else-if="!data.items.length" class="mt-6" icon="i-lucide-bell-off" :title="t('notifications.empty')" />
    <div v-else class="mt-6 divide-y divide-slate-200 border-y border-slate-200 dark:divide-zinc-800 dark:border-zinc-800">
      <NuxtLink v-for="item in data.items" :key="item.id" :to="itemLink(item)" class="flex items-start gap-3 px-2 py-4 hover:bg-slate-50 dark:hover:bg-zinc-900" @click="!item.readAt && markRead(item.id)">
        <span class="mt-1 size-2 shrink-0 rounded-full" :class="item.readAt ? 'bg-slate-300 dark:bg-zinc-700' : 'bg-[var(--sf-accent)]'" />
        <div class="min-w-0"><p class="font-medium text-slate-900 dark:text-zinc-100">{{ itemLabel(item.type) }}</p><time class="text-xs text-slate-500">{{ new Date(item.createdAt).toLocaleString() }}</time></div>
      </NuxtLink>
    </div>
  </main>
</template>
