<script setup lang="ts">
import { useSystemErrorPageContext } from '~/composables/errors/useSystemErrorPageContext'
import type { NuxtError } from '#app'
import { resolveErrorPageContent } from '~/utils/errors/errorPage'
import SFButton from '../SFButton.vue'

const context = useSystemErrorPageContext()
const error = computed(() => context?.error.value || ({ statusCode: 500 } as NuxtError))
const { t } = useI18n()
const localePath = useLocalePath()
const canGoBack = ref(false)
const page = computed(() => resolveErrorPageContent(error.value?.statusCode))
const title = computed(() => t(page.value.titleKey))

onMounted(() => {
  canGoBack.value = window.history.length > 1
})

function clearResolvedDevErrorOverlay() {
  if (import.meta.dev && import.meta.client) {
    document.querySelector('nuxt-error-overlay.pip-hidden')?.remove()
  }
}

async function goHome() {
  await clearError({ redirect: localePath('/') })
  clearResolvedDevErrorOverlay()
}

async function goBack() {
  if (import.meta.client && window.history.length > 1) {
    await clearError()
    clearResolvedDevErrorOverlay()
    window.history.back()
    return
  }
  await goHome()
}

async function retry() {
  await clearError()
  clearResolvedDevErrorOverlay()
  if (import.meta.client) {
    window.location.reload()
  }
}
</script>

<template>
  <div class="sforum-system-error__actions" :aria-label="title" data-system-error-region="actions">
    <SFButton size="md" @click="goHome">
      <template #leading><UIcon name="i-lucide-home" class="sforum-system-error__button-icon" /></template>
      {{ t('errors.page.home') }}
    </SFButton>
    <SFButton v-if="canGoBack" variant="ghost" size="md" @click="goBack">
      <template #leading><UIcon name="i-lucide-arrow-left" class="sforum-system-error__button-icon" /></template>
      {{ t('errors.page.back') }}
    </SFButton>
    <SFButton v-if="page.showRetry" variant="ghost" size="md" @click="retry">
      <template #leading><UIcon name="i-lucide-refresh-cw" class="sforum-system-error__button-icon" /></template>
      {{ t('errors.page.retry') }}
    </SFButton>
  </div>
</template>
