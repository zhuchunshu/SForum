<script setup lang="ts">
const { t, locale, locales } = useI18n()
const localePath = useLocalePath()
const switchLocalePath = useSwitchLocalePath()
const { user, refresh } = useAuthSession()
const { siteName } = useWebOptions()
const apiBaseUrl = useRuntimeConfig().public.apiBaseUrl as string
const router = useRouter()
const colorMode = useColorMode()
const { can } = usePermissions()

// 仅对有发帖权限的登录用户显示“发帖”入口。
const canCreateTopic = computed(() => can(FORUM_PERMISSIONS.topicCreate))

// 控制用户下拉菜单的显示
const menuOpen = ref(false)
const menuRef = ref<HTMLElement | null>(null)

// 控制语言切换下拉菜单的显示
const langMenuOpen = ref(false)
const langMenuRef = ref<HTMLElement | null>(null)
const resolvedColorMode = ref<'light' | 'dark'>(
  colorMode.value === 'dark' ? 'dark' : 'light'
)
let colorModeObserver: MutationObserver | null = null

// 点击页面其他区域关闭菜单
onMounted(() => {
  document.addEventListener('click', onClickOutside)
  syncResolvedColorMode()
  colorModeObserver = new MutationObserver(syncResolvedColorMode)
  colorModeObserver.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['class']
  })
})
onUnmounted(() => {
  document.removeEventListener('click', onClickOutside)
  colorModeObserver?.disconnect()
})
function onClickOutside(e: MouseEvent) {
  if (menuRef.value && !menuRef.value.contains(e.target as Node)) {
    menuOpen.value = false
  }
  if (langMenuRef.value && !langMenuRef.value.contains(e.target as Node)) {
    langMenuOpen.value = false
  }
}

watch(
  () => colorMode.value,
  () => {
    syncResolvedColorMode()
  },
  { immediate: true }
)

function syncResolvedColorMode() {
  if (!import.meta.client) {
    resolvedColorMode.value = colorMode.value === 'dark' ? 'dark' : 'light'
    return
  }

  // Nuxt Color Mode 会先改 <html> 类名，再完成组件水合；按钮以真实页面类名为准。
  resolvedColorMode.value =
    colorMode.value === 'dark' ||
    document.documentElement.classList.contains('dark')
      ? 'dark'
      : 'light'
}

// 退出登录
async function logout() {
  menuOpen.value = false
  try {
    await $fetch(`${apiBaseUrl}/auth/logout`, {
      method: 'POST',
      credentials: 'include'
    })
  } catch {
    // 即使接口失败也清空本地状态
  }
  await refresh()
  await router.push(localePath('/login'))
}

// 用户显示名：优先 displayName，没有就用 username 首字母大写
const displayName = computed(() =>
  user.value?.displayName || user.value?.username || ''
)
const avatarLetter = computed(() =>
  displayName.value.charAt(0).toUpperCase()
)

// 当前语言的名称，比如 "简体中文" 或 "English"
const currentLocaleName = computed(() => {
  const currentLoc = (locales.value as any[]).find(
    (loc) => (typeof loc === 'object' ? loc.code : loc) === locale.value
  )
  return typeof currentLoc === 'object' ? currentLoc.name : currentLoc || ''
})

const isDarkMode = computed(() => resolvedColorMode.value === 'dark')
const themeToggleLabel = computed(() =>
  isDarkMode.value ? t('nav.lightMode') : t('nav.darkMode')
)
const themeToggleIcon = computed(() =>
  isDarkMode.value ? 'i-lucide-sun' : 'i-lucide-moon'
)

function toggleColorMode() {
  const nextMode = isDarkMode.value ? 'light' : 'dark'
  colorMode.preference = nextMode
  resolvedColorMode.value = nextMode
}
</script>

