<script setup lang="ts">
import type { CurrentUser } from '~/composables/useAuthSession'

const { t, locale } = useI18n()
const localePath = useLocalePath()
const apiBaseUrl = useRuntimeConfig().public.apiBaseUrl as string
const { refresh, can } = useAuthSession()

const form = reactive({
  username: '',
  email: '',
  password: '',
  displayName: ''
})
const submitting = ref(false)
const errorMessage = ref('')

useSeoMeta({
  title: t('auth.registerTitle')
})

async function submitRegister() {
  errorMessage.value = ''
  submitting.value = true

  try {
    await $fetch<CurrentUser>(`${apiBaseUrl}/auth/register`, {
      method: 'POST',
      credentials: 'include',
      body: {
        username: form.username,
        email: form.email,
        password: form.password,
        displayName: form.displayName,
        locale: locale.value
      }
    })
    await refresh()
    await navigateTo(localePath(can('admin.access') ? '/admin' : '/'))
  } catch {
    errorMessage.value = t('errors.registerFailed')
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
          <NuxtLink :to="localePath('/login')" class="auth-tab">
            {{ t('auth.loginTitle') }}
          </NuxtLink>
          <NuxtLink :to="localePath('/register')" class="auth-tab auth-tab--active">
            {{ t('auth.registerTitle') }}
          </NuxtLink>
        </nav>

        <h2 class="auth-form-title">{{ t('auth.registerHeading') }}</h2>
        <p class="auth-form-sub">{{ t('auth.registerIntro') }}</p>

        <form @submit.prevent="submitRegister">
          <SFAlert
            v-if="errorMessage"
            :title="errorMessage"
            variant="danger"
            compact
            class="auth-alert"
          />

          <div class="auth-field">
            <label class="auth-label" for="username-input">
              {{ t('auth.username') }}
            </label>
            <input
              id="username-input"
              v-model="form.username"
              class="auth-input"
              type="text"
              name="username"
              :placeholder="t('auth.usernamePlaceholder')"
              autocomplete="username"
              required
            />
          </div>

          <div class="auth-field">
            <label class="auth-label" for="email-input">
              {{ t('auth.email') }}
            </label>
            <input
              id="email-input"
              v-model="form.email"
              class="auth-input"
              type="email"
              name="email"
              :placeholder="t('auth.emailPlaceholder')"
              autocomplete="email"
              required
            />
          </div>

          <div class="auth-field">
            <label class="auth-label" for="displayname-input">
              {{ t('auth.displayName') }}
              <span class="auth-label-optional">（{{ t('auth.optional') }}）</span>
            </label>
            <input
              id="displayname-input"
              v-model="form.displayName"
              class="auth-input"
              type="text"
              name="displayName"
              :placeholder="t('auth.displayNamePlaceholder')"
            />
          </div>

          <div class="auth-field">
            <label class="auth-label" for="reg-password-input">
              {{ t('auth.password') }}
            </label>
            <input
              id="reg-password-input"
              v-model="form.password"
              class="auth-input"
              type="password"
              name="password"
              :placeholder="t('auth.passwordPlaceholder')"
              autocomplete="new-password"
              required
            />
          </div>

          <button class="auth-btn" type="submit" :disabled="submitting">
            {{ submitting ? t('auth.registering') : t('auth.submitRegister') }}
          </button>
        </form>

        <p class="auth-terms">
          {{ t('auth.agreeTo') }}
          <a href="#">{{ t('auth.terms') }}</a>
          {{ t('auth.and') }}
          <a href="#">{{ t('auth.privacy') }}</a>
        </p>

        <p class="auth-switch">
          {{ t('auth.haveAccount') }}
          <NuxtLink :to="localePath('/login')">{{ t('auth.goLogin') }}</NuxtLink>
        </p>

      </div>
    </div>

  </div>
</template>

<style scoped>
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

.auth-alert {
  margin-bottom: 16px;
}

.auth-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 14px;
}

.auth-label {
  font-size: 12px;
  font-weight: 600;
  color: #374151;
}

.auth-label-optional {
  font-weight: 400;
  color: #9ca3af;
}

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

.auth-terms {
  margin-top: 14px;
  font-size: 12px;
  color: #9ca3af;
  text-align: center;
  line-height: 1.6;
}

.auth-terms a {
  color: #4b5563;
  text-decoration: none;
  border-bottom: 1px solid #e5e7eb;
}

.auth-switch {
  margin-top: 14px;
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
