<script setup lang="ts">
import type { AltchaWidgetElement } from 'altcha'

import SFAuthShell from '~/components/identity/auth/SFAuthShell.vue'
import { useAuthReturnNavigation } from '~/composables/identity/useAuthReturnNavigation'
import { useAuthSession } from '~/composables/identity/useAuthSession'
import { useMailResendCooldown } from '~/composables/identity/useMailResendCooldown'
import { apiErrorFields, apiErrorMessage } from '~/composables/useApiClient'
import { useSForumSeo } from '~/composables/seo/useSForumSeo'

const { t, locale } = useI18n()
const route = useRoute()
const toast = useToast()
const { apiBaseUrl, request } = useApiClient()
const { user, refresh } = useAuthSession()
const { destination } = useAuthReturnNavigation({ explicitRedirect: route.query.redirect })
const { humanVerificationEnabledFor, altchaWidgetSettings } = useWebOptions()
const resendCooldown = useMailResendCooldown('auth.email_verification_rate_limited')

const state = ref<'pending' | 'verified'>(user.value?.emailVerified ? 'verified' : 'pending')
const sent = ref(false)
const resending = ref(false)
const checking = ref(false)
const errorMessage = ref('')
const fieldErrors = ref<Record<string, string[]>>({})
const humanVerificationToken = ref('')
const altchaWidget = ref<AltchaWidgetElement | null>(null)

const altchaConfiguration = computed(() => JSON.stringify({
  hideLogo: altchaWidgetSettings.value.hideLogo,
  hideFooter: altchaWidgetSettings.value.hideFooter,
  minDuration: altchaWidgetSettings.value.minDuration
}))
const altchaChallengeUrl = computed(() => `${apiBaseUrl}/human-verification/challenge?purpose=email_verification`)
const emailVerificationHumanVerificationEnabled = computed(() => humanVerificationEnabledFor('email_verification'))
const humanVerificationEnabled = computed(() => {
  return emailVerificationHumanVerificationEnabled.value || Boolean(fieldError('humanVerification'))
})
const sendLabel = computed(() => {
  if (resending.value) return t('auth.emailVerificationSending')
  if (resendCooldown.active.value) return t('auth.emailVerificationResendCooldown', { seconds: resendCooldown.remainingSeconds.value })
  return sent.value ? t('auth.emailVerificationResend') : t('auth.emailVerificationSend')
})

useSForumSeo({
  title: () => t('auth.emailVerificationTitle'),
  noindex: true
})

watch(() => user.value?.emailVerified, (verified) => {
  if (verified) state.value = 'verified'
}, { immediate: true })

onMounted(async () => {
  if (route.query.verified === '1') {
    await checkVerification()
  }
})

function fieldError(name: string) {
  return fieldErrors.value[name]?.[0] || ''
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

async function sendVerification() {
  if (resending.value || resendCooldown.active.value) return
  if (humanVerificationEnabled.value && !humanVerificationToken.value) return
  resending.value = true
  errorMessage.value = ''
  fieldErrors.value = {}
  const wasSent = sent.value
  const submittedHumanVerificationToken = humanVerificationEnabled.value ? humanVerificationToken.value : ''
  const body: Record<string, unknown> = {}
  if (humanVerificationEnabled.value) {
    body.humanVerification = {
      provider: 'altcha',
      token: submittedHumanVerificationToken
    }
  }
  try {
    const result = await request<{ sent: boolean, retryAfterSeconds?: number, retryAt?: string }>('/auth/email-verification/request', { method: 'POST', body })
    sent.value = true
    resendCooldown.start(result)
    if (humanVerificationEnabled.value) resetHumanVerification()
    toast.add({
      color: 'success',
      icon: 'i-lucide-mail-check',
      title: wasSent ? t('auth.emailVerificationResent') : t('auth.emailVerificationSent'),
      duration: 10000
    })
  } catch (error) {
    fieldErrors.value = apiErrorFields(error)
    if (humanVerificationEnabled.value && (submittedHumanVerificationToken || fieldError('humanVerification'))) {
      resetHumanVerification()
    }
    resendCooldown.capture(error)
    errorMessage.value = apiErrorMessage(error) || t('auth.emailVerificationSendFailed')
  } finally {
    resending.value = false
  }
}

async function checkVerification() {
  if (checking.value) return
  checking.value = true
  try {
    const currentUser = await refresh()
    if (currentUser?.emailVerified) {
      state.value = 'verified'
      toast.add({
        color: 'success',
        icon: 'i-lucide-badge-check',
        title: t('auth.emailVerificationConfirmed'),
        duration: 10000
      })
      return
    }
    toast.add({
      color: 'neutral',
      icon: 'i-lucide-mail-search',
      title: t('auth.emailVerificationStillPending'),
      duration: 10000
    })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-circle-alert',
      title: apiErrorMessage(error) || t('auth.emailVerificationCheckFailed'),
      duration: 0
    })
  } finally {
    checking.value = false
  }
}

async function continueToDestination() {
  await navigateTo(destination.value, { replace: true })
}
</script>