<template>
  <header class="navbar">
    <div class="navbar__inner">

      <!-- Logo -->
      <NuxtLink :to="localePath('/')" class="navbar__logo">
        <div class="navbar__logo-mark" aria-hidden="true">
          <UIcon name="i-lucide-message-circle" class="navbar__logo-icon" />
        </div>
        <span class="navbar__logo-text">{{ siteName }}</span>
      </NuxtLink>

      <!-- 主导航（占位，后续扩展版块） -->
      <nav class="navbar__nav" :aria-label="t('nav.mainNav')">
        <NuxtLink :to="localePath('/')" class="navbar__nav-link">
          {{ t('nav.home') }}
        </NuxtLink>
        <NuxtLink
          v-if="canCreateTopic"
          :to="localePath('/topics/new')"
          class="navbar__nav-link navbar__nav-link--create"
        >
          <UIcon name="i-lucide-plus" class="size-4" />
          <span>{{ t('nav.newTopic') }}</span>
        </NuxtLink>
      </nav>

      <!-- 语言切换 -->
      <div ref="langMenuRef" class="navbar__lang">
        <button
          class="navbar__lang-btn"
          :aria-label="t('nav.language')"
          @click="langMenuOpen = !langMenuOpen"
        >
          <UIcon name="i-lucide-globe" class="navbar__lang-icon" aria-hidden="true" />
          <span class="navbar__lang-text">{{ currentLocaleName }}</span>
          <UIcon
            name="i-lucide-chevron-down"
            class="navbar__chevron"
            :class="{ 'navbar__chevron--open': langMenuOpen }"
            aria-hidden="true"
          />
        </button>

        <!-- 语言选择下拉菜单 -->
        <Transition name="menu">
          <div v-if="langMenuOpen" class="navbar__dropdown" role="menu">
            <NuxtLink
              v-for="loc in locales"
              :key="loc.code"
              :to="switchLocalePath(loc.code)"
              class="navbar__dropdown-item navbar__dropdown-item--lang"
              :class="{ 'navbar__dropdown-item--active': locale === loc.code }"
              role="menuitem"
              @click="langMenuOpen = false"
            >
              <span>{{ loc.name }}</span>
              <UIcon
                v-if="locale === loc.code"
                name="i-lucide-check"
                class="navbar__selected-icon"
                aria-hidden="true"
              />
            </NuxtLink>
          </div>
        </Transition>
      </div>

      <!-- 夜间模式切换 -->
      <ClientOnly>
        <button
          type="button"
          class="navbar__theme-btn"
          :aria-label="themeToggleLabel"
          :aria-pressed="isDarkMode ? 'true' : 'false'"
          :title="themeToggleLabel"
          @click="toggleColorMode"
        >
          <UIcon :name="themeToggleIcon" class="navbar__theme-icon" aria-hidden="true" />
        </button>
        <template #fallback>
          <span class="navbar__theme-placeholder" aria-hidden="true" />
        </template>
      </ClientOnly>

      <!-- 右侧用户区 -->
      <div class="navbar__right">

        <!-- 未登录 -->
        <template v-if="!user">
          <NuxtLink :to="localePath('/login')" class="navbar__btn navbar__btn--ghost">
            {{ t('nav.login') }}
          </NuxtLink>
          <NuxtLink :to="localePath('/register')" class="navbar__btn navbar__btn--primary">
            {{ t('nav.register') }}
          </NuxtLink>
        </template>

        <!-- 已登录 -->
        <template v-else>
          <div ref="menuRef" class="navbar__user">
            <button
              class="navbar__avatar-btn"
              :aria-label="t('nav.userMenu')"
              :aria-expanded="menuOpen"
              @click="menuOpen = !menuOpen"
            >
              <span class="navbar__avatar">{{ avatarLetter }}</span>
              <span class="navbar__username">{{ displayName }}</span>
              <UIcon
                name="i-lucide-chevron-down"
                class="navbar__chevron"
                :class="{ 'navbar__chevron--open': menuOpen }"
                aria-hidden="true"
              />
            </button>

            <!-- 下拉菜单 -->
            <Transition name="menu">
              <div v-if="menuOpen" class="navbar__dropdown" role="menu">
                <div class="navbar__dropdown-header">
                  <span class="navbar__dropdown-name">{{ displayName }}</span>
                  <span class="navbar__dropdown-username">@{{ user.username }}</span>
                </div>
                <div class="navbar__dropdown-divider" />
                <NuxtLink
                  :to="localePath(`/u/${user.username}`)"
                  class="navbar__dropdown-item"
                  role="menuitem"
                  @click="menuOpen = false"
                >
                  <UIcon name="i-lucide-user" class="size-3.5" />
                  <span>{{ t('nav.myProfile') }}</span>
                </NuxtLink>
                <NuxtLink
                  :to="localePath('/settings/profile')"
                  class="navbar__dropdown-item"
                  role="menuitem"
                  @click="menuOpen = false"
                >
                  <UIcon name="i-lucide-settings" class="size-3.5" />
                  <span>{{ t('nav.profileSettings') }}</span>
                </NuxtLink>
                <div class="navbar__dropdown-divider" />
                <button
                  class="navbar__dropdown-item navbar__dropdown-item--danger"
                  role="menuitem"
                  @click="logout"
                >
                  {{ t('nav.logout') }}
                </button>
              </div>
            </Transition>
          </div>
        </template>

      </div>
    </div>
  </header>
