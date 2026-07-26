<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { isUnauthenticatedAuthError } from '~/composables/useAuthSession'
import {
  filterNotifications,
  groupNotificationsByDate,
  notificationFilters,
  notificationPresentation,
  unreadLoadedCount,
  type NotificationFilter,
  type NotificationPresentation
} from '~/utils/notificationsPresentation'

/**
 * 宿主 body 岛：forum.notifications。主题 L1 挂载；路由页仅 outlet + fail-closed 回退。
 */

const PAGE_LIMIT = 20

const { t, locale } = useI18n()
const localePath = useLocalePath()
const router = useRouter()
const toast = useToast()
const notifications = useNotifications()
const forumApi = useForumApi()
const { can } = usePermissions()
const { format: formatSiteDateTime, timezone } = useSiteDateTime()

const renderedAt = useState<number>('notification-inbox-rendered-at', () => Date.now())
const activeFilter = ref<NotificationFilter>('all')
const selectedId = ref<number | null>(null)
const actionError = ref('')
const loadingMoreError = ref('')
const loadingMore = ref(false)
const markingAll = ref(false)
const markingIds = ref(new Set<number>())
const mobileMenuOpen = useState<boolean>('forum-mobile-menu-open', () => false)
const mobileInfoOpen = useState<boolean>('forum-mobile-info-open', () => false)

const [
  notificationAsync,
  unreadAsync,
  categoryAsync
] = await Promise.all([
  useAsyncData(
    'notification-inbox',
    () => notifications.list(0, PAGE_LIMIT),
    { default: () => ({ items: [], hasMore: false }) }
  ),
  useAsyncData(
    'notification-inbox-unread-count',
    () => notifications.refreshUnreadCount(),
    { default: () => notifications.unreadCount.value }
  ),
  useAsyncData(
    'notification-inbox-categories',
    () => forumApi.listCategoryGroups(),
    { default: () => [] }
  )
])

const items = ref([...notificationAsync.data.value.items])
const hasMore = ref(notificationAsync.data.value.hasMore)

if (items.value.length > 0) {
  selectedId.value = items.value[0]?.id ?? null
}

watch(() => notificationAsync.data.value, (page) => {
  items.value = [...page.items]
  hasMore.value = page.hasMore
  selectedId.value = page.items[0]?.id ?? null
})

const categories = computed(() => categoryAsync.data.value.flatMap(group => group.categories || []))
const categoriesPending = computed(() => categoryAsync.pending.value)
const categoryTopicTotal = computed(() => categories.value.reduce((sum, category) => sum + category.topicCount, 0))
const canCreateTopic = computed(() => can(FORUM_PERMISSIONS.topicCreate))
const unreadTotal = computed(() => notifications.unreadCount.value)
const presentedItems = computed(() => items.value.map(notificationPresentation))
const filteredItems = computed(() => filterNotifications(presentedItems.value, activeFilter.value))
const visibleGroups = computed(() => groupNotificationsByDate(
  filteredItems.value,
  new Date(renderedAt.value),
  timezone.value,
  String(locale.value || 'zh-CN')
))
const selectedNotification = computed(() => {
  if (selectedId.value == null) return presentedItems.value[0] || null
  return presentedItems.value.find(item => item.id === selectedId.value) || presentedItems.value[0] || null
})
const selectedRawNotification = computed(() => items.value.find(item => item.id === selectedNotification.value?.id) || null)
const loadedUnread = computed(() => unreadLoadedCount(presentedItems.value))
const loadedTypeCounts = computed(() => {
  const counts = new Map<NotificationFilter, number>()
  counts.set('all', presentedItems.value.length)
  counts.set('unread', loadedUnread.value)
  for (const item of presentedItems.value) {
    counts.set(item.type, (counts.get(item.type) || 0) + 1)
  }
  return counts
})
const isInitialLoading = computed(() => notificationAsync.pending.value || unreadAsync.pending.value)
const initialError = computed(() => notificationAsync.error.value || unreadAsync.error.value)
const hasActionableUnread = computed(() => unreadTotal.value > 0 || loadedUnread.value > 0)
const filterScopeLabel = computed(() => t('notifications.filter.loadedScope', { count: presentedItems.value.length }))
const unreadSummaryLabel = computed(() => t('notifications.unreadSummary', { count: unreadTotal.value }))

