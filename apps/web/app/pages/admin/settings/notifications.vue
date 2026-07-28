<script setup lang="ts">
import { useAdminRoutes } from '~/composables/admin/useAdminRoutes'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminNotificationSettings' })

const route = useRoute()
const adminRoutes = useAdminRoutes()
const tab = route.query.tab === 'channels' ? 'channels' : 'policy'

await navigateTo({
  path: adminRoutes.path('/settings/mail'),
  query: { tab },
  replace: true
})
</script>

<template>
  <div class="py-10 text-center text-sm text-slate-500 dark:text-zinc-400">
    Redirecting…
  </div>
</template>