</template>

<style scoped>
/* ====== Navbar 外框 ====== */
.navbar {
  position: sticky;
  top: 0;
  z-index: 50;
  height: 56px;
  background: #ffffff;
  border-top: 3px solid var(--sf-accent);
  border-bottom: 1px solid #e4e8ef;
  /* 轻微阴影，页面滚动时有层次感 */
  box-shadow: 0 1px 0 #e4e8ef;
}

.navbar__inner {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 100%;
  /* 与首页/详情页内容容器一致，保证 topbar 左右边缘对齐。 */
  max-width: 1376px;
  margin: 0 auto;
  padding: 0 16px;
}
/* sm 以上用 24px，和页面容器的 px-4 sm:px-6 对齐。 */
@media (min-width: 640px) {
  .navbar__inner {
    padding: 0 24px;
  }
}

/* ====== Logo ====== */
.navbar__logo {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  text-decoration: none;
  color: #111827;
  font-size: 15px;
  font-weight: 700;
  letter-spacing: -0.01em;
  flex-shrink: 0;
  margin-right: 8px;
}

.navbar__logo-mark {
  width: 26px;
  height: 26px;
  border-radius: 7px;
  background: transparent;
  border: 2px solid var(--sf-accent);
  color: var(--sf-accent);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
}

.navbar__logo-icon {
  width: 15px;
  height: 15px;
}

/* ====== 主导航 ====== */
.navbar__nav {
  display: flex;
  align-items: center;
  gap: 2px;
  flex: 1;
}

.navbar__nav-link {
  display: inline-flex;
  align-items: center;
  height: 32px;
  padding: 0 10px;
  border-radius: 6px;
  color: #4b5563;
  font-size: 14px;
  font-weight: 500;
  text-decoration: none;
  transition: color 0.15s, background 0.15s;
}

.navbar__nav-link:hover {
  color: #111827;
  background: #f3f4f6;
}

/* NuxtLink 激活状态 */
.navbar__nav-link.router-link-active {
  color: var(--sf-accent);
  background: var(--sf-accent-soft);
  font-weight: 600;
}

/* ====== 语言切换 ====== */
.navbar__lang {
  position: relative;
}

.navbar__lang-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 32px;
  padding: 0 8px;
  border: 1px solid transparent;
  border-radius: 6px;
  background: transparent;
  cursor: pointer;
  color: #4b5563;
  transition: background 0.15s, color 0.15s;
  font-family: inherit;
}

.navbar__lang-btn:hover {
  background: #f3f4f6;
  color: #111827;
}

.navbar__lang-btn svg {
  color: #6b7280;
  transition: color 0.15s;
}

.navbar__lang-btn:hover svg {
  color: #111827;
}

