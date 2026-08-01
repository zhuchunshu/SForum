<script setup lang="ts">
import { apiErrorMessage, apiErrorReason } from '~/composables/useApiClient'
import { useAccountSecurityApi } from '~/composables/identity/useAccountSecurityApi'
import { useSForumSeo } from '~/composables/seo/useSForumSeo'
import {
  passwordPolicyProgress,
  passwordPolicyProgressLevel,
  passwordPolicyRequirements
} from '~/composables/useWebOptions'
import { buildAuthPageLink } from '~/utils/identity/authReturn'
import SFSettingsShell from '~/components/settings/SFSettingsShell.vue'

/**
 * 宿主 body 岛：forum.settings.password。
 * 本地密码独立于外部登录方式和设备/令牌安全页；密码材料只提交给 Host API。
 */
const { t } = useI18n()
const toast = useToast()
const localePath = useLocalePath()
const securityApi = useAccountSecurityApi()
const { passwordPolicy } = useWebOptions()

useSForumSeo({
  title: () => t('localPasswordSettings.metaTitle'),
  description: () => t('localPasswordSettings.metaDescription'),
  type: 'website',
  noindex: true
})

const form = reactive({
  password: '',
  confirm: ''
})
const submitting = ref(false)
const fieldError = ref('')
const surfaceError = ref('')
const recentAuthRequired = ref(false)
const showPassword = ref(false)
const showConfirm = ref(false)

const passwordsMatch = computed(() => form.password === form.confirm)
const passwordProgress = computed(() => passwordPolicyProgress(form.password, passwordPolicy.value))
const passwordProgressLevel = computed(() => passwordPolicyProgressLevel(passwordProgress.value))
const passwordRequirementRows = computed(() =>
  passwordPolicyRequirements(form.password, passwordPolicy.value).map(item => ({
    ...item,
    label: passwordRequirementLabel(item.key)
  }))
)
const newPasswordMeetsPolicy = computed(() => passwordRequirementRows.value.every(item => item.met))
const canSubmit = computed(() =>
  newPasswordMeetsPolicy.value
  && passwordsMatch.value
  && form.password.length > 0
  && !submitting.value
)

const reauthLoginLink = computed(() => buildAuthPageLink(localePath('/login'), '/settings/password'))
const passwordInputType = computed(() => showPassword.value ? 'text' : 'password')
const confirmInputType = computed(() => showConfirm.value ? 'text' : 'password')

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

function clearMessages() {
  fieldError.value = ''
  surfaceError.value = ''
}

function resetForm() {
  form.password = ''
  form.confirm = ''
  clearMessages()
}

