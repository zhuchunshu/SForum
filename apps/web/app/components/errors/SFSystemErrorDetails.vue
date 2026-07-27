<script setup lang="ts">
import { useSystemErrorPageContext } from '~/composables/errors/useSystemErrorPageContext'
import type { NuxtError } from '#app'
import { resolveErrorPageContent } from '~/utils/errors/errorPage'

const context = useSystemErrorPageContext()
const error = computed(() => context?.error.value || ({ statusCode: 500 } as NuxtError))
const { t } = useI18n()
const { siteName } = useWebOptions()
const headingId = 'sforum-system-error-title'
const page = computed(() => resolveErrorPageContent(error.value?.statusCode))
const statusLabel = computed(() => t('errors.page.statusLabel', { statusCode: page.value.statusCode }))
const title = computed(() => t(page.value.titleKey, { siteName: siteName.value }))
const description = computed(() => t(page.value.descriptionKey, { siteName: siteName.value }))
</script>

<template>
  <section class="sforum-system-error__details" :aria-labelledby="headingId" data-system-error-region="details">
    <div class="sforum-system-error__status" :aria-label="statusLabel">
      <span class="sforum-system-error__status-code">{{ page.statusCode }}</span>
      <span class="sforum-system-error__status-caption">{{ t('errors.page.statusCaption') }}</span>
    </div>
    <div class="sforum-system-error__copy">
      <h1 :id="headingId" class="sforum-system-error__title">{{ title }}</h1>
      <p class="sforum-system-error__description">{{ description }}</p>
    </div>
  </section>
</template>
