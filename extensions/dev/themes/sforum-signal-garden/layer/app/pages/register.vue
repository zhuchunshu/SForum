<script setup lang="ts">
import type { AltchaWidgetElement } from 'altcha'
import type { CurrentUser } from '~/composables/useAuthSession'

definePageMeta({ layout: 'auth' })

type RegistrationStatus = {
  nextUserIsInitialSuperAdmin: boolean
}

const { t, locale } = useI18n()
const localePath = useLocalePath()
const { apiBaseUrl, request } = useApiClient()
const { user, setUser } = useAuthSession()
const { returnFromAuth } = useAuthReturnNavigation()
const { siteName, humanVerificationEnabledFor, altchaWidgetSettings } = useWebOptions()

if (user.value) {
  await returnFromAuth()
}

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
const altchaChallengeUrl = computed(() => `${apiBaseUrl}/human-verification/challenge?purpose=register`)
const humanVerificationEnabled = computed(() => {
  return humanVerificationEnabledFor('register') || Boolean(fieldError('humanVerification'))
})
const { data: registrationStatus } = await useAsyncData('auth-registration-status-signal-garden', async () => {
  try {
    return await request<RegistrationStatus>('/auth/registration-status')
  } catch {
    return { nextUserIsInitialSuperAdmin: false }
  }
})
const isBootstrapRegistration = computed(() => registrationStatus.value?.nextUserIsInitialSuperAdmin === true)
const passwordDescription = computed(() => {
  return ['password-hint', fieldDescription('password')].filter(Boolean).join(' ')
})

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
        <p class="sg-auth-story__copy">{{ t('auth.brandDesc') }}</p>
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
          <NuxtLink :to="localePath('/login')" class="sg-auth-tab">
            {{ t('auth.loginTitle') }}
          </NuxtLink>
          <NuxtLink :to="localePath('/register')" class="sg-auth-tab sg-auth-tab--active">
            {{ t('auth.registerTitle') }}
          </NuxtLink>
        </nav>

        <h2 class="sg-auth-title">{{ t('auth.registerHeading') }}</h2>
        <p class="sg-auth-subtitle">{{ t('auth.registerIntro') }}</p>

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
            <NuxtLink v-if="sessionUnavailable" :to="localePath('/login')" class="sg-auth-help">
              {{ t('auth.goLogin') }}
            </NuxtLink>
          </SFAlert>

          <div class="sg-field">
            <label class="sg-label" for="username-input">{{ t('auth.username') }}</label>
            <input
              id="username-input"
              v-model="form.username"
              :class="['sg-input', fieldError('username') ? 'sg-input--invalid' : '']"
              type="text"
              name="username"
              :placeholder="t('auth.usernamePlaceholder')"
              autocomplete="username"
              required
              :aria-invalid="fieldError('username') ? 'true' : undefined"
              :aria-describedby="fieldDescription('username')"
            />
            <p v-if="fieldError('username')" id="username-error" class="sg-field-message">
              {{ fieldError('username') }}
            </p>
          </div>

          <div class="sg-field">
            <label class="sg-label" for="email-input">{{ t('auth.email') }}</label>
            <input
              id="email-input"
              v-model="form.email"
              :class="['sg-input', fieldError('email') ? 'sg-input--invalid' : '']"
              type="email"
              name="email"
              :placeholder="t('auth.emailPlaceholder')"
              autocomplete="email"
              required
              :aria-invalid="fieldError('email') ? 'true' : undefined"
              :aria-describedby="fieldDescription('email')"
            />
            <p v-if="fieldError('email')" id="email-error" class="sg-field-message">
              {{ fieldError('email') }}
            </p>
          </div>

          <div class="sg-field">
            <label class="sg-label" for="displayname-input">
              {{ t('auth.displayName') }}
              <span class="sg-field-hint">（{{ t('auth.optional') }}）</span>
            </label>
            <input
              id="displayname-input"
              v-model="form.displayName"
              class="sg-input"
              type="text"
              name="displayName"
              :placeholder="t('auth.displayNamePlaceholder')"
            />
          </div>

          <div class="sg-field">
            <label class="sg-label" for="reg-password-input">{{ t('auth.password') }}</label>
            <input
              id="reg-password-input"
              v-model="form.password"
              :class="['sg-input', fieldError('password') ? 'sg-input--invalid' : '']"
              type="password"
              name="password"
              :placeholder="t('auth.passwordPlaceholder')"
              autocomplete="new-password"
              required
              :aria-invalid="fieldError('password') ? 'true' : undefined"
              :aria-describedby="passwordDescription"
            />
            <p id="password-hint" class="sg-field-hint">{{ t('auth.passwordHint') }}</p>
            <p v-if="fieldError('password')" id="password-error" class="sg-field-message">
              {{ fieldError('password') }}
            </p>
          </div>

          <div v-if="humanVerificationEnabled" class="sg-field">
            <label id="human-verification-label" class="sg-label">
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
            <p v-if="fieldError('humanVerification')" id="humanVerification-error" class="sg-field-message">
              {{ fieldError('humanVerification') }}
            </p>
          </div>

          <button class="sg-auth-submit" type="submit" :disabled="submitting">
            {{ submitting ? t('auth.registering') : t('auth.submitRegister') }}
          </button>
        </form>

        <p class="sg-auth-terms">
          {{ t('auth.agreeTo') }}
          <a href="#">{{ t('auth.terms') }}</a>
          {{ t('auth.and') }}
          <a href="#">{{ t('auth.privacy') }}</a>
        </p>

        <p class="sg-auth-switch">
          {{ t('auth.haveAccount') }}
          <NuxtLink :to="localePath('/login')">{{ t('auth.goLogin') }}</NuxtLink>
        </p>
      </div>
    </section>
  </main>
</template>