function filterLabel(filter: NotificationFilter) {
  return t(`notifications.filter.${filter}`)
}

function filterIcon(filter: NotificationFilter) {
  const icons: Record<NotificationFilter, string> = {
    all: 'i-lucide-bell',
    unread: 'i-lucide-dot',
    reply: 'i-lucide-reply',
    mention: 'i-lucide-at-sign',
    moderation_approved: 'i-lucide-shield-check',
    moderation_rejected: 'i-lucide-shield-alert',
    admin_test: 'i-lucide-bell-ring'
  }
  return icons[filter]
}

function filterCount(filter: NotificationFilter) {
  return loadedTypeCounts.value.get(filter) || 0
}

function closeMobileDrawers() {
  mobileMenuOpen.value = false
  mobileInfoOpen.value = false
}

function selectFilter(filter: NotificationFilter) {
  activeFilter.value = filter
  closeMobileDrawers()
}

function notificationTime(item: NotificationPresentation) {
  return formatSiteDateTime(item.createdAt, { now: new Date(renderedAt.value) })
}

function actionLabel(item: NotificationPresentation | null) {
  if (!item || item.target.unavailable) return t('notifications.targetUnavailable')
  if (item.type === 'reply') return t('notifications.actions.viewReply')
  if (item.type === 'mention') return t('notifications.actions.viewMention')
  if (item.type === 'moderation_approved' || item.type === 'moderation_rejected') return t('notifications.actions.viewTarget')
  return t('notifications.actions.openTarget')
}

function translatedBody(item: NotificationPresentation) {
  return t(item.bodyKey, item.bodyParams)
}

function groupTitle(key: 'today' | 'earlier') {
  return t(`notifications.groups.${key}`)
}

function groupCount(count: number) {
  return t('notifications.groupCount', { count })
}

function setNotificationRead(id: number, readAt: string | undefined) {
  const item = items.value.find(entry => entry.id === id)
  if (item) {
    item.readAt = readAt
  }
}

async function markRead(item: NotificationPresentation | null) {
  if (!item || item.read || markingIds.value.has(item.id)) {
    return
  }

  const previousReadAt = selectedRawNotification.value?.readAt
  const previousUnreadCount = notifications.unreadCount.value
  const readAt = new Date().toISOString()
  markingIds.value = new Set(markingIds.value).add(item.id)
  actionError.value = ''
  setNotificationRead(item.id, readAt)

  try {
    await notifications.markRead(item.id)
  } catch (error) {
    setNotificationRead(item.id, previousReadAt)
    notifications.unreadCount.value = previousUnreadCount
    actionError.value = apiErrorMessage(error) || t('notifications.markReadFailed')
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: actionError.value,
      duration: 0
    })
  } finally {
    const next = new Set(markingIds.value)
    next.delete(item.id)
    markingIds.value = next
  }
}

async function selectNotification(item: NotificationPresentation) {
  selectedId.value = item.id
  await markRead(item)
}

async function markAllRead() {
  if (!hasActionableUnread.value || markingAll.value) {
    return
  }

  const previousItems = items.value.map(item => ({ id: item.id, readAt: item.readAt }))
  const previousUnreadCount = notifications.unreadCount.value
  const readAt = new Date().toISOString()
  markingAll.value = true
  actionError.value = ''
  items.value.forEach(item => {
    item.readAt ||= readAt
  })

  try {
    await notifications.markAllRead()
    toast.add({
      color: 'primary',
      icon: 'i-lucide-check-check',
      title: t('notifications.allRead'),
      duration: 10000
    })
  } catch (error) {
    for (const previous of previousItems) {
      setNotificationRead(previous.id, previous.readAt)
    }
    notifications.unreadCount.value = previousUnreadCount
    actionError.value = apiErrorMessage(error) || t('notifications.markAllReadFailed')
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: actionError.value,
      duration: 0
    })
  } finally {
    markingAll.value = false
  }
}

