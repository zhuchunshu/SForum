<script setup lang="ts">
import { buildForumHomeQuery } from '~/utils/forumHome'
import type { SiteExtensionNavItem, SiteNavItem, SitePublicNav } from '~/composables/useSiteChromeApi'
import {
  forumCategoriesIndexPath,
  forumTagsIndexPath,
  forumTopicExtensionLabel,
  parseForumTagPublicPagesOption
} from '~/utils/forumTaxonomy'

const { t, locale, locales, setLocale } = useI18n()
const localePath = useLocalePath()
const { user, status, refresh } = useAuthSession()
const {
  siteName,
  siteTagline,
  siteLogoUrl,
  webOption
} = useWebOptions()
const { request } = useApiClient()
const chromeApi = useSiteChromeApi()
const router = useRouter()
const route = useRoute()
const colorMode = useColorMode()
const { can } = usePermissions()
const notifications = useNotifications()

// 标签公开列表受运行时选项控制；关闭时隐藏导航入口（详情页同样 404）。
const publicTagPagesEnabled = computed(() => parseForumTagPublicPagesOption(
  webOption('forum.tags.public_pages', 'enabled')
))

// Wave 2 + E2.3：运营配置 items 在前；forum.nav.items 贡献次之。失败时回退内置 Home/Categories/Tags。
type DesktopNavItem = {
  key: string
  label: string
  href: string
  openInNewTab: boolean
  icon?: string
}

const { data: publicNav } = await useAsyncData('site-public-nav-items', async () => {
  try {
    return await chromeApi.listPublicNav()
  } catch {
    return { items: [], extensionItems: [] } as SitePublicNav
  }
}, { default: () => ({ items: [], extensionItems: [] }) as SitePublicNav })

const isEnglishLocale = computed(() => String(locale.value).toLowerCase().startsWith('en'))

function operatorNavLabel(item: SiteNavItem) {
  return isEnglishLocale.value ? item.labelEnUS : item.labelZhCN
}

function extensionNavLabel(item: SiteExtensionNavItem) {
  return forumTopicExtensionLabel(item, String(locale.value || 'zh-CN')) || item.id
}

// 仅消费宿主已校验的站内相对路径或扩展代理 URL；拒绝外链与 admin。
function isSafePublicNavHref(href: string) {
  const value = `${href || ''}`.trim()
  if (!value.startsWith('/') || value.startsWith('//') || value.includes('://')) {
    return false
  }
  if (value === '/api' || value.startsWith('/api/')) {
    return false
  }
  if (value === '/admin' || value.startsWith('/admin/')) {
    return false
  }
  return true
}

function filterTagNav(href: string) {
  if (publicTagPagesEnabled.value) {
    return true
  }
  return !href.replace(/\/$/, '').endsWith('/tags')
}

const desktopNavItems = computed((): DesktopNavItem[] => {
  const configured = (publicNav.value?.items || [])
    .map((item) => ({
      key: `core-${item.id}`,
      label: operatorNavLabel(item),
      href: item.href,
      openInNewTab: item.openInNewTab
    }))
    .filter((item) => item.label && item.href)
    .filter((item) => filterTagNav(item.href))
    // 顶栏已有完整搜索框，避免运营默认项再次占用一个“搜索”导航位。
    .filter((item) => item.href.replace(/\/$/, '') !== '/search')

  // 核心/运营项在前；无运营配置时回退内置三项。
  const core: DesktopNavItem[] = configured.length > 0
    ? configured
    : [
        { key: 'fallback-home', label: t('home.filter.latest'), href: '/', openInNewTab: false },
        { key: 'fallback-categories', label: t('home.filter.categories'), href: forumCategoriesIndexPath(), openInNewTab: false },
        ...(publicTagPagesEnabled.value
          ? [{ key: 'fallback-tags', label: t('home.filter.tags'), href: forumTagsIndexPath(), openInNewTab: false }]
          : [])
      ]

  // E2.3：扩展贡献次之，按 order 追加。
  const extensions = [...(publicNav.value?.extensionItems || [])]
    .slice()
    .sort((a, b) => (a.order || 0) - (b.order || 0))
    .map((item) => ({
      key: `ext-${item.extensionId}:${item.id}`,
      label: extensionNavLabel(item),
      href: item.url,
      openInNewTab: false,
      icon: item.icon
    }))
    .filter((item) => item.label && isSafePublicNavHref(item.href))
    .filter((item) => filterTagNav(item.href))

  return [...core, ...extensions]
})

