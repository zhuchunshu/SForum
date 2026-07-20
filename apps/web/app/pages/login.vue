<script setup lang="ts">
/** auth.login 路由壳：auth layout/middleware + outlet；表单岛为 Host 组件。 */
definePageMeta({ layout: 'auth', middleware: 'guest' })
</script>

<template>
  <SFPageOutlet page="auth.login">
    <SFLoginFormPage />
  </SFPageOutlet>
</template>
