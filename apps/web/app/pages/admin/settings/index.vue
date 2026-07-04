<script setup lang="ts">
definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

const { t } = useI18n()
const toast = useToast()
const { options, fetchEnvelope, save } = useWebOptions()

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
  <UDashboardNavbar :title="t('admin.settings.title')" icon="i-lucide-settings-2">
    <template #right>
      <UButton
        color="neutral"
        variant="outline"
        leading-icon="i-lucide-refresh-cw"
        :loading="pending"
        @click="refresh()"
      >
        {{ t('admin.settings.refresh') }}
      </UButton>
    </template>
  </UDashboardNavbar>

  <UDashboardToolbar>
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm text-muted">
        <UIcon name="i-lucide-database" class="size-4" />
        <span class="truncate">{{ t('admin.settings.intro') }}</span>
      </div>
    </template>
  </UDashboardToolbar>

  <div class="flex flex-1 flex-col gap-4 p-4 sm:p-6">
    <UAlert
      v-if="error"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="t('admin.settings.loadFailed')"
    />

    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div>
            <h2 class="text-base font-semibold text-highlighted">
              {{ t('admin.settings.basic.title') }}
            </h2>
            <p class="mt-1 text-sm text-muted">
              {{ t('admin.settings.basic.description') }}
            </p>
          </div>
          <UBadge color="neutral" variant="soft">
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
          />
        </UFormField>

        <div class="flex justify-end">
          <UButton
            type="submit"
            leading-icon="i-lucide-save"
            :loading="saving"
          >
            {{ t('admin.settings.save') }}
          </UButton>
        </div>
      </form>
    </UCard>
  </div>
</template>