async function submitPassword() {
  if (!canSubmit.value) {
    return
  }
  clearMessages()
  recentAuthRequired.value = false
  submitting.value = true
  try {
    await securityApi.setupPassword(form.password)
    resetForm()
    toast.add({
      color: 'primary',
      icon: 'i-lucide-check',
      title: t('localPasswordSettings.saved'),
      duration: 10000
    })
  } catch (error) {
    const reason = apiErrorReason(error)
    if (reason === 'auth.recent_auth_required' || reason === 'auth.required') {
      recentAuthRequired.value = true
      surfaceError.value = reason === 'auth.required'
        ? t('localPasswordSettings.sessionRequired')
        : t('localPasswordSettings.recentAuthRequired')
      return
    }
    fieldError.value = apiErrorMessage(error) || t('localPasswordSettings.saveFailed')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <SFSettingsShell
    class="sforum-settings-local-password"
    data-sforum-island-body="identity.component.local_password_settings"
    active="password"
    title-id="local-password-settings-title"
    :title="t('localPasswordSettings.title')"
    :description="t('localPasswordSettings.intro')"
    :rail-label="t('localPasswordSettings.rail.ariaLabel')"
    :rail-open-label="t('localPasswordSettings.rail.open')"
  >
    <section
      class="sf-local-password__notice"
      data-testid="local-password-recommendation"
    >
      <UIcon name="i-lucide-lock-keyhole" class="sf-local-password__notice-icon" aria-hidden="true" />
      <div class="min-w-0">
        <h2>{{ t('localPasswordSettings.recommendedTitle') }}</h2>
        <p>{{ t('localPasswordSettings.recommendedDescription') }}</p>
      </div>
    </section>

    <SFAlert
      v-if="recentAuthRequired"
      variant="warning"
      class="mt-4"
      :title="t('localPasswordSettings.recentAuthTitle')"
      :description="surfaceError"
      closable
      @close="recentAuthRequired = false"
    >
      <NuxtLink
        :to="reauthLoginLink"
        class="mt-2 inline-flex items-center gap-1 text-sm font-semibold text-amber-800 underline dark:text-amber-200"
      >
        <UIcon name="i-lucide-log-in" class="size-4" aria-hidden="true" />
        {{ t('localPasswordSettings.reauthenticate') }}
      </NuxtLink>
    </SFAlert>

    <SFCard class="sf-local-password__card" data-testid="local-password-form">
      <div class="sf-local-password__form-head">
        <div>
          <h2>{{ t('localPasswordSettings.formTitle') }}</h2>
          <p>{{ t('localPasswordSettings.formDescription') }}</p>
        </div>
        <UBadge color="neutral" variant="soft" class="sf-local-password__policy-badge">
          {{ t('localPasswordSettings.policyBadge', {
            min: passwordPolicy.minLength,
            max: passwordPolicy.maxLength
          }) }}
        </UBadge>
      </div>

      <form class="sf-local-password__form" @submit.prevent="submitPassword">
        <label class="sf-local-password__field" for="local-password-input">
          <span>{{ t('localPasswordSettings.password') }}</span>
          <div class="sf-local-password__input-wrap">
            <input
              id="local-password-input"
              v-model="form.password"
              :type="passwordInputType"
              autocomplete="new-password"
              data-testid="password-setup-input"
              @input="clearMessages"
            >
            <button
              type="button"
              class="sf-local-password__visibility"
              :aria-label="showPassword ? t('localPasswordSettings.hidePassword') : t('localPasswordSettings.showPassword')"
              :title="showPassword ? t('localPasswordSettings.hidePassword') : t('localPasswordSettings.showPassword')"
              @click="showPassword = !showPassword"
            >
              <UIcon :name="showPassword ? 'i-lucide-eye-off' : 'i-lucide-eye'" class="size-4" aria-hidden="true" />
            </button>
          </div>
        </label>

        <div class="sf-local-password__meter" aria-live="polite">
          <div class="sf-local-password__meter-head">
            <span>{{ t('auth.passwordStrength') }}</span>
            <strong :class="`is-${passwordProgressLevel}`">{{ passwordProgress }}%</strong>
          </div>
          <div
            class="sf-local-password__meter-bar"
            :class="`is-${passwordProgressLevel}`"
            aria-hidden="true"
          >
            <span :style="{ width: `${passwordProgress}%` }" />
          </div>
          <ul class="sf-local-password__requirements">
            <li
              v-for="row in passwordRequirementRows"
              :key="row.key"
              :class="{ 'is-met': row.met }"
            >
              <UIcon
                :name="row.met ? 'i-lucide-check' : 'i-lucide-circle'"
                class="size-[14px] shrink-0"
                aria-hidden="true"
              />
              <span>{{ row.label }}</span>
            </li>
          </ul>
        </div>

        <label class="sf-local-password__field" for="local-password-confirm">
          <span>{{ t('localPasswordSettings.confirm') }}</span>
          <div class="sf-local-password__input-wrap">
            <input
              id="local-password-confirm"
              v-model="form.confirm"
              :type="confirmInputType"
              autocomplete="new-password"
              data-testid="password-setup-confirm"
              @input="clearMessages"
            >
            <button
              type="button"
              class="sf-local-password__visibility"
              :aria-label="showConfirm ? t('localPasswordSettings.hidePassword') : t('localPasswordSettings.showPassword')"
              :title="showConfirm ? t('localPasswordSettings.hidePassword') : t('localPasswordSettings.showPassword')"
              @click="showConfirm = !showConfirm"
            >
              <UIcon :name="showConfirm ? 'i-lucide-eye-off' : 'i-lucide-eye'" class="size-4" aria-hidden="true" />
            </button>
          </div>
          <p v-if="form.confirm && !passwordsMatch" class="sf-local-password__field-error">
            {{ t('auth.passwordsDoNotMatch') }}
          </p>
        </label>

        <p
          v-if="fieldError"
          class="sf-local-password__field-error"
          data-testid="password-setup-error"
        >
          {{ fieldError }}
        </p>

        <div class="sf-local-password__actions">
          <SFButton
            type="submit"
            variant="primary"
            size="sm"
            :loading="submitting"
            :disabled="!canSubmit"
            data-testid="password-setup-submit"
          >
            <UIcon name="i-lucide-save" class="mr-1" aria-hidden="true" />
            {{ submitting ? t('localPasswordSettings.submitting') : t('localPasswordSettings.submit') }}
          </SFButton>
          <SFButton
            type="button"
            variant="ghost"
            size="sm"
            :disabled="submitting || (!form.password && !form.confirm)"
            @click="resetForm"
          >
            {{ t('localPasswordSettings.reset') }}
          </SFButton>
        </div>
      </form>
    </SFCard>

    <template #rail>
      <section class="sforum-settings__rail-section">
        <div class="sforum-settings__rail-head">
          <h2>{{ t('localPasswordSettings.rail.passwordTitle') }}</h2>
          <span>{{ t('localPasswordSettings.rail.currentPage') }}</span>
        </div>
        <p class="sforum-settings__rail-help">{{ t('localPasswordSettings.rail.passwordHelp') }}</p>
      </section>

      <section class="sforum-settings__rail-section">
        <div class="sforum-settings__rail-head">
          <h2>{{ t('localPasswordSettings.rail.policyTitle') }}</h2>
          <span>{{ t('localPasswordSettings.rail.hostOwned') }}</span>
        </div>
        <dl class="sforum-settings__stats">
          <div>
            <dt>{{ t('localPasswordSettings.rail.length') }}</dt>
            <dd>{{ passwordPolicy.minLength }}-{{ passwordPolicy.maxLength }}</dd>
          </div>
          <div>
            <dt>{{ t('localPasswordSettings.rail.reauth') }}</dt>
            <dd>{{ t('localPasswordSettings.rail.required') }}</dd>
          </div>
        </dl>
      </section>
    </template>
  </SFSettingsShell>
</template>

<style scoped>
.sf-local-password__notice {
  display: flex;
  gap: 0.75rem;
  align-items: flex-start;
  margin-top: 0.5rem;
  padding: 1rem;
  border: 1px solid rgb(15 118 110 / 0.22);
  border-radius: 0.5rem;
  background: rgb(240 253 250 / 0.72);
}

.sf-local-password__notice h2,
.sf-local-password__form-head h2 {
  margin: 0;
  font-size: 1rem;
  line-height: 1.5;
  font-weight: 700;
  color: var(--sf-fg, #0f172a);
}

.sf-local-password__notice p,
.sf-local-password__form-head p {
  margin: 0.25rem 0 0;
  font-size: 0.875rem;
  line-height: 1.6;
  color: var(--sf-fg-secondary, #64748b);
}

.sf-local-password__notice-icon {
  width: 1.25rem;
  height: 1.25rem;
  flex: 0 0 1.25rem;
  color: #0f766e;
}

.sf-local-password__card {
  margin-top: 1rem;
  padding: 1rem;
}

.sf-local-password__form-head {
  display: flex;
  gap: 1rem;
  align-items: flex-start;
  justify-content: space-between;
}

.sf-local-password__policy-badge {
  flex: 0 0 auto;
}

.sf-local-password__form {
  display: grid;
  gap: 1rem;
  margin-top: 1rem;
}

.sf-local-password__field {
  display: grid;
  gap: 0.375rem;
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--sf-fg-secondary, #475569);
}

.sf-local-password__input-wrap {
  position: relative;
}

.sf-local-password__input-wrap input {
  width: 100%;
  min-height: 2.75rem;
  border: 1px solid var(--sf-border-subtle, #e2e8f0);
  border-radius: 0.5rem;
  background: var(--sf-public-surface, #fff);
  padding: 0.625rem 2.75rem 0.625rem 0.75rem;
  font-size: 0.9375rem;
  color: var(--sf-fg, #0f172a);
  outline: none;
}

.sf-local-password__input-wrap input:focus {
  border-color: var(--sf-accent, #0f766e);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--sf-accent, #0f766e) 18%, transparent);
}

.sf-local-password__visibility {
  position: absolute;
  right: 0.375rem;
  top: 50%;
  display: inline-flex;
  width: 2rem;
  height: 2rem;
  align-items: center;
  justify-content: center;
  transform: translateY(-50%);
  border-radius: 0.375rem;
  color: var(--sf-fg-tertiary, #64748b);
}

.sf-local-password__visibility:hover {
  background: var(--sf-hover-surface, #f1f5f9);
  color: var(--sf-fg, #0f172a);
}

.sf-local-password__meter {
  padding: 0.875rem;
  border: 1px solid var(--sf-border-subtle, #e2e8f0);
  border-radius: 0.5rem;
  background: var(--sf-public-muted-surface, #f8fafc);
}

.sf-local-password__meter-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 0.75rem;
  color: var(--sf-fg-tertiary, #64748b);
}

.sf-local-password__meter-head strong.is-weak {
  color: #b45309;
}

.sf-local-password__meter-head strong.is-medium {
  color: #0d9488;
}

.sf-local-password__meter-head strong.is-strong {
  color: #0f766e;
}

.sf-local-password__meter-bar {
  height: 0.375rem;
  margin-top: 0.5rem;
  overflow: hidden;
  border-radius: 999px;
  background: #e2e8f0;
}

.sf-local-password__meter-bar span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: #94a3b8;
  transition: width 0.15s ease;
}

.sf-local-password__meter-bar.is-weak span {
  background: #f59e0b;
}

.sf-local-password__meter-bar.is-medium span {
  background: #0d9488;
}

.sf-local-password__meter-bar.is-strong span {
  background: #0f766e;
}

.sf-local-password__requirements {
  display: grid;
  gap: 0.375rem;
  margin: 0.75rem 0 0;
  padding: 0;
  list-style: none;
}

.sf-local-password__requirements li {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.75rem;
  color: var(--sf-fg-tertiary, #94a3b8);
}

.sf-local-password__requirements li.is-met {
  color: #0f766e;
}

.sf-local-password__field-error {
  margin: 0.125rem 0 0;
  font-size: 0.875rem;
  font-weight: 500;
  color: #dc2626;
}

.sf-local-password__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
}

:global(.dark) .sf-local-password__notice {
  border-color: rgb(20 184 166 / 0.28);
  background: rgb(19 78 74 / 0.22);
}

:global(.dark) .sf-local-password__meter {
  border-color: var(--sf-border-subtle, #27272a);
  background: rgb(24 24 27 / 0.48);
}

:global(.dark) .sf-local-password__meter-bar {
  background: #3f3f46;
}

@media (max-width: 640px) {
  .sf-local-password__form-head {
    display: grid;
  }

  .sf-local-password__policy-badge {
    justify-self: start;
  }

  .sf-local-password__actions :deep(.sf-button) {
    width: 100%;
    min-height: 2.75rem;
  }
}
</style>
