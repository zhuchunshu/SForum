<script setup lang="ts">
import type { AltchaWidgetElement } from 'altcha'

import SFRecoveryShell from '~/components/identity/recovery/SFRecoveryShell.vue'
import { useSForumSeo } from '~/composables/seo/useSForumSeo'
import { apiErrorFields, apiErrorMessage } from '~/composables/useApiClient'

/**
 * 宿主 body 岛：auth.forgot_password（凭证表单仍为 Host 组件，不经主题可执行代码）。
 * 路由页保留 layout/middleware meta + fail-closed 回退。
 */

const resendCooldownSeconds = 30

const { t, locale } = useI18n()
const localePath = useLocalePath()
const toast = useToast()
const { humanVerificationEnabledFor, altchaWidgetSettings } = useWebOptions()
const { apiBaseUrl, request } = useApiClient()

useSForumSeo({
  title: () => t('auth.forgotPassword'),
  description: () => t('auth.forgotPasswordDesc'),
  type: 'website',
  noindex: true
})

const email = ref('')
const submittedEmail = ref('')
const emailInput = ref<HTMLInputElement | null>(null)
const submitting = ref(false)
const submitted = ref(false)
const resending = ref(false)
const resendRemaining = ref(0)
const emailInvalid = ref(false)
const errorMessage = ref('')
const fieldErrors = ref<Record<string, string[]>>({})
let resendTimer: ReturnType<typeof setInterval> | null = null

// 人机验证：当运营者启用 password_reset 场景时接入 ALTCHA，与注册页保持一致。
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
const altchaChallengeUrl = computed(() => `${apiBaseUrl}/human-verification/challenge?purpose=password_reset`)
const passwordResetHumanVerificationEnabled = computed(() => humanVerificationEnabledFor('password_reset'))
const humanVerificationEnabled = computed(() => {
  return passwordResetHumanVerificationEnabled.value || Boolean(fieldError('humanVerification'))
})
const emailError = computed(() => {
  if (fieldError('email')) return fieldError('email')
  return emailInvalid.value ? t('auth.recovery.emailInvalid') : ''
})
const maskedEmail = computed(() => maskEmail(submittedEmail.value))
const resendLabel = computed(() => {
  if (resending.value) return t('auth.recovery.resending')
  if (resendRemaining.value > 0) {
    return t('auth.recovery.resendIn', { seconds: resendRemaining.value })
  }
  return t('auth.recovery.resend')
})

function fieldError(name: string) {
  return fieldErrors.value[name]?.[0] || ''
}

function fieldDescription(name: string) {
  return fieldError(name) ? `${name}-error` : undefined
}

