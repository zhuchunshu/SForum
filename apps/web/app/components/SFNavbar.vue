<script setup lang="ts">
import SFNotificationPreview from '~/components/notifications/SFNotificationPreview.vue'
import SFPublicMobileNavigation from '~/components/navigation/SFPublicMobileNavigation.vue'
import SFPublicNavigationLinks from '~/components/navigation/SFPublicNavigationLinks.vue'
import { useNotifications } from '~/composables/notifications/useNotifications'
import { FORUM_PERMISSIONS, usePermissions } from '~/composables/identity/usePermissions'
import { useAuthSession } from '~/composables/identity/useAuthSession'
import {
  type NavbarMenuItem,
  useNavbarLanguageMenu
} from '~/composables/navigation/useNavbarLanguageMenu'
import { usePublicNavigation } from '~/composables/navigation/usePublicNavigation'
import { usePublicSidebarDrawer } from '~/composables/navigation/usePublicSidebarDrawer'
import { useColorModePreference } from '~/composables/appearance/useColorModePreference'
import { buildForumHomeQuery } from '~/utils/forum/forumHome'
import { parseForumTagPublicPagesOption } from '~/utils/forum/forumTaxonomy'

const props = withDefaults(defineProps<{
  /** Core 404 应急页不得在 API 已失效时继续启动 chrome 请求。 */
  fetchRemoteChrome?: boolean
  /** Host 回退壳的桌面导航几何；主题路径继续由主题 CSS 管理。 */
  layout?: 'default' | 'fullwidth-3col'
}>(), {
  fetchRemoteChrome: true,
  layout: 'default'
})

const { t } = useI18n()
const localePath = useLocalePath()
const { user, status, refresh } = useAuthSession()
const notifications = useNotifications()
const {
  siteName,
  siteTagline,
  siteLogoUrl,
  webOption
} = useWebOptions()
const { request } = useApiClient()
const router = useRouter()
const route = useRoute()
const {
  preference: colorModePreference,
  options: colorModeOptions,
  cyclePreference: cycleColorModePreference
} = useColorModePreference()
const { can } = usePermissions()
const { topbarItems, sidebarItems } = usePublicNavigation(props.fetchRemoteChrome)

// 标签公开列表受运行时选项控制；关闭时隐藏导航入口（详情页同样 404）。
const publicTagPagesEnabled = computed(() => parseForumTagPublicPagesOption(
  webOption('forum.tags.public_pages', 'enabled')
))

function filterTagNav(href: string) {
  if (publicTagPagesEnabled.value) {
    return true
  }
  return !href.replace(/\/$/, '').endsWith('/tags')
}

const visibleTopbarItems = computed(() => topbarItems.value
  .filter(item => !item.href || filterTagNav(item.href)))
const visibleSidebarItems = computed(() => sidebarItems.value
  .filter(item => !item.href || filterTagNav(item.href)))

// 导航栏注册入口以 registration-status 为准（含 bootstrap 覆盖）。
type RegistrationStatus = {
  nextUserIsInitialSuperAdmin: boolean
  registrationEnabled?: boolean
}
const defaultRegistrationStatus = (): RegistrationStatus => ({
  nextUserIsInitialSuperAdmin: false,
  registrationEnabled: true
})
const registrationStatus = props.fetchRemoteChrome
  ? useAsyncData('auth-registration-status-navbar', async () => {
      try {
        return await request<RegistrationStatus>('/auth/registration-status')
      } catch {
        // 接口失败时保守显示注册，避免误关 bootstrap 入口。
        return defaultRegistrationStatus()
      }
    }, { default: defaultRegistrationStatus }).data
  : shallowRef<RegistrationStatus>(defaultRegistrationStatus())
const showRegisterLinks = computed(() => registrationStatus.value?.registrationEnabled !== false)
const logoAriaLabel = computed(() => {
  const tagline = siteTagline.value
  return tagline ? `${siteName.value} — ${tagline}` : siteName.value
})

const routeSearchQuery = computed(() => typeof route.query.q === 'string' ? route.query.q.trim() : '')
const searchQuery = ref(routeSearchQuery.value)
const mobileSearchOpen = ref(false)
const {
  open: mobileMenuOpen,
  hasPageOwner: hasPageSidebarOwner,
  closeDrawer: closeMobileMenu,
  toggleDrawer: toggleMobileSidebar
} = usePublicSidebarDrawer()
const mobileInfoOpen = useState<boolean>('forum-mobile-info-open', () => false)

