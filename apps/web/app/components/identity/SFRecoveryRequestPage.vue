<script setup lang="ts">
import { useSForumSeo } from '~/composables/seo/useSForumSeo'
/**
 * 宿主 body 岛：auth.forgot_password（凭证表单仍为 Host 组件，不经主题可执行代码）。
 * 路由页保留 layout/middleware meta + fail-closed 回退。
 */

import type { AltchaWidgetElement } from 'altcha'


const { t, locale } = useI18n()
const { siteName, humanVerificationEnabledFor, altchaWidgetSettings } = useWebOptions()
const { apiBaseUrl, request } = useApiClient()

useSForumSeo({
  title: () => `${t('auth.forgotPassword')} - ${siteName.value}`,
  description: () => t('auth.forgotPasswordDesc'),
  type: 'website'
})

const email = ref('')
const submitting = ref(false)
const submitted = ref(false)
const errorMessage = ref('')
const fieldErrors = ref<Record<string, string[]>>({})

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
const altchaChallengeUrl = computed(() => {
  return `${apiBaseUrl}/human-verification/challenge?purpose=password_reset`
})
const passwordResetHumanVerificationEnabled = computed(() => {
  // 前端只负责体验开关；API verifier 仍按同一场景配置做权威校验。
  return humanVerificationEnabledFor('password_reset')
})
const humanVerificationEnabled = computed(() => {
  return passwordResetHumanVerificationEnabled.value || Boolean(fieldError('humanVerification'))
})

function fieldError(name: string) {
  return fieldErrors.value[name]?.[0] || ''
}

function fieldDescription(name: string) {
  return fieldError(name) ? `${name}-error` : undefined
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

async function submit() {
  if (submitting.value || !email.value.trim()) {
    return
  }
  submitting.value = true
  errorMessage.value = ''
  fieldErrors.value = {}
  const submittedHumanVerificationToken = humanVerificationEnabled.value ? humanVerificationToken.value : ''
  const body: Record<string, unknown> = { email: email.value.trim() }
  if (humanVerificationEnabled.value) {
    body.humanVerification = {
      provider: 'altcha',
      token: submittedHumanVerificationToken
    }
  }
  try {
    await request('/auth/password-reset/request', {
      method: 'POST',
      body
    })
    // 无论邮箱是否存在都显示成功提示（隐私保护）。
    submitted.value = true
  } catch (error) {
    fieldErrors.value = apiErrorFields(error)
    if (humanVerificationEnabled.value && (submittedHumanVerificationToken || fieldError('humanVerification'))) {
      resetHumanVerification()
    }
    errorMessage.value = apiErrorMessage(error) || t('auth.forgotPasswordFailed')
  } finally {
    submitting.value = false
  }
}
</script>

<template>

<main class="sf-public-page min-h-screen flex items-center justify-center px-4 py-12">
    <div class="w-full max-w-md">
      <h1 class="text-2xl font-bold text-slate-900 mb-2 dark:text-zinc-50">
        {{ t('auth.forgotPassword') }}
      </h1>
      <p class="text-sm text-slate-500 mb-6 dark:text-zinc-400">
        {{ t('auth.forgotPasswordDesc') }}
      </p>

      <SFCard v-if="submitted" class="p-6">
        <SFAlert variant="success" :title="t('auth.forgotPasswordSent')" />
        <p class="text-sm text-slate-600 mt-3 dark:text-zinc-400">
          {{ t('auth.forgotPasswordSentDesc') }}
        </p>
      </SFCard>

      <SFCard v-else class="p-6">
        <SFAlert v-if="errorMessage" variant="danger" :title="errorMessage" closable class="mb-4" @close="errorMessage = ''" />
        <div class="mb-4">
          <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
            {{ t('auth.email') }}
          </label>
          <input
            v-model="email"
            type="email"
            class="sf-input w-full"
            :placeholder="t('auth.emailPlaceholder')"
            @keydown.enter="submit"
          >
        </div>

        <div v-if="humanVerificationEnabled" class="mb-4">
          <label id="human-verification-label" class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
            {{ t('auth.humanVerification') }}
          </label>
          <ClientOnly>
            <altcha-widget
              ref="altchaWidget"
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
              <p class="text-sm text-slate-500 dark:text-zinc-400">
                {{ t('auth.humanVerificationLoading') }}
              </p>
            </template>
          </ClientOnly>
          <p v-if="fieldError('humanVerification')" id="humanVerification-error" class="text-sm text-red-600 mt-2 dark:text-red-400">
            {{ fieldError('humanVerification') }}
          </p>
        </div>

        <SFButton variant="primary" class="w-full" :disabled="submitting || !email.trim()" @click="submit">
          {{ submitting ? t('auth.submitting') : t('auth.sendResetLink') }}
        </SFButton>
      </SFCard>

      <p class="text-center text-sm text-slate-500 mt-4 dark:text-zinc-400">
        <NuxtLink :to="useLocalePath()('/login')" class="text-[#0F766E] hover:underline dark:text-teal-300">
          {{ t('auth.backToLogin') }}
        </NuxtLink>
      </p>
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
</style>