.navbar__lang-icon {
  width: 15px;
  height: 15px;
  color: #6b7280;
  transition: color 0.15s;
}

.navbar__lang-btn:hover .navbar__lang-icon {
  color: #111827;
}

.navbar__theme-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  border: 1px solid transparent;
  border-radius: 6px;
  background: transparent;
  color: #4b5563;
  cursor: pointer;
  transition: background 0.15s, color 0.15s, border-color 0.15s;
}

.navbar__theme-btn:hover {
  background: #f3f4f6;
  color: #111827;
}

.navbar__theme-icon {
  width: 15px;
  height: 15px;
}

.navbar__theme-placeholder {
  display: inline-block;
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
}

.navbar__lang-text {
  font-size: 13px;
  font-weight: 500;
  color: inherit;
}

/* 语言选项 */
.navbar__dropdown-item--lang {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.navbar__dropdown-item--active {
  color: var(--sf-accent);
  font-weight: 600;
  background: var(--sf-accent-soft);
}

.navbar__dropdown-item--active:hover {
  background: var(--sf-accent-soft);
  color: var(--sf-accent);
}

/* ====== 右侧区域 ====== */
.navbar__right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

/* 按钮 */
.navbar__btn {
  display: inline-flex;
  align-items: center;
  height: 32px;
  padding: 0 14px;
  border-radius: 7px;
  font-size: 13px;
  font-weight: 600;
  text-decoration: none;
  transition: background 0.15s, color 0.15s, box-shadow 0.15s;
  cursor: pointer;
  border: none;
  font-family: inherit;
}

.navbar__btn--ghost {
  background: transparent;
  color: #4b5563;
  border: 1px solid #d1d5db;
}

.navbar__btn--ghost:hover {
  background: #f9fafb;
  border-color: #9ca3af;
  color: #111827;
}

.navbar__btn--primary {
  background: var(--sf-accent);
  color: #ffffff;
}

.navbar__btn--primary:hover {
  background: var(--sf-accent-hover);
  box-shadow: 0 2px 8px rgb(var(--sf-accent-rgb) / 0.22);
}

/* ====== 用户头像按钮 ====== */
.navbar__user {
  position: relative;
}

.navbar__avatar-btn {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  height: 32px;
  padding: 0 10px 0 4px;
  border: 1px solid #e4e8ef;
  border-radius: 8px;
  background: #ffffff;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
  font-family: inherit;
}

.navbar__avatar-btn:hover {
  border-color: #d1d5db;
  background: #f9fafb;
}

.navbar__avatar {
  width: 24px;
  height: 24px;
  border-radius: 6px;
  background: var(--sf-accent);
  color: #ffffff;
  font-size: 11px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.navbar__username {
  font-size: 13px;
  font-weight: 600;
  color: #374151;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.navbar__chevron {
  width: 12px;
  height: 12px;
  color: #9ca3af;
  flex-shrink: 0;
  transition: transform 0.18s;
}

.navbar__selected-icon {
  width: 12px;
  height: 12px;
  flex-shrink: 0;
}

.navbar__chevron--open {
  transform: rotate(180deg);
}

/* ====== 下拉菜单 ====== */
.navbar__dropdown {
  position: absolute;
  top: calc(100% + 4px);
  right: 0;
  min-width: 150px;
  background: #ffffff;
  border: 1px solid #e4e8ef;
  border-radius: 10px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
  overflow: hidden;
  /* 防止点击内部元素触发 outside click */
  z-index: 100;
  padding: 4px;
}

.navbar__dropdown-header {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 12px 14px;
}

.navbar__dropdown-name {
  font-size: 13px;
  font-weight: 700;
  color: #111827;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.navbar__dropdown-username {
  font-size: 11px;
  color: #9ca3af;
}

.navbar__dropdown-divider {
  height: 1px;
  background: #f3f4f6;
  margin: 4px 0;
}

.navbar__dropdown-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 8px 12px;
  font-size: 13px;
  font-weight: 500;
  color: #374151;
  text-decoration: none;
  background: transparent;
  border: none;
  border-radius: 6px;
  text-align: left;
  cursor: pointer;
  font-family: inherit;
  transition: background 0.12s, color 0.12s;
}

.navbar__dropdown-item:hover {
  background: #f3f4f6;
  color: #111827;
}

.navbar__dropdown-item--danger {
  color: #b91c1c;
}

.navbar__dropdown-item--danger:hover {
  background: #fef2f2;
  color: #991b1b;
}

/* ====== 深色模式 ====== */
.dark .navbar {
  background: #09090b;
  border-bottom-color: #27272a;
  box-shadow: 0 1px 0 #27272a;
}

.dark .navbar__logo {
  color: #f4f4f5;
}

.dark .navbar__logo-mark {
  border-color: var(--sf-accent-dark);
  color: var(--sf-accent-dark);
}

.dark .navbar__nav-link,
.dark .navbar__lang-btn,
.dark .navbar__theme-btn {
  color: #d4d4d8;
}

.dark .navbar__nav-link:hover,
.dark .navbar__lang-btn:hover,
.dark .navbar__theme-btn:hover {
  background: #18181b;
  color: #ffffff;
}

.dark .navbar__nav-link.router-link-active {
  color: var(--sf-accent-dark);
  background: rgb(var(--sf-accent-rgb) / 0.2);
}

.dark .navbar__lang-btn svg,
.dark .navbar__lang-icon {
  color: #a1a1aa;
}

.dark .navbar__lang-btn:hover svg,
.dark .navbar__lang-btn:hover .navbar__lang-icon {
  color: #ffffff;
}

.dark .navbar__btn--ghost {
  border-color: #3f3f46;
  color: #d4d4d8;
}

.dark .navbar__btn--ghost:hover {
  background: #18181b;
  border-color: #52525b;
  color: #ffffff;
}

.dark .navbar__btn--primary {
  background: var(--sf-accent-dark);
  color: #052e2b;
}

.dark .navbar__btn--primary:hover {
  background: #5eead4;
}

.dark .navbar__avatar-btn {
  border-color: #27272a;
  background: #18181b;
}

.dark .navbar__avatar-btn:hover {
  border-color: #3f3f46;
  background: #27272a;
}

.dark .navbar__avatar {
  background: var(--sf-accent-dark);
  color: #052e2b;
}

.dark .navbar__username,
.dark .navbar__dropdown-name {
  color: #f4f4f5;
}

.dark .navbar__chevron,
.dark .navbar__dropdown-username {
  color: #a1a1aa;
}

.dark .navbar__dropdown {
  background: #18181b;
  border-color: #27272a;
  box-shadow: 0 18px 36px rgba(0, 0, 0, 0.32);
}

.dark .navbar__dropdown-divider {
  background: #27272a;
}

.dark .navbar__dropdown-item {
  color: #d4d4d8;
}

.dark .navbar__dropdown-item:hover {
  background: #27272a;
  color: #ffffff;
}

.dark .navbar__dropdown-item--active {
  color: var(--sf-accent-dark);
  background: rgb(var(--sf-accent-rgb) / 0.2);
}

.dark .navbar__dropdown-item--active:hover {
  color: var(--sf-accent-dark);
  background: rgb(var(--sf-accent-rgb) / 0.26);
}

.dark .navbar__dropdown-item--danger {
  color: #fca5a5;
}

.dark .navbar__dropdown-item--danger:hover {
  background: rgba(127, 29, 29, 0.24);
  color: #fecaca;
}

/* ====== 下拉动画 ====== */
.menu-enter-active,
.menu-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.menu-enter-from,
.menu-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

/* ====== 响应式 ====== */
@media (max-width: 640px) {
  .navbar__inner {
    padding: 0 16px;
  }

  .navbar__logo-text {
    display: none;
  }

  .navbar__username {
    display: none;
  }

  .navbar__btn--ghost {
    display: none;
  }

  /* 移动端仅保留地球图标 */
  .navbar__lang-text {
    display: none;
  }
}
</style>
