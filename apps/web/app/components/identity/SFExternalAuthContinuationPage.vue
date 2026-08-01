<script setup lang="ts">
import SFAuthShell from '~/components/identity/auth/SFAuthShell.vue'
import { useAuthReturnNavigation } from '~/composables/identity/useAuthReturnNavigation'
import { useAuthSession, type CurrentUser } from '~/composables/identity/useAuthSession'
import { useSForumSeo } from '~/composables/seo/useSForumSeo'
import { apiErrorMessage, apiErrorReason } from '~/composables/useApiClient'

type ExternalAuthContinuationPreparation = {
  providerId: string
  providerLabel: string
  providerIcon: string
  usernameHint: string
  displayName: string
  emailHint: string
  emailVerified: boolean
  canLinkExisting: boolean
  canRegister: boolean
}

const route = useRoute()
const router = useRouter()
const localePath = useLocalePath()
const { t } = useI18n()
const toast = useToast()
const { request } = useApiClient()
const { user, status, refresh, setUser } = useAuthSession()
const { destination, returnFromAuth } = useAuthReturnNavigation()

const ticket = computed(() => typeof route.query.ticket === 'string' ? route.query.ticket.trim() : '')
const shouldBind = computed(() => route.query.bind === '1')
const binding = ref(false)
const bindAttempted = ref(false)
const errorMessage = ref(ticket.value ? '' : t('auth.external.continuation.invalid'))

const { data: preparation, pending } = await useAsyncData(
  `auth-external-continuation-${ticket.value || 'missing'}`,
  async () => {
    if (!ticket.value) {
      return null
    }
    try {
      return await request<ExternalAuthContinuationPreparation>('/auth/external-continuation/prepare', {
        method: 'POST',
        body: { ticket: ticket.value }
      })
    } catch (error) {
      errorMessage.value = continuationErrorMessage(error)
      return null
    }
  }
)

const providerName = computed(() => preparation.value?.providerLabel.trim() || t('auth.providers.genericName'))
const providerIcon = computed(() => preparation.value?.providerIcon.trim() || 'i-lucide-key-round')
const hasChoice = computed(() => Boolean(preparation.value?.canLinkExisting || preparation.value?.canRegister))
const busy = computed(() => pending.value || binding.value)

useSForumSeo({
  title: () => t('auth.external.continuation.title'),
  noindex: true,
  nofollow: true
})

function continuationErrorMessage(error: unknown) {
  switch (apiErrorReason(error)) {
    case 'auth.external_registration_ticket_invalid':
      return t('auth.external.continuation.invalid')
    case 'auth.external_registration_ticket_expired':
      return t('auth.external.continuation.expired')
    case 'auth.external_subject_conflict':
      return t('auth.external.reasons.subjectConflict')
    case 'auth.recent_auth_required':
      return t('auth.external.reasons.recentAuthRequired')
    case 'auth.required':
      return t('auth.external.reasons.authRequired')
    default:
      return apiErrorMessage(error) || t('auth.external.reasons.providerUnavailable')
  }
}

function continuationReturnPath() {
  return router.resolve({
    path: localePath('/auth/continue'),
    query: {
      ticket: ticket.value,
      redirect: destination.value,
      bind: '1'
    }
  }).fullPath
}

async function signInAndBind() {
  if (busy.value || !preparation.value?.canLinkExisting) {
    return
  }
  if (user.value) {
    await completeBinding()
    return
  }
  await navigateTo({
    path: localePath('/login'),
    query: {
      redirect: continuationReturnPath(),
      continuation_provider: preparation.value.providerId
    }
  })
}

async function registerAndBind() {
  if (busy.value || !preparation.value?.canRegister) {
    return
  }
  await navigateTo({
    path: localePath('/register'),
    query: { ticket: ticket.value, redirect: destination.value }
  })
}

