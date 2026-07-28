<script setup lang="ts">
import { useNotifications, type NotificationPreviewContent } from '~/composables/notifications/useNotifications'
import { FORUM_PERMISSIONS, usePermissions } from '~/composables/identity/usePermissions'
import { useForumApi } from '~/composables/forum/useForumApi'
import { apiErrorMessage, apiErrorStatusCode } from '~/composables/useApiClient'
import SFHomeNavigation from '~/components/forum/SFHomeNavigation.vue'
import { notificationPresentation } from '~/utils/notifications/notificationsPresentation'

/** forum.notification.show：独立路由详情岛，只呈现 Host 已重新授权的当前摘要。 */

const { t } = useI18n()
const localePath = useLocalePath()
const route = useRoute()
const router = useRouter()
const toast = useToast()
const notifications = useNotifications()
const forumApi = useForumApi()
const { can } = usePermissions()
const { format: formatSiteDateTime } = useSiteDateTime()

const notificationId = Number.parseInt(String(route.params.notificationId || ''), 10)
if (!Number.isSafeInteger(notificationId) || notificationId <= 0) {
  throw createError({ statusCode: 404, statusMessage: 'Notification not found' })
}

const [detailAsync, unreadAsync, categoryAsync] = await Promise.all([
  useAsyncData(`notification-detail:${notificationId}`, () => notifications.get(notificationId)),
  useAsyncData('notification-detail-unread-count', () => notifications.refreshUnreadCount(), {
    default: () => notifications.unreadCount.value
  }),
  useAsyncData('notification-detail-categories', () => forumApi.listCategoryGroups(), {
    default: () => []
  })
])

if (detailAsync.error.value && apiErrorStatusCode(detailAsync.error.value) === 404) {
  throw createError({ statusCode: 404, statusMessage: 'Notification not found' })
}

const detail = detailAsync.data
const presented = computed(() => detail.value ? notificationPresentation(detail.value) : null)
const preview = computed(() => detail.value?.preview)
const categories = computed(() => categoryAsync.data.value.flatMap(group => group.categories || []))
const categoryTopicTotal = computed(() => categories.value.reduce((sum, category) => sum + category.topicCount, 0))
const canCreateTopic = computed(() => can(FORUM_PERMISSIONS.topicCreate))
const unreadTotal = computed(() => notifications.unreadCount.value)
const markingRead = ref(false)
const actionError = ref('')
const mobileMenuOpen = useState<boolean>('forum-mobile-menu-open', () => false)
const mobileInfoOpen = useState<boolean>('forum-mobile-info-open', () => false)

const contentHeading = computed(() => {
  if (detail.value?.type === 'reply') return t('notifications.detailPage.replyContent')
  if (detail.value?.type === 'mention') return t('notifications.detailPage.mentionContent')
  return t('notifications.detailPage.targetContent')
})

const actionLabel = computed(() => {
  if (detail.value?.type === 'reply') return t('notifications.actions.viewReply')
  if (detail.value?.type === 'mention') return t('notifications.actions.viewMention')
  return t('notifications.actions.viewTarget')
})

function authorName(content?: NotificationPreviewContent) {
  return content?.author?.displayName || content?.author?.username || t('notifications.detailPage.unknownAuthor')
}

function contextHeading(content?: NotificationPreviewContent) {
  return content?.type === 'comment'
    ? t('notifications.detailPage.repliedComment')
    : t('notifications.detailPage.originalTopic')
}

function notificationTime(value: string) {
  return formatSiteDateTime(value, { now: new Date() })
}

function closeMobileDrawers() {
  mobileMenuOpen.value = false
  mobileInfoOpen.value = false
}

async function openTarget() {
  if (!presented.value || presented.value.target.unavailable) return
  closeMobileDrawers()
  await router.push(localePath(presented.value.target.path))
}

onMounted(async () => {
  if (!detail.value || detail.value.readAt || markingRead.value) return
  markingRead.value = true
  try {
    await notifications.markRead(detail.value.id)
    detail.value.readAt = new Date().toISOString()
  } catch (error) {
    actionError.value = apiErrorMessage(error) || t('notifications.markReadFailed')
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: actionError.value,
      duration: 0
    })
  } finally {
    markingRead.value = false
  }
})

useHead(() => ({ title: preview.value?.topicTitle || t('notifications.detailPage.title') }))
</script>

