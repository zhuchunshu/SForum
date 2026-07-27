<script setup lang="ts">
import SFRecoveryRequestPage from '~/components/identity/SFRecoveryRequestPage.vue'
/** auth.forgot_password 路由壳：auth layout/middleware + outlet；表单岛为 Host 组件。 */
definePageMeta({ public: true })
</script>

<template>
  <SFPageOutlet page="auth.forgot_password">
    <SFRecoveryRequestPage />
  </SFPageOutlet>
</template>