async function completeBinding() {
  if (binding.value || !ticket.value || !preparation.value?.canLinkExisting) {
    return
  }
  binding.value = true
  errorMessage.value = ''
  try {
    const current = await request<CurrentUser>('/auth/external-continuation/link', {
      method: 'POST',
      body: { ticket: ticket.value }
    })
    setUser(current)
    toast.add({
      color: 'success',
      icon: 'i-lucide-link-2',
      title: t('auth.external.continuation.linkSuccess', { name: providerName.value }),
      duration: 10000
    })
    await returnFromAuth()
  } catch (error) {
    errorMessage.value = continuationErrorMessage(error)
  } finally {
    binding.value = false
  }
}

onMounted(async () => {
  if (!shouldBind.value || bindAttempted.value || !preparation.value?.canLinkExisting) {
    return
  }
  bindAttempted.value = true
  if (!user.value && status.value === 'unknown') {
    await refresh({ timeout: 1200 })
  }
  if (!user.value) {
    errorMessage.value = t('auth.external.reasons.authRequired')
    return
  }
  await completeBinding()
})
</script>

<template>
  <SFAuthShell>
    <div class="continuation" data-testid="external-auth-continuation">
      <div v-if="pending || (shouldBind && !bindAttempted)" class="continuation-state" aria-live="polite">
        <UIcon name="i-lucide-loader-circle" class="continuation-spinner" aria-hidden="true" />
        <p>{{ t('auth.external.continuation.loading') }}</p>
      </div>

      <template v-else-if="preparation">
        <div class="continuation-provider">
          <span class="continuation-provider-icon">
            <UIcon :name="providerIcon" aria-hidden="true" />
          </span>
          <span>{{ providerName }}</span>
          <UIcon name="i-lucide-badge-check" class="continuation-verified" aria-hidden="true" />
        </div>

        <h1>{{ t('auth.external.continuation.heading') }}</h1>
        <p class="continuation-intro">
          {{ t('auth.external.continuation.intro', { name: providerName }) }}
        </p>

        <SFAlert
          v-if="errorMessage"
          :title="errorMessage"
          variant="danger"
          compact
          class="continuation-alert"
        />

        <div v-if="binding" class="continuation-state continuation-state--inline" aria-live="polite">
          <UIcon name="i-lucide-loader-circle" class="continuation-spinner" aria-hidden="true" />
          <p>{{ t('auth.external.continuation.binding') }}</p>
        </div>

        <div v-else-if="hasChoice" class="continuation-actions">
          <button
            v-if="preparation.canLinkExisting"
            type="button"
            class="continuation-choice continuation-choice--primary"
            :disabled="busy"
            data-testid="continue-with-existing"
            @click="signInAndBind"
          >
            <span class="continuation-choice-icon">
              <UIcon name="i-lucide-log-in" aria-hidden="true" />
            </span>
            <span class="continuation-choice-copy">
              <strong>{{ t('auth.external.continuation.existingTitle') }}</strong>
              <small>{{ t('auth.external.continuation.existingDescription') }}</small>
            </span>
            <UIcon name="i-lucide-chevron-right" class="continuation-chevron" aria-hidden="true" />
          </button>

          <button
            v-if="preparation.canRegister"
            type="button"
            class="continuation-choice"
            :disabled="busy"
            data-testid="continue-with-registration"
            @click="registerAndBind"
          >
            <span class="continuation-choice-icon">
              <UIcon name="i-lucide-user-plus" aria-hidden="true" />
            </span>
            <span class="continuation-choice-copy">
              <strong>{{ t('auth.external.continuation.registerTitle') }}</strong>
              <small>{{ t('auth.external.continuation.registerDescription') }}</small>
            </span>
            <UIcon name="i-lucide-chevron-right" class="continuation-chevron" aria-hidden="true" />
          </button>
        </div>

        <SFAlert
          v-else
          :title="t('auth.external.continuation.unavailable')"
          variant="danger"
          compact
          class="continuation-alert"
        />
      </template>

      <div v-else class="continuation-error">
        <span class="continuation-error-icon">
          <UIcon name="i-lucide-circle-alert" aria-hidden="true" />
        </span>
        <h1>{{ t('auth.external.continuation.cannotContinue') }}</h1>
        <p>{{ errorMessage || t('auth.external.continuation.invalid') }}</p>
        <UButton
          :to="localePath('/login')"
          color="neutral"
          variant="outline"
          icon="i-lucide-arrow-left"
        >
          {{ t('auth.external.continuation.backToLogin') }}
        </UButton>
      </div>
    </div>
  </SFAuthShell>
