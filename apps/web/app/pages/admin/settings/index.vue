<script setup lang="ts">
import { useAdminTabs } from '~/composables/useAdminTabs'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminSettings'
})

const { t } = useI18n()
const toast = useToast()
const { options, fetchEnvelope, save } = useWebOptions()
const adminTabs = useAdminTabs()

onMounted(() => {
  adminTabs.openTab('/settings', 'admin.nav.settings', 'i-lucide-settings-2', 'AdminSettings')
})

const siteName = ref(options.value['site.name'] || 'SForum')
const saving = ref(false)

const { pending, error, refresh } = await useAsyncData('admin-web-options', async () => {
  const envelope = await fetchEnvelope()
  options.value = {
    ...options.value,
    ...Object.fromEntries(envelope.data.map((item) => [item.name, item.value]))
  }
  siteName.value = options.value['site.name'] || 'SForum'
  return envelope.data
})

useSeoMeta({
  title: t('admin.settings.metaTitle')
})

async function submit() {
  saving.value = true
  try {
    await save('site.name', siteName.value)
    toast.add({
      color: 'success',
      icon: 'i-lucide-check',
      title: t('admin.settings.saved')
    })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: apiErrorMessage(error) || t('admin.settings.saveFailed')
    })
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <!-- 局部标题 -->
  <div class="mb-4">
    <h2 class="text-xl font-bold flex items-center gap-2 text-slate-900 dark:text-zinc-100">
      <UIcon name="i-lucide-settings-2" class="size-5 text-teal-600 dark:text-teal-400" />
      {{ t('admin.settings.title') }}
    </h2>
  </div>

  <!-- 整合刷新按钮的统一 Toolbar -->
  <UDashboardToolbar class="border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 rounded-lg px-4 py-2.5 mb-6 text-slate-500 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm text-slate-500 dark:text-zinc-400">
        <UIcon name="i-lucide-database" class="size-4" />
        <span class="truncate">{{ t('admin.settings.intro') }}</span>
      </div>
    </template>
    <template #right>
      <UButton
        color="neutral"
        variant="outline"
        leading-icon="i-lucide-refresh-cw"
        :loading="pending"
        class="border-slate-200 dark:border-zinc-700"
        @click="refresh()"
      >
        {{ t('admin.settings.refresh') }}
      </UButton>
    </template>
  </UDashboardToolbar>

  <div class="flex flex-col gap-4">
    <UAlert
      v-if="error"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="t('admin.settings.loadFailed')"
    />

    <UCard class="border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div>
            <h2 class="text-base font-bold text-slate-900 dark:text-white">
              {{ t('admin.settings.basic.title') }}
            </h2>
            <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.settings.basic.description') }}
            </p>
          </div>
          <UBadge color="neutral" variant="soft" class="border border-slate-200 dark:border-zinc-800 font-mono">
            site.name
          </UBadge>
        </div>
      </template>

      <form class="grid max-w-2xl gap-4" @submit.prevent="submit">
        <UFormField :label="t('admin.settings.siteName')" name="site-name">
          <UInput
            v-model="siteName"
            icon="i-lucide-message-square-text"
            :placeholder="t('admin.settings.siteNamePlaceholder')"
            maxlength="80"
            required
            class="w-full"
          />
        </UFormField>

        <div class="flex justify-end mt-2">
          <UButton
            type="submit"
            leading-icon="i-lucide-save"
            :loading="saving"
            class="bg-teal-600 hover:bg-teal-700 dark:bg-teal-500 dark:hover:bg-teal-600 text-white font-medium"
          >
            {{ t('admin.settings.save') }}
          </UButton>
        </div>
      </form>
    </UCard>
  </div>
</template>
