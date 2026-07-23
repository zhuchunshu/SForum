<script setup lang="ts">
import type { AvatarView, PublicProfile } from '~/composables/useProfileApi'

defineProps<{
  profile: PublicProfile
  displayName: string
  avatar: AvatarView | null
  bio: string
  location: string
  websiteText: string
  websiteHref: string
  joinedLabel: string
  dirty: boolean
  publicProfilePath: string
  showScope?: boolean
}>()

const { t } = useI18n()
</script>

<template>
  <section class="sforum-settings__rail-section">
    <div class="sforum-settings__rail-head">
      <h2>{{ t('profileSettings.preview.title') }}</h2>
      <NuxtLink :to="publicProfilePath">{{ t('profileSettings.viewPublicProfile') }}</NuxtLink>
    </div>

    <div class="sforum-settings-preview__person">
      <SFAvatar :name="displayName" :avatar="avatar" size="sm" />
      <div>
        <strong>{{ displayName }}</strong>
        <span>@{{ profile.username }}</span>
      </div>
    </div>

    <span v-if="dirty" class="sforum-settings-preview__dirty">
      {{ t('profileSettings.preview.unsaved') }}
    </span>

    <p v-if="bio.trim()" class="sforum-settings-preview__bio">
      {{ bio }}
    </p>
    <p v-else class="sforum-settings-preview__bio">
      {{ t('profile.bioEmpty') }}
    </p>

    <div v-if="location.trim() || websiteText || joinedLabel" class="sforum-settings-preview__meta">
      <span v-if="location.trim()">
        <UIcon name="i-lucide-map-pin" class="size-3.5" aria-hidden="true" />
        {{ location }}
      </span>
      <span v-if="websiteText">
        <UIcon name="i-lucide-globe" class="size-3.5" aria-hidden="true" />
        <a v-if="!dirty && websiteHref" :href="websiteHref" target="_blank" rel="noopener noreferrer nofollow">{{ websiteText }}</a>
        <span v-else>{{ websiteText }}</span>
      </span>
      <span v-if="joinedLabel">
        <UIcon name="i-lucide-calendar" class="size-3.5" aria-hidden="true" />
        {{ t('profile.joinedOn', { date: joinedLabel }) }}
      </span>
    </div>
  </section>

  <section v-if="showScope" class="sforum-settings__rail-section">
    <div class="sforum-settings__rail-head">
      <h2>{{ t('profileSettings.scope.title') }}</h2>
      <span>{{ t('profileSettings.scope.current') }}</span>
    </div>
    <ul class="sforum-settings-preview__scope">
      <li>
        <UIcon name="i-lucide-eye" class="size-4" aria-hidden="true" />
        <span>{{ t('profileSettings.scope.publicFields') }}</span>
      </li>
      <li>
        <UIcon name="i-lucide-lock-keyhole" class="size-4" aria-hidden="true" />
        <span>{{ t('profileSettings.scope.privateFields') }}</span>
      </li>
    </ul>
  </section>
</template>
