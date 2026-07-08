<script setup lang="ts">
import type { AltchaWidgetElement } from 'altcha'
import type { CurrentUser } from '~/composables/useAuthSession'

definePageMeta({ layout: 'auth' })

type RegistrationStatus = {
  nextUserIsInitialSuperAdmin: boolean
}

const { t, locale } = useI18n()
const toast = useToast()
const localePath = useLocalePath()
const adminRoutes = useAdminRoutes()
const { apiBaseUrl, request } = useApiClient()
const { setUser, can } = useAuthSession()
const { siteName, humanVerificationEnabledFor, altchaWidgetSettings, passwordPolicy } = useWebOptions()

const form = reactive({
  username: '',
  email: '',
  password: '',
  displayName: ''
})
const submitting = ref(false)
const errorMessage = ref('')
const sessionUnavailable = ref(false)
const fieldErrors = ref<Record<string, string[]>>({})
const humanVerificationToken = ref('')
const altchaWidget = ref<AltchaWidgetElement | null>(null)
const altchaConfiguration = computed(() => JSON.stringify({
  hideLogo: altchaWidgetSettings.value.hideLogo,
  hideFooter: altchaWidgetSettings.value.hideFooter,
  minDuration: altchaWidgetSettings.value.minDuration
}))
const altchaWidgetType = computed(() => altchaWidgetSettings.value.type)
const altchaWidgetAuto = computed(() => altchaWidgetSettings.value.auto)
const altchaWidgetDisplay = computed(() => altchaWidgetSettings.value.display)
const altchaWidgetWorkers = computed(() => altchaWidgetSettings.value.workers)
const altchaWidgetClass = computed(() => [
  'auth-altcha',
  `auth-altcha--${altchaWidgetDisplay.value}`,
  fieldError('humanVerification') ? 'auth-altcha--invalid' : ''
].filter(Boolean))
const altchaChallengeUrl = computed(() => {
  return `${apiBaseUrl}/human-verification/challenge?purpose=register`
})
const registerHumanVerificationEnabled = computed(() => {
  // 前端只负责体验开关；API verifier 仍按同一场景配置做权威校验。
  return humanVerificationEnabledFor('register')
})
const humanVerificationEnabled = computed(() => {
  return registerHumanVerificationEnabled.value || Boolean(fieldError('humanVerification'))
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

function registerSuccessTitle() {
  const title = t('auth.registerSuccess')
  if (title !== 'auth.registerSuccess') {
    return title
  }
  return locale.value.toLowerCase().startsWith('en')
    ? 'Registration successful. Welcome aboard.'
    : '注册成功，欢迎加入。'
}

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

const passwordDescription = computed(() => {
  return ['password-hint', fieldDescription('password')].filter(Boolean).join(' ')
})
const passwordProgress = computed(() => passwordPolicyProgress(form.password, passwordPolicy.value))
const passwordRequirementRows = computed(() => {
  return passwordPolicyRequirements(form.password, passwordPolicy.value).map(item => ({
    ...item,
    label: passwordRequirementLabel(item.key)
  }))
})

function passwordRequirementLabel(key: string) {
  switch (key) {
    case 'lowercase':
      return t('auth.passwordRequirementLowercase')
    case 'uppercase':
      return t('auth.passwordRequirementUppercase')
    case 'number':
      return t('auth.passwordRequirementNumber')
    case 'symbol':
      return t('auth.passwordRequirementSymbol')
    default:
      return t('auth.passwordRequirementLength', {
        min: passwordPolicy.value.minLength,
        max: passwordPolicy.value.maxLength
      })
  }
}

async function submitRegister() {
  if (submitting.value) {
    return
  }

  errorMessage.value = ''
  sessionUnavailable.value = false
  fieldErrors.value = {}
  submitting.value = true
  const submittedHumanVerificationToken = humanVerificationEnabled.value ? humanVerificationToken.value : ''
  const body: Record<string, unknown> = {
    username: form.username,
    email: form.email,
    password: form.password,
    displayName: form.displayName,
    locale: locale.value
  }
  if (humanVerificationEnabled.value) {
    body.humanVerification = {
      provider: 'altcha',
      token: submittedHumanVerificationToken
    }
  }

  let currentUser: CurrentUser | null = null
  try {
    currentUser = await request<CurrentUser>('/auth/register', {
      method: 'POST',
      body
    })
  } catch (error) {
    fieldErrors.value = apiErrorFields(error)
    sessionUnavailable.value = apiErrorReason(error) === 'auth.session_unavailable'
    if (humanVerificationEnabled.value && (submittedHumanVerificationToken || fieldError('humanVerification'))) {
      resetHumanVerification()
    }
    errorMessage.value = registerErrorMessage(error, t)
  } finally {
    submitting.value = false
  }

  if (!currentUser) {
    return
  }

  setUser(currentUser)
  toast.add({
    color: 'success',
    icon: 'i-lucide-check',
    title: registerSuccessTitle(),
    duration: 10000
  })
  await navigateTo(can('admin.access') ? adminRoutes.path('/') : localePath('/'))
}
</script>

<template>
  <div class="auth-shell">

    <!-- 左侧品牌区 -->
    <div class="auth-left">
      <NuxtLink :to="localePath('/')" class="auth-logo">
        <span class="auth-logo-mark" aria-hidden="true">
          <UIcon name="i-lucide-message-circle" class="auth-logo-icon" />
        </span>
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
            <span class="auth-feature-icon" aria-hidden="true">
              <UIcon name="i-lucide-check" class="auth-feature-icon-svg" />
            </span>
            {{ t('auth.feature1') }}
          </li>
          <li class="auth-feature">
            <span class="auth-feature-icon" aria-hidden="true">
              <UIcon name="i-lucide-check" class="auth-feature-icon-svg" />
            </span>
            {{ t('auth.feature2') }}
          </li>
          <li class="auth-feature">
            <span class="auth-feature-icon" aria-hidden="true">
              <UIcon name="i-lucide-check" class="auth-feature-icon-svg" />
            </span>
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
          >
            <NuxtLink
              v-if="sessionUnavailable"
              :to="localePath('/login')"
              class="auth-alert-action"
            >
              {{ t('auth.goLogin') }}
            </NuxtLink>
          </SFAlert>

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
              :aria-describedby="passwordDescription"
            />
            <div id="password-hint" class="auth-password-policy">
              <div class="auth-password-policy__header">
                <span>{{ t('auth.passwordStrength') }}</span>
                <span>{{ passwordProgress }}%</span>
              </div>
              <div class="auth-password-policy__bar" aria-hidden="true">
                <span :style="{ width: `${passwordProgress}%` }" />
              </div>
              <p class="auth-field-hint">
                {{ t('auth.passwordPolicySummary', { min: passwordPolicy.minLength, max: passwordPolicy.maxLength }) }}
              </p>
              <ul class="auth-password-policy__list">
                <li v-for="item in passwordRequirementRows" :key="item.key" :class="{ 'is-met': item.met }">
                  <UIcon :name="item.met ? 'i-lucide-check' : 'i-lucide-circle'" class="auth-password-policy__icon" />
                  <span>{{ item.label }}</span>
                </li>
              </ul>
            </div>
            <p v-if="fieldError('password')" id="password-error" class="auth-field-message">
              {{ fieldError('password') }}
            </p>
          </div>

          <div v-if="humanVerificationEnabled" class="auth-field">
            <label id="human-verification-label" class="auth-label">
              {{ t('auth.humanVerification') }}
            </label>
            <ClientOnly>
              <altcha-widget
                ref="altchaWidget"
                :class="altchaWidgetClass"
                :challenge="altchaChallengeUrl"
                :configuration="altchaConfiguration"
                :auto="altchaWidgetAuto"
                :display="altchaWidgetDisplay"
                :language="locale === 'zh-CN' ? 'zh-cn' : 'en'"
                :type="altchaWidgetType"
                :workers="altchaWidgetWorkers"
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
  background: var(--sf-muted);
  border-right: 1px solid var(--sf-border);
}

.auth-logo {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  font-size: 15px;
  font-weight: 700;
  color: var(--sf-fg);
  letter-spacing: -0.01em;
  text-decoration: none;
}

.auth-logo-mark {
  width: 28px;
  height: 28px;
  border-radius: 7px;
  background: var(--sf-accent);
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
  color: var(--sf-fg);
  margin: 0;
}

.auth-desc {
  font-size: 15px;
  color: var(--sf-fg-secondary);
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
  color: var(--sf-fg-secondary);
  line-height: 1.5;
}

.auth-feature-icon {
  width: 20px;
  height: 20px;
  border-radius: 5px;
  border: 1px solid var(--sf-border);
  background: var(--sf-card);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  flex-shrink: 0;
  margin-top: 1px;
  color: var(--sf-accent);
}

.auth-left-footer {
  font-size: 12px;
  color: var(--sf-fg-tertiary);
  margin: 0;
}

/* ====== 右侧 ====== */
.auth-right {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 40px;
  background: var(--sf-card);
}

.auth-form-wrap {
  width: 100%;
  max-width: 380px;
}

.auth-tabs {
  display: flex;
  border-bottom: 1px solid var(--sf-border);
  margin-bottom: 32px;
}

.auth-tab {
  padding: 10px 0;
  margin-right: 24px;
  margin-bottom: -1px;
  border-bottom: 2px solid transparent;
  color: var(--sf-fg-tertiary);
  font-size: 14px;
  font-weight: 600;
  text-decoration: none;
  transition: color 0.18s, border-color 0.18s;
}

.auth-tab:hover { color: var(--sf-fg-secondary); }

.auth-tab--active {
  color: var(--sf-fg);
  border-bottom-color: var(--sf-accent);
}

.auth-form-title {
  font-size: 20px;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--sf-fg);
  margin: 0 0 5px;
}

