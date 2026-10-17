<script setup lang="ts">
import { useSForumSeo } from '~/composables/seo/useSForumSeo'
import SFLinkedAccountsSection from '~/components/identity/SFLinkedAccountsSection.vue'
import SFSettingsShell from '~/components/settings/SFSettingsShell.vue'
import SFSettingsSecurityTabNav from '~/components/settings/SFSettingsSecurityTabNav.vue'

/**
 * 宿主 body 岛：forum.settings.login_methods。
 * 「账号安全」的登录方式 tab：专注外部身份绑定；密码跳转到兄弟 tab 路由。
 */
const { t } = useI18n()
const localePath = useLocalePath()

useSForumSeo({
  title: () => t('loginMethodsSettings.metaTitle'),
  description: () => t('loginMethodsSettings.metaDescription'),
  type: 'website',
  noindex: true
})
</script>

<template>
  <SFSettingsShell
    class="sforum-settings-login-methods"
    data-sforum-island-body="identity.component.login_methods_settings"
    active="security"
    title-id="login-methods-settings-title"
    :title="t('loginMethodsSettings.title')"
    :description="t('loginMethodsSettings.intro')"
    :rail-label="t('loginMethodsSettings.rail.ariaLabel')"
  >
    <SFSettingsSecurityTabNav />

    <section class="rounded-md border border-teal-200 bg-teal-50/70 p-4 dark:border-teal-900/60 dark:bg-teal-950/25">
      <div class="flex gap-3">
        <UIcon name="i-lucide-shield-check" class="size-5 shrink-0 text-teal-700 dark:text-teal-300" aria-hidden="true" />
        <div class="min-w-0">
          <h2 class="font-semibold text-slate-900 dark:text-zinc-50">
            {{ t('loginMethodsSettings.recommendedTitle') }}
          </h2>
          <p class="mt-1 text-sm text-muted">
            {{ t('loginMethodsSettings.recommendedDescription') }}
          </p>
        </div>
      </div>
    </section>

    <SFLinkedAccountsSection
      class="mt-5"
      return-path="/settings/security/login-methods"
      :show-heading="false"
    />

    <section class="mt-5 rounded-md border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div class="min-w-0">
          <h2 class="text-base font-semibold text-slate-900 dark:text-zinc-50">
            {{ t('loginMethodsSettings.passwordCtaTitle') }}
          </h2>
          <p class="mt-1 text-sm text-muted">
            {{ t('loginMethodsSettings.passwordCtaDescription') }}
          </p>
        </div>
        <NuxtLink
          :to="localePath('/settings/security/password')"
          class="inline-flex min-h-9 shrink-0 items-center justify-center gap-1.5 rounded-md border border-slate-200 px-3 text-sm font-semibold text-slate-700 hover:bg-slate-50 dark:border-zinc-700 dark:text-zinc-200 dark:hover:bg-zinc-800"
        >
          <UIcon name="i-lucide-lock-keyhole" class="size-4" aria-hidden="true" />
          {{ t('loginMethodsSettings.passwordCtaAction') }}
        </NuxtLink>
      </div>
    </section>

    <template #rail>
      <section class="sforum-settings__rail-section">
        <div class="sforum-settings__rail-head">
          <h2>{{ t('loginMethodsSettings.rail.methodsTitle') }}</h2>
          <span>{{ t('loginMethodsSettings.rail.currentPage') }}</span>
        </div>
        <p class="sforum-settings__rail-help">{{ t('loginMethodsSettings.rail.methodsHelp') }}</p>
      </section>

      <section class="sforum-settings__rail-section">
        <div class="sforum-settings__rail-head">
          <h2>{{ t('loginMethodsSettings.rail.passwordTitle') }}</h2>
          <span>{{ t('loginMethodsSettings.rail.backup') }}</span>
        </div>
        <p class="sforum-settings__rail-help">{{ t('loginMethodsSettings.rail.passwordHelp') }}</p>
      </section>
    </template>
  </SFSettingsShell>
</template>
