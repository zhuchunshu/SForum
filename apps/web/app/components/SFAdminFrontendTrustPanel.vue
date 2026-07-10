<script setup lang="ts">
import type { AdminExtension } from '~/utils/adminExtensions'

const props = defineProps<{ extension: AdminExtension }>()
const extension = computed(() => props.extension)
const { user } = useAuthSession()
const { t } = useI18n()
const { status, error, busy, mutate } = useAdminFrontendTrust(extension)
const isSuperAdmin = computed(() => user.value?.roleKeys?.includes('super_admin') === true)
</script>

<template>
  <section v-if="extension.manifest.frontend?.admin" class="mt-4 border-t border-slate-200 pt-4 dark:border-zinc-800">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h4 class="text-sm font-semibold">{{ t('admin.extensions.frontend.title') }}</h4>
        <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ status?.trustState || t('admin.extensions.frontend.loading') }}</p>
      </div>
      <div v-if="isSuperAdmin" class="flex gap-2">
        <UButton v-if="['required', 'invalidated', 'revoked'].includes(status?.trustState || '')" size="xs" icon="i-lucide-shield-check" :loading="busy" @click="mutate('grant')">{{ t('admin.extensions.frontend.grant') }}</UButton>
        <UButton v-if="['trusted', 'revocation_pending'].includes(status?.trustState || '')" size="xs" color="error" variant="subtle" icon="i-lucide-shield-x" :loading="busy" @click="mutate('revoke')">{{ t('admin.extensions.frontend.revoke') }}</UButton>
      </div>
    </div>
    <UAlert v-if="error" class="mt-3" color="error" icon="i-lucide-triangle-alert" :description="error" />
    <dl v-if="status?.declaration" class="mt-3 grid gap-2 text-xs sm:grid-cols-2">
      <div><dt class="text-slate-500">Digest</dt><dd class="break-all font-mono">{{ status.digest }}</dd></div>
      <div><dt class="text-slate-500">API / Root</dt><dd>{{ status.declaration.apiVersion }} / {{ status.declaration.root }}</dd></div>
      <div class="sm:col-span-2"><dt class="text-slate-500">Components</dt><dd class="break-all font-mono">{{ Object.keys(status.declaration.components).join(', ') }}</dd></div>
    </dl>
  </section>
</template>