.auth-form-sub {
  font-size: 13px;
  color: var(--sf-fg-secondary);
  margin: 0 0 26px;
  line-height: 1.55;
}

.auth-alert {
  margin-bottom: 16px;
}

.auth-alert-action {
  display: inline-flex;
  margin-top: 8px;
  color: #b91c1c;
  font-size: 12px;
  font-weight: 700;
  text-decoration: underline;
  text-underline-offset: 3px;
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
  color: var(--sf-fg-secondary);
}

.auth-label-optional {
  font-weight: 400;
  color: var(--sf-fg-tertiary);
}

.auth-input {
  height: 40px;
  padding: 0 13px;
  border: 1px solid var(--sf-border);
  border-radius: 7px;
  background: var(--sf-card);
  color: var(--sf-fg);
  font-size: 14px;
  font-family: inherit;
  outline: none;
  transition: border-color 0.18s, box-shadow 0.18s;
  width: 100%;
}

.auth-input:hover { border-color: var(--sf-fg-tertiary); }

.auth-input:focus {
  border-color: var(--sf-accent);
  box-shadow: 0 0 0 3px var(--sf-accent-focus);
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

.auth-field-hint {
  margin: 0;
  color: var(--sf-fg-tertiary);
  font-size: 12px;
  line-height: 1.45;
}

.auth-password-policy {
  display: grid;
  gap: 7px;
  margin-top: 1px;
}

.auth-password-policy__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--sf-fg-secondary);
  font-size: 12px;
  font-weight: 600;
}

