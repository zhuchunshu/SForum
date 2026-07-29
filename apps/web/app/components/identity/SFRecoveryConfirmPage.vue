<script setup lang="ts">
import SFRecoveryShell from '~/components/identity/recovery/SFRecoveryShell.vue'
import { useSForumSeo } from '~/composables/seo/useSForumSeo'
import { apiErrorFields, apiErrorMessage, apiErrorReason } from '~/composables/useApiClient'

/**
 * 宿主 body 岛：auth.reset_password（凭证表单仍为 Host 组件，不经主题可执行代码）。
 * 路由页保留 layout/middleware meta + fail-closed 回退。
 */

const route = useRoute()
const { t } = useI18n()
const localePath = useLocalePath()
const toast = useToast()
const { siteName, passwordPolicy } = useWebOptions()
const { request } = useApiClient()

useSForumSeo({
  title: () => `${t('auth.resetPassword')} - ${siteName.value}`,
  description: () => t('auth.resetPasswordDesc'),
  type: 'website'
})

const token = computed(() => {
  const rawToken = route.query.token
  return String(Array.isArray(rawToken) ? rawToken[0] ?? '' : rawToken ?? '').trim()
})
const newPassword = ref('')
const confirmPassword = ref('')
const showNewPassword = ref(false)
const showConfirmPassword = ref(false)
const submitting = ref(false)
const completed = ref(false)
const tokenRejected = ref(false)
const errorMessage = ref('')
const fieldErrors = ref<Record<string, string[]>>({})

const passwordsMatch = computed(() => newPassword.value === confirmPassword.value)
const passwordRequirementRows = computed(() => {
  return passwordPolicyRequirements(newPassword.value, passwordPolicy.value).map(item => ({
    ...item,
    label: passwordRequirementLabel(item.key)
  }))
})
const metRequirementCount = computed(() => passwordRequirementRows.value.filter(item => item.met).length)
const passwordProgress = computed(() => passwordPolicyProgress(newPassword.value, passwordPolicy.value))
const passwordProgressLevel = computed(() => passwordPolicyProgressLevel(passwordProgress.value))
const newPasswordMeetsPolicy = computed(() => passwordRequirementRows.value.every(item => item.met))
const tokenUnavailable = computed(() => token.value === '' || tokenRejected.value)
const phase = computed<1 | 2 | 3>(() => completed.value ? 3 : tokenUnavailable.value ? 1 : 2)
const confirmPasswordError = computed(() => {
  if (fieldError('confirmPassword')) return fieldError('confirmPassword')
  if (confirmPassword.value && !passwordsMatch.value) return t('auth.passwordsDoNotMatch')
  return ''
})
const passwordMeterLabel = computed(() => {
  if (!newPassword.value) return t('auth.recovery.passwordNotStarted')
  if (newPasswordMeetsPolicy.value) return t('auth.recovery.passwordReady')
  return t('auth.recovery.passwordProgress', {
    met: metRequirementCount.value,
    total: passwordRequirementRows.value.length
  })
})
const canSubmit = computed(() => {
  return !tokenUnavailable.value
    && newPasswordMeetsPolicy.value
    && Boolean(confirmPassword.value)
    && passwordsMatch.value
    && !submitting.value
})

function fieldError(name: string) {
  return fieldErrors.value[name]?.[0] || ''
}

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

function isRejectedToken(error: unknown) {
  const reason = apiErrorReason(error)
  const message = apiErrorMessage(error)
  return reason === 'auth.password_reset_invalid' || message === 'auth.password_reset_invalid'
}

function clearFieldFeedback(name: string) {
  errorMessage.value = ''
  delete fieldErrors.value[name]
}

