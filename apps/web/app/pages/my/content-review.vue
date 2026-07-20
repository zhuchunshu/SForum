<script setup lang="ts">
/** forum.my.content_review 路由壳：outlet + fail-closed；呈现由主题 L1 + SFMyContentReviewPage 岛拥有。 */
definePageMeta({ requiresAuth: true })
</script>

<template>
  <SFPageOutlet page="forum.my.content_review">
    <SFMyContentReviewPage />
  </SFPageOutlet>
</template>