async function loadMore() {
  if (!hasMore.value || loadingMore.value || items.value.length === 0) {
    return
  }
  loadingMore.value = true
  loadingMoreError.value = ''
  try {
    const beforeId = items.value[items.value.length - 1]?.id || 0
    const page = await notifications.list(beforeId, PAGE_LIMIT)
    const seen = new Set(items.value.map(item => item.id))
    items.value = [...items.value, ...page.items.filter(item => !seen.has(item.id))]
    hasMore.value = page.hasMore
  } catch (error) {
    loadingMoreError.value = apiErrorMessage(error) || t('notifications.loadMoreFailed')
  } finally {
    loadingMore.value = false
  }
}

async function openSelectedTarget() {
  const item = selectedNotification.value
  if (!item || item.target.unavailable) {
    actionError.value = t('notifications.targetUnavailableHelp')
    return
  }
  await markRead(item)
  closeMobileDrawers()
  await router.push(localePath(item.target.path))
}
</script>

<template>
  <main
    class="sforum-notifications"
    data-sforum-island-body="forum.component.notifications"
    data-layout="fullwidth-3col"
  >
    <div class="sforum-notifications__layout">
      <div class="sforum-notifications__sidebar sforum-home__sidebar">
        <SFHomeNavigation
          desktop-only
          navigation-mode="route"
          :categories="categories"
          :total-topics="categoryTopicTotal"
          :pending="categoriesPending"
          :can-create-topic="canCreateTopic"
          :show-categories="false"
        >
          <template #after-navigation>
            <nav class="sforum-notifications__type-nav" :aria-label="t('notifications.filter.aria')">
              <div class="sforum-notifications__rail-label">{{ t('notifications.filter.title') }}</div>
              <button
                v-for="filter in notificationFilters"
                :key="filter"
                type="button"
                class="sforum-notifications__rail-link"
                :class="{ 'is-active': activeFilter === filter }"
                :aria-pressed="activeFilter === filter"
                @click="selectFilter(filter)"
              >
                <span class="sforum-notifications__rail-link-main">
                  <UIcon :name="filterIcon(filter)" class="size-[18px]" aria-hidden="true" />
                  {{ filterLabel(filter) }}
                </span>
                <span class="sforum-notifications__rail-count">{{ filterCount(filter) }}</span>
              </button>
              <p class="sforum-notifications__filter-scope">{{ filterScopeLabel }}</p>
            </nav>
          </template>
        </SFHomeNavigation>
      </div>

      <section class="sforum-notifications__main" aria-labelledby="notification-page-title">
        <div class="sforum-notifications__mobile-nav">
          <SFHomeNavigation
            mobile-only
            navigation-mode="route"
            :categories="categories"
            :total-topics="categoryTopicTotal"
            :pending="categoriesPending"
            :can-create-topic="canCreateTopic"
          />
        </div>

        <SFRegionOutlet page="forum.notifications" region="content_before" />

        <header class="sforum-notifications__head">
          <div class="sforum-notifications__head-copy">
            <h1 id="notification-page-title">{{ t('notifications.title') }}</h1>
            <p>{{ unreadSummaryLabel }}</p>
          </div>
          <div class="sforum-notifications__head-actions">
            <button
              type="button"
              class="sforum-notifications__text-button"
              :disabled="!hasActionableUnread || markingAll"
              @click="markAllRead"
            >
              <UIcon name="i-lucide-check-check" class="size-4" aria-hidden="true" />
              <span>{{ markingAll ? t('notifications.markingAllRead') : t('notifications.markAllRead') }}</span>
            </button>
            <button
              type="button"
              class="sforum-notifications__icon-button sforum-notifications__desktop-hidden"
              :aria-label="t('notifications.detail.open')"
              @click="mobileInfoOpen = true"
            >
              <UIcon name="i-lucide-panel-right" class="size-[18px]" aria-hidden="true" />
            </button>
          </div>
        </header>

        <div class="sforum-notifications__filter-strip" :aria-label="t('notifications.filter.aria')">
          <button
            v-for="filter in notificationFilters"
            :key="filter"
            type="button"
            class="sforum-notifications__filter-button"
            :class="{ 'is-active': activeFilter === filter }"
            :aria-pressed="activeFilter === filter"
            @click="selectFilter(filter)"
          >
            {{ filterLabel(filter) }}
          </button>
        </div>
        <p class="sforum-notifications__filter-note">{{ filterScopeLabel }}</p>

        <SFAlert
          v-if="actionError"
          class="sforum-notifications__alert"
          variant="danger"
          :title="actionError"
          closable
          @close="actionError = ''"
        />

        <SFAlert
          v-if="initialError"
          class="sforum-notifications__alert"
          variant="danger"
          :title="isUnauthenticatedAuthError(initialError) ? t('notifications.authRequired') : t('notifications.loadFailed')"
        />

        <div v-else-if="isInitialLoading" class="sforum-notifications__loading" aria-busy="true">
          <SFSkeleton v-for="item in 5" :key="item" avatar :lines="2" />
        </div>

        <SFEmptyState
          v-else-if="!items.length"
          class="sforum-notifications__empty"
          icon="i-lucide-bell-off"
          :title="t('notifications.empty')"
          :description="t('notifications.emptyDescription')"
        />

        <SFEmptyState
          v-else-if="!filteredItems.length"
          class="sforum-notifications__empty"
          icon="i-lucide-filter-x"
          :title="t('notifications.filter.empty')"
          :description="t('notifications.filter.emptyDescription', { count: presentedItems.length })"
        />

        <div v-else class="sforum-notifications__stream">
          <section
            v-for="group in visibleGroups"
            :key="group.key"
            class="sforum-notifications__group"
            :aria-labelledby="`notification-group-${group.key}`"
          >
            <div class="sforum-notifications__group-head">
              <span :id="`notification-group-${group.key}`">{{ groupTitle(group.key) }}</span>
              <span>{{ groupCount(group.items.length) }}</span>
            </div>

            <div class="sforum-notifications__list">
              <button
                v-for="item in group.items"
                :key="item.id"
                type="button"
                class="sforum-notifications__row"
                :class="{ 'is-unread': !item.read, 'is-selected': selectedNotification?.id === item.id }"
                :aria-pressed="selectedNotification?.id === item.id"
                @click="selectNotification(item)"
              >
                <SFAvatar :name="filterLabel(item.type)" :alt="filterLabel(item.type)" size="list" />
                <span class="sforum-notifications__row-body">
                  <span class="sforum-notifications__row-lead">
                    {{ t(item.titleKey) }}
                    <span v-if="!item.read" class="sforum-notifications__unread-text">{{ t('notifications.unread') }}</span>
                  </span>
                  <span class="sforum-notifications__row-excerpt">{{ translatedBody(item) }}</span>
                  <span class="sforum-notifications__row-meta">
                    <span class="sforum-notifications__type-label" :data-type="item.type">
                      <UIcon :name="item.icon" class="size-3.5" aria-hidden="true" />
                      {{ t(item.typeLabelKey) }}
                    </span>
                    <span v-if="item.targetLabel">{{ item.targetLabel }}</span>
                    <span v-else>{{ t('notifications.targetUnavailable') }}</span>
                  </span>
                </span>
                <time class="sforum-notifications__row-time" :datetime="item.createdAt">{{ notificationTime(item) }}</time>
              </button>
            </div>
          </section>

          <div v-if="items.length && !initialError" class="sforum-notifications__more">
            <button
              v-if="hasMore"
              type="button"
              class="sforum-notifications__secondary-button"
              :disabled="loadingMore"
              @click="loadMore"
            >
              <UIcon name="i-lucide-chevron-down" class="size-4" aria-hidden="true" />
              {{ loadingMore ? t('notifications.loadingMore') : t('notifications.loadMore') }}
            </button>
            <span v-else>{{ t('notifications.end') }}</span>
            <div v-if="loadingMoreError" class="sforum-notifications__inline-error" role="alert">
              <span>{{ loadingMoreError }}</span>
              <button type="button" @click="loadMore">{{ t('home.feed.retryLoadMore') }}</button>
            </div>
          </div>
        </div>

        <SFRegionOutlet page="forum.notifications" region="content_after" />

        <SFContentColumnFooter />
      </section>

      <aside class="sforum-notifications__right" :aria-label="t('notifications.detail.aria')">
        <section class="sforum-notifications__rail-section">
          <div class="sforum-notifications__rail-head">
            <h2>{{ t('notifications.inbox') }}</h2>
            <span>{{ t('notifications.unreadAuthority') }}</span>
          </div>
          <div class="sforum-notifications__unread-summary">
            <strong>{{ unreadTotal }}</strong>
            <span>{{ t('notifications.unreadCountLabel') }}</span>
          </div>
          <p class="sforum-notifications__rail-help">{{ t('notifications.unreadSource') }}</p>
        </section>

        <section class="sforum-notifications__rail-section">
          <div class="sforum-notifications__rail-head">
            <h2>{{ t('notifications.detail.title') }}</h2>
            <span v-if="selectedNotification">{{ t(selectedNotification.typeLabelKey) }}</span>
          </div>

          <div v-if="selectedNotification" class="sforum-notifications__detail">
            <div class="sforum-notifications__detail-person">
              <SFAvatar :name="filterLabel(selectedNotification.type)" :alt="filterLabel(selectedNotification.type)" size="sm" />
              <div>
                <strong>{{ t(selectedNotification.titleKey) }}</strong>
                <time :datetime="selectedNotification.createdAt">{{ notificationTime(selectedNotification) }}</time>
              </div>
            </div>

            <h3>{{ selectedNotification.targetLabel || t('notifications.targetUnavailable') }}</h3>
            <p>{{ translatedBody(selectedNotification) }}</p>
            <p v-if="selectedNotification.detailMeta && selectedNotification.detailMeta !== selectedNotification.targetLabel" class="sforum-notifications__detail-note">
              {{ selectedNotification.detailMeta }}
            </p>

            <div class="sforum-notifications__detail-actions">
              <button
                type="button"
                class="sforum-notifications__primary-button"
                :disabled="selectedNotification.target.unavailable"
                @click="openSelectedTarget"
              >
                {{ actionLabel(selectedNotification) }}
                <UIcon name="i-lucide-arrow-right" class="size-4" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="sforum-notifications__icon-button"
                :disabled="selectedNotification.read || markingIds.has(selectedNotification.id)"
                :aria-label="t('notifications.actions.markRead')"
                @click="markRead(selectedNotification)"
              >
                <UIcon name="i-lucide-check" class="size-[18px]" aria-hidden="true" />
              </button>
            </div>
          </div>

          <div v-else class="sforum-notifications__detail-empty">
            <UIcon name="i-lucide-bell" class="size-7" aria-hidden="true" />
            <p>{{ t('notifications.detail.empty') }}</p>
          </div>
        </section>

        <section class="sforum-notifications__rail-section">
          <div class="sforum-notifications__rail-head">
            <h2>{{ t('notifications.loadedOverview') }}</h2>
            <span>{{ t('notifications.loadedOnly') }}</span>
          </div>
          <dl class="sforum-notifications__loaded-stats">
            <div>
              <dt>{{ t('notifications.loadedTotal') }}</dt>
              <dd>{{ presentedItems.length }}</dd>
            </div>
            <div>
              <dt>{{ t('notifications.loadedUnread') }}</dt>
              <dd>{{ loadedUnread }}</dd>
            </div>
          </dl>
        </section>
      </aside>
    </div>

    <button
      v-if="mobileMenuOpen || mobileInfoOpen"
      type="button"
      class="sforum-mobile-drawer__backdrop"
      :aria-label="t('common.close')"
      @click="closeMobileDrawers"
    />

    <aside v-if="mobileMenuOpen" class="sforum-mobile-drawer sforum-mobile-drawer--left">
      <header class="sforum-mobile-drawer__head">
        <strong>{{ t('home.sidebar.drawerTitle') }}</strong>
        <button type="button" :aria-label="t('common.close')" @click="closeMobileDrawers">
          <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
        </button>
      </header>
      <SFHomeNavigation
        desktop-only
        navigation-mode="route"
        :categories="categories"
        :total-topics="categoryTopicTotal"
        :pending="categoriesPending"
        :can-create-topic="canCreateTopic"
      />
    </aside>

    <aside v-if="mobileInfoOpen" class="sforum-mobile-drawer sforum-mobile-drawer--right">
      <header class="sforum-mobile-drawer__head">
        <strong>{{ t('notifications.detail.title') }}</strong>
        <button type="button" :aria-label="t('common.close')" @click="closeMobileDrawers">
          <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
        </button>
      </header>
      <aside class="sforum-notifications__right sforum-notifications__right--drawer" :aria-label="t('notifications.detail.aria')">
        <section class="sforum-notifications__rail-section">
          <div class="sforum-notifications__rail-head">
            <h2>{{ t('notifications.inbox') }}</h2>
            <span>{{ t('notifications.unreadAuthority') }}</span>
          </div>
          <div class="sforum-notifications__unread-summary">
            <strong>{{ unreadTotal }}</strong>
            <span>{{ t('notifications.unreadCountLabel') }}</span>
          </div>
          <p class="sforum-notifications__rail-help">{{ t('notifications.unreadSource') }}</p>
        </section>
        <section class="sforum-notifications__rail-section">
          <div class="sforum-notifications__rail-head">
            <h2>{{ t('notifications.detail.title') }}</h2>
            <span v-if="selectedNotification">{{ t(selectedNotification.typeLabelKey) }}</span>
          </div>
          <div v-if="selectedNotification" class="sforum-notifications__detail">
            <div class="sforum-notifications__detail-person">
              <SFAvatar :name="filterLabel(selectedNotification.type)" :alt="filterLabel(selectedNotification.type)" size="sm" />
              <div>
                <strong>{{ t(selectedNotification.titleKey) }}</strong>
                <time :datetime="selectedNotification.createdAt">{{ notificationTime(selectedNotification) }}</time>
              </div>
            </div>
            <h3>{{ selectedNotification.targetLabel || t('notifications.targetUnavailable') }}</h3>
            <p>{{ translatedBody(selectedNotification) }}</p>
            <div class="sforum-notifications__detail-actions">
              <button
                type="button"
                class="sforum-notifications__primary-button"
                :disabled="selectedNotification.target.unavailable"
                @click="openSelectedTarget"
              >
                {{ actionLabel(selectedNotification) }}
                <UIcon name="i-lucide-arrow-right" class="size-4" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="sforum-notifications__icon-button"
                :disabled="selectedNotification.read || markingIds.has(selectedNotification.id)"
                :aria-label="t('notifications.actions.markRead')"
                @click="markRead(selectedNotification)"
              >
                <UIcon name="i-lucide-check" class="size-[18px]" aria-hidden="true" />
              </button>
            </div>
          </div>
          <div v-else class="sforum-notifications__detail-empty">
            <UIcon name="i-lucide-bell" class="size-7" aria-hidden="true" />
            <p>{{ t('notifications.detail.empty') }}</p>
          </div>
        </section>
      </aside>
    </aside>
  </main>
</template>

<style scoped src="./SFNotificationsPage.css"></style>
