<script setup lang="ts">
import type { CurrentUser } from '~/composables/useAuthSession'

const { t, locale } = useI18n()
const localePath = useLocalePath()
const apiBaseUrl = useRuntimeConfig().public.apiBaseUrl as string
const { refresh, can } = useAuthSession()

const form = reactive({
  username: '',
  email: '',
  password: '',
  displayName: ''
})
const submitting = ref(false)
const errorMessage = ref('')

useSeoMeta({
  title: t('auth.registerTitle')
})

async function submitRegister() {
  errorMessage.value = ''
  submitting.value = true

  try {
    await $fetch<CurrentUser>(`${apiBaseUrl}/auth/register`, {
      method: 'POST',
      credentials: 'include',
      body: {
        username: form.username,
        email: form.email,
        password: form.password,
        displayName: form.displayName,
        locale: locale.value
      }
    })
    await refresh()
    await navigateTo(localePath(can('admin.access') ? '/admin' : '/'))
  } catch {
    errorMessage.value = t('errors.registerFailed')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main class="page-shell auth-page">
    <section class="forum-board auth-board" aria-labelledby="register-title">
      <p class="eyebrow">
        {{ t('auth.registerEyebrow') }}
      </p>
      <h1 id="register-title">
        {{ t('auth.registerTitle') }}
      </h1>
      <p class="intro">
        {{ t('auth.registerIntro') }}
      </p>

      <form class="auth-form" @submit.prevent="submitRegister">
        <SFAlert
          v-if="errorMessage"
          :title="errorMessage"
          variant="danger"
          compact
        />
        <SFInput
          v-model="form.username"
          name="username"
          :label="t('auth.username')"
          required
        />
        <SFInput
          v-model="form.email"
          name="email"
          type="email"
          :label="t('auth.email')"
          required
        />
        <SFInput
          v-model="form.displayName"
          name="displayName"
          :label="t('auth.displayName')"
        />
        <SFInput
          v-model="form.password"
          name="password"
          type="password"
          :label="t('auth.password')"
          required
        />
        <SFButton type="submit" :loading="submitting" block>
          {{ t('auth.submitRegister') }}
        </SFButton>
      </form>

      <p class="auth-switch">
        {{ t('auth.haveAccount') }}
        <NuxtLink :to="localePath('/login')">
          {{ t('auth.goLogin') }}
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
