<script setup lang="ts">
const { t } = useI18n()
const localePath = useLocalePath()
const { user, refresh } = useAuthSession()
const apiBaseUrl = useRuntimeConfig().public.apiBaseUrl as string
const router = useRouter()

// 控制用户下拉菜单的显示
const menuOpen = ref(false)
const menuRef = ref<HTMLElement | null>(null)

// 点击页面其他区域关闭菜单
onMounted(() => {
  document.addEventListener('click', onClickOutside)
})
onUnmounted(() => {
  document.removeEventListener('click', onClickOutside)
})
function onClickOutside(e: MouseEvent) {
  if (menuRef.value && !menuRef.value.contains(e.target as Node)) {
    menuOpen.value = false
  }
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
</script>

<template>
  <header class="navbar">
    <div class="navbar__inner">

      <!-- Logo -->
      <NuxtLink :to="localePath('/')" class="navbar__logo">
        <div class="navbar__logo-mark">💬</div>
        <span class="navbar__logo-text">SForum</span>
      </NuxtLink>

      <!-- 主导航（占位，后续扩展版块） -->
      <nav class="navbar__nav" :aria-label="t('nav.mainNav')">
        <NuxtLink :to="localePath('/')" class="navbar__nav-link">
          {{ t('nav.home') }}
        </NuxtLink>
      </nav>

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
              <!-- 箭头图标 -->
              <svg
                class="navbar__chevron"
                :class="{ 'navbar__chevron--open': menuOpen }"
                width="12" height="12"
                viewBox="0 0 12 12"
                fill="none"
                aria-hidden="true"
              >
                <path d="M2 4l4 4 4-4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
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
                  :to="localePath('/admin')"
                  class="navbar__dropdown-item"
                  role="menuitem"
                  @click="menuOpen = false"
                >
                  {{ t('nav.admin') }}
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
  height: 52px;
  background: #ffffff;
  border-bottom: 1px solid #e4e8ef;
  /* 轻微阴影，页面滚动时有层次感 */
  box-shadow: 0 1px 0 #e4e8ef;
}

.navbar__inner {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 100%;
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 24px;
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
  background: #0f766e;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
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
  color: #0f766e;
  background: #f0faf9;
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
  background: #0f766e;
  color: #ffffff;
}

.navbar__btn--primary:hover {
  background: #0b5f59;
  box-shadow: 0 2px 8px rgba(15, 118, 110, 0.22);
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
  background: #0f766e;
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
  color: #9ca3af;
  flex-shrink: 0;
  transition: transform 0.18s;
}

.navbar__chevron--open {
  transform: rotate(180deg);
}

/* ====== 下拉菜单 ====== */
.navbar__dropdown {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  min-width: 180px;
  background: #ffffff;
  border: 1px solid #e4e8ef;
  border-radius: 10px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
  overflow: hidden;
  /* 防止点击内部元素触发 outside click */
  z-index: 100;
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
}

.navbar__dropdown-item {
  display: block;
  width: 100%;
  padding: 9px 14px;
  font-size: 13px;
  font-weight: 500;
  color: #374151;
  text-decoration: none;
  background: transparent;
  border: none;
  text-align: left;
  cursor: pointer;
  font-family: inherit;
  transition: background 0.12s, color 0.12s;
}

.navbar__dropdown-item:hover {
  background: #f9fafb;
  color: #111827;
}

.navbar__dropdown-item--danger {
  color: #b91c1c;
}

.navbar__dropdown-item--danger:hover {
  background: #fef2f2;
  color: #991b1b;
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
}
</style>