</template>

<style scoped>
.continuation {
  width: 100%;
  max-width: 430px;
}

.continuation-provider {
  display: flex;
  align-items: center;
  gap: 9px;
  margin-bottom: 24px;
  color: var(--sf-fg-secondary);
  font-size: 13px;
  font-weight: 600;
}

.continuation-provider-icon,
.continuation-error-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: 1px solid var(--sf-border);
  border-radius: 6px;
  color: var(--sf-fg);
  font-size: 18px;
}

.continuation-verified {
  color: var(--sf-accent);
  font-size: 16px;
}

h1 {
  margin: 0 0 8px;
  color: var(--sf-fg);
  font-size: 22px;
  font-weight: 700;
  letter-spacing: 0;
  line-height: 1.35;
}

.continuation-intro,
.continuation-error p {
  margin: 0;
  color: var(--sf-fg-secondary);
  font-size: 14px;
  line-height: 1.65;
}

.continuation-alert {
  margin-top: 20px;
}

.continuation-actions {
  margin-top: 28px;
  border-top: 1px solid var(--sf-border);
}

.continuation-choice {
  display: grid;
  grid-template-columns: 40px minmax(0, 1fr) 20px;
  align-items: center;
  gap: 12px;
  width: 100%;
  min-height: 82px;
  padding: 14px 4px;
  border: 0;
  border-bottom: 1px solid var(--sf-border);
  background: transparent;
  color: var(--sf-fg);
  text-align: left;
  cursor: pointer;
  transition: background-color 0.16s, color 0.16s;
}

.continuation-choice:hover:not(:disabled) {
  background: color-mix(in srgb, var(--sf-accent) 6%, transparent);
}

.continuation-choice:focus-visible {
  outline: 2px solid var(--sf-accent);
  outline-offset: 2px;
}

.continuation-choice:disabled {
  cursor: wait;
  opacity: 0.6;
}

.continuation-choice-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border: 1px solid var(--sf-border);
  border-radius: 6px;
  color: var(--sf-fg-secondary);
  font-size: 19px;
}

.continuation-choice--primary .continuation-choice-icon {
  border-color: color-mix(in srgb, var(--sf-accent) 45%, var(--sf-border));
  color: var(--sf-accent);
}

.continuation-choice-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
}

.continuation-choice-copy strong {
  font-size: 14px;
  font-weight: 650;
  line-height: 1.4;
}

.continuation-choice-copy small {
  color: var(--sf-fg-tertiary);
  font-size: 12px;
  line-height: 1.5;
}

.continuation-chevron {
  color: var(--sf-fg-tertiary);
  font-size: 18px;
}

.continuation-state {
  display: flex;
  min-height: 180px;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: var(--sf-fg-secondary);
  font-size: 14px;
}

.continuation-state--inline {
  min-height: 110px;
  margin-top: 20px;
  border-top: 1px solid var(--sf-border);
  border-bottom: 1px solid var(--sf-border);
}

.continuation-state p {
  margin: 0;
}

.continuation-spinner {
  animation: continuation-spin 0.8s linear infinite;
  font-size: 20px;
}

.continuation-error {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 14px;
}

.continuation-error h1 {
  margin: 2px 0 0;
}

@keyframes continuation-spin {
  to { transform: rotate(360deg); }
}

@media (max-width: 480px) {
  .continuation {
    max-width: none;
  }

  .continuation-choice {
    min-height: 88px;
  }
}
</style>
