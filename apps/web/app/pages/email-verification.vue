<script setup lang="ts">
import SFEmailVerificationPage from '~/components/identity/SFEmailVerificationPage.vue'

definePageMeta({ layout: 'auth', requiresAuth: true })
</script>

<template>
  <SFEmailVerificationPage />
</template>
