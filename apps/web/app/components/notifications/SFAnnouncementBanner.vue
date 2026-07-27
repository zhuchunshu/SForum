<script setup lang="ts">
import { useSiteChromeApi } from '~/composables/admin/useSiteChromeApi'
import type { SiteAnnouncement } from '~/composables/admin/useSiteChromeApi'

const { locale } = useI18n()
const chromeApi = useSiteChromeApi()

const STORAGE_KEY = 'sforum.dismissed-announcements'

const { data: announcements } = await useAsyncData('site-public-announcements', async () => {
  try {
    return await chromeApi.listPublicAnnouncements()
  } catch {
    return [] as SiteAnnouncement[]
  }
}, { default: () => [] as SiteAnnouncement[] })

const dismissedIds = ref<number[]>([])

onMounted(() => {
  if (!import.meta.client) {
    return
  }
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) {
      return
    }
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) {
      dismissedIds.value = parsed.filter((id): id is number => typeof id === 'number')
    }
  } catch {
    dismissedIds.value = []
  }
})

const isEnglish = computed(() => String(locale.value).toLowerCase().startsWith('en'))

const visibleAnnouncements = computed(() => {
  const dismissed = new Set(dismissedIds.value)
  return (announcements.value || []).filter((item) => {
    if (!item.dismissible) {
      return true
    }
    return !dismissed.has(item.id)
  })
})

function titleOf(item: SiteAnnouncement) {
  return isEnglish.value
    ? (item.titleEnUS || item.titleZhCN)
    : (item.titleZhCN || item.titleEnUS)
}

function bodyOf(item: SiteAnnouncement) {
  return isEnglish.value
    ? (item.bodyEnUS || item.bodyZhCN)
    : (item.bodyZhCN || item.bodyEnUS)
}

function dismiss(item: SiteAnnouncement) {
  if (!item.dismissible) {
    return
  }
  dismissedIds.value = [...new Set([...dismissedIds.value, item.id])]
  if (import.meta.client) {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(dismissedIds.value))
  }
}
</script>

<template>
  <div v-if="visibleAnnouncements.length" class="sf-announcements" role="region" aria-label="Announcements">
    <div
      v-for="item in visibleAnnouncements"
      :key="item.id"
      class="sf-announcement"
      :class="`sf-announcement--${item.style || 'info'}`"
    >
      <div class="sf-announcement__body">
        <p v-if="titleOf(item)" class="sf-announcement__title">
          <a
            v-if="item.href"
            :href="item.href"
            class="sf-announcement__link"
            :target="item.href.startsWith('http') ? '_blank' : undefined"
            :rel="item.href.startsWith('http') ? 'noopener noreferrer' : undefined"
          >
            {{ titleOf(item) }}
          </a>
          <span v-else>{{ titleOf(item) }}</span>
        </p>
        <p v-if="bodyOf(item)" class="sf-announcement__text">
          {{ bodyOf(item) }}
        </p>
      </div>
      <button
        v-if="item.dismissible"
        type="button"
        class="sf-announcement__dismiss"
        aria-label="Dismiss"
        @click="dismiss(item)"
      >
        <UIcon name="i-lucide-x" class="size-4" />
      </button>
    </div>
  </div>
</template>

<style scoped>
.sf-announcements {
  display: flex;
  flex-direction: column;
  gap: 0;
  border-bottom: 1px solid var(--sf-public-border, #e4e8ef);
}

.sf-announcement {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 20px;
  font-size: 13px;
  line-height: 1.45;
}

.sf-announcement--info {
  background: color-mix(in srgb, var(--sf-accent) 10%, transparent);
  color: var(--sf-public-text, #0f172a);
}

.sf-announcement--success {
  background: #ecfdf5;
  color: #065f46;
}

.sf-announcement--warning {
  background: #fffbeb;
  color: #92400e;
}

.sf-announcement--danger {
  background: #fef2f2;
  color: #991b1b;
}

.dark .sf-announcement--success {
  background: rgb(16 185 129 / 0.12);
  color: #6ee7b7;
}

.dark .sf-announcement--warning {
  background: rgb(245 158 11 / 0.12);
  color: #fcd34d;
}

.dark .sf-announcement--danger {
  background: rgb(239 68 68 / 0.12);
  color: #fca5a5;
}

.sf-announcement__title {
  margin: 0;
  font-weight: 700;
}

.sf-announcement__text {
  margin: 2px 0 0;
  opacity: 0.92;
}

.sf-announcement__link {
  color: inherit;
  text-decoration: underline;
  text-underline-offset: 2px;
}

.sf-announcement__dismiss {
  flex-shrink: 0;
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: inherit;
  cursor: pointer;
  opacity: 0.7;
}

.sf-announcement__dismiss:hover {
  opacity: 1;
  background: rgb(0 0 0 / 0.06);
}
</style>