<template>
  <main
    class="sforum-notifications sforum-notification-detail"
    data-sforum-island-body="forum.component.notification_detail"
    data-layout="fullwidth-3col"
  >
    <div class="sforum-notifications__layout">
      <div class="sforum-notifications__sidebar sforum-home__sidebar">
        <SFHomeNavigation
          desktop-only
          navigation-mode="route"
          :categories="categories"
          :total-topics="categoryTopicTotal"
          :pending="categoryAsync.pending.value"
          :can-create-topic="canCreateTopic"
          :show-categories="false"
        />
      </div>

      <section class="sforum-notifications__main" aria-labelledby="notification-detail-title">
        <div class="sforum-notifications__mobile-nav">
          <SFHomeNavigation
            mobile-only
            navigation-mode="route"
            :categories="categories"
            :total-topics="categoryTopicTotal"
            :pending="categoryAsync.pending.value"
            :can-create-topic="canCreateTopic"
          />
        </div>

        <header class="sforum-notifications__head sforum-notification-detail__head">
          <div class="sforum-notifications__head-copy">
            <NuxtLink class="sforum-notification-detail__back" :to="localePath('/notifications')">
              <UIcon name="i-lucide-arrow-left" class="size-4" aria-hidden="true" />
              {{ t('notifications.detailPage.back') }}
            </NuxtLink>
            <h1 id="notification-detail-title">{{ t('notifications.detailPage.title') }}</h1>
            <p v-if="presented">{{ t(presented.titleKey) }}</p>
          </div>
          <button
            type="button"
            class="sforum-notifications__icon-button sforum-notifications__desktop-hidden"
            :aria-label="t('notifications.detail.open')"
            @click="mobileInfoOpen = true"
          >
            <UIcon name="i-lucide-panel-right" class="size-[18px]" aria-hidden="true" />
          </button>
        </header>

        <SFAlert
          v-if="actionError"
          class="sforum-notifications__alert"
          variant="danger"
          :title="actionError"
          closable
          @close="actionError = ''"
        />

        <div v-if="detailAsync.pending.value || unreadAsync.pending.value" class="sforum-notifications__loading" aria-busy="true">
          <SFSkeleton avatar :lines="3" />
          <SFSkeleton :lines="4" />
        </div>

        <SFAlert
          v-else-if="detailAsync.error.value"
          class="sforum-notifications__alert"
          variant="danger"
          :title="apiErrorMessage(detailAsync.error.value) || t('notifications.loadFailed')"
        />

        <article v-else-if="detail && presented" class="sforum-notification-detail__reader">
          <div class="sforum-notification-detail__event">
            <SFAvatar
              :name="authorName(preview?.content)"
              :avatar="preview?.content.author?.avatar"
              size="sm"
            />
            <div>
              <strong>{{ t(presented.titleKey) }}</strong>
              <time :datetime="detail.createdAt">{{ notificationTime(detail.createdAt) }}</time>
            </div>
            <span class="sforum-notifications__type-label" :data-type="detail.type">
              <UIcon :name="presented.icon" class="size-3.5" aria-hidden="true" />
              {{ t(presented.typeLabelKey) }}
            </span>
          </div>

          <template v-if="preview">
            <h2>{{ preview.topicTitle }}</h2>

            <section class="sforum-notification-detail__content-block" :aria-labelledby="`notification-content-${detail.id}`">
              <div class="sforum-notification-detail__section-head">
                <h3 :id="`notification-content-${detail.id}`">{{ contentHeading }}</h3>
                <span>{{ authorName(preview.content) }}</span>
              </div>
              <blockquote>{{ preview.content.excerpt || t('notifications.detailPage.emptyExcerpt') }}</blockquote>
            </section>

            <section v-if="preview.context" class="sforum-notification-detail__content-block is-context">
              <div class="sforum-notification-detail__section-head">
                <h3>{{ contextHeading(preview.context) }}</h3>
                <span>{{ authorName(preview.context) }}</span>
              </div>
              <blockquote>{{ preview.context.excerpt || t('notifications.detailPage.emptyExcerpt') }}</blockquote>
            </section>

            <div class="sforum-notification-detail__actions">
              <button type="button" class="sforum-notifications__primary-button" @click="openTarget">
                {{ actionLabel }}
                <UIcon name="i-lucide-arrow-right" class="size-4" aria-hidden="true" />
              </button>
            </div>
          </template>

          <SFAlert
            v-else
            variant="info"
            :title="t('notifications.targetUnavailable')"
            :description="t('notifications.targetUnavailableHelp')"
          />
        </article>

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
        </section>
        <section v-if="detail && presented" class="sforum-notifications__rail-section">
          <div class="sforum-notifications__rail-head">
            <h2>{{ t('notifications.detailPage.status') }}</h2>
            <span>{{ detail.readAt ? t('notifications.detailPage.read') : t('notifications.unread') }}</span>
          </div>
          <dl class="sforum-notification-detail__meta">
            <div>
              <dt>{{ t('notifications.detailPage.type') }}</dt>
              <dd>{{ t(presented.typeLabelKey) }}</dd>
            </div>
            <div>
              <dt>{{ t('notifications.detailPage.target') }}</dt>
              <dd>{{ detail.targetAvailable ? t('notifications.detailPage.available') : t('notifications.targetUnavailable') }}</dd>
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
        :pending="categoryAsync.pending.value"
        :can-create-topic="canCreateTopic"
      />
    </aside>

    <aside v-if="mobileInfoOpen" class="sforum-mobile-drawer sforum-mobile-drawer--right">
      <header class="sforum-mobile-drawer__head">
        <strong>{{ t('notifications.detailPage.title') }}</strong>
        <button type="button" :aria-label="t('common.close')" @click="closeMobileDrawers">
          <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
        </button>
      </header>
      <div class="sforum-notification-detail__drawer-summary">
        <strong>{{ unreadTotal }}</strong>
        <span>{{ t('notifications.unreadCountLabel') }}</span>
        <p v-if="detail && presented">{{ t(presented.typeLabelKey) }}</p>
      </div>
    </aside>
  </main>
</template>

<style scoped src="../SFNotificationsPage.css"></style>
<style scoped src="./SFNotificationDetailPage.css"></style>
