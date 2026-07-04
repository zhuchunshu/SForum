<script setup lang="ts">
import type { CurrentUser } from '~/composables/useAuthSession'

definePageMeta({ layout: 'auth' })

const { t } = useI18n()
const localePath = useLocalePath()
const adminRoutes = useAdminRoutes()
const { request } = useApiClient()
const { refresh, can } = useAuthSession()

const form = reactive({
  login: '',
  password: ''
})
const submitting = ref(false)
const errorMessage = ref('')

useSeoMeta({
  title: t('auth.loginTitle')
})

async function submitLogin() {
  errorMessage.value = ''
  submitting.value = true

  try {
    await request<CurrentUser>('/auth/login', {
      method: 'POST',
      body: {
        login: form.login,
        password: form.password
      }
    })
    await refresh()
    await navigateTo(can('admin.access') ? adminRoutes.path('/') : localePath('/'))
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('errors.loginFailed')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="auth-shell">

    <!-- 左侧品牌区 -->
    <div class="auth-left">
      <NuxtLink :to="localePath('/')" class="auth-logo">
        <div class="auth-logo-mark">💬</div>
        SForum
      </NuxtLink>

      <div class="auth-left-body">
        <h1 class="auth-headline">
          {{ t('auth.brandHeadlineL1') }}<br />{{ t('auth.brandHeadlineL2') }}
        </h1>
        <p class="auth-desc">
          {{ t('auth.brandDesc') }}
        </p>
        <ul class="auth-features">
          <li class="auth-feature">
            <span class="auth-feature-icon">✓</span>
            {{ t('auth.feature1') }}
          </li>
          <li class="auth-feature">
            <span class="auth-feature-icon">✓</span>
            {{ t('auth.feature2') }}
          </li>
          <li class="auth-feature">
            <span class="auth-feature-icon">✓</span>
            {{ t('auth.feature3') }}
          </li>
        </ul>
      </div>

      <p class="auth-left-footer">
        © 2026 SForum
      </p>
    </div>

    <!-- 右侧表单区 -->
    <div class="auth-right">
      <div class="auth-form-wrap">

        <!-- 页面导航 tab -->
        <nav class="auth-tabs" aria-label="登录或注册">
          <NuxtLink :to="localePath('/login')" class="auth-tab auth-tab--active">
            {{ t('auth.loginTitle') }}
          </NuxtLink>
          <NuxtLink :to="localePath('/register')" class="auth-tab">
            {{ t('auth.registerTitle') }}
          </NuxtLink>
        </nav>

        <h2 class="auth-form-title">{{ t('auth.loginHeading') }}</h2>
        <p class="auth-form-sub">{{ t('auth.loginIntro') }}</p>

        <form @submit.prevent="submitLogin">
          <SFAlert
            v-if="errorMessage"
            :title="errorMessage"
            variant="danger"
            compact
            class="auth-alert"
          />

          <div class="auth-field">
            <label class="auth-label" for="login-input">
              {{ t('auth.loginName') }}
            </label>
            <input
              id="login-input"
              v-model="form.login"
              class="auth-input"
              type="text"
              name="login"
              :placeholder="t('auth.loginNamePlaceholder')"
              autocomplete="username"
              required
            />
          </div>

          <div class="auth-field">
            <div class="auth-field-head">
              <label class="auth-label" for="password-input">
                {{ t('auth.password') }}
              </label>
              <a href="#" class="auth-forgot">{{ t('auth.forgotPassword') }}</a>
            </div>
            <input
              id="password-input"
              v-model="form.password"
              class="auth-input"
              type="password"
              name="password"
              placeholder="••••••••"
              autocomplete="current-password"
              required
            />
          </div>

          <button class="auth-btn" type="submit" :disabled="submitting">
            {{ submitting ? t('auth.loggingIn') : t('auth.submitLogin') }}
          </button>
        </form>

        <p class="auth-switch">
          {{ t('auth.needAccount') }}
          <NuxtLink :to="localePath('/register')">{{ t('auth.goRegister') }}</NuxtLink>
        </p>

      </div>
    </div>

  </div>
</template>

<style scoped>
/* 整体双栏布局 */
.auth-shell {
  display: grid;
  grid-template-columns: 1fr 1fr;
  min-height: 100vh;
}

/* ====== 左侧 ====== */
.auth-left {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: 48px 52px;
  background: #f5f7fa;
  border-right: 1px solid #e4e8ef;
}

.auth-logo {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  font-size: 15px;
  font-weight: 700;
  color: #111827;
  letter-spacing: -0.01em;
  text-decoration: none;
}

.auth-logo-mark {
  width: 28px;
  height: 28px;
  border-radius: 7px;
  background: #0f766e;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
}

.auth-left-body {
  display: flex;
  flex-direction: column;
  gap: 28px;
}

.auth-headline {
  font-size: clamp(26px, 2.6vw, 40px);
  font-weight: 700;
  letter-spacing: -0.03em;
  line-height: 1.15;
  color: #111827;
  margin: 0;
}

.auth-desc {
  font-size: 15px;
  color: #4b5563;
  line-height: 1.75;
  margin: 0;
  max-width: 340px;
}

.auth-features {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 13px;
  padding: 0;
  margin: 0;
}

.auth-feature {
  display: flex;
  align-items: flex-start;
  gap: 11px;
  font-size: 13.5px;
  color: #4b5563;
  line-height: 1.5;
}

.auth-feature-icon {
  width: 20px;
  height: 20px;
  border-radius: 5px;
  border: 1px solid #d1d5db;
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  flex-shrink: 0;
  margin-top: 1px;
  color: #0f766e;
}

.auth-left-footer {
  font-size: 12px;
  color: #9ca3af;
  margin: 0;
}

/* ====== 右侧 ====== */
.auth-right {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 40px;
  background: #ffffff;
}

.auth-form-wrap {
  width: 100%;
  max-width: 380px;
}

/* 页面 tab 导航 */
.auth-tabs {
  display: flex;
  border-bottom: 1px solid #e4e8ef;
  margin-bottom: 32px;
}

.auth-tab {
  padding: 10px 0;
  margin-right: 24px;
  margin-bottom: -1px;
  border-bottom: 2px solid transparent;
  color: #9ca3af;
  font-size: 14px;
  font-weight: 600;
  text-decoration: none;
  transition: color 0.18s, border-color 0.18s;
}

.auth-tab:hover { color: #4b5563; }

/* 当前页高亮：login 页永远高亮登录 tab */
.auth-tab--active {
  color: #111827;
  border-bottom-color: #0f766e;
}

.auth-form-title {
  font-size: 20px;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: #111827;
  margin: 0 0 5px;
}

.auth-form-sub {
  font-size: 13px;
  color: #4b5563;
  margin: 0 0 26px;
  line-height: 1.55;
}

/* 错误提示 */
.auth-alert {
  margin-bottom: 16px;
}

/* 表单字段 */
.auth-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 14px;
}

.auth-field-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.auth-label {
  font-size: 12px;
  font-weight: 600;
  color: #374151;
}

.auth-forgot {
  font-size: 12px;
  color: #9ca3af;
  text-decoration: none;
  transition: color 0.15s;
}

.auth-forgot:hover { color: #4b5563; }

.auth-input {
  height: 40px;
  padding: 0 13px;
  border: 1px solid #d1d5db;
  border-radius: 7px;
  background: #fff;
  color: #111827;
  font-size: 14px;
  font-family: inherit;
  outline: none;
  transition: border-color 0.18s, box-shadow 0.18s;
  width: 100%;
}

.auth-input:hover { border-color: #9ca3af; }

.auth-input:focus {
  border-color: #0f766e;
  box-shadow: 0 0 0 3px rgba(15, 118, 110, 0.14);
}

.auth-input::placeholder { color: #d1d5db; }

/* 提交按钮 */
.auth-btn {
  display: block;
  width: 100%;
  height: 40px;
  border: none;
  border-radius: 7px;
  background: #0f766e;
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  font-family: inherit;
  cursor: pointer;
  letter-spacing: 0.01em;
  transition: background 0.18s, box-shadow 0.18s;
  margin-top: 8px;
}

.auth-btn:hover:not(:disabled) {
  background: #0b5f59;
  box-shadow: 0 4px 14px rgba(15, 118, 110, 0.25);
}

.auth-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* 底部切换提示 */
.auth-switch {
  margin-top: 20px;
  font-size: 13px;
  color: #9ca3af;
  text-align: center;
}

.auth-switch a {
  color: #4b5563;
  font-weight: 600;
  text-decoration: none;
  border-bottom: 1px solid #d1d5db;
  transition: color 0.15s, border-color 0.15s;
}

.auth-switch a:hover {
  color: #111827;
  border-bottom-color: #9ca3af;
}

@media (max-width: 720px) {
  .auth-shell {
    grid-template-columns: 1fr;
  }

  .auth-left {
    display: none;
  }

  .auth-right {
    min-height: 100vh;
  }
}
</style>