function maskEmail(value: string) {
  const [local, domain] = value.split('@')
  if (!local || !domain) return 'n•••@example.com'
  const hiddenLength = Math.min(3, Math.max(2, local.length - 1))
  return `${local.slice(0, 1)}${'•'.repeat(hiddenLength)}@${domain}`
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

function clearResendTimer() {
  if (resendTimer) clearInterval(resendTimer)
  resendTimer = null
}

function startResendCountdown() {
  clearResendTimer()
  resendRemaining.value = resendCooldownSeconds
  resendTimer = setInterval(() => {
    resendRemaining.value = Math.max(0, resendRemaining.value - 1)
    if (resendRemaining.value === 0) clearResendTimer()
  }, 1000)
}

function clearEmailFeedback() {
  emailInvalid.value = false
  errorMessage.value = ''
  delete fieldErrors.value.email
}

function validateEmail() {
  const valid = Boolean(email.value.trim()) && (emailInput.value?.checkValidity() ?? false)
  emailInvalid.value = !valid
  return valid
}

async function sendResetRequest(isResend = false) {
  if (submitting.value || resending.value || (isResend && resendRemaining.value > 0)) return
  if (!validateEmail()) {
    emailInput.value?.focus()
    return
  }

  const pending = isResend ? resending : submitting
  pending.value = true
  errorMessage.value = ''
  fieldErrors.value = {}
  const submittedHumanVerificationToken = humanVerificationEnabled.value ? humanVerificationToken.value : ''
  const normalizedEmail = email.value.trim()
  const body: Record<string, unknown> = { email: normalizedEmail }
  if (humanVerificationEnabled.value) {
    body.humanVerification = {
      provider: 'altcha',
      token: submittedHumanVerificationToken
    }
  }

  try {
    await request('/auth/password-reset/request', { method: 'POST', body })
    submittedEmail.value = normalizedEmail
    submitted.value = true
    startResendCountdown()
    toast.add({
      color: 'success',
      icon: 'i-lucide-mail-check',
      title: t('auth.recovery.requestAcceptedToast'),
      duration: 10000
    })
  } catch (error) {
    fieldErrors.value = apiErrorFields(error)
    if (humanVerificationEnabled.value && (submittedHumanVerificationToken || fieldError('humanVerification'))) {
      resetHumanVerification()
    }
    errorMessage.value = apiErrorMessage(error) || t('auth.forgotPasswordFailed')
  } finally {
    pending.value = false
  }
}

async function changeEmail() {
  clearResendTimer()
  resendRemaining.value = 0
  submitted.value = false
  errorMessage.value = ''
  await nextTick()
  emailInput.value?.focus()
}

onBeforeUnmount(clearResendTimer)
</script>

<template>
  <SFRecoveryShell :phase="1">
    <section v-if="!submitted" class="sf-recovery-view" data-testid="recovery-request-view">
      <div class="sf-recovery-form-icon" aria-hidden="true">
        <UIcon name="i-lucide-mail" />
      </div>
      <h2 class="sf-recovery-title">{{ t('auth.recovery.requestTitle') }}</h2>
      <p class="sf-recovery-description">{{ t('auth.recovery.requestDescription') }}</p>

      <SFAlert
        v-if="errorMessage"
        variant="danger"
        :title="errorMessage"
        closable
        class="sf-recovery-alert"
        @close="errorMessage = ''"
      />

      <form novalidate @submit.prevent="sendResetRequest(false)">
        <div class="sf-recovery-field">
          <label class="sf-recovery-label" for="recovery-email">{{ t('auth.recovery.emailLabel') }}</label>
          <input
            id="recovery-email"
            ref="emailInput"
            v-model="email"
            class="sf-recovery-input"
            name="email"
            type="email"
            inputmode="email"
            autocomplete="email"
            :placeholder="t('auth.emailPlaceholder')"
            :aria-invalid="emailError ? 'true' : undefined"
            :aria-describedby="emailError ? 'recovery-email-error' : 'recovery-email-hint'"
            required
            @input="clearEmailFeedback"
          >
          <p v-if="emailError" id="recovery-email-error" class="sf-recovery-field-error">
            {{ emailError }}
          </p>
          <p v-else id="recovery-email-hint" class="sf-recovery-field-hint">
            {{ t('auth.recovery.emailExpiryHint') }}
          </p>
        </div>

        <div v-if="humanVerificationEnabled" class="sf-recovery-field">
          <label id="human-verification-label" class="sf-recovery-label">
            {{ t('auth.humanVerification') }}
          </label>
          <ClientOnly>
            <altcha-widget
              ref="altchaWidget"
              class="sf-recovery-altcha"
              :challenge="altchaChallengeUrl"
              :configuration="altchaConfiguration"
              :auto="altchaWidgetAuto"
              :display="altchaWidgetDisplay"
              :language="locale === 'zh-CN' ? 'zh-cn' : 'en'"
              :type="altchaWidgetType"
              :workers="altchaWidgetWorkers"
              :aria-invalid="fieldError('humanVerification') ? 'true' : undefined"
              aria-labelledby="human-verification-label"
              :aria-describedby="fieldDescription('humanVerification')"
              @verified="handleAltchaVerified"
              @expired="resetHumanVerification"
              @statechange="handleAltchaStateChange"
            />
            <template #fallback>
              <p class="sf-recovery-field-hint">{{ t('auth.humanVerificationLoading') }}</p>
            </template>
          </ClientOnly>
          <p
            v-if="fieldError('humanVerification')"
            id="humanVerification-error"
            class="sf-recovery-field-error"
          >
            {{ fieldError('humanVerification') }}
          </p>
        </div>

        <button
          class="sf-recovery-button"
          type="submit"
          :disabled="submitting || !email.trim()"
        >
          <span v-if="submitting" class="sf-recovery-spinner" aria-hidden="true" />
          <UIcon v-else name="i-lucide-send" aria-hidden="true" />
          <span>{{ submitting ? t('auth.recovery.sending') : t('auth.sendResetLink') }}</span>
        </button>
      </form>

      <p class="sf-recovery-note">
        <UIcon name="i-lucide-shield" aria-hidden="true" />
        <span>{{ t('auth.recovery.nonEnumerationNote') }}</span>
      </p>

      <div class="sf-recovery-help">
        <span>{{ t('auth.recovery.noEmailAccess') }}</span>
        <NuxtLink :to="localePath('/')" class="sf-recovery-inline-link">
          {{ t('auth.recovery.contactFromCommunity') }}
        </NuxtLink>
      </div>
    </section>

    <section v-else class="sf-recovery-view" data-testid="recovery-sent-view" aria-live="polite">
      <div class="sf-recovery-result-icon is-success" aria-hidden="true">
        <UIcon name="i-lucide-mail-check" />
      </div>
      <h2 class="sf-recovery-title">{{ t('auth.recovery.sentTitle') }}</h2>
      <p class="sf-recovery-description">{{ t('auth.recovery.sentDescription') }}</p>

      <SFAlert
        v-if="errorMessage"
        variant="danger"
        :title="errorMessage"
        closable
        class="sf-recovery-alert"
        @close="errorMessage = ''"
      />

      <div class="sf-recovery-mail-address">
        <UIcon name="i-lucide-mail" aria-hidden="true" />
        <span>{{ maskedEmail }}</span>
      </div>

      <div class="sf-recovery-button-row">
        <button type="button" class="sf-recovery-button is-secondary" @click="changeEmail">
          {{ t('auth.recovery.changeEmail') }}
        </button>
        <button
          type="button"
          class="sf-recovery-button"
          :disabled="resending || resendRemaining > 0"
          @click="sendResetRequest(true)"
        >
          <span v-if="resending" class="sf-recovery-spinner" aria-hidden="true" />
          <span>{{ resendLabel }}</span>
        </button>
      </div>

      <p class="sf-recovery-note is-accent">
        <UIcon name="i-lucide-info" aria-hidden="true" />
        <span>{{ t('auth.recovery.sentHint') }}</span>
      </p>

      <div class="sf-recovery-help">
        <span>{{ t('auth.recovery.alreadyReset') }}</span>
        <NuxtLink :to="localePath('/login')" class="sf-recovery-inline-link">
          {{ t('auth.backToLogin') }}
        </NuxtLink>
      </div>
    </section>
  </SFRecoveryShell>
</template>
