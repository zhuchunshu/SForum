<script setup lang="ts">
import SFExternalAuthContinuationPage from '~/components/identity/SFExternalAuthContinuationPage.vue'

/** auth.external_continuation 路由壳：游客选择与登录后绑定共用同一页面。 */
definePageMeta({ layout: 'auth' })
</script>

<template>
  <SFPageOutlet page="auth.external_continuation">
    <SFExternalAuthContinuationPage />
  </SFPageOutlet>
</template>
