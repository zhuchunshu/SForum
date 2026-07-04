<script setup lang="ts">
import type { CurrentUser } from '~/composables/useAuthSession'

const { t } = useI18n()
const localePath = useLocalePath()
const apiBaseUrl = useRuntimeConfig().public.apiBaseUrl as string
const { refresh, can } = useAuthSession()

const form = reactive({
  login: '',
  password: ''
})
const submitting = ref(false)
const errorMessage = ref('')

useSeoMeta({
  title: t('auth.loginTitle')
})

async function submitLogin() {
  errorMessage.value = ''
  submitting.value = true

  try {
    await $fetch<CurrentUser>(`${apiBaseUrl}/auth/login`, {
      method: 'POST',
      credentials: 'include',
      body: {
        login: form.login,
        password: form.password
      }
    })
    await refresh()
    await navigateTo(localePath(can('admin.access') ? '/admin' : '/'))
  } catch {
    errorMessage.value = t('errors.loginFailed')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main class="page-shell auth-page">
    <section class="forum-board auth-board" aria-labelledby="login-title">
      <p class="eyebrow">
        {{ t('auth.loginEyebrow') }}
      </p>
      <h1 id="login-title">
        {{ t('auth.loginTitle') }}
      </h1>
      <p class="intro">
        {{ t('auth.loginIntro') }}
      </p>

      <form class="auth-form" @submit.prevent="submitLogin">
        <SFAlert
          v-if="errorMessage"
          :title="errorMessage"
          variant="danger"
          compact
        />
        <SFInput
          v-model="form.login"
          name="login"
          :label="t('auth.loginName')"
          required
        />
        <SFInput
          v-model="form.password"
          name="password"
          type="password"
          :label="t('auth.password')"
          required
        />
        <SFButton type="submit" :loading="submitting" block>
          {{ t('auth.submitLogin') }}
        </SFButton>
      </form>

      <p class="auth-switch">
        {{ t('auth.needAccount') }}
        <NuxtLink :to="localePath('/register')">
          {{ t('auth.goRegister') }}
        </NuxtLink>
      </p>
    </section>
  </main>
</template>

<style scoped>
.auth-page {
  display: grid;
  align-items: center;
}

.auth-board {
  max-width: 560px;
}

.auth-form {
  display: grid;
  gap: 16px;
  margin-top: 28px;
}

.auth-switch {
  margin: 18px 0 0;
  color: var(--sf-fg-secondary);
  font-size: 14px;
}

.auth-switch a {
  color: var(--sf-accent);
  font-weight: 700;
}
</style>
