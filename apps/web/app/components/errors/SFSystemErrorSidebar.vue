<script setup lang="ts">
import { FORUM_PERMISSIONS, usePermissions } from '~/composables/identity/usePermissions'
import { useSystemErrorRecoveryData } from '~/composables/errors/useSystemErrorRecoveryData'
import SFHomeNavigation from '~/components/forum/SFHomeNavigation.vue'
const { categories, categoriesPending, totalTopics } = useSystemErrorRecoveryData()
const { can } = usePermissions()
const canCreateTopic = computed(() => can(FORUM_PERMISSIONS.topicCreate))
</script>

<template>
  <SFHomeNavigation
    desktop-only
    navigation-mode="route"
    :categories="categories"
    :total-topics="totalTopics"
    :pending="categoriesPending"
    :can-create-topic="canCreateTopic"
  />
</template>
