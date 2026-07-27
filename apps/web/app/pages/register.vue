<script setup lang="ts">
import SFRegisterFormPage from '~/components/identity/SFRegisterFormPage.vue'
/** auth.register 路由壳：auth layout/middleware + outlet；表单岛为 Host 组件。 */
definePageMeta({ layout: 'auth', middleware: 'guest' })
</script>

<template>
  <SFPageOutlet page="auth.register">
    <SFRegisterFormPage />
  </SFPageOutlet>
</template>
