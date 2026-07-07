<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'

definePageMeta({ public: true })

const { t } = useI18n()
const { siteName } = useWebOptions()
const { request } = useApiClient()

useSForumSeo({
  title: () => `${t('auth.forgotPassword')} - ${siteName.value}`,
  description: () => t('auth.forgotPasswordDesc'),
  type: 'website'
})

const email = ref('')
const submitting = ref(false)
const submitted = ref(false)
const errorMessage = ref('')

async function submit() {
  if (submitting.value || !email.value.trim()) {
    return
  }
  submitting.value = true
  errorMessage.value = ''
  try {
    await request('/auth/password-reset/request', {
      method: 'POST',
      body: { email: email.value.trim() }
    })
    // 无论邮箱是否存在都显示成功提示（隐私保护）。
    submitted.value = true
  } catch (error) {
    errorMessage.value = apiErrorMessage(error) || t('auth.forgotPasswordFailed')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main class="min-h-screen flex items-center justify-center px-4 py-12" style="background-color: var(--sf-surface)">
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