// 发帖入口只对拥有论坛发帖权限的用户显示，API 仍负责最终鉴权。
const canCreateTopic = computed(() => can(FORUM_PERMISSIONS.topicCreate))
const canReviewContent = computed(() => can(FORUM_PERMISSIONS.moderationReview))
const displayName = computed(() =>
  user.value?.displayName || user.value?.username || ''
)
const { currentLocaleName, languageMenuItems } = useNavbarLanguageMenu()
const currentColorModeOption = computed(() =>
  colorModeOptions.find(option => option.value === colorModePreference.value) || colorModeOptions[0]!
)
const colorModePreferenceLabel = computed(() => t(currentColorModeOption.value.labelKey))
const colorModeTriggerLabel = computed(() => t('appearance.colorMode.currentPreference', {
  preference: colorModePreferenceLabel.value
}))
const colorModeTriggerIcon = computed(() => currentColorModeOption.value.icon)

const userMenuItems = computed<NavbarMenuItem[][]>(() => {
  if (!user.value) {
    return []
  }

  return [
    [
      {
        label: displayName.value,
        description: `@${user.value.username}`,
        type: 'label'
      }
    ],
    [
      {
        label: t('nav.myProfile'),
        icon: 'i-lucide-user',
        to: localePath(`/u/${user.value.username}`)
      },
      {
        label: t('nav.profileSettings'),
        icon: 'i-lucide-settings',
        to: localePath('/settings/profile')
      },
      ...(canReviewContent.value
        ? [{ label: t('nav.moderationWorkbench'), icon: 'i-lucide-shield-check', to: localePath('/moderation') }]
        : [])
    ],
    [
      {
        label: t('nav.logout'),
        icon: 'i-lucide-log-out',
        color: 'error',
        onSelect: () => {
          void logout()
        }
      }
    ]
  ]
})

function closeMobileDrawers() {
  closeMobileMenu()
  mobileInfoOpen.value = false
}

function toggleMobileMenu() {
  const opening = !mobileMenuOpen.value
  closeMobileDrawers()
  if (opening) toggleMobileSidebar()
}

function toggleMobileInfo() {
  const opening = !mobileInfoOpen.value
  closeMobileDrawers()
  mobileInfoOpen.value = opening
}

watch(() => route.fullPath, closeMobileDrawers)
watch(routeSearchQuery, (query) => {
  if (searchQuery.value !== query) {
    searchQuery.value = query
  }
})

function submitSearch(query: string) {
  const normalizedQuery = query.trim()
  return navigateTo({
    path: localePath(normalizedQuery ? '/search' : '/'),
    query: buildForumHomeQuery({
      query: normalizedQuery,
      categorySlug: '',
      tagSlug: ''
    })
  })
}

function submitMobileSearch(query: string) {
  mobileSearchOpen.value = false
  return submitSearch(query)
}

async function logout() {
  try {
    await request('/auth/logout', { method: 'POST' })
  } catch {
    // 服务端退出失败时仍刷新会话，以服务端实际状态为准。
  }

  await refresh()
  await router.push(localePath('/login'))
}
</script>

