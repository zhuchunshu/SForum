<script setup lang="ts">
/** moderation.review 路由壳：outlet + fail-closed；呈现由主题 L1 + SFModerationReviewPage 岛拥有。 */
import SFModerationReviewPage from '~/components/moderation/SFModerationReviewPage.vue'

definePageMeta({ requiresAuth: true, middleware: ['moderation-review'] })
</script>

<template>
  <SFPageOutlet page="moderation.review">
    <SFModerationReviewPage />
  </SFPageOutlet>
</template>