.auth-password-policy__bar {
  height: 5px;
  overflow: hidden;
  border-radius: 999px;
  background: var(--sf-muted);
  border: 1px solid var(--sf-border);
}

.auth-password-policy__bar span {
  display: block;
  height: 100%;
  min-width: 0;
  border-radius: inherit;
  background: var(--sf-accent);
  transition: width 0.18s ease;
}

.auth-password-policy__list {
  display: grid;
  gap: 5px;
  margin: 0;
  padding: 0;
  list-style: none;
  color: var(--sf-fg-tertiary);
  font-size: 12px;
  line-height: 1.4;
}

.auth-password-policy__list li {
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 18px;
}

.auth-password-policy__list li.is-met {
  color: var(--sf-accent);
}

.auth-password-policy__icon {
  width: 13px;
  height: 13px;
  flex: 0 0 13px;
}

/* 覆盖 WebKit 自动填充默认底色，保持注册表单跟随当前主题表面。 */
.auth-input:-webkit-autofill,
.auth-input:-webkit-autofill:hover,
.auth-input:-webkit-autofill:active {
  -webkit-box-shadow: 0 0 0 1000px var(--sf-card) inset;
  box-shadow: 0 0 0 1000px var(--sf-card) inset;
  -webkit-text-fill-color: var(--sf-fg);
  caret-color: var(--sf-fg);
  transition: background-color 9999s ease-out, color 9999s ease-out;
}

.auth-input:-webkit-autofill:focus {
  border-color: var(--sf-accent);
  -webkit-box-shadow: 0 0 0 1000px var(--sf-card) inset, 0 0 0 3px var(--sf-accent-focus);
  box-shadow: 0 0 0 1000px var(--sf-card) inset, 0 0 0 3px var(--sf-accent-focus);
  -webkit-text-fill-color: var(--sf-fg);
  caret-color: var(--sf-fg);
}

.auth-input::placeholder { color: var(--sf-fg-tertiary); }

.auth-altcha {
  width: 100%;
  --altcha-max-width: 100%;
  --altcha-border-radius: 7px;
  --altcha-border-color: var(--sf-border);
  --altcha-color-primary: var(--sf-accent);
  --altcha-color-primary-content: #ffffff;
  --altcha-color-success: var(--sf-accent);
}

.auth-altcha--invalid {
  --altcha-border-color: #dc2626;
}

.auth-altcha-fallback {
  min-height: 52px;
  display: flex;
  align-items: center;
  padding: 0 13px;
  border: 1px solid var(--sf-border);
  border-radius: 7px;
  color: var(--sf-fg-tertiary);
  font-size: 13px;
}

.auth-btn {
  display: block;
  width: 100%;
  height: 40px;
  border: none;
  border-radius: 7px;
  background: var(--sf-accent);
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
  background: var(--sf-accent-hover);
  box-shadow: 0 4px 14px rgb(var(--sf-accent-rgb) / 0.25);
}

.auth-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.auth-terms {
  margin-top: 14px;
  font-size: 12px;
  color: var(--sf-fg-tertiary);
  text-align: center;
  line-height: 1.6;
}

.auth-terms a {
  color: var(--sf-fg-secondary);
  text-decoration: none;
  border-bottom: 1px solid var(--sf-border);
}

.auth-switch {
  margin-top: 14px;
  font-size: 13px;
  color: var(--sf-fg-tertiary);
  text-align: center;
}

.auth-switch a {
  color: var(--sf-fg-secondary);
  font-weight: 600;
  text-decoration: none;
  border-bottom: 1px solid var(--sf-border);
  transition: color 0.15s, border-color 0.15s;
}

.auth-switch a:hover {
  color: var(--sf-fg);
  border-bottom-color: var(--sf-fg-tertiary);
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
