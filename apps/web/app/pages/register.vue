<script setup lang="ts">
import type { AltchaWidgetElement } from 'altcha'
import type { CurrentUser } from '~/composables/useAuthSession'

definePageMeta({ layout: 'auth' })

type RegistrationStatus = {
  nextUserIsInitialSuperAdmin: boolean
}

const { t, locale } = useI18n()
const localePath = useLocalePath()
const adminRoutes = useAdminRoutes()
const { apiBaseUrl, request } = useApiClient()
const { refresh, can } = useAuthSession()
const { siteName } = useWebOptions()

const form = reactive({
  username: '',
  email: '',
  password: '',
  displayName: ''
})
const submitting = ref(false)
const errorMessage = ref('')
const fieldErrors = ref<Record<string, string[]>>({})
const humanVerificationToken = ref('')
const altchaWidget = ref<AltchaWidgetElement | null>(null)
// ALTCHA 仅从 configuration JSON 读取 hideLogo/hideFooter，自动刷新时也会沿用。
const altchaConfiguration = JSON.stringify({
  hideLogo: true,
  hideFooter: true
})
const { data: registrationStatus } = await useAsyncData('auth-registration-status', async () => {
  try {
    return await request<RegistrationStatus>('/auth/registration-status')
  } catch {
    return { nextUserIsInitialSuperAdmin: false }
  }
})
const isBootstrapRegistration = computed(() => registrationStatus.value?.nextUserIsInitialSuperAdmin === true)

useSeoMeta({
  title: t('auth.registerTitle')
})

function handleAltchaVerified(event: Event) {
  const detail = (event as CustomEvent<{ payload?: string }>).detail
  humanVerificationToken.value = detail?.payload || ''
}

function handleAltchaStateChange(event: Event) {
  const detail = (event as CustomEvent<{ state?: string, payload?: string }>).detail
  if (detail?.state === 'verified' && detail.payload) {
    humanVerificationToken.value = detail.payload
    return
  }
  if (detail?.state === 'expired' || detail?.state === 'error' || detail?.state === 'unverified') {
    humanVerificationToken.value = ''
  }
}

function resetHumanVerification() {
  humanVerificationToken.value = ''
  altchaWidget.value?.reset()
}

function fieldError(name: string) {
  return fieldErrors.value[name]?.[0] || ''
}

function fieldDescription(name: string) {
  return fieldError(name) ? `${name}-error` : undefined
}

function registerErrorMessage(error: unknown) {
  const message = apiErrorMessage(error)
  if (message) {
    return message
  }

  switch (apiErrorReason(error)) {
    case 'human_verification.required':
      return t('errors.humanVerificationRequired')
    case 'human_verification.invalid':
      return t('errors.humanVerificationInvalid')
    case 'human_verification.expired':
      return t('errors.humanVerificationExpired')
    case 'human_verification.replayed':
      return t('errors.humanVerificationReplayed')
    case 'rate_limit.exceeded':
      return t('errors.rateLimited')
    default:
      return t('errors.registerFailed')
  }
}

