<script setup lang="ts">
import { useColorModePreference } from '~/composables/appearance/useColorModePreference'

const props = defineProps<{
  recoveryPhase?: 1 | 2 | 3
})

const { t } = useI18n()
const localePath = useLocalePath()
const { siteLogoUrl, siteName, siteTagline } = useWebOptions()
const {
  preference: colorModePreference,
  options: colorModeOptions,
  cyclePreference: cycleColorModePreference
} = useColorModePreference()

const isRecovery = computed(() => props.recoveryPhase !== undefined)
const currentRecoveryPhase = computed(() => props.recoveryPhase ?? 1)
const brandDescription = computed(() => siteTagline.value || t(
  isRecovery.value ? 'auth.recovery.brandDescription' : 'auth.brandDesc'
))
const currentColorModeOption = computed(() =>
  colorModeOptions.find(option => option.value === colorModePreference.value) || colorModeOptions[0]!
)
const colorModeTriggerLabel = computed(() => t('appearance.colorMode.currentPreference', {
  preference: t(currentColorModeOption.value.labelKey)
}))
const recoverySteps = computed(() => [
  { phase: 1 as const, title: t('auth.recovery.stepEmailTitle'), description: t('auth.recovery.stepEmailDescription') },
  { phase: 2 as const, title: t('auth.recovery.stepPasswordTitle'), description: t('auth.recovery.stepPasswordDescription') },
  { phase: 3 as const, title: t('auth.recovery.stepLoginTitle'), description: t('auth.recovery.stepLoginDescription') }
])
</script>

<template>
  <main class="sf-auth-shell" :class="{ 'sf-auth-shell--recovery': isRecovery }">
    <aside class="sf-auth-shell__brand" :aria-labelledby="isRecovery ? `sf-auth-brand-${currentRecoveryPhase}` : 'sf-auth-brand'">
      <NuxtLink :to="localePath('/')" class="sf-auth-shell__brand-link">
        <span class="sf-auth-shell__logo" aria-hidden="true">
          <img v-if="siteLogoUrl" :src="siteLogoUrl" alt="">
          <UIcon v-else name="i-tabler-message-circle-filled" />
        </span>
        <span>{{ siteName }}</span>
      </NuxtLink>

      <div class="sf-auth-shell__brand-content">
        <h1 :id="isRecovery ? `sf-auth-brand-${currentRecoveryPhase}` : 'sf-auth-brand'" class="sf-auth-shell__headline">
          <template v-if="isRecovery">
            {{ t('auth.recovery.brandHeadlineL1') }}<br>{{ t('auth.recovery.brandHeadlineL2') }}
          </template>
          <template v-else>
            {{ t('auth.brandHeadlineL1') }}<br>{{ t('auth.brandHeadlineL2') }}
          </template>
        </h1>
        <p class="sf-auth-shell__description">{{ brandDescription }}</p>

        <ol v-if="isRecovery" class="sf-auth-shell__progress" :aria-label="t('auth.recovery.progressLabel')">
          <li
            v-for="step in recoverySteps"
            :key="step.phase"
            class="sf-auth-shell__progress-step"
            :class="{ 'is-current': step.phase === currentRecoveryPhase, 'is-complete': step.phase < currentRecoveryPhase }"
            :aria-current="step.phase === currentRecoveryPhase ? 'step' : undefined"
          >
            <span class="sf-auth-shell__step-marker" aria-hidden="true">
              <UIcon v-if="step.phase < currentRecoveryPhase" name="i-lucide-check" />
              <span v-else>{{ step.phase }}</span>
            </span>
            <span class="sf-auth-shell__step-title">{{ step.title }}</span>
            <span class="sf-auth-shell__step-description">{{ step.description }}</span>
          </li>
        </ol>
        <p v-if="isRecovery" class="sf-auth-shell__recovery-note">
          <UIcon name="i-lucide-shield-check" aria-hidden="true" />
          {{ t('auth.recovery.privacyProtected') }}
        </p>

        <ul v-else class="sf-auth-shell__features">
          <li v-for="key in ['auth.feature1', 'auth.feature2', 'auth.feature3']" :key="key">
            <span aria-hidden="true"><UIcon name="i-lucide-check" /></span>
            {{ t(key) }}
          </li>
        </ul>
      </div>
    </aside>

    <section class="sf-auth-shell__panel" :aria-label="isRecovery ? t('auth.recovery.formRegionLabel') : undefined">
      <NuxtLink :to="localePath('/')" class="sf-auth-shell__mobile-brand">
        <span class="sf-auth-shell__logo" aria-hidden="true">
          <img v-if="siteLogoUrl" :src="siteLogoUrl" alt="">
          <UIcon v-else name="i-tabler-message-circle-filled" />
        </span>
        <span>{{ siteName }}</span>
      </NuxtLink>

      <ClientOnly>
        <button
          type="button"
          class="sf-auth-shell__color-mode"
          :aria-label="colorModeTriggerLabel"
          :title="colorModeTriggerLabel"
          @click="cycleColorModePreference"
        >
          <UIcon :name="currentColorModeOption.icon" aria-hidden="true" />
        </button>
        <template #fallback><span class="sf-auth-shell__color-mode-placeholder" aria-hidden="true" /></template>
      </ClientOnly>

      <div class="sf-auth-shell__content">
        <NuxtLink v-if="isRecovery" :to="localePath('/login')" class="sf-auth-shell__back-link">
          <UIcon name="i-lucide-arrow-left" aria-hidden="true" />
          {{ t('auth.backToLogin') }}
        </NuxtLink>
        <slot />
      </div>
    </section>
  </main>
