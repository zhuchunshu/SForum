<script setup lang="ts">
import type { CurrentUser } from '~/composables/useAuthSession'

definePageMeta({ layout: 'auth' })

const { t } = useI18n()
const localePath = useLocalePath()
const { request } = useApiClient()
const { user, setUser } = useAuthSession()
const { returnFromAuth } = useAuthReturnNavigation()
const { siteName } = useWebOptions()

if (user.value) {
  await returnFromAuth()
}

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
  let currentUser: CurrentUser

  try {
    currentUser = await request<CurrentUser>('/auth/login', {
      method: 'POST',
      body: {
        login: form.login,
        password: form.password
      }
    })
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('errors.loginFailed')
    return
  } finally {
    submitting.value = false
  }

  setUser(currentUser)
  await returnFromAuth()
}
</script>

<template>
  <main class="sg-auth-shell">
    <section class="sg-auth-story">
      <NuxtLink :to="localePath('/')" class="sg-brand">
        <span class="sg-brand__mark" aria-hidden="true">
          <UIcon name="i-lucide-sprout" class="sg-brand__icon" />
        </span>
        <span>{{ siteName }}</span>
      </NuxtLink>

      <div>
        <h1 class="sg-auth-story__title">
          {{ t('auth.brandHeadlineL1') }}<br />{{ t('auth.brandHeadlineL2') }}
        </h1>
        <p class="sg-auth-story__copy">
          {{ t('auth.brandDesc') }}
        </p>
        <ul class="sg-auth-feature-list">
          <li class="sg-auth-feature">
            <span class="sg-auth-feature__icon" aria-hidden="true">
              <UIcon name="i-lucide-check" class="sg-icon" />
            </span>
            {{ t('auth.feature1') }}
          </li>
          <li class="sg-auth-feature">
            <span class="sg-auth-feature__icon" aria-hidden="true">
              <UIcon name="i-lucide-check" class="sg-icon" />
            </span>
            {{ t('auth.feature2') }}
          </li>
          <li class="sg-auth-feature">
            <span class="sg-auth-feature__icon" aria-hidden="true">
              <UIcon name="i-lucide-check" class="sg-icon" />
            </span>
            {{ t('auth.feature3') }}
          </li>
        </ul>
      </div>

      <p class="sg-auth-help">© 2026 {{ siteName }}</p>
    </section>

    <section class="sg-auth-panel">
      <div class="sg-auth-form">
        <nav class="sg-auth-tabs" aria-label="登录或注册">
          <NuxtLink :to="localePath('/login')" class="sg-auth-tab sg-auth-tab--active">
            {{ t('auth.loginTitle') }}
          </NuxtLink>
          <NuxtLink :to="localePath('/register')" class="sg-auth-tab">
            {{ t('auth.registerTitle') }}
          </NuxtLink>
        </nav>

        <h2 class="sg-auth-title">{{ t('auth.loginHeading') }}</h2>
        <p class="sg-auth-subtitle">{{ t('auth.loginIntro') }}</p>

        <form @submit.prevent="submitLogin">
          <SFAlert
            v-if="errorMessage"
            :title="errorMessage"
            variant="danger"
            compact
            class="auth-alert"
          />

          <div class="sg-field">
            <label class="sg-label" for="login-input">{{ t('auth.loginName') }}</label>
            <input
              id="login-input"
              v-model="form.login"
              class="sg-input"
              type="text"
              name="login"
              :placeholder="t('auth.loginNamePlaceholder')"
              autocomplete="username"
              required
            />
          </div>

          <div class="sg-field">
            <div class="sg-field__head">
              <label class="sg-label" for="password-input">{{ t('auth.password') }}</label>
              <a href="#" class="sg-auth-help">{{ t('auth.forgotPassword') }}</a>
            </div>
            <input
              id="password-input"
              v-model="form.password"
              class="sg-input"
              type="password"
              name="password"
              placeholder="••••••••"
              autocomplete="current-password"
              required
            />
          </div>

          <button class="sg-auth-submit" type="submit" :disabled="submitting">
            {{ submitting ? t('auth.loggingIn') : t('auth.submitLogin') }}
          </button>
        </form>

        <p class="sg-auth-switch">
          {{ t('auth.needAccount') }}
          <NuxtLink :to="localePath('/register')">{{ t('auth.goRegister') }}</NuxtLink>
        </p>
      </div>
    </section>
  </main>
</template>