<template>
  <header
    class="navbar"
    :class="{ 'navbar--fullwidth-3col': props.layout === 'fullwidth-3col' }"
  >
    <div class="navbar__inner">
      <button
        type="button"
        class="navbar__mobile-shell-button"
        :aria-label="t('nav.openMenu')"
        :aria-expanded="mobileMenuOpen"
        @click="toggleMobileMenu"
      >
        <UIcon name="i-lucide-menu" class="size-5" aria-hidden="true" />
      </button>

      <NuxtLink
        :to="localePath('/')"
        class="navbar__logo"
        :aria-label="logoAriaLabel"
      >
        <span v-if="siteLogoUrl" class="navbar__logo-image-wrap" aria-hidden="true">
          <img :src="siteLogoUrl" alt="" class="navbar__logo-image">
        </span>
        <span v-else class="navbar__logo-mark" aria-hidden="true">
          <UIcon name="i-tabler-message-circle-filled" class="size-5" />
        </span>
        <span class="navbar__logo-text-wrap">
          <span class="navbar__logo-text">{{ siteName }}</span>
          <span v-if="siteTagline" class="navbar__logo-tagline">{{ siteTagline }}</span>
        </span>
      </NuxtLink>

      <SFPublicNavigationLinks
        class="navbar__desktop-nav"
        :items="visibleTopbarItems"
      />

      <SFSearch
        v-model="searchQuery"
        class="navbar__search"
        :placeholder="t('home.searchPlaceholder')"
        :aria-label="t('nav.search')"
        @submit="submitSearch"
      />

      <NuxtLink
        v-if="canCreateTopic"
        :to="localePath('/topics/new')"
        class="navbar__new-topic"
        :aria-label="t('nav.newTopic')"
      >
        <UIcon name="i-lucide-square-pen" class="size-4" aria-hidden="true" />
        <span>{{ t('nav.newTopic') }}</span>
      </NuxtLink>
      <span v-else class="navbar__new-topic-placeholder" aria-hidden="true" />

      <!-- 工具区：通知 / 语言 / 日夜模式；会话区单独一列与右侧栏对齐 -->
      <div class="navbar__utility">
        <button
          type="button"
          class="navbar__mobile-info-button"
          :aria-label="t('home.rightRail.ariaLabel')"
          :aria-expanded="mobileInfoOpen"
          @click="toggleMobileInfo"
        >
          <UIcon name="i-lucide-panel-right" class="size-5" aria-hidden="true" />
        </button>
        <SFNotificationPreview v-if="user" />
        <NuxtLink
          v-if="canCreateTopic"
          :to="localePath('/topics/new')"
          class="navbar__mobile-new-topic"
          :aria-label="t('nav.newTopic')"
        >
          <UIcon name="i-lucide-square-pen" class="size-5" aria-hidden="true" />
        </NuxtLink>

        <ClientOnly>
          <UDropdownMenu
            :items="languageMenuItems"
            :content="{ align: 'end' }"
          >
            <UButton
              color="neutral"
              variant="ghost"
              square
              class="navbar__control navbar__language-control"
              :aria-label="t('nav.language')"
              :title="currentLocaleName"
            >
              <UIcon name="i-tabler-language" class="size-5" aria-hidden="true" />
            </UButton>
          </UDropdownMenu>
          <template #fallback>
            <UButton
              color="neutral"
              variant="ghost"
              square
              class="navbar__control"
              :aria-label="t('nav.language')"
              :title="currentLocaleName"
              aria-hidden="true"
              tabindex="-1"
              data-ssr-fallback="navbar-language"
            >
              <UIcon name="i-tabler-language" class="size-5" aria-hidden="true" />
            </UButton>
          </template>
        </ClientOnly>

        <ClientOnly>
          <UButton
            color="neutral"
            variant="ghost"
            square
            class="navbar__control navbar__appearance-control"
            :aria-label="colorModeTriggerLabel"
            :title="colorModeTriggerLabel"
            @click="cycleColorModePreference"
          >
            <UIcon :name="colorModeTriggerIcon" class="size-5" aria-hidden="true" />
          </UButton>
          <template #fallback>
            <UButton
              color="neutral"
              variant="ghost"
              square
              class="navbar__control navbar__appearance-control"
              :aria-label="t('nav.appearance')"
              :title="t('nav.appearance')"
              aria-hidden="true"
              tabindex="-1"
              data-ssr-fallback="navbar-appearance"
            >
              <UIcon name="i-tabler-brightness-filled" class="size-5" aria-hidden="true" />
            </UButton>
          </template>
        </ClientOnly>
      </div>

      <div class="navbar__session">
        <template v-if="status === 'guest'">
          <NuxtLink
            :to="localePath('/login')"
            class="navbar__auth-link navbar__auth-link--quiet"
            :aria-label="t('nav.login')"
          >
            {{ t('nav.login') }}
          </NuxtLink>
          <NuxtLink
            v-if="showRegisterLinks"
            :to="localePath('/register')"
            class="navbar__auth-link navbar__auth-link--primary"
            :aria-label="t('nav.register')"
          >
            {{ t('nav.register') }}
          </NuxtLink>
        </template>

        <ClientOnly
          v-else-if="user"
        >
          <UDropdownMenu
            :items="userMenuItems"
            :content="{ align: 'end' }"
          >
            <UButton
              color="neutral"
              variant="ghost"
              class="navbar__user-trigger"
              :aria-label="t('nav.userMenu')"
            >
              <SFAvatar
                :name="displayName"
                :avatar="user.avatar"
                size="sm"
                shape="circle"
              />
              <span class="navbar__username">{{ displayName }}</span>
              <UIcon name="i-lucide-chevron-down" class="size-3.5" aria-hidden="true" />
            </UButton>
          </UDropdownMenu>
          <template #fallback>
            <UButton
              color="neutral"
              variant="ghost"
              class="navbar__user-trigger"
              :aria-label="t('nav.userMenu')"
              aria-hidden="true"
              tabindex="-1"
              data-ssr-fallback="navbar-user"
            >
              <SFAvatar
                :name="displayName"
                :avatar="user.avatar"
                size="sm"
                shape="circle"
              />
              <span class="navbar__username">{{ displayName }}</span>
              <UIcon name="i-lucide-chevron-down" class="size-3.5" aria-hidden="true" />
            </UButton>
          </template>
        </ClientOnly>
        <span
          v-else
          class="navbar__session-loading"
          :aria-label="t('nav.userMenu')"
          aria-busy="true"
        >
          <span class="navbar__session-loading-avatar" aria-hidden="true" />
          <span class="navbar__session-loading-name" aria-hidden="true" />
        </span>
        <NuxtLink
          v-if="status === 'guest'"
          :to="localePath('/login')"
          class="navbar__mobile-guest-avatar"
          :aria-label="t('nav.login')"
        >
          <SFAvatar :name="siteName" size="sm" shape="circle" />
        </NuxtLink>
      </div>
    </div>

    <div class="navbar__mobile-search-panel">
      <div class="navbar__mobile-search-inner">
        <SFSearch
          v-model="searchQuery"
          class="navbar__mobile-search"
          :placeholder="t('home.searchPlaceholder')"
          :aria-label="t('nav.search')"
          @submit="submitMobileSearch"
        />
        <NuxtLink
          :to="canCreateTopic ? localePath('/topics/new') : localePath('/login')"
          class="navbar__mobile-compose"
        >
          <UIcon :name="canCreateTopic ? 'i-lucide-square-pen' : 'i-lucide-log-in'" class="size-4" aria-hidden="true" />
          <span>{{ canCreateTopic ? t('nav.newTopic') : t('nav.login') }}</span>
        </NuxtLink>
      </div>
    </div>
  </header>
  <nav class="navbar__mobile-bottom-nav" :aria-label="t('nav.mainNav')">
    <NuxtLink
      :to="localePath('/')"
      class="navbar__mobile-bottom-link"
      :class="{ 'is-active': route.path === localePath('/') }"
    >
      <UIcon name="i-lucide-house" class="size-5" aria-hidden="true" />
      <span>{{ t('nav.home') }}</span>
    </NuxtLink>
    <NuxtLink
      :to="canCreateTopic ? localePath('/topics/new') : localePath('/login')"
      class="navbar__mobile-bottom-link"
      :class="{ 'is-active': route.path.startsWith(localePath('/topics/new')) }"
      :aria-label="t('nav.newTopic')"
    >
      <UIcon name="i-lucide-square-pen" class="size-5" aria-hidden="true" />
      <span>{{ t('nav.newTopic') }}</span>
    </NuxtLink>
    <NuxtLink
      :to="user ? localePath('/notifications') : localePath('/login')"
      class="navbar__mobile-bottom-link navbar__mobile-bottom-notifications"
      :class="{ 'is-active': route.path.startsWith(localePath('/notifications')) }"
      :aria-label="t('nav.notifications')"
    >
      <span class="navbar__mobile-bottom-icon-wrap">
        <UIcon name="i-lucide-bell" class="size-5" aria-hidden="true" />
        <span v-if="user && notifications.unreadCount.value" class="navbar__mobile-bottom-badge">
          {{ notifications.unreadCount.value > 99 ? '99+' : notifications.unreadCount.value }}
        </span>
      </span>
      <span>{{ t('nav.notifications') }}</span>
    </NuxtLink>
  </nav>
  <SFPublicMobileNavigation
    :open="mobileMenuOpen && !hasPageSidebarOwner"
    :items="visibleSidebarItems"
    @close="closeMobileMenu"
  />