function resolveNavTo(href: string) {
  const value = href.trim()
  if (!value) {
    return localePath('/')
  }
  if (value.startsWith('http://') || value.startsWith('https://')) {
    return value
  }
  // 扩展代理路径走 API 前缀以外的站内路由；/extensions/* 由宿主代理。
  return localePath(value.startsWith('/') ? value : `/${value}`)
}

function isExternalHref(href: string) {
  return href.startsWith('http://') || href.startsWith('https://')
}

/** 顶栏 active 与 demo 一致：首页精确匹配，其它路径前缀匹配。 */
function isDesktopNavActive(href: string) {
  if (isExternalHref(href)) {
    return false
  }
  const resolved = resolveNavTo(href)
  if (typeof resolved !== 'string' || !resolved.startsWith('/')) {
    return false
  }
  const targetPath = resolved.split('?')[0]?.replace(/\/$/, '') || '/'
  const currentPath = route.path.replace(/\/$/, '') || '/'
  const homePath = String(localePath('/')).replace(/\/$/, '') || '/'
  if (targetPath === homePath) {
    return currentPath === homePath
  }
  return currentPath === targetPath || currentPath.startsWith(`${targetPath}/`)
}

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

type LocaleCode = string
type LocaleOption = {
  code: LocaleCode
  name?: string
}
type NavbarMenuItem = {
  label: string
  description?: string
  icon?: string
  to?: string
  /** 覆盖 ULink 的路由 active，语言项按 i18n locale 判定 */
  active?: boolean
  type?: 'label'
  color?: 'error'
  onSelect?: (event: Event) => void
  children?: NavbarMenuItem[]
}

const searchQuery = ref('')
const mobileSearchOpen = ref(false)
const mobileMenuOpen = useState<boolean>('forum-mobile-menu-open', () => false)
const mobileInfoOpen = useState<boolean>('forum-mobile-info-open', () => false)
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

