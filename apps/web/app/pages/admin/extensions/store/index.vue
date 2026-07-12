<script setup lang="ts">
/**
 * 旧「应用商城」入口：重定向到插件商城（默认子页）。
 * 侧栏已拆成「应用商城 → 主题 / 插件」一级菜单。
 */
definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminExtensionStoreRedirect'
})

const adminRoutes = useAdminRoutes()

await navigateTo(adminRoutes.path('/extensions/store/plugins'), { replace: true })
</script>

<template>
  <div class="min-w-0 shrink-0" />
</template>