<template>
  <SFAuthShell>
    <div class="email-verification-page" data-testid="email-verification-page">
      <template v-if="state === 'pending'">
        <header class="email-verification-header">
          <span class="email-verification-icon" aria-hidden="true">
            <UIcon name="i-lucide-mail-check" class="size-7" />
          </span>
          <div>
            <h2 class="email-verification-title">{{ t('auth.emailVerificationHeading') }}</h2>
            <p class="email-verification-subtitle">{{ t('auth.emailVerificationDescription') }}</p>
          </div>
        </header>

        <SFAlert
          v-if="errorMessage"
          variant="danger"
          :title="errorMessage"
          closable
          class="email-verification-alert"
          @close="errorMessage = ''"
        />

        <div class="email-verification-status" role="status" aria-live="polite">
          <UIcon :name="sent ? 'i-lucide-send' : 'i-lucide-mail-question'" class="size-5" aria-hidden="true" />
          <div>
            <strong>{{ sent ? t('auth.emailVerificationLinkSent') : t('auth.emailVerificationNotSent') }}</strong>
            <p>{{ sent ? t('auth.emailVerificationLinkHint') : t('auth.emailVerificationNotSentHint') }}</p>
          </div>
        </div>

        <div v-if="humanVerificationEnabled" class="email-verification-human-verification">
          <label id="email-verification-human-label" class="email-verification-label">
            {{ t('auth.humanVerification') }}
          </label>
          <ClientOnly>
            <altcha-widget
              ref="altchaWidget"
              class="auth-altcha"
              :challenge="altchaChallengeUrl"
              :configuration="altchaConfiguration"
              :auto="altchaWidgetSettings.auto"
              :display="altchaWidgetSettings.display"
              :language="locale === 'zh-CN' ? 'zh-cn' : 'en'"
              :type="altchaWidgetSettings.type"
              :workers="altchaWidgetSettings.workers"
              :aria-invalid="fieldError('humanVerification') ? 'true' : undefined"
              aria-labelledby="email-verification-human-label"
              :aria-describedby="fieldError('humanVerification') ? 'email-verification-human-error' : undefined"
              @verified="handleAltchaVerified"
              @expired="resetHumanVerification"
              @statechange="handleAltchaStateChange"
            />
            <template #fallback>
              <p class="email-verification-field-hint">{{ t('auth.humanVerificationLoading') }}</p>
            </template>
          </ClientOnly>
          <p v-if="fieldError('humanVerification')" id="email-verification-human-error" class="email-verification-field-error">
            {{ fieldError('humanVerification') }}
          </p>
        </div>

        <div class="email-verification-actions">
          <UButton
            block
            size="lg"
            color="primary"
            :loading="resending"
            :disabled="resendCooldown.active.value || (humanVerificationEnabled && !humanVerificationToken)"
            leading-icon="i-lucide-send"
            data-testid="email-verification-send"
            @click="sendVerification"
          >
            {{ sendLabel }}
          </UButton>
          <UButton
            v-if="sent"
            block
            size="lg"
            color="neutral"
            variant="outline"
            :loading="checking"
            leading-icon="i-lucide-refresh-cw"
            data-testid="email-verification-check"
            @click="checkVerification"
          >
            {{ t('auth.emailVerificationCheck') }}
          </UButton>
        </div>

        <p v-if="sent" class="email-verification-help">
          <UIcon name="i-lucide-circle-help" class="size-4" aria-hidden="true" />
          {{ t('auth.emailVerificationHelp') }}
        </p>
      </template>

      <template v-else>
        <div class="email-verification-complete" data-testid="email-verification-complete">
          <span class="email-verification-icon email-verification-icon--success" aria-hidden="true">
            <UIcon name="i-lucide-badge-check" class="size-8" />
          </span>
          <h2 class="email-verification-title">{{ t('auth.emailVerificationSuccessTitle') }}</h2>
          <p class="email-verification-subtitle">{{ t('auth.emailVerificationSuccessDescription') }}</p>
          <UButton block size="lg" color="primary" trailing-icon="i-lucide-arrow-right" data-testid="email-verification-continue" @click="continueToDestination">
            {{ t('auth.emailVerificationContinue') }}
          </UButton>
        </div>
      </template>
    </div>
  </SFAuthShell>
</template>

<style scoped>
.email-verification-page { width: min(100%, 410px); }
.email-verification-header { display: flex; align-items: flex-start; gap: 14px; }
.email-verification-icon { display: grid; width: 52px; height: 52px; flex: 0 0 52px; place-items: center; border: 1px solid var(--sf-accent); border-radius: 14px; background: var(--sf-accent-soft); color: var(--sf-accent); }
.email-verification-icon--success { margin: 0 auto 20px; border-color: var(--sf-accent); }
.email-verification-title { margin: 0; color: var(--sf-fg); font-size: 24px; font-weight: 720; line-height: 1.3; }
.email-verification-subtitle { margin: 8px 0 0; color: var(--sf-fg-secondary); font-size: 14px; line-height: 1.7; }
.email-verification-alert { margin-top: 20px; }
.email-verification-status { display: flex; align-items: flex-start; gap: 12px; margin: 28px 0 22px; padding: 16px; border: 1px solid var(--sf-border); border-radius: 10px; background: var(--sf-muted); color: var(--sf-accent); }
.email-verification-status strong { display: block; color: var(--sf-fg); font-size: 14px; }
.email-verification-status p { margin: 4px 0 0; color: var(--sf-fg-secondary); font-size: 12px; line-height: 1.6; }
.email-verification-human-verification { margin-bottom: 18px; }
.email-verification-label { display: block; margin-bottom: 8px; color: var(--sf-fg); font-size: 13px; font-weight: 650; }
.email-verification-field-hint { margin: 6px 0 0; color: var(--sf-fg-tertiary); font-size: 12px; }
.email-verification-field-error { margin: 6px 0 0; color: var(--sf-danger); font-size: 12px; }
.email-verification-actions { display: grid; gap: 10px; }
.email-verification-help { display: flex; align-items: flex-start; gap: 7px; margin: 28px 0 0; color: var(--sf-fg-tertiary); font-size: 12px; line-height: 1.6; }
.email-verification-complete { text-align: center; }
.email-verification-complete .email-verification-subtitle { margin: 0 auto 28px; }
</style>