// 语言切换必须走 setLocale：写 sforum_locale cookie 并导航到目标 locale 路径。
// 仅用 switchLocalePath 链接时，prefix_except_default 下从 /en 回到 / 会因
// cookie 仍为 en 被 detectBrowserLanguage(redirectOn:root) 再 302 回 /en。
const languageMenuItems = computed<NavbarMenuItem[]>(() =>
  localeOptions.value.map((entry) => {
    const isCurrent = entry.code === locale.value
    return {
      label: entry.name || entry.code,
      icon: isCurrent ? 'i-lucide-check' : 'i-lucide-languages',
      // 按当前 i18n locale 标记，避免默认语无前缀时 Router 把 `/` 误判为 active
      active: isCurrent,
      onSelect: (event: Event) => {
        if (isCurrent) {
          event.preventDefault()
          return
        }
        void setLocale(entry.code)
      }
    }
  })
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

function closeMobileDrawers() {
  mobileMenuOpen.value = false
  mobileInfoOpen.value = false
}

function toggleMobileMenu() {
  const opening = !mobileMenuOpen.value
  closeMobileDrawers()
  mobileMenuOpen.value = opening
}

function toggleMobileInfo() {
  const opening = !mobileInfoOpen.value
  closeMobileDrawers()
  mobileInfoOpen.value = opening
}

watch(() => route.fullPath, closeMobileDrawers)

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

      <nav class="navbar__desktop-nav" :aria-label="t('nav.mainNav')">
        <template v-for="item in desktopNavItems" :key="item.key">
          <a
            v-if="item.openInNewTab || isExternalHref(item.href)"
            :href="resolveNavTo(item.href)"
            class="navbar__nav-link"
            :class="{ 'is-active': isDesktopNavActive(item.href) }"
            :target="item.openInNewTab || isExternalHref(item.href) ? '_blank' : undefined"
            :rel="item.openInNewTab || isExternalHref(item.href) ? 'noopener noreferrer' : undefined"
          >
            <UIcon v-if="item.icon" :name="item.icon" class="size-3.5" aria-hidden="true" />
            {{ item.label }}
          </a>
          <NuxtLink
            v-else
            :to="resolveNavTo(item.href)"
            class="navbar__nav-link"
            :class="{ 'is-active': isDesktopNavActive(item.href) }"
            active-class=""
            exact-active-class=""
          >
            <UIcon v-if="item.icon" :name="item.icon" class="size-3.5" aria-hidden="true" />
            {{ item.label }}
          </NuxtLink>
        </template>
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
            square
            class="navbar__control"
            :aria-label="t('nav.language')"
            :title="currentLocaleName"
          >
            <UIcon name="i-lucide-globe" class="size-4" aria-hidden="true" />
          </UButton>
        </UDropdownMenu>

        <ClientOnly>
          <UButton
            color="neutral"
            variant="ghost"
            square
            class="navbar__control"
            :aria-label="themeToggleLabel"
            :aria-pressed="isDarkMode"
            :title="themeToggleLabel"
            @click="toggleColorMode"
          >
            <UIcon :name="themeToggleIcon" class="size-4" aria-hidden="true" />
          </UButton>
          <template #fallback>
            <span class="navbar__control-placeholder" aria-hidden="true" />
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

        <UDropdownMenu
          v-else-if="user"
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
        <span v-else class="navbar__session-placeholder" aria-hidden="true" />
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
.navbar__nav-link,
.navbar__new-topic,
.navbar__mobile-new-topic,
.navbar__utility,
.navbar__session,
.navbar__auth-link {
  display: flex;
  align-items: center;
}

.navbar__notification { position: relative; display: grid; width: 36px; height: 36px; place-items: center; color: #64748b; }
.navbar__notification-badge { position: absolute; top: -2px; right: -4px; min-width: 18px; height: 18px; padding: 0 4px; border: 2px solid #fff; border-radius: 9px; background: var(--sf-accent); color: #fff; font-size: 10px; line-height: 14px; text-align: center; }

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
  flex-shrink: 0;
  gap: 24px;
}

.navbar__nav-link {
  position: relative;
  gap: 6px;
  min-height: 100%;
  padding: 0;
  border-radius: 0;
  color: var(--sf-public-text-secondary, #4f5869);
  font-size: 13px;
  font-weight: 600;
  text-decoration: none;
  cursor: pointer;
}

.navbar__nav-link:hover,
.navbar__nav-link.is-active {
  color: var(--sf-public-text, #151922);
  background: transparent;
}

/* demo .top-nav a.is-active::after：底边 2px 强调色下划线 */
.navbar__nav-link.is-active::after {
  content: "";
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  height: 2px;
  background: var(--sf-accent);
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

.navbar__search {
  width: min(360px, 30vw);
  min-width: 160px;
  margin-left: auto;
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

.navbar__session-placeholder {
  width: 112px;
  height: 36px;
  display: block;
  flex: 0 0 112px;
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

.navbar__control-placeholder {
  width: 36px;
  height: 36px;
  display: block;
  flex: 0 0 36px;
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

.dark .navbar__nav-link,
.dark .navbar__control,
.dark .navbar__user-trigger,
.dark .navbar__mobile-trigger,
.dark .navbar__mobile-search-close {
  color: #d4d4d8;
}

.dark .navbar__nav-link:hover,
.dark .navbar__nav-link.is-active {
  color: #f4f4f5;
  background: transparent;
}

.dark .navbar__nav-link.is-active::after {
  background: var(--sf-accent-dark);
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
  }

  .navbar__session-placeholder {
    width: 40px;
    flex-basis: 40px;
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
    border-top: 1px solid var(--sf-public-border);
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
    padding: 7px 12px;
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
    padding: 2px;
  }

  .navbar__notification {
    display: none;
  }
}
</style>
