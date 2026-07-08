<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'

definePageMeta({ public: true })

const route = useRoute()
const { t } = useI18n()
const { siteName, passwordPolicy } = useWebOptions()
const { request } = useApiClient()

useSForumSeo({
  title: () => `${t('auth.resetPassword')} - ${siteName.value}`,
  description: () => t('auth.resetPasswordDesc'),
  type: 'website'
})

// 令牌可来自查询参数或路由。
const token = computed(() => String(route.query.token ?? ''))
const newPassword = ref('')
const confirmPassword = ref('')
const submitting = ref(false)
const completed = ref(false)
const errorMessage = ref('')
const fieldError = ref('')

const passwordsMatch = computed(() => newPassword.value === confirmPassword.value)
const passwordProgress = computed(() => passwordPolicyProgress(newPassword.value, passwordPolicy.value))
// 进度条颜色档位：空=中性灰、弱=红、中=黄、强(100%)=主题色。
const passwordProgressLevel = computed(() => passwordPolicyProgressLevel(passwordProgress.value))
const passwordRequirementRows = computed(() => {
  return passwordPolicyRequirements(newPassword.value, passwordPolicy.value).map(item => ({
    ...item,
    label: passwordRequirementLabel(item.key)
  }))
})
const newPasswordMeetsPolicy = computed(() => passwordRequirementRows.value.every(item => item.met))
const canSubmit = computed(() => token.value !== '' && newPasswordMeetsPolicy.value && passwordsMatch.value && !submitting.value)

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

