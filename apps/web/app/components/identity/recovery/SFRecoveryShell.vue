<script setup lang="ts">
import { useColorModePreference } from '~/composables/appearance/useColorModePreference'

const props = defineProps<{
  phase: 1 | 2 | 3
}>()

const { t } = useI18n()
const localePath = useLocalePath()
const { siteName, siteLogoUrl } = useWebOptions()
const {
  preference: colorModePreference,
  options: colorModeOptions,
  cyclePreference: cycleColorModePreference
} = useColorModePreference()

const currentYear = new Date().getFullYear()
const currentColorModeOption = computed(() =>
  colorModeOptions.find(option => option.value === colorModePreference.value) || colorModeOptions[0]!
)
const colorModeTriggerLabel = computed(() => t('appearance.colorMode.currentPreference', {
  preference: t(currentColorModeOption.value.labelKey)
}))

const steps = computed(() => [
  {
    phase: 1 as const,
    title: t('auth.recovery.stepEmailTitle'),
    description: t('auth.recovery.stepEmailDescription')
  },
  {
    phase: 2 as const,
    title: t('auth.recovery.stepPasswordTitle'),
    description: t('auth.recovery.stepPasswordDescription')
  },
  {
    phase: 3 as const,
    title: t('auth.recovery.stepLoginTitle'),
    description: t('auth.recovery.stepLoginDescription')
  }
])
</script>

<template>
  <main class="sf-recovery-shell">
    <section class="sf-recovery-brand" :aria-labelledby="`sf-recovery-brand-${phase}`">
      <NuxtLink :to="localePath('/')" class="sf-recovery-brand-link">
        <span class="sf-recovery-logo" aria-hidden="true">
          <img v-if="siteLogoUrl" :src="siteLogoUrl" alt="">
          <UIcon v-else name="i-tabler-message-circle-filled" />
        </span>
        <span>{{ siteName }}</span>
      </NuxtLink>

      <div class="sf-recovery-brand-content">
        <h1 :id="`sf-recovery-brand-${phase}`" class="sf-recovery-brand-heading">
          {{ t('auth.recovery.brandHeadlineL1') }}<br>{{ t('auth.recovery.brandHeadlineL2') }}
        </h1>
        <p class="sf-recovery-brand-description">
          {{ t('auth.recovery.brandDescription') }}
        </p>

        <ol class="sf-recovery-progress" :aria-label="t('auth.recovery.progressLabel')">
          <li
            v-for="step in steps"
            :key="step.phase"
            class="sf-recovery-progress-step"
            :class="{
              'is-current': step.phase === props.phase,
              'is-complete': step.phase < props.phase
            }"
            :aria-current="step.phase === props.phase ? 'step' : undefined"
          >
            <span class="sf-recovery-step-marker" aria-hidden="true">
              <UIcon v-if="step.phase < props.phase" name="i-lucide-check" />
              <span v-else>{{ step.phase }}</span>
            </span>
            <span class="sf-recovery-step-title">{{ step.title }}</span>
            <span class="sf-recovery-step-description">{{ step.description }}</span>
          </li>
        </ol>
      </div>

      <footer class="sf-recovery-brand-footer">
        <span class="sf-recovery-privacy">
          <UIcon name="i-lucide-shield-check" aria-hidden="true" />
          {{ t('auth.recovery.privacyProtected') }}
        </span>
        <span>© {{ currentYear }} {{ siteName }}</span>
      </footer>
    </section>

    <section class="sf-recovery-panel" :aria-label="t('auth.recovery.formRegionLabel')">
      <NuxtLink :to="localePath('/')" class="sf-recovery-mobile-brand">
        <span class="sf-recovery-logo" aria-hidden="true">
          <img v-if="siteLogoUrl" :src="siteLogoUrl" alt="">
          <UIcon v-else name="i-tabler-message-circle-filled" />
        </span>
        <span>{{ siteName }}</span>
      </NuxtLink>

      <div class="sf-recovery-panel-tools">
        <ClientOnly>
          <button
            type="button"
            class="sf-recovery-tool-button"
            :aria-label="colorModeTriggerLabel"
            :title="colorModeTriggerLabel"
            @click="cycleColorModePreference"
          >
            <UIcon :name="currentColorModeOption.icon" aria-hidden="true" />
          </button>
          <template #fallback>
            <span class="sf-recovery-tool-placeholder" aria-hidden="true" />
          </template>
        </ClientOnly>
      </div>

      <div class="sf-recovery-form-container">
        <NuxtLink :to="localePath('/login')" class="sf-recovery-back-link">
          <UIcon name="i-lucide-arrow-left" aria-hidden="true" />
          {{ t('auth.backToLogin') }}
        </NuxtLink>
        <slot />
      </div>
    </section>
  </main>
</template>

<style src="./SFRecoveryShell.css" lang="css"></style>
