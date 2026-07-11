<script setup lang="ts">
import { buildForumHomeQuery } from '~/utils/forumHome'
import {
  forumCategoriesIndexPath,
  forumTagsIndexPath,
  parseForumTagPublicPagesOption
} from '~/utils/forumTaxonomy'

const { t, locale, locales } = useI18n()
const localePath = useLocalePath()
const switchLocalePath = useSwitchLocalePath()
const { user, refresh } = useAuthSession()
const { siteName, siteTagline, webOption } = useWebOptions()
const { request } = useApiClient()
const router = useRouter()
const colorMode = useColorMode()
const { can } = usePermissions()
const notifications = useNotifications()

// 标签公开列表受运行时选项控制；关闭时隐藏导航入口（详情页同样 404）。
const publicTagPagesEnabled = computed(() => parseForumTagPublicPagesOption(
  webOption('forum.tags.public_pages', 'enabled')
))

// 导航栏注册入口以 registration-status 为准（含 bootstrap 覆盖）。
type RegistrationStatus = {
  nextUserIsInitialSuperAdmin: boolean
  registrationEnabled?: boolean
}
const { data: registrationStatus } = await useAsyncData('auth-registration-status-navbar', async () => {
  try {
    return await request<RegistrationStatus>('/auth/registration-status')
  } catch {
    // 接口失败时保守显示注册，避免误关 bootstrap 入口。
    return { nextUserIsInitialSuperAdmin: false, registrationEnabled: true }
  }
})
const showRegisterLinks = computed(() => registrationStatus.value?.registrationEnabled !== false)
const logoAriaLabel = computed(() => {
  const tagline = siteTagline.value
  return tagline ? `${siteName.value} — ${tagline}` : siteName.value
})

type LocaleCode = Parameters<typeof switchLocalePath>[0]
type LocaleOption = {
  code: LocaleCode
  name?: string
}
type NavbarMenuItem = {
  label: string
  description?: string
  icon?: string
  to?: string
  type?: 'label'
  color?: 'error'
  onSelect?: (event: Event) => void
  children?: NavbarMenuItem[]
}

const searchQuery = ref('')
const mobileSearchOpen = ref(false)
const resolvedColorMode = ref<'light' | 'dark'>(
  colorMode.value === 'dark' ? 'dark' : 'light'
)
let colorModeObserver: MutationObserver | null = null

// 发帖入口只对拥有论坛发帖权限的用户显示，API 仍负责最终鉴权。
const canCreateTopic = computed(() => can(FORUM_PERMISSIONS.topicCreate))
const canReviewContent = computed(() => can(FORUM_PERMISSIONS.moderationReview))
const displayName = computed(() =>
  user.value?.displayName || user.value?.username || ''
)
const localeOptions = computed(() =>
  (locales.value as readonly (LocaleCode | LocaleOption)[]).map((entry) => {
    if (typeof entry === 'string') {
      return { code: entry, name: entry }
    }

    return {
      code: entry.code,
      name: entry.name || entry.code
    }
  })
)
const currentLocaleName = computed(() =>
  localeOptions.value.find((entry) => entry.code === locale.value)?.name || locale.value
)
const isDarkMode = computed(() => resolvedColorMode.value === 'dark')
const themeToggleLabel = computed(() =>
  isDarkMode.value ? t('nav.lightMode') : t('nav.darkMode')
)
const themeToggleIcon = computed(() =>
  isDarkMode.value ? 'i-lucide-sun' : 'i-lucide-moon'
)

const languageMenuItems = computed<NavbarMenuItem[]>(() =>
  localeOptions.value.map((entry) => ({
    label: entry.name,
    icon: entry.code === locale.value ? 'i-lucide-check' : 'i-lucide-languages',
    to: switchLocalePath(entry.code)
  }))
)

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

const mobileMenuItems = computed<NavbarMenuItem[][]>(() => {
  const destinations: NavbarMenuItem[] = [
    {
      label: t('nav.home'),
      icon: 'i-lucide-house',
      to: localePath('/')
    },
    {
      label: t('nav.search'),
      icon: 'i-lucide-search',
      onSelect: () => {
        mobileSearchOpen.value = true
      }
    }
  ]

  const account: NavbarMenuItem[] = []
  if (!user.value) {
    account.push({
      label: t('nav.login'),
      icon: 'i-lucide-log-in',
      to: localePath('/login')
    })
    if (showRegisterLinks.value) {
      account.push({
        label: t('nav.register'),
        icon: 'i-lucide-user-plus',
        to: localePath('/register')
      })
    }
  }
  if (user.value && canReviewContent.value) {
    destinations.push({
      label: t('nav.moderationWorkbench'),
      icon: 'i-lucide-shield-check',
      to: localePath('/moderation')
    })
  }

  const controls: NavbarMenuItem[] = [
    {
      label: t('nav.appearance'),
      description: themeToggleLabel.value,
      icon: themeToggleIcon.value,
      onSelect: () => {
        toggleColorMode()
      }
    },
    {
      label: t('nav.language'),
      icon: 'i-lucide-globe',
      children: languageMenuItems.value
    }
  ]

  return account.length
    ? [destinations, account, controls]
    : [destinations, controls]
})

