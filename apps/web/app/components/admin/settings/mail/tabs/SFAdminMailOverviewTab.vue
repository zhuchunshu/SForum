<script setup lang="ts">
import type { MailProvider } from '../model'
const { t } = useI18n()
const { request } = useApiClient()
const pending = ref(true)
const configured = ref(false)
const errorMessage = ref('')
onMounted(load)
async function load() {
  pending.value = true
  try { configured.value = (await request<{ items: MailProvider[], configured: boolean }>('/admin/mail/providers')).configured } catch (error) { errorMessage.value = apiErrorMessage(error) || t('admin.mailSettings.loadFailed') } finally { pending.value = false }
}
defineExpose({ refresh: load, pending })
</script>
<template>
  <UCard class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
    <template #header><div><h2 class="text-base font-bold">{{ t('admin.mailSettings.gettingStarted') }}</h2><p class="mt-1 text-xs text-muted">{{ t('admin.mailSettings.description') }}</p></div></template>
    <SFAlert v-if="errorMessage" variant="danger" :title="errorMessage" />
    <div class="grid gap-4 md:grid-cols-2">
      <div class="rounded-md border border-emerald-200 bg-emerald-50/60 p-4 dark:border-emerald-900 dark:bg-emerald-950/20"><div class="flex justify-between gap-3"><strong>{{ t('admin.mailSettings.inAppStatus') }}</strong><UBadge color="success" variant="soft">{{ t('admin.mailSettings.ready') }}</UBadge></div><p class="mt-2 text-sm">{{ t('admin.mailSettings.inAppReady') }}</p></div>
      <div class="rounded-md border p-4" :class="configured ? 'border-emerald-200' : 'border-amber-200'"><div class="flex justify-between gap-3"><strong>{{ t('admin.mailSettings.mailStatus') }}</strong><UBadge :color="configured ? 'success' : 'warning'" variant="soft">{{ configured ? t('admin.mailSettings.ready') : t('admin.mailSettings.needsSetup') }}</UBadge></div><p class="mt-2 text-sm">{{ configured ? t('admin.mailSettings.providerReady') : t('admin.mailSettings.inAppContinues') }}</p></div>
    </div>
    <ol class="mt-6 grid gap-3 lg:grid-cols-3">
      <li
        v-for="(step, index) in [
          { title: t('admin.mailSettings.stepProvider'), help: t('admin.mailSettings.stepProviderHelp') },
          { title: t('admin.mailSettings.stepConfigure'), help: t('admin.mailSettings.stepConfigureHelp') },
          { title: t('admin.mailSettings.stepTest'), help: t('admin.mailSettings.stepTestHelp') }
        ]"
        :key="step.title"
        class="flex gap-3 rounded-md border border-slate-200 p-4 dark:border-zinc-800"
      >
        <span class="grid size-7 shrink-0 place-items-center rounded-full bg-[var(--sf-accent)] text-xs font-bold text-white">{{ index + 1 }}</span>
        <span><strong class="block text-sm">{{ step.title }}</strong><span class="mt-1 block text-xs leading-5 text-muted">{{ step.help }}</span></span>
      </li>
    </ol>
  </UCard>
</template>
