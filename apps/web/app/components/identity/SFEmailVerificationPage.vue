<script setup lang="ts">
import SFAuthShell from '~/components/identity/auth/SFAuthShell.vue'
import { useAuthReturnNavigation } from '~/composables/identity/useAuthReturnNavigation'
import { useAuthSession } from '~/composables/identity/useAuthSession'
import { apiErrorMessage } from '~/composables/useApiClient'
import { useSForumSeo } from '~/composables/seo/useSForumSeo'

const { t } = useI18n()
const route = useRoute()
const toast = useToast()
const { request } = useApiClient()
const { user, refresh } = useAuthSession()
const { destination } = useAuthReturnNavigation({ explicitRedirect: route.query.redirect })

const state = ref<'pending' | 'verified'>(user.value?.emailVerified ? 'verified' : 'pending')
const resending = ref(false)
const checking = ref(false)
const resendCooldown = ref(0)
let cooldownTimer: ReturnType<typeof setInterval> | undefined

useSForumSeo({
  title: () => t('auth.emailVerificationTitle'),
  noindex: true
})

watch(() => user.value?.emailVerified, (verified) => {
  if (verified) state.value = 'verified'
}, { immediate: true })

onMounted(async () => {
  cooldownTimer = setInterval(() => {
    if (resendCooldown.value > 0) resendCooldown.value -= 1
  }, 1000)
  if (route.query.verified === '1') {
    await checkVerification()
  }
})

onBeforeUnmount(() => {
  if (cooldownTimer) clearInterval(cooldownTimer)
})

async function resendVerification() {
  if (resending.value || resendCooldown.value > 0) return
  resending.value = true
  try {
    await request('/auth/email-verification/request', { method: 'POST' })
    resendCooldown.value = 30
    toast.add({
      color: 'success',
      icon: 'i-lucide-mail-check',
      title: t('auth.emailVerificationResent'),
      duration: 10000
    })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-circle-alert',
      title: apiErrorMessage(error) || t('auth.emailVerificationSendFailed'),
      duration: 0
    })
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

        <div class="email-verification-status" role="status" aria-live="polite">
          <UIcon name="i-lucide-send" class="size-5" aria-hidden="true" />
          <div>
            <strong>{{ t('auth.emailVerificationLinkSent') }}</strong>
            <p>{{ t('auth.emailVerificationLinkHint') }}</p>
          </div>
        </div>

        <UButton
          block
          size="lg"
          color="primary"
          :loading="checking"
          leading-icon="i-lucide-refresh-cw"
          data-testid="email-verification-check"
          @click="checkVerification"
        >
          {{ t('auth.emailVerificationCheck') }}
        </UButton>

        <div class="email-verification-delivery">
          <span>{{ t('auth.emailVerificationNotReceived') }}</span>
          <UButton
            variant="link"
            color="neutral"
            :loading="resending"
            :disabled="resendCooldown > 0"
            leading-icon="i-lucide-rotate-cw"
            data-testid="email-verification-resend"
            @click="resendVerification"
          >
            {{ resendCooldown > 0 ? t('auth.emailVerificationResendCooldown', { seconds: resendCooldown }) : t('auth.emailVerificationResend') }}
          </UButton>
        </div>

        <p class="email-verification-help">
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
.email-verification-status { display: flex; align-items: flex-start; gap: 12px; margin: 28px 0 22px; padding: 16px; border: 1px solid var(--sf-border); border-radius: 10px; background: var(--sf-muted); color: var(--sf-accent); }
.email-verification-status strong { display: block; color: var(--sf-fg); font-size: 14px; }
.email-verification-status p { margin: 4px 0 0; color: var(--sf-fg-secondary); font-size: 12px; line-height: 1.6; }
.email-verification-delivery { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-top: 18px; color: var(--sf-fg-tertiary); font-size: 12px; }
.email-verification-help { display: flex; align-items: flex-start; gap: 7px; margin: 28px 0 0; color: var(--sf-fg-tertiary); font-size: 12px; line-height: 1.6; }
.email-verification-complete { text-align: center; }
.email-verification-complete .email-verification-subtitle { margin: 0 auto 28px; }
@media (max-width: 420px) { .email-verification-delivery { align-items: flex-start; flex-direction: column; } }
</style>