</template>

<style scoped>
.navbar {
  position: sticky;
  top: 0;
  z-index: 50;
  min-height: var(--sf-public-topbar-height, 52px);
  border-bottom: 1px solid var(--sf-public-border);
  background: var(--sf-public-surface);
  box-shadow: none;
}

.navbar__inner {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: var(--sf-public-topbar-height, 52px);
  max-width: none;
  margin: 0;
  padding: 0 18px 0 20px;
}

.navbar__logo,
.navbar__desktop-nav,
.navbar__new-topic,
.navbar__mobile-new-topic,
.navbar__utility,
.navbar__session,
.navbar__auth-link {
  display: flex;
  align-items: center;
}

.navbar__logo {
  min-width: 0;
  flex-shrink: 0;
  gap: 10px;
  color: #111827;
  font-size: 15px;
  font-weight: 700;
  text-decoration: none;
}

.navbar__logo-mark {
  display: grid;
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  place-items: center;
  border: 1px solid var(--sf-accent);
  border-radius: 7px;
  background: var(--sf-accent);
  color: #ffffff;
}

.navbar__logo-image-wrap {
  display: grid;
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  place-items: center;
  overflow: hidden;
  border-radius: 7px;
}

.navbar__logo-image {
  width: 32px;
  height: 32px;
  object-fit: contain;
}

