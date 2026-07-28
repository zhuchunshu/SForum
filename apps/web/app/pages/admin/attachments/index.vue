<script setup lang="ts">
import { useAdminRoutes } from '~/composables/admin/useAdminRoutes'
import { useAuthSession } from '~/composables/identity/useAuthSession'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminAttachmentsRedirect' })

const route = useRoute()
const adminRoutes = useAdminRoutes()
const { can } = useAuthSession()

const requestedPage = route.query.tab === 'manager'
  ? '/attachments/manager'
  : '/attachments/settings'
const fallbackPage = can('attachment.settings.manage')
  ? '/attachments/settings'
  : can('attachment.manage')
    ? '/attachments/manager'
    : '/'
const targetPage = requestedPage === '/attachments/manager'
  ? (can('attachment.manage') ? requestedPage : fallbackPage)
  : (can('attachment.settings.manage') ? requestedPage : fallbackPage)

await navigateTo(adminRoutes.path(targetPage), { replace: true })
</script>

<template>
  <div class="min-w-0 shrink-0" />
</template>