async function submitRegister() {
  errorMessage.value = ''
  fieldErrors.value = {}
  submitting.value = true
  const submittedHumanVerificationToken = humanVerificationToken.value

  try {
    await request<CurrentUser>('/auth/register', {
      method: 'POST',
      body: {
        username: form.username,
        email: form.email,
        password: form.password,
        displayName: form.displayName,
        locale: locale.value,
        humanVerification: {
          provider: 'altcha',
          token: submittedHumanVerificationToken
        }
      }
    })
    await refresh()
    await navigateTo(can('admin.access') ? adminRoutes.path('/') : localePath('/'))
  } catch (error) {
    fieldErrors.value = apiErrorFields(error)
    if (submittedHumanVerificationToken || fieldError('humanVerification')) {
      resetHumanVerification()
    }
    errorMessage.value = registerErrorMessage(error)
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
        {{ siteName }}
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
        © 2026 {{ siteName }}
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
            v-if="isBootstrapRegistration"
            :title="t('auth.firstUserAdminNotice')"
            variant="warning"
            compact
            class="auth-alert"
          />

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
              :class="['auth-input', fieldError('username') ? 'auth-input--invalid' : '']"
              type="text"
              name="username"
              :placeholder="t('auth.usernamePlaceholder')"
              autocomplete="username"
              required
              :aria-invalid="fieldError('username') ? 'true' : undefined"
              :aria-describedby="fieldDescription('username')"
            />
            <p v-if="fieldError('username')" id="username-error" class="auth-field-message">
              {{ fieldError('username') }}
            </p>
          </div>

          <div class="auth-field">
            <label class="auth-label" for="email-input">
              {{ t('auth.email') }}
            </label>
            <input
              id="email-input"
              v-model="form.email"
              :class="['auth-input', fieldError('email') ? 'auth-input--invalid' : '']"
              type="email"
              name="email"
              :placeholder="t('auth.emailPlaceholder')"
              autocomplete="email"
              required
              :aria-invalid="fieldError('email') ? 'true' : undefined"
              :aria-describedby="fieldDescription('email')"
            />
            <p v-if="fieldError('email')" id="email-error" class="auth-field-message">
              {{ fieldError('email') }}
            </p>
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
              :class="['auth-input', fieldError('password') ? 'auth-input--invalid' : '']"
              type="password"
              name="password"
              :placeholder="t('auth.passwordPlaceholder')"
              autocomplete="new-password"
              required
              :aria-invalid="fieldError('password') ? 'true' : undefined"
              :aria-describedby="fieldDescription('password')"
            />
            <p v-if="fieldError('password')" id="password-error" class="auth-field-message">
              {{ fieldError('password') }}
            </p>
          </div>

          <div class="auth-field">
            <label id="human-verification-label" class="auth-label">
              {{ t('auth.humanVerification') }}
            </label>
            <ClientOnly>
              <altcha-widget
                ref="altchaWidget"
                :class="['auth-altcha', fieldError('humanVerification') ? 'auth-altcha--invalid' : '']"
                :challenge="`${apiBaseUrl}/human-verification/challenge?purpose=register`"
                :configuration="altchaConfiguration"
                :language="locale === 'zh-CN' ? 'zh-cn' : 'en'"
                type="checkbox"
                :aria-invalid="fieldError('humanVerification') ? 'true' : undefined"
                :aria-labelledby="'human-verification-label'"
                :aria-describedby="fieldDescription('humanVerification')"
                @verified="handleAltchaVerified"
                @expired="resetHumanVerification"
                @statechange="handleAltchaStateChange"
              />
              <template #fallback>
                <div class="auth-altcha-fallback">
                  {{ t('auth.humanVerificationLoading') }}
                </div>
              </template>
            </ClientOnly>
            <p v-if="fieldError('humanVerification')" id="humanVerification-error" class="auth-field-message">
              {{ fieldError('humanVerification') }}
            </p>
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

.auth-input--invalid,
.auth-input--invalid:hover,
.auth-input--invalid:focus {
  border-color: #dc2626;
}

.auth-input--invalid:focus {
  box-shadow: 0 0 0 3px rgba(220, 38, 38, 0.12);
}

.auth-field-message {
  margin: 0;
  color: #b91c1c;
  font-size: 12px;
  line-height: 1.45;
}

/* 覆盖 WebKit 自动填充默认底色，保持注册表单白底。 */
.auth-input:-webkit-autofill,
.auth-input:-webkit-autofill:hover,
.auth-input:-webkit-autofill:active {
  -webkit-box-shadow: 0 0 0 1000px #fff inset;
  box-shadow: 0 0 0 1000px #fff inset;
  -webkit-text-fill-color: #111827;
  caret-color: #111827;
  transition: background-color 9999s ease-out, color 9999s ease-out;
}

.auth-input:-webkit-autofill:focus {
  border-color: #0f766e;
  -webkit-box-shadow: 0 0 0 1000px #fff inset, 0 0 0 3px rgba(15, 118, 110, 0.14);
  box-shadow: 0 0 0 1000px #fff inset, 0 0 0 3px rgba(15, 118, 110, 0.14);
  -webkit-text-fill-color: #111827;
  caret-color: #111827;
}

.auth-input::placeholder { color: #d1d5db; }

.auth-altcha {
  width: 100%;
  --altcha-max-width: 100%;
  --altcha-border-radius: 7px;
  --altcha-border-color: #d1d5db;
  --altcha-color-primary: #0f766e;
  --altcha-color-primary-content: #ffffff;
  --altcha-color-success: #0f766e;
}

.auth-altcha--invalid {
  --altcha-border-color: #dc2626;
}

.auth-altcha-fallback {
  min-height: 52px;
  display: flex;
  align-items: center;
  padding: 0 13px;
  border: 1px solid #d1d5db;
  border-radius: 7px;
  color: #6b7280;
  font-size: 13px;
}

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