.navbar__logo-text-wrap {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 1px;
  line-height: 1.15;
}

.navbar__logo-text,
.navbar__username {
  max-width: 150px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.navbar__logo-tagline {
  display: none;
}

.navbar__desktop-nav {
  align-self: stretch;
  flex: 0 1 auto;
}

.navbar__new-topic,
.navbar__mobile-new-topic {
  justify-content: center;
  gap: 6px;
  flex-shrink: 0;
  border-radius: 7px;
  color: #fff;
  background: var(--sf-accent);
  font-size: 13px;
  font-weight: 700;
  text-decoration: none;
}

.navbar__new-topic-placeholder {
  width: 92px;
  height: 38px;
  display: block;
  flex: 0 0 92px;
}

.navbar__new-topic {
  min-height: 38px;
  padding: 0 15px;
}

.navbar__new-topic:hover,
.navbar__mobile-new-topic:hover {
  background: var(--sf-accent-hover);
}

.navbar__mobile-new-topic {
  display: none;
}

/* 紧凑搜索：不抢占顶栏中间全部空间；默认主题网格内再 cap max-width */
.navbar__search {
  width: min(260px, 24vw);
  min-width: 140px;
  max-width: 280px;
  margin-left: auto;
  flex: 0 1 260px;
}

.navbar__search :deep(.sf-search__box) {
  min-height: 38px;
  border-radius: 7px;
}

/* 工具按钮（语言 / 日夜 / 通知）与会话区分离，便于与右侧栏对齐 */
.navbar__utility {
  flex-shrink: 0;
  justify-content: flex-end;
  gap: 4px;
}

.navbar__session {
  flex-shrink: 0;
  justify-content: flex-end;
  gap: 6px;
  min-width: 0;
}

.navbar__session-loading {
  width: 112px;
  height: 36px;
  display: flex;
  flex: 0 0 112px;
  align-items: center;
  gap: 7px;
  padding: 0 10px;
  box-sizing: border-box;
  overflow: hidden;
  border-radius: 7px;
}

.navbar__session-loading-avatar,
.navbar__session-loading-name {
  display: block;
  background: var(--sf-public-surface-muted);
}

.navbar__session-loading-avatar {
  width: 28px;
  height: 28px;
  flex: 0 0 28px;
  border-radius: 50%;
}

.navbar__session-loading-name {
  width: 56px;
  height: 10px;
  border-radius: 4px;
}

[data-ssr-fallback] {
  pointer-events: none;
}

@media (min-width: 981px) {
  .navbar--fullwidth-3col .navbar__inner {
    display: grid;
    min-height: 58px;
    grid-template-columns:
      var(--sf-public-sidebar-width, 230px)
      auto
      minmax(0, 1fr)
      auto
      auto
      var(--sf-public-right-rail-width, 270px);
    gap: 10px;
    padding: 0 var(--sf-public-edge-inset, 24px);
  }

  .navbar--fullwidth-3col .navbar__logo {
    padding-left: 16px;
  }

  .navbar--fullwidth-3col .navbar__desktop-nav {
    gap: 28px;
  }

  .navbar--fullwidth-3col .navbar__search {
    width: min(260px, 100%);
    min-width: 0;
    max-width: 280px;
    margin-left: 0;
    justify-self: start;
  }

  .navbar--fullwidth-3col .navbar__new-topic,
  .navbar--fullwidth-3col .navbar__session {
    box-sizing: border-box;
    width: 100%;
  }

  .navbar--fullwidth-3col .navbar__utility,
  .navbar--fullwidth-3col .navbar__session {
    justify-content: flex-end;
  }

  .navbar--fullwidth-3col .navbar__utility {
    gap: 2px;
  }

  .navbar--fullwidth-3col .navbar__session {
    min-width: 0;
    padding: 0 16px 0 12px;
  }

  .navbar--fullwidth-3col .navbar__user-trigger {
    max-width: 100%;
  }
}

.navbar__control,
.navbar__user-trigger,
.navbar__mobile-trigger {
  min-height: 36px;
  border-radius: 7px;
}

.navbar__control {
  width: 36px;
  color: #4b5563;
}

.navbar__auth-link {
  min-height: 34px;
  padding: 0 12px;
  border-radius: 7px;
  font-size: 13px;
  font-weight: 650;
  text-decoration: none;
}

.navbar__auth-link--quiet {
  border: 1px solid #d1d5db;
  color: #4b5563;
}

.navbar__auth-link--quiet:hover {
  border-color: #9ca3af;
  color: #111827;
  background: #f9fafb;
}

.navbar__auth-link--primary {
  color: #fff;
  background: var(--sf-accent);
}

.navbar__auth-link--primary:hover {
  background: var(--sf-accent-hover);
}

.navbar__user-trigger {
  max-width: 190px;
  gap: 7px;
  color: #374151;
}

.navbar__username {
  font-size: 13px;
  font-weight: 650;
}

.navbar__mobile-trigger {
  display: none;
  color: #374151;
}

.navbar__mobile-shell-button,
.navbar__mobile-info-button {
  display: none;
  width: 40px;
  height: 40px;
  flex: 0 0 40px;
  place-items: center;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: var(--sf-public-text-secondary);
  cursor: pointer;
}

.navbar__mobile-shell-button:hover,
.navbar__mobile-info-button:hover {
  background: var(--sf-public-surface-muted);
  color: var(--sf-public-text);
}

.navbar__mobile-search-panel {
  display: none;
  border-top: 1px solid #e4e8ef;
  background: #fff;
}

.navbar__mobile-search-inner {
  display: flex;
  align-items: center;
  gap: 8px;
  max-width: var(--sf-public-container);
  margin: 0 auto;
  padding: 8px 24px;
}

.navbar__mobile-search {
  flex: 1;
}

.navbar__mobile-search-close {
  flex: 0 0 40px;
  border-radius: 7px;
  color: #374151;
}

.navbar__mobile-bottom-nav,
.navbar__mobile-guest-avatar {
  display: none;
}

.navbar__mobile-compose {
  min-height: 40px;
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border-radius: 7px;
  padding: 0 13px;
  background: var(--sf-accent);
  color: #fff;
  font-size: 12px;
  font-weight: 700;
  text-decoration: none;
}

.dark .navbar {
  border-bottom-color: #27272a;
  background: #09090b;
  box-shadow: 0 1px 0 #27272a;
}

.dark .navbar__logo {
  color: #f4f4f5;
}

.dark .navbar__logo-mark {
  border-color: var(--sf-accent-dark);
  background: var(--sf-accent-dark);
  color: #052e2b;
}

.dark .navbar__control,
.dark .navbar__user-trigger,
.dark .navbar__mobile-trigger,
.dark .navbar__mobile-search-close {
  color: #d4d4d8;
}

.dark .navbar__auth-link--quiet {
  border-color: #3f3f46;
  color: #d4d4d8;
}

.dark .navbar__auth-link--quiet:hover {
  border-color: #52525b;
  color: #fff;
  background: #18181b;
}

.dark .navbar__auth-link--primary {
  color: #052e2b;
  background: var(--sf-accent-dark);
}

.dark .navbar__new-topic,
.dark .navbar__mobile-new-topic {
  color: #052e2b;
  background: var(--sf-accent-dark);
}

.dark .navbar__mobile-search-panel {
  border-top-color: #27272a;
  background: #09090b;
}

@media (max-width: 980px) {
  .navbar {
    min-height: 108px;
  }

  .navbar__inner {
    min-height: 54px;
    padding: 0 12px;
  }

  .navbar__desktop-nav,
  .navbar__search,
  .navbar__new-topic,
  .navbar__new-topic-placeholder,
  .navbar__auth-link {
    display: none;
  }

  .navbar__utility,
  .navbar__session {
    min-width: 0;
  }

  .navbar__utility {
    margin-left: auto;
    order: 0;
    gap: 3px;
  }

  .navbar__session {
    order: 3;
  }

  .navbar__utility :deep(.sf-notification-preview),
  .navbar__language-control,
  .navbar__mobile-new-topic {
    display: none !important;
  }

  .navbar__appearance-control {
    order: 1;
  }

  .navbar__mobile-info-button {
    order: 2;
  }

  .navbar__session-loading {
    width: 40px;
    flex-basis: 40px;
    padding: 0 4px;
  }

  .navbar__session-loading-name {
    display: none;
  }

  .navbar__username {
    display: none;
  }

  .navbar__mobile-shell-button,
  .navbar__mobile-info-button {
    display: grid;
  }

  .navbar__mobile-search-panel {
    display: block;
    height: 54px;
    border-top: 0;
    background: transparent;
  }

  .navbar__logo,
  .navbar__mobile-new-topic,
  .navbar__user-trigger,
  .navbar__mobile-trigger,
  .navbar__mobile-search-close,
  .navbar__mobile-search {
    min-width: 40px;
    min-height: 40px;
  }

  .navbar__mobile-search :deep(.sf-search__box) {
    min-height: 40px;
  }

  .navbar__mobile-search-inner {
    height: 54px;
    max-width: none;
    padding: 10px 12px 8px;
  }

  .navbar__mobile-search .sf-search__box {
    min-height: 38px;
    border: 0;
    border-radius: 12px;
    background: var(--sf-public-surface);
    box-shadow: 0 2px 12px rgb(51 56 80 / 0.055);
  }

  .navbar__mobile-compose {
    min-height: 38px;
    border-radius: 10px;
  }

  .navbar__user-trigger,
  .navbar__mobile-guest-avatar {
    display: inline-flex;
    width: 32px;
    height: 32px;
    align-items: center;
    justify-content: center;
    padding: 2px;
    border-radius: 50%;
  }

  .navbar__mobile-guest-avatar {
    color: var(--sf-public-text-secondary);
  }

  .navbar__mobile-bottom-nav {
    position: fixed;
    right: 0;
    bottom: 0;
    left: 0;
    z-index: 40;
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    height: 60px;
    padding: 8px 28px 7px;
    background: color-mix(in srgb, var(--sf-public-surface) 97%, transparent);
    box-shadow: 0 -2px 14px rgb(51 56 80 / 0.08);
    backdrop-filter: blur(12px);
  }

  .navbar__mobile-bottom-link {
    position: relative;
    display: flex;
    min-width: 0;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 2px;
    color: var(--sf-public-text-muted);
    font-size: 10px;
    font-weight: 650;
    text-decoration: none;
  }

  .navbar__mobile-bottom-link.is-active {
    color: var(--sf-accent);
  }

  .navbar__mobile-bottom-icon-wrap {
    position: relative;
    display: inline-flex;
  }

  .navbar__mobile-bottom-badge {
    position: absolute;
    top: -5px;
    left: 13px;
    min-width: 14px;
    height: 14px;
    display: grid;
    place-items: center;
    border-radius: 50%;
    padding: 0 3px;
    background: #e05260;
    color: #fff;
    font-size: 9px;
    font-style: normal;
    font-weight: 700;
    line-height: 1;
  }

  .navbar__mobile-new-topic,
  .navbar__mobile-trigger {
    display: none;
  }
}

@media (max-width: 520px) {
  .navbar__inner {
    gap: 6px;
    padding: 0 10px;
  }

  .navbar__mobile-search-inner {
    padding: 7px 10px;
  }

  .navbar__logo-text,
  .navbar__username,
  .navbar__user-trigger > :deep(svg) {
    display: none;
  }

  .navbar__user-trigger {
    width: 30px;
    height: 30px;
    padding: 1px;
  }

  .navbar__mobile-guest-avatar {
    width: 30px;
    height: 30px;
  }

  .navbar__mobile-bottom-nav {
    padding-right: 24px;
    padding-left: 24px;
  }
}
</style>
