<script setup lang="ts">
// 品牌与前台壳已并入个性化设置；保留路由以兼容旧书签与侧栏历史。
definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminSiteChromeRedirect'
})

const route = useRoute()
const adminRoutes = useAdminRoutes()

const tab = typeof route.query.tab === 'string' ? route.query.tab : 'brand'
const target = adminRoutes.path('/personalization')

await navigateTo({
  path: target,
  query: { tab },
  replace: true
})
</script>

<template>
  <div class="py-10 text-center text-sm text-slate-500 dark:text-zinc-400">
    Redirecting…
  </div>
</template>