watch(
  () => colorMode.value,
  syncResolvedColorMode,
  { immediate: true }
)

onMounted(() => {
  syncResolvedColorMode()
  colorModeObserver = new MutationObserver(syncResolvedColorMode)
  colorModeObserver.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['class']
  })
  if (user.value) void notifications.refreshUnreadCount().catch(() => {})
})

onUnmounted(() => {
  colorModeObserver?.disconnect()
})

function syncResolvedColorMode() {
  if (!import.meta.client) {
    resolvedColorMode.value = colorMode.value === 'dark' ? 'dark' : 'light'
    return
  }

  // 颜色模式可能先更新 html class，菜单图标以页面实际状态为准。
  resolvedColorMode.value =
    colorMode.value === 'dark' ||
    document.documentElement.classList.contains('dark')
      ? 'dark'
      : 'light'
}

function toggleColorMode() {
  const nextMode = isDarkMode.value ? 'light' : 'dark'
  colorMode.preference = nextMode
  resolvedColorMode.value = nextMode
}

function submitSearch(query: string) {
  return navigateTo({
    path: localePath('/'),
    query: buildForumHomeQuery({
      query,
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
  <header class="navbar">
    <div class="navbar__inner">
      <NuxtLink
        :to="localePath('/')"
        class="navbar__logo"
        :aria-label="logoAriaLabel"
      >
        <span class="navbar__logo-mark" aria-hidden="true">
          <UIcon name="i-lucide-message-circle" class="size-4" />
        </span>
        <span class="navbar__logo-text-wrap">
          <span class="navbar__logo-text">{{ siteName }}</span>
          <span v-if="siteTagline" class="navbar__logo-tagline">{{ siteTagline }}</span>
        </span>
      </NuxtLink>

      <nav class="navbar__desktop-nav" :aria-label="t('nav.mainNav')">
        <NuxtLink
          :to="localePath('/')"
          class="navbar__nav-link"
          :aria-label="t('home.filter.latest')"
        >
          {{ t('home.filter.latest') }}
        </NuxtLink>
        <NuxtLink
          :to="localePath(forumCategoriesIndexPath())"
          class="navbar__nav-link"
          :aria-label="t('home.filter.categories')"
        >
          {{ t('home.filter.categories') }}
        </NuxtLink>
        <NuxtLink
          v-if="publicTagPagesEnabled"
          :to="localePath(forumTagsIndexPath())"
          class="navbar__nav-link"
          :aria-label="t('home.filter.tags')"
        >
          {{ t('home.filter.tags') }}
        </NuxtLink>
      </nav>

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

      <div class="navbar__actions">
        <NuxtLink v-if="user" :to="localePath('/notifications')" class="navbar__notification" :aria-label="t('nav.notifications')">
          <UIcon name="i-lucide-bell" class="size-5" aria-hidden="true" />
          <span v-if="notifications.unreadCount.value" class="navbar__notification-badge">{{ notifications.unreadCount.value > 99 ? '99+' : notifications.unreadCount.value }}</span>
        </NuxtLink>
        <NuxtLink
          v-if="canCreateTopic"
          :to="localePath('/topics/new')"
          class="navbar__mobile-new-topic"
          :aria-label="t('nav.newTopic')"
        >
          <UIcon name="i-lucide-square-pen" class="size-5" aria-hidden="true" />
        </NuxtLink>

        <UDropdownMenu
          :items="languageMenuItems"
          :content="{ align: 'end' }"
        >
          <UButton
            color="neutral"
            variant="ghost"
            class="navbar__control navbar__desktop-control"
            :aria-label="t('nav.language')"
          >
            <UIcon name="i-lucide-globe" class="size-4" aria-hidden="true" />
            <span class="navbar__control-label">{{ currentLocaleName }}</span>
            <UIcon name="i-lucide-chevron-down" class="size-3.5" aria-hidden="true" />
          </UButton>
        </UDropdownMenu>

        <ClientOnly>
          <UButton
            color="neutral"
            variant="ghost"
            square
            class="navbar__control navbar__desktop-control"
            :aria-label="themeToggleLabel"
            :aria-pressed="isDarkMode"
            @click="toggleColorMode"
          >
            <UIcon :name="themeToggleIcon" class="size-4" aria-hidden="true" />
          </UButton>
          <template #fallback>
            <span class="navbar__control-placeholder navbar__desktop-control" aria-hidden="true" />
          </template>
        </ClientOnly>

        <template v-if="!user">
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

        <UDropdownMenu
          v-else
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
              shape="square"
            />
            <span class="navbar__username">{{ displayName }}</span>
            <UIcon name="i-lucide-chevron-down" class="size-3.5" aria-hidden="true" />
          </UButton>
        </UDropdownMenu>

        <UDropdownMenu
          :items="mobileMenuItems"
          :content="{ align: 'end' }"
        >
          <UButton
            color="neutral"
            variant="ghost"
            square
            class="navbar__mobile-trigger"
            :aria-label="t('nav.openMenu')"
          >
            <UIcon name="i-lucide-menu" class="size-5" aria-hidden="true" />
          </UButton>
        </UDropdownMenu>
      </div>
    </div>

    <div v-if="mobileSearchOpen" class="navbar__mobile-search-panel">
      <div class="navbar__mobile-search-inner">
        <SFSearch
          v-model="searchQuery"
          class="navbar__mobile-search"
          :placeholder="t('home.searchPlaceholder')"
          :aria-label="t('nav.search')"
          @submit="submitMobileSearch"
        />
        <UButton
          color="neutral"
          variant="ghost"
          square
          class="navbar__mobile-search-close"
          :aria-label="t('nav.closeSearch')"
          @click="mobileSearchOpen = false"
        >
          <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
        </UButton>
      </div>
    </div>
  </header>
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
  gap: 18px;
  min-height: var(--sf-public-topbar-height, 52px);
  max-width: none;
  margin: 0;
  padding: 0 20px;
}

.navbar__logo,
.navbar__desktop-nav,
.navbar__nav-link,
.navbar__new-topic,
.navbar__mobile-new-topic,
.navbar__actions,
.navbar__auth-link {
  display: flex;
  align-items: center;
}

.navbar__notification { position: relative; display: grid; width: 36px; height: 36px; place-items: center; color: #64748b; }
.navbar__notification-badge { position: absolute; top: -2px; right: -4px; min-width: 18px; height: 18px; padding: 0 4px; border: 2px solid #fff; border-radius: 9px; background: var(--sf-accent); color: #fff; font-size: 10px; line-height: 14px; text-align: center; }

.navbar__logo {
  min-width: 0;
  flex-shrink: 0;
  gap: 8px;
  color: #111827;
  font-size: 15px;
  font-weight: 700;
  text-decoration: none;
}

.navbar__logo-mark {
  display: grid;
  width: 28px;
  height: 28px;
  flex: 0 0 28px;
  place-items: center;
  border: 1px solid var(--sf-accent);
  border-radius: 7px;
  background: var(--sf-accent);
  color: #ffffff;
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
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--sf-public-text-muted, #64748b);
  font-size: 11px;
  font-weight: 500;
}

.navbar__desktop-nav {
  flex-shrink: 0;
  gap: 2px;
}

.navbar__nav-link {
  gap: 6px;
  min-height: 34px;
  padding: 7px 11px;
  border-radius: 7px;
  color: var(--sf-public-text-muted, #64748b);
  font-size: 13px;
  font-weight: 650;
  text-decoration: none;
  cursor: pointer;
}

.navbar__nav-link:hover,
.navbar__nav-link.router-link-active {
  color: var(--sf-accent);
  background: var(--sf-accent-soft);
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
  font-weight: 650;
  text-decoration: none;
}

.navbar__new-topic {
  min-height: 36px;
  padding: 0 12px;
}

.navbar__new-topic:hover,
.navbar__mobile-new-topic:hover {
  background: var(--sf-accent-hover);
}

.navbar__mobile-new-topic {
  display: none;
}

.navbar__search {
  width: min(260px, 30vw);
  min-width: 160px;
  margin-left: auto;
}

.navbar__search :deep(.sf-search__box) {
  min-height: 36px;
  border-radius: 7px;
}

.navbar__actions {
  flex-shrink: 0;
  gap: 6px;
}

.navbar__control,
.navbar__user-trigger,
.navbar__mobile-trigger {
  min-height: 36px;
  border-radius: 7px;
}

.navbar__control {
  color: #4b5563;
}

.navbar__control-label {
  max-width: 96px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}

.navbar__control-placeholder {
  width: 36px;
  height: 36px;
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

.dark .navbar__nav-link,
.dark .navbar__control,
.dark .navbar__user-trigger,
.dark .navbar__mobile-trigger,
.dark .navbar__mobile-search-close {
  color: #d4d4d8;
}

.dark .navbar__nav-link:hover,
.dark .navbar__nav-link.router-link-active {
  color: var(--sf-accent-dark);
  background: rgb(var(--sf-accent-rgb) / 0.2);
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
  .navbar__desktop-nav,
  .navbar__search,
  .navbar__new-topic,
  .navbar__desktop-control,
  .navbar__auth-link {
    display: none;
  }

  .navbar__actions {
    margin-left: auto;
  }

  .navbar__mobile-new-topic,
  .navbar__mobile-trigger {
    display: inline-flex;
  }

  .navbar__mobile-search-panel {
    display: block;
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
}

@media (max-width: 520px) {
  .navbar__inner {
    gap: 6px;
    padding: 0 16px;
  }

  .navbar__mobile-search-inner {
    padding: 8px 16px;
  }

  .navbar__logo-text,
  .navbar__username,
  .navbar__user-trigger > :deep(svg) {
    display: none;
  }

  .navbar__user-trigger {
    padding: 2px;
  }
}
</style>