</template>

<style scoped>
.sf-auth-shell { display: grid; min-height: 100svh; grid-template-columns: minmax(360px, 46%) minmax(0, 54%); background: var(--sf-card); color: var(--sf-fg); }
.sf-auth-shell__brand { display: flex; min-height: 100%; flex-direction: column; padding: 42px clamp(40px, 5vw, 76px); border-right: 1px solid var(--sf-border); background: color-mix(in srgb, var(--sf-muted) 72%, var(--sf-card)); }
.sf-auth-shell__brand-link, .sf-auth-shell__mobile-brand { display: inline-flex; width: fit-content; align-items: center; gap: 10px; color: var(--sf-fg); font-size: 16px; font-weight: 740; text-decoration: none; }
.sf-auth-shell__logo { display: grid; width: 32px; height: 32px; flex: 0 0 32px; place-items: center; overflow: hidden; border-radius: 7px; color: var(--sf-accent); }
.sf-auth-shell__logo img, .sf-auth-shell__logo svg { display: block; width: 100%; height: 100%; object-fit: contain; }
.sf-auth-shell__brand-content { width: min(100%, 470px); margin: auto 0; }
.sf-auth-shell__headline { max-width: 420px; margin: 0; color: var(--sf-fg); font-size: 38px; font-weight: 720; line-height: 1.28; letter-spacing: 0; }
.sf-auth-shell__description { max-width: 430px; margin: 18px 0 0; color: var(--sf-fg-secondary); font-size: 15px; line-height: 1.8; }
.sf-auth-shell__features { display: grid; gap: 13px; margin: 34px 0 0; padding: 0; list-style: none; color: var(--sf-fg-secondary); font-size: 13.5px; line-height: 1.5; }
.sf-auth-shell__features li { display: flex; align-items: flex-start; gap: 11px; }
.sf-auth-shell__features li > span { display: grid; width: 20px; height: 20px; flex: 0 0 20px; place-items: center; border: 1px solid var(--sf-border); border-radius: 5px; background: var(--sf-card); color: var(--sf-accent); }
.sf-auth-shell__features svg { width: 12px; height: 12px; }
.sf-auth-shell__progress { display: grid; gap: 0; margin: 52px 0 0; padding: 0; list-style: none; }
.sf-auth-shell__progress-step { position: relative; display: grid; min-height: 72px; grid-template-columns: 34px 1fr; grid-template-rows: auto auto; column-gap: 14px; }
.sf-auth-shell__progress-step:not(:last-child)::after { position: absolute; top: 30px; bottom: 3px; left: 16px; width: 1px; background: var(--sf-border); content: ''; }
.sf-auth-shell__progress-step.is-complete:not(:last-child)::after { background: var(--sf-accent); }
.sf-auth-shell__step-marker { z-index: 1; display: grid; width: 32px; height: 32px; grid-row: 1 / span 2; place-items: center; border: 1px solid var(--sf-border); border-radius: 50%; background: var(--sf-muted); color: var(--sf-fg-tertiary); font-size: 12px; font-weight: 700; }
.sf-auth-shell__progress-step.is-current .sf-auth-shell__step-marker { border-color: var(--sf-accent); background: var(--sf-accent); color: var(--sf-accent-contrast); box-shadow: 0 0 0 4px var(--sf-accent-soft); }
.sf-auth-shell__progress-step.is-complete .sf-auth-shell__step-marker { border-color: var(--sf-accent); background: var(--sf-accent-soft); color: var(--sf-accent); }
.sf-auth-shell__step-title { margin-top: 1px; color: var(--sf-fg-secondary); font-size: 14px; font-weight: 680; line-height: 1.4; }
.sf-auth-shell__step-description { margin-top: 4px; color: var(--sf-fg-tertiary); font-size: 12px; line-height: 1.5; }
.sf-auth-shell__recovery-note { display: inline-flex; align-items: center; gap: 7px; margin: 30px 0 0; color: var(--sf-fg-tertiary); font-size: 12px; }
.sf-auth-shell__recovery-note svg { width: 15px; height: 15px; }
.sf-auth-shell__panel { position: relative; display: flex; min-width: 0; align-items: center; justify-content: center; padding: 48px 40px; }
.sf-auth-shell__content { width: min(100%, 410px); }
.sf-auth-shell__color-mode { position: absolute; top: 28px; right: 32px; display: grid; width: 38px; height: 38px; place-items: center; border: 1px solid var(--sf-border); border-radius: 7px; background: var(--sf-card); color: var(--sf-fg-secondary); cursor: pointer; }
.sf-auth-shell__color-mode:hover { background: var(--sf-muted); color: var(--sf-fg); }
.sf-auth-shell__color-mode svg { width: 18px; height: 18px; }
.sf-auth-shell__color-mode-placeholder { position: absolute; top: 28px; right: 32px; width: 38px; height: 38px; }
.sf-auth-shell__mobile-brand { display: none; }
.sf-auth-shell__back-link { display: inline-flex; align-items: center; gap: 8px; margin-bottom: 34px; color: var(--sf-fg-secondary); font-size: 13px; font-weight: 620; text-decoration: none; }
.sf-auth-shell__back-link:hover { color: var(--sf-accent); }
.sf-auth-shell__back-link svg { width: 16px; height: 16px; }
@media (max-width: 720px) { .sf-auth-shell { display: block; } .sf-auth-shell__brand { display: none; } .sf-auth-shell__panel { min-height: 100svh; align-items: flex-start; padding: 94px 20px 48px; } .sf-auth-shell__mobile-brand { position: absolute; top: 24px; left: 20px; display: inline-flex; font-size: 15px; } .sf-auth-shell__mobile-brand .sf-auth-shell__logo { width: 29px; height: 29px; flex-basis: 29px; } .sf-auth-shell__color-mode, .sf-auth-shell__color-mode-placeholder { top: 20px; right: 20px; } .sf-auth-shell__content { max-width: 430px; margin: 0 auto; } .sf-auth-shell__back-link { margin-bottom: 28px; } }
</style>
