<script setup lang="ts">
import type { NuxtError } from '#app'
import { resolveErrorPageContent } from '~/utils/errorPage'

const props = defineProps<{
  error: NuxtError
}>()

const { t } = useI18n()
const localePath = useLocalePath()
const { siteName } = useWebOptions()
const route = useRoute()
const canGoBack = ref(false)
const headingId = 'sforum-error-title'

const page = computed(() => resolveErrorPageContent(props.error?.statusCode))
const statusLabel = computed(() => t('errors.page.statusLabel', { statusCode: page.value.statusCode }))
const title = computed(() => t(page.value.titleKey, { siteName: siteName.value }))
const description = computed(() => t(page.value.descriptionKey, { siteName: siteName.value }))

useSForumSeo({
  title,
  description,
  path: () => route.path,
  noindex: true,
  schema: { type: 'WebPage' }
})

onMounted(() => {
  canGoBack.value = window.history.length > 1
})

async function goHome() {
  await clearError({ redirect: localePath('/') })
}

async function goBack() {
  if (import.meta.client && window.history.length > 1) {
    await clearError()
    window.history.back()
    return
  }

  await goHome()
}

async function retry() {
  await clearError()
  if (import.meta.client) {
    window.location.reload()
  }
}
</script>

<template>
  <div class="sforum-error-page">
    <SFNavbar />

    <main class="sforum-error-page__main">
      <section class="sforum-error-page__panel" :aria-labelledby="headingId">
        <div class="sforum-error-page__icon" aria-hidden="true">
          <UIcon :name="page.icon" class="sforum-error-page__icon-svg" />
        </div>

        <p class="sforum-error-page__status">
          {{ statusLabel }}
        </p>

        <h1 :id="headingId" class="sforum-error-page__title">
          {{ title }}
        </h1>

        <p class="sforum-error-page__description">
          {{ description }}
        </p>

        <div class="sforum-error-page__actions" :aria-label="title">
          <SFButton size="md" @click="goHome">
            <template #leading>
              <UIcon name="i-lucide-home" class="sforum-error-page__button-icon" />
            </template>
            {{ t('errors.page.home') }}
          </SFButton>

          <SFButton v-if="canGoBack" variant="ghost" size="md" @click="goBack">
            <template #leading>
              <UIcon name="i-lucide-arrow-left" class="sforum-error-page__button-icon" />
            </template>
            {{ t('errors.page.back') }}
          </SFButton>

          <SFButton v-if="page.showRetry" variant="ghost" size="md" @click="retry">
            <template #leading>
              <UIcon name="i-lucide-refresh-cw" class="sforum-error-page__button-icon" />
            </template>
            {{ t('errors.page.retry') }}
          </SFButton>
        </div>
      </section>
    </main>

    <SFFooter />
  </div>
</template>

<style scoped>
.sforum-error-page {
  display: flex;
  min-height: 100vh;
  flex-direction: column;
  background: var(--sf-surface);
  color: var(--sf-fg);
}

.sforum-error-page__main {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  padding: 48px 16px;
}

.sforum-error-page__panel {
  width: min(100%, 620px);
  padding: 32px;
  border: 1px solid var(--sf-border);
  border-radius: 12px;
  background: var(--sf-card);
  box-shadow: 0 18px 45px rgba(15, 23, 42, 0.08);
  text-align: center;
}

.dark .sforum-error-page__panel {
  box-shadow: none;
}

.sforum-error-page__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  margin-bottom: 18px;
  border: 1px solid var(--sf-accent-soft-border);
  border-radius: 14px;
  background: var(--sf-accent-soft);
  color: var(--sf-accent);
}

.sforum-error-page__icon-svg {
  width: 28px;
  height: 28px;
}

.sforum-error-page__status {
  display: inline-flex;
  align-items: center;
  min-height: 28px;
  margin: 0 0 14px;
  padding: 0 10px;
  border: 1px solid var(--sf-accent-soft-border);
  border-radius: 7px;
  background: var(--sf-accent-soft);
  color: var(--sf-accent);
  font-size: 0.78rem;
  font-weight: 800;
  line-height: 1;
}

.sforum-error-page__title {
  margin: 0;
  color: var(--sf-fg);
  font-size: clamp(1.8rem, 4vw, 2.65rem);
  font-weight: 850;
  line-height: 1.12;
  letter-spacing: 0;
}

.sforum-error-page__description {
  max-width: 34rem;
  margin: 14px auto 0;
  color: var(--sf-fg-tertiary);
  font-size: 0.98rem;
  line-height: 1.7;
}

.sforum-error-page__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 10px;
  margin-top: 26px;
}

.sforum-error-page__button-icon {
  width: 1rem;
  height: 1rem;
}

@media (max-width: 640px) {
  .sforum-error-page__main {
    align-items: stretch;
    padding: 28px 14px;
  }

  .sforum-error-page__panel {
    padding: 26px 18px;
  }

  .sforum-error-page__actions {
    flex-direction: column;
  }
}
</style>