async function submit() {
  if (!canSubmit.value) return
  submitting.value = true
  errorMessage.value = ''
  fieldErrors.value = {}
  try {
    await request('/auth/password-reset/confirm', {
      method: 'POST',
      body: { token: token.value, newPassword: newPassword.value }
    })
    completed.value = true
    toast.add({
      color: 'success',
      icon: 'i-lucide-circle-check',
      title: t('auth.resetPasswordSuccess'),
      duration: 10000
    })
  } catch (error) {
    if (isRejectedToken(error)) {
      tokenRejected.value = true
      return
    }
    fieldErrors.value = apiErrorFields(error)
    errorMessage.value = apiErrorMessage(error) || t('auth.resetPasswordFailed')
  } finally {
    submitting.value = false
  }
}

watch(token, () => {
  tokenRejected.value = false
  errorMessage.value = ''
})
</script>

<template>
  <SFRecoveryShell :phase="phase">
    <section v-if="completed" class="sf-recovery-view" data-testid="recovery-complete-view" aria-live="polite">
      <div class="sf-recovery-result-icon is-success" aria-hidden="true">
        <UIcon name="i-lucide-circle-check" />
      </div>
      <h2 class="sf-recovery-title">{{ t('auth.recovery.completedTitle') }}</h2>
      <p class="sf-recovery-description">
        {{ t('auth.recovery.completedDescription', { siteName }) }}
      </p>

      <NuxtLink :to="localePath('/login')" class="sf-recovery-button">
        <UIcon name="i-lucide-log-in" aria-hidden="true" />
        {{ t('auth.backToLogin') }}
      </NuxtLink>

      <p class="sf-recovery-note is-accent">
        <UIcon name="i-lucide-shield-check" aria-hidden="true" />
        <span>{{ t('auth.recovery.completedSecurityNote') }}</span>
      </p>
    </section>

    <section v-else-if="tokenUnavailable" class="sf-recovery-view" data-testid="recovery-invalid-view">
      <div class="sf-recovery-result-icon is-danger" aria-hidden="true">
        <UIcon name="i-lucide-link-2-off" />
      </div>
      <h2 class="sf-recovery-title">{{ t('auth.recovery.invalidTitle') }}</h2>
      <p class="sf-recovery-description">{{ t('auth.recovery.invalidDescription') }}</p>

      <NuxtLink :to="localePath('/forgot-password')" class="sf-recovery-button">
        <UIcon name="i-lucide-rotate-ccw" aria-hidden="true" />
        {{ t('auth.recovery.requestNewLink') }}
      </NuxtLink>

      <p class="sf-recovery-note is-danger">
        <UIcon name="i-lucide-triangle-alert" aria-hidden="true" />
        <span>{{ t('auth.recovery.invalidSecurityNote') }}</span>
      </p>

      <div class="sf-recovery-help">
        <span>{{ t('auth.recovery.noLongerResetting') }}</span>
        <NuxtLink :to="localePath('/login')" class="sf-recovery-inline-link">
          {{ t('auth.backToLogin') }}
        </NuxtLink>
      </div>
    </section>

    <section v-else class="sf-recovery-view" data-testid="recovery-confirm-view">
      <div class="sf-recovery-form-icon" aria-hidden="true">
        <UIcon name="i-lucide-key-round" />
      </div>
      <h2 class="sf-recovery-title">{{ t('auth.recovery.resetTitle') }}</h2>
      <p class="sf-recovery-description">{{ t('auth.recovery.resetDescription') }}</p>

      <SFAlert
        v-if="errorMessage"
        variant="danger"
        :title="errorMessage"
        closable
        class="sf-recovery-alert"
        @close="errorMessage = ''"
      />

      <form novalidate @submit.prevent="submit">
        <div class="sf-recovery-field">
          <label class="sf-recovery-label" for="new-password">{{ t('auth.newPassword') }}</label>
          <div class="sf-recovery-input-wrap">
            <input
              id="new-password"
              v-model="newPassword"
              class="sf-recovery-input has-action"
              name="new-password"
              :type="showNewPassword ? 'text' : 'password'"
              autocomplete="new-password"
              :placeholder="t('auth.newPasswordPlaceholder')"
              :aria-invalid="fieldError('newPassword') ? 'true' : undefined"
              aria-describedby="password-requirements"
              required
              @input="clearFieldFeedback('newPassword')"
            >
            <button
              type="button"
              class="sf-recovery-input-action"
              :aria-label="t(showNewPassword ? 'auth.recovery.hideNewPassword' : 'auth.recovery.showNewPassword')"
              :title="t(showNewPassword ? 'auth.recovery.hideNewPassword' : 'auth.recovery.showNewPassword')"
              @click="showNewPassword = !showNewPassword"
            >
              <UIcon :name="showNewPassword ? 'i-lucide-eye-off' : 'i-lucide-eye'" aria-hidden="true" />
            </button>
          </div>
          <p v-if="fieldError('newPassword')" class="sf-recovery-field-error">
            {{ fieldError('newPassword') }}
          </p>

          <div id="password-requirements" class="sf-recovery-meter">
            <div class="sf-recovery-meter-header">
              <span>{{ t('auth.passwordStrength') }}</span>
              <span>{{ passwordMeterLabel }}</span>
            </div>
            <div class="sf-recovery-meter-track" aria-hidden="true">
              <span
                v-for="(item, index) in passwordRequirementRows"
                :key="item.key"
                class="sf-recovery-meter-segment"
                :class="{
                  'is-filled': index < metRequirementCount,
                  [`is-${passwordProgressLevel}`]: index < metRequirementCount
                }"
              />
            </div>
            <ul class="sf-recovery-requirements">
              <li
                v-for="item in passwordRequirementRows"
                :key="item.key"
                class="sf-recovery-requirement"
                :class="{ 'is-met': item.met }"
              >
                <UIcon :name="item.met ? 'i-lucide-circle-check' : 'i-lucide-circle'" aria-hidden="true" />
                <span>{{ item.label }}</span>
              </li>
            </ul>
          </div>
        </div>

        <div class="sf-recovery-field">
          <label class="sf-recovery-label" for="confirm-password">{{ t('auth.confirmPassword') }}</label>
          <div class="sf-recovery-input-wrap">
            <input
              id="confirm-password"
              v-model="confirmPassword"
              class="sf-recovery-input has-action"
              name="confirm-password"
              :type="showConfirmPassword ? 'text' : 'password'"
              autocomplete="new-password"
              :placeholder="t('auth.confirmPasswordPlaceholder')"
              :aria-invalid="confirmPasswordError ? 'true' : undefined"
              :aria-describedby="confirmPasswordError ? 'confirm-password-error' : undefined"
              required
              @input="clearFieldFeedback('confirmPassword')"
            >
            <button
              type="button"
              class="sf-recovery-input-action"
              :aria-label="t(showConfirmPassword ? 'auth.recovery.hideConfirmPassword' : 'auth.recovery.showConfirmPassword')"
              :title="t(showConfirmPassword ? 'auth.recovery.hideConfirmPassword' : 'auth.recovery.showConfirmPassword')"
              @click="showConfirmPassword = !showConfirmPassword"
            >
              <UIcon :name="showConfirmPassword ? 'i-lucide-eye-off' : 'i-lucide-eye'" aria-hidden="true" />
            </button>
          </div>
          <p v-if="confirmPasswordError" id="confirm-password-error" class="sf-recovery-field-error" aria-live="polite">
            {{ confirmPasswordError }}
          </p>
        </div>

        <button class="sf-recovery-button" type="submit" :disabled="!canSubmit">
          <span v-if="submitting" class="sf-recovery-spinner" aria-hidden="true" />
          <UIcon v-else name="i-lucide-lock-keyhole" aria-hidden="true" />
          <span>{{ submitting ? t('auth.submitting') : t('auth.resetPassword') }}</span>
        </button>
      </form>
    </section>
  </SFRecoveryShell>
</template>
