<script setup lang="ts">
import SFEmailVerificationPage from '~/components/identity/SFEmailVerificationPage.vue'

/** auth.email_verification 路由壳：登录态守卫 + outlet；验证行为由 Host 岛持有。 */
definePageMeta({ layout: 'auth', requiresAuth: true })
</script>

<template>
  <SFPageOutlet page="auth.email_verification">
    <SFEmailVerificationPage />
  </SFPageOutlet>
</template>
