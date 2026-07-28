<script setup lang="ts">
/**
 * 账号设置左栏 after-nav：资料 / 安全（及可选公开主页）。
 * 样式复用 sf-home-navigation 链接语言，与通知/审核 after-nav 一致。
 */
defineProps<{
  active: 'profile' | 'security' | 'notifications'
  publicProfilePath?: string
}>()

const emit = defineEmits<{
  navigate: []
}>()

const { t } = useI18n()
const localePath = useLocalePath()
</script>

<template>
  <nav class="sforum-settings__account-nav" :aria-label="t('profileSettings.nav.ariaLabel')">
    <div class="sf-home-navigation__label">{{ t('profileSettings.nav.account') }}</div>
    <NuxtLink
      :to="localePath('/settings/profile')"
      class="sf-home-navigation__link"
      :class="{ 'is-active': active === 'profile' }"
      @click="emit('navigate')"
    >
      <span class="sf-home-navigation__link-main">
        <UIcon name="i-lucide-user-round" class="size-[18px]" aria-hidden="true" />
        {{ t('profileSettings.title') }}
      </span>
    </NuxtLink>
    <NuxtLink
      :to="localePath('/settings/security')"
      class="sf-home-navigation__link"
      :class="{ 'is-active': active === 'security' }"
      @click="emit('navigate')"
    >
      <span class="sf-home-navigation__link-main">
        <UIcon name="i-lucide-shield-check" class="size-[18px]" aria-hidden="true" />
        {{ t('accountSecurity.title') }}
      </span>
    </NuxtLink>
    <NuxtLink
      :to="localePath('/settings/notifications')"
      class="sf-home-navigation__link"
      :class="{ 'is-active': active === 'notifications' }"
      @click="emit('navigate')"
    >
      <span class="sf-home-navigation__link-main">
        <UIcon name="i-lucide-bell" class="size-[18px]" aria-hidden="true" />
        {{ t('notificationSettings.title') }}
      </span>
    </NuxtLink>

    <template v-if="publicProfilePath">
      <div class="sf-home-navigation__label">{{ t('profileSettings.nav.space') }}</div>
      <NuxtLink
        :to="publicProfilePath"
        class="sf-home-navigation__link"
        @click="emit('navigate')"
      >
        <span class="sf-home-navigation__link-main">
          <UIcon name="i-lucide-external-link" class="size-[18px]" aria-hidden="true" />
          {{ t('profileSettings.viewPublicProfile') }}
        </span>
      </NuxtLink>
    </template>
  </nav>
</template>
