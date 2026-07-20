<script setup lang="ts">
/** auth.reset_password 路由壳：auth layout/middleware + outlet；表单岛为 Host 组件。 */
definePageMeta({ public: true })
</script>

<template>
  <SFPageOutlet page="auth.reset_password">
    <SFRecoveryConfirmPage />
  </SFPageOutlet>
</template>
