<script setup lang="ts">
import { FORUM_PERMISSIONS, usePermissions } from '~/composables/identity/usePermissions'
import { useSystemErrorRecoveryData } from '~/composables/errors/useSystemErrorRecoveryData'
import SFHomeNavigation from '~/components/forum/SFHomeNavigation.vue'
import SFResponsivePublicSidebar from '~/components/forum/navigation/SFResponsivePublicSidebar.vue'
const { t } = useI18n()
const { categories, categoriesPending, totalTopics } = useSystemErrorRecoveryData()
const { can } = usePermissions()
const canCreateTopic = computed(() => can(FORUM_PERMISSIONS.topicCreate))
</script>

<template>
  <SFResponsivePublicSidebar
    owner-id="system.error"
    :title="t('home.sidebar.drawerTitle')"
  >
    <SFHomeNavigation
      desktop-only
      navigation-mode="route"
      :categories="categories"
      :total-topics="totalTopics"
      :pending="categoriesPending"
      :can-create-topic="canCreateTopic"
    />
  </SFResponsivePublicSidebar>
</template>
