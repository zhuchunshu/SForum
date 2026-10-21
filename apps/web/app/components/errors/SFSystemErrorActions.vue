<script setup lang="ts">
import { useSystemErrorPageContext } from '~/composables/errors/useSystemErrorPageContext'
import { useErrorPageActions } from '~/composables/errors/useErrorPageActions'
import type { NuxtError } from '#app'
import { resolveErrorPageContent } from '~/utils/errors/errorPage'
import SFButton from '../SFButton.vue'

const context = useSystemErrorPageContext()
const error = computed(() => context?.error.value || ({ statusCode: 500 } as NuxtError))
const { t } = useI18n()
const page = computed(() => resolveErrorPageContent(error.value?.statusCode))
const title = computed(() => t(page.value.titleKey))
// 动作渲染成真链接：入口 chunk 被拒载时仍能返回首页或整页重试。
const { canGoBack, homeHref, retryHref, goHome, goBack, retry } = useErrorPageActions()
</script>

<template>
  <div class="sforum-system-error__actions" :aria-label="title" data-system-error-region="actions">
    <SFButton size="md" :href="homeHref" @click="goHome">
      <template #leading><UIcon name="i-lucide-home" class="sforum-system-error__button-icon" /></template>
      {{ t('errors.page.home') }}
    </SFButton>
    <SFButton v-if="canGoBack" variant="ghost" size="md" :href="homeHref" @click="goBack">
      <template #leading><UIcon name="i-lucide-arrow-left" class="sforum-system-error__button-icon" /></template>
      {{ t('errors.page.back') }}
    </SFButton>
    <SFButton v-if="page.showRetry" variant="ghost" size="md" :href="retryHref" @click="retry">
      <template #leading><UIcon name="i-lucide-refresh-cw" class="sforum-system-error__button-icon" /></template>
      {{ t('errors.page.retry') }}
    </SFButton>
  </div>
</template>
