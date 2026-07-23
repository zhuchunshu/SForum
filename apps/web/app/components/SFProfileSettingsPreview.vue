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
  <section class="sf-profile-settings-canvas__preview-card">
    <header>
      <h3>{{ t('profileSettings.preview.title') }}</h3>
      <NuxtLink :to="publicProfilePath">{{ t('profileSettings.viewPublicProfile') }}</NuxtLink>
    </header>
    <div class="sf-profile-settings-canvas__preview">
      <SFAvatar :name="displayName" :avatar="avatar" size="lg" />
      <h4>{{ displayName }}</h4>
      <p class="sf-profile-settings-canvas__handle">@{{ profile.username }}</p>
      <span v-if="dirty" class="sf-profile-settings-canvas__dirty-badge">
        {{ t('profileSettings.preview.unsaved') }}
      </span>
      <p v-if="bio.trim()" class="sf-profile-settings-canvas__preview-bio">
        {{ bio }}
      </p>
      <div v-if="location.trim() || websiteText || joinedLabel" class="sf-profile-settings-canvas__preview-meta">
        <span v-if="location.trim()">
          <UIcon name="i-lucide-map-pin" class="size-3.5" />
          {{ location }}
        </span>
        <span v-if="websiteText">
          <UIcon name="i-lucide-globe" class="size-3.5" />
          <a v-if="!dirty && websiteHref" :href="websiteHref" target="_blank" rel="noopener noreferrer nofollow">{{ websiteText }}</a>
          <span v-else>{{ websiteText }}</span>
        </span>
        <span v-if="joinedLabel">
          <UIcon name="i-lucide-calendar" class="size-3.5" />
          {{ t('profile.joinedOn', { date: joinedLabel }) }}
        </span>
      </div>
    </div>
  </section>

  <section v-if="showScope" class="sf-profile-settings-canvas__scope-card">
    <header>
      <h3>{{ t('profileSettings.scope.title') }}</h3>
      <span>{{ t('profileSettings.scope.current') }}</span>
    </header>
    <ul>
      <li>
        <UIcon name="i-lucide-eye" class="size-4" />
        <span>{{ t('profileSettings.scope.publicFields') }}</span>
      </li>
      <li>
        <UIcon name="i-lucide-lock-keyhole" class="size-4" />
        <span>{{ t('profileSettings.scope.privateFields') }}</span>
      </li>
    </ul>
  </section>
</template>