async function submit() {
  if (!canSubmit.value) {
    return
  }
  submitting.value = true
  errorMessage.value = ''
  fieldError.value = ''
  try {
    await request('/auth/password-reset/confirm', {
      method: 'POST',
      body: { token: token.value, newPassword: newPassword.value }
    })
    completed.value = true
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('auth.resetPasswordFailed')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main class="min-h-screen flex items-center justify-center px-4 py-12" style="background-color: var(--sf-surface)">
    <div class="w-full max-w-md">
      <h1 class="text-2xl font-bold text-slate-900 mb-2 dark:text-zinc-50">
        {{ t('auth.resetPassword') }}
      </h1>
      <p class="text-sm text-slate-500 mb-6 dark:text-zinc-400">
        {{ t('auth.resetPasswordDesc') }}
      </p>

      <SFCard v-if="completed" class="p-6">
        <SFAlert variant="success" :title="t('auth.resetPasswordSuccess')" />
        <p class="text-sm text-slate-600 mt-3 dark:text-zinc-400">
          {{ t('auth.resetPasswordSuccessDesc') }}
        </p>
        <NuxtLink :to="useLocalePath()('/login')" class="sf-button sf-button--primary block text-center mt-4">
          {{ t('auth.backToLogin') }}
        </NuxtLink>
      </SFCard>

      <SFCard v-else-if="!token" class="p-6">
        <SFAlert variant="danger" :title="t('auth.resetTokenMissing')" />
      </SFCard>

      <SFCard v-else class="p-6">
        <SFAlert v-if="errorMessage" variant="danger" :title="errorMessage" closable class="mb-4" @close="errorMessage = ''" />
        <div class="mb-4">
          <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
            {{ t('auth.newPassword') }}
          </label>
          <input
            v-model="newPassword"
            type="password"
            class="sf-input w-full"
            :placeholder="t('auth.newPasswordPlaceholder')"
          >
          <p class="text-xs text-slate-400 mt-1 dark:text-zinc-500">
            {{ t('auth.passwordPolicySummary', { min: passwordPolicy.minLength, max: passwordPolicy.maxLength }) }}
          </p>
          <div class="reset-password-policy">
            <div class="reset-password-policy__header">
              <span>{{ t('auth.passwordStrength') }}</span>
              <span :class="['reset-password-policy__value', `is-${passwordProgressLevel}`]">{{ passwordProgress }}%</span>
            </div>
            <div class="reset-password-policy__bar" :class="[`is-${passwordProgressLevel}`]" aria-hidden="true">
              <span :style="{ width: `${passwordProgress}%` }" />
            </div>
            <ul class="reset-password-policy__list">
              <li v-for="item in passwordRequirementRows" :key="item.key" :class="{ 'is-met': item.met }">
                <UIcon :name="item.met ? 'i-lucide-check' : 'i-lucide-circle'" class="reset-password-policy__icon" />
                <span>{{ item.label }}</span>
              </li>
            </ul>
          </div>
        </div>
        <div class="mb-4">
          <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
            {{ t('auth.confirmPassword') }}
          </label>
          <input
            v-model="confirmPassword"
            type="password"
            class="sf-input w-full"
            :placeholder="t('auth.confirmPasswordPlaceholder')"
          >
          <p v-if="confirmPassword && !passwordsMatch" class="text-sm text-red-600 mt-1 dark:text-red-400">
            {{ t('auth.passwordsDoNotMatch') }}
          </p>
        </div>
        <SFButton variant="primary" class="w-full" :disabled="!canSubmit" @click="submit">
          {{ submitting ? t('auth.submitting') : t('auth.resetPassword') }}
        </SFButton>
      </SFCard>
    </div>
  </main>
</template>

<style scoped>
.sf-input {
  border: 1px solid #d1d5db;
  border-radius: 0.5rem;
  padding: 0.5rem 0.75rem;
  font-size: 0.95rem;
  background: #ffffff;
  color: #111827;
  outline: none;
  transition: border-color 0.15s;
}
.sf-input:focus {
  border-color: #0f766e;
  box-shadow: 0 0 0 3px rgba(15, 118, 110, 0.12);
}
:global(.dark) .sf-input {
  background: #18181b;
  border-color: #3f3f46;
  color: #f4f4f5;
}
.reset-password-policy {
  display: grid;
  gap: 0.45rem;
  margin-top: 0.55rem;
}
.reset-password-policy__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  color: #475569;
  font-size: 0.75rem;
  font-weight: 600;
}
.reset-password-policy__bar {
  height: 0.35rem;
  overflow: hidden;
  border-radius: 999px;
  background: #f1f5f9;
  border: 1px solid #e2e8f0;
}
.reset-password-policy__bar span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: #cbd5e1;
  transition: width 0.18s ease, background-color 0.18s ease;
}
/* 进度条按合格度分档变色：弱=亮红、中=亮琥珀、强=主题色；空输入保持中性灰。
   弱/中档用亮色阶而非 --sf-danger/--sf-warning：后者偏暗、色相接近，
   大色块上难辨（红↔棕橙糊在一起）。 */
.reset-password-policy__bar.is-weak span {
  background: #ef4444;
}
.reset-password-policy__bar.is-medium span {
  background: #f59e0b;
}
.reset-password-policy__bar.is-strong span {
  background: var(--sf-accent);
}
/* 百分比文字与进度条同档变色，保持视觉一致。 */
.reset-password-policy__value.is-weak {
  color: var(--sf-danger);
}
.reset-password-policy__value.is-medium {
  color: var(--sf-warning);
}
.reset-password-policy__value.is-strong {
  color: var(--sf-accent);
}
.reset-password-policy__list {
  display: grid;
  gap: 0.3rem;
  margin: 0;
  padding: 0;
  list-style: none;
  color: #64748b;
  font-size: 0.75rem;
  line-height: 1.4;
}
.reset-password-policy__list li {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  min-height: 1.125rem;
}
.reset-password-policy__list li.is-met {
  color: var(--sf-accent);
}
.reset-password-policy__icon {
  width: 0.8125rem;
  height: 0.8125rem;
  flex: 0 0 0.8125rem;
}
:global(.dark) .reset-password-policy__header {
  color: #d4d4d8;
}
:global(.dark) .reset-password-policy__bar {
  background: #27272a;
  border-color: #3f3f46;
}
:global(.dark) .reset-password-policy__list {
  color: #a1a1aa;
}
</style>
