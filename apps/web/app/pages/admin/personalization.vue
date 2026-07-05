<script setup lang="ts">
import {
  appearanceThemes,
  buildCustomAppearanceThemeValue,
  customColorFromAppearanceTheme,
  defaultCustomThemeColor,
  normalizeAppearanceThemeValue,
  normalizeHexColor,
  resolveAppearanceTheme,
  type AdminWebOption,
  type AppearanceTheme,
  type AppearanceThemePreset,
  type FooterLinkKey,
  type FooterLinkOption,
  type WebOption
} from '~/composables/useWebOptions'
import { useAdminPage } from '~/composables/useAdminPage'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminPersonalization'
})

type ThemePreview = {
  accent: string
  hover: string
  soft: string
}

const { t } = useI18n()
const toast = useToast()
const { options, fetchAdminEnvelope, saveMany } = useWebOptions()
const adminPage = useAdminPage('/personalization')

const themePreviews: Record<AppearanceThemePreset, ThemePreview> = {
  pine_teal: { accent: '#0f766e', hover: '#0b5f59', soft: '#e6f4f1' },
  ocean_blue: { accent: '#2563eb', hover: '#1d4ed8', soft: '#eff6ff' },
  violet: { accent: '#7c3aed', hover: '#6d28d9', soft: '#f3e8ff' },
  rose: { accent: '#e11d48', hover: '#be123c', soft: '#fff1f2' },
  amber: { accent: '#d97706', hover: '#b45309', soft: '#fffbeb' }
}

const defaultFooterLinks: FooterLinkOption[] = [
  {
    key: 'terms',
    labels: { 'zh-CN': '服务条款', 'en-US': 'Terms of Service' },
    url: '#'
  },
  {
    key: 'privacy',
    labels: { 'zh-CN': '隐私政策', 'en-US': 'Privacy Policy' },
    url: '#'
  },
  {
    key: 'guidelines',
    labels: { 'zh-CN': '社区指南', 'en-US': 'Guidelines' },
    url: '#'
  }
]

const form = reactive({
  themeMode: 'pine_teal' as AppearanceThemePreset | 'custom',
  customColor: defaultCustomThemeColor,
  footerCopyrightZHCN: '© {year} {siteName}。保留所有权利。',
  footerCopyrightENUS: '© {year} {siteName}. All rights reserved.',
  footerLinks: cloneFooterLinks(defaultFooterLinks)
})

const saving = ref(false)
const savedSnapshot = ref('')
const lastAdminItems = ref<AdminWebOption[]>([])

const themeChoices = computed(() => {
  return appearanceThemes.map((theme) => ({
    value: theme,
    label: t(`admin.personalization.themes.${theme}.label`),
    description: t(`admin.personalization.themes.${theme}.description`),
    preview: themePreviews[theme]
  }))
})

const selectedAppearanceTheme = computed<AppearanceTheme>(() => {
  return form.themeMode === 'custom'
    ? buildCustomAppearanceThemeValue(form.customColor)
    : form.themeMode
})

const customThemePreview = computed<ThemePreview>(() => {
  const vars = resolveAppearanceTheme(buildCustomAppearanceThemeValue(form.customColor)).cssVars
  return {
    accent: vars['--sf-accent'] || defaultCustomThemeColor,
    hover: vars['--sf-accent-hover'] || defaultCustomThemeColor,
    soft: vars['--sf-accent-soft'] || '#eff6ff'
  }
})

const hasChanges = computed(() => formSnapshot() !== savedSnapshot.value)

const { pending, error, refresh } = await useAsyncData('admin-personalization-options', async () => {
  const envelope = await fetchAdminEnvelope()
  applyAdminOptions(envelope.data)
  return envelope.data
})

useSeoMeta({
  title: t('admin.personalization.metaTitle')
})

function applyAdminOptions(items: AdminWebOption[]) {
  lastAdminItems.value = items
  const publicOptions = items.filter((item) => item.public && !item.secret)
  options.value = {
    ...options.value,
    ...Object.fromEntries(publicOptions.map((item) => [item.name, item.value]))
  }

  const map = Object.fromEntries(items.map((item) => [item.name, item]))
  applyThemeValue(map['appearance.theme']?.value)
  form.footerCopyrightZHCN = map['footer.copyright.zh-CN']?.value ?? defaultFooterText('zh-CN')
  form.footerCopyrightENUS = map['footer.copyright.en-US']?.value ?? defaultFooterText('en-US')
  form.footerLinks = parseFooterLinks(map['footer.links']?.value)
  savedSnapshot.value = formSnapshot()
}

async function savePersonalizationSettings() {
  saving.value = true
  try {
    normalizeCustomColorInput()
    await saveAndApply([
      { name: 'appearance.theme', value: selectedAppearanceTheme.value },
      { name: 'footer.copyright.zh-CN', value: form.footerCopyrightZHCN },
      { name: 'footer.copyright.en-US', value: form.footerCopyrightENUS },
      { name: 'footer.links', value: JSON.stringify(normalizedFooterLinks()) }
    ])
    toast.add({
      color: 'success',
      icon: 'i-lucide-check',
      title: t('admin.personalization.saved')
    })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: apiErrorMessage(error) || t('admin.personalization.saveFailed')
    })
  } finally {
    saving.value = false
  }
}

async function saveAndApply(items: WebOption[]) {
  const updated = await saveMany(items)
  applyAdminOptions(updated)
}

function resetForm() {
  applyAdminOptions(lastAdminItems.value)
}

function applyThemeValue(value: string | undefined) {
  const theme = normalizeAppearanceThemeValue(value)
  const customColor = customColorFromAppearanceTheme(theme)
  if (customColor) {
    form.themeMode = 'custom'
    form.customColor = customColor
    return
  }
  form.themeMode = theme as AppearanceThemePreset
}

function normalizeCustomColorInput() {
  form.customColor = normalizeHexColor(form.customColor) || defaultCustomThemeColor
}

function parseFooterLinks(value: string | undefined) {
  if (!value) {
    return cloneFooterLinks(defaultFooterLinks)
  }

  try {
    const parsed = JSON.parse(value)
    if (!Array.isArray(parsed)) {
      return cloneFooterLinks(defaultFooterLinks)
    }
    const byKey = new Map<FooterLinkKey, FooterLinkOption>()
    for (const item of parsed) {
      const link = normalizeFooterLink(item)
      if (link) {
        byKey.set(link.key, link)
      }
    }
    if (byKey.size !== defaultFooterLinks.length) {
      return cloneFooterLinks(defaultFooterLinks)
    }
    return defaultFooterLinks.map((link) => byKey.get(link.key) || cloneFooterLink(link))
  } catch {
    return cloneFooterLinks(defaultFooterLinks)
  }
}

function normalizeFooterLink(value: unknown): FooterLinkOption | null {
  if (!value || typeof value !== 'object') {
    return null
  }
  const item = value as Partial<FooterLinkOption>
  if (!isFooterLinkKey(item.key) || !item.labels || typeof item.labels !== 'object') {
    return null
  }
  const labels = item.labels as Partial<Record<'zh-CN' | 'en-US', unknown>>
  const zhCN = typeof labels['zh-CN'] === 'string' ? labels['zh-CN'].trim() : ''
  const enUS = typeof labels['en-US'] === 'string' ? labels['en-US'].trim() : ''
  const url = typeof item.url === 'string' ? item.url.trim() : ''
  if (!zhCN || !enUS) {
    return null
  }
  return {
    key: item.key,
    labels: {
      'zh-CN': zhCN,
      'en-US': enUS
    },
    url
  }
}

function normalizedFooterLinks() {
  return form.footerLinks.map((link) => ({
    key: link.key,
    labels: {
      'zh-CN': link.labels['zh-CN'].trim(),
      'en-US': link.labels['en-US'].trim()
    },
    url: link.url.trim()
  }))
}

function cloneFooterLinks(links: FooterLinkOption[]) {
  return links.map(cloneFooterLink)
}

function cloneFooterLink(link: FooterLinkOption): FooterLinkOption {
  return {
    key: link.key,
    labels: { ...link.labels },
    url: link.url
  }
}

function isFooterLinkKey(value: unknown): value is FooterLinkKey {
  return value === 'terms' || value === 'privacy' || value === 'guidelines'
}

function defaultFooterText(locale: 'zh-CN' | 'en-US') {
  return locale === 'en-US'
    ? '© {year} {siteName}. All rights reserved.'
    : '© {year} {siteName}。保留所有权利。'
}

function formSnapshot() {
  return JSON.stringify({
    theme: selectedAppearanceTheme.value,
    footerCopyrightZHCN: form.footerCopyrightZHCN,
    footerCopyrightENUS: form.footerCopyrightENUS,
    footerLinks: normalizedFooterLinks()
  })
}
</script>

<template>
  <div class="mb-4 flex flex-col gap-1">
    <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ t('admin.personalization.title') }}
    </h2>
    <p class="text-sm text-slate-500 dark:text-zinc-400">
      {{ t('admin.personalization.intro') }}
    </p>
  </div>

  <UDashboardToolbar class="mb-6 rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-500 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm">
        <UIcon name="i-lucide-swatch-book" class="size-4" />
        <span class="truncate">{{ t('admin.personalization.toolbar') }}</span>
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
        {{ t('admin.personalization.refresh') }}
      </UButton>
    </template>
  </UDashboardToolbar>

  <form class="flex flex-col gap-5" @submit.prevent="savePersonalizationSettings">
    <UAlert
      v-if="error"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="t('admin.personalization.loadFailed')"
    />

    <UCard
      class="border-slate-200 bg-white text-slate-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100"
    >
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div>
            <h2 class="text-base font-bold text-slate-900 dark:text-white">
              {{ t('admin.personalization.theme.title') }}
            </h2>
            <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.personalization.theme.description') }}
            </p>
          </div>
          <UBadge color="neutral" variant="soft" class="border border-slate-200 font-mono dark:border-zinc-800">
            {{ t('admin.personalization.appearancePreset') }}
          </UBadge>
        </div>
      </template>

      <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-6">
        <label
          v-for="choice in themeChoices"
          :key="choice.value"
          class="group flex min-h-36 cursor-pointer flex-col justify-between gap-4 rounded-lg border bg-white p-4 text-sm transition dark:bg-zinc-950"
          :class="form.themeMode === choice.value
            ? 'border-[var(--sf-accent)] shadow-sm ring-2 ring-[var(--sf-accent-focus)]'
            : 'border-slate-200 hover:border-[var(--sf-accent-soft-border)] dark:border-zinc-800'"
        >
          <input
            v-model="form.themeMode"
            class="sr-only"
            type="radio"
            name="appearance-theme"
            :value="choice.value"
          >
          <span class="flex items-center justify-between gap-3">
            <span class="font-bold text-slate-900 dark:text-zinc-100">{{ choice.label }}</span>
            <span class="flex items-center gap-1">
              <span class="size-4 rounded-full border border-black/5" :style="{ background: choice.preview.soft }"></span>
              <span class="size-4 rounded-full border border-black/5" :style="{ background: choice.preview.accent }"></span>
              <span class="size-4 rounded-full border border-black/5" :style="{ background: choice.preview.hover }"></span>
            </span>
          </span>
          <span class="text-xs leading-5 text-slate-500 dark:text-zinc-400">
            {{ choice.description }}
          </span>
        </label>

        <div
          class="group flex min-h-36 cursor-pointer flex-col justify-between gap-4 rounded-lg border bg-white p-4 text-sm transition dark:bg-zinc-950"
          :class="form.themeMode === 'custom'
            ? 'border-[var(--sf-accent)] shadow-sm ring-2 ring-[var(--sf-accent-focus)]'
            : 'border-slate-200 hover:border-[var(--sf-accent-soft-border)] dark:border-zinc-800'"
          role="radio"
          tabindex="0"
          :aria-checked="form.themeMode === 'custom'"
          @click="form.themeMode = 'custom'"
          @keydown.enter.prevent="form.themeMode = 'custom'"
          @keydown.space.prevent="form.themeMode = 'custom'"
        >
          <input
            v-model="form.themeMode"
            class="sr-only"
            type="radio"
            name="appearance-theme"
            value="custom"
          >
          <span class="flex items-center justify-between gap-3">
            <span class="font-bold text-slate-900 dark:text-zinc-100">
              {{ t('admin.personalization.themes.custom.label') }}
            </span>
            <span class="flex items-center gap-1">
              <span class="size-4 rounded-full border border-black/5" :style="{ background: customThemePreview.soft }"></span>
              <span class="size-4 rounded-full border border-black/5" :style="{ background: customThemePreview.accent }"></span>
              <span class="size-4 rounded-full border border-black/5" :style="{ background: customThemePreview.hover }"></span>
            </span>
          </span>
          <span class="text-xs leading-5 text-slate-500 dark:text-zinc-400">
            {{ t('admin.personalization.themes.custom.description') }}
          </span>
          <div class="flex items-center gap-2" @click.stop>
            <input
              v-model="form.customColor"
              type="color"
              class="h-9 w-12 shrink-0 cursor-pointer rounded-md border border-slate-200 bg-white p-1 dark:border-zinc-700 dark:bg-zinc-900"
              :aria-label="t('admin.personalization.themes.custom.colorLabel')"
              @focus="form.themeMode = 'custom'"
              @input="form.themeMode = 'custom'"
            >
            <UInput
              v-model="form.customColor"
              size="sm"
              maxlength="7"
              class="min-w-0 flex-1 font-mono"
              :placeholder="defaultCustomThemeColor"
              @focus="form.themeMode = 'custom'"
              @blur="normalizeCustomColorInput"
            />
          </div>
        </div>
      </div>
    </UCard>

    <UCard
      class="border-slate-200 bg-white text-slate-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100"
      :ui="{ footer: 'sticky bottom-0 z-20 bg-white/95 dark:bg-zinc-900/95 backdrop-blur-sm border-t border-slate-200 dark:border-zinc-800 p-4 sm:px-6' }"
    >
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div>
            <h2 class="text-base font-bold text-slate-900 dark:text-white">
              {{ t('admin.personalization.footer.title') }}
            </h2>
            <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.personalization.footer.description') }}
            </p>
          </div>
          <UBadge color="neutral" variant="soft" class="border border-slate-200 font-mono dark:border-zinc-800">
            footer.*
          </UBadge>
        </div>
      </template>

      <div class="grid max-w-5xl gap-5">
        <div class="grid gap-4 lg:grid-cols-2">
          <UFormField :label="t('admin.personalization.footer.copyrightZh')" name="footer-copyright-zh">
            <UTextarea
              v-model="form.footerCopyrightZHCN"
              :rows="3"
              :maxlength="200"
              class="w-full"
            />
          </UFormField>
          <UFormField :label="t('admin.personalization.footer.copyrightEn')" name="footer-copyright-en">
            <UTextarea
              v-model="form.footerCopyrightENUS"
              :rows="3"
              :maxlength="200"
              class="w-full"
            />
          </UFormField>
        </div>

        <div class="grid gap-3">
          <div class="flex items-center gap-2 text-sm font-bold text-slate-900 dark:text-zinc-100">
            <UIcon name="i-lucide-link" class="size-4 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
            {{ t('admin.personalization.footer.links') }}
          </div>

          <div
            v-for="link in form.footerLinks"
            :key="link.key"
            class="grid gap-3 rounded-lg border border-slate-200 bg-slate-50 p-3 dark:border-zinc-800 dark:bg-zinc-950/60 lg:grid-cols-[10rem_1fr_1fr_1.2fr]"
          >
            <div class="flex items-center gap-2 text-sm font-semibold text-slate-700 dark:text-zinc-300">
              <UIcon name="i-lucide-corner-down-right" class="size-4 text-slate-400" />
              {{ t(`admin.personalization.footer.linkKeys.${link.key}`) }}
            </div>
            <UInput
              v-model="link.labels['zh-CN']"
              :placeholder="t('admin.personalization.footer.labelZh')"
              maxlength="40"
            />
            <UInput
              v-model="link.labels['en-US']"
              :placeholder="t('admin.personalization.footer.labelEn')"
              maxlength="40"
            />
            <UInput
              v-model="link.url"
              icon="i-lucide-link"
              :placeholder="t('admin.personalization.footer.urlPlaceholder')"
            />
          </div>
        </div>
      </div>

      <template #footer>
        <SFAdminFormFooter
          :saving="saving"
          :show-unsaved-alert="hasChanges"
          :submit-text="t('admin.personalization.save')"
          @reset="resetForm"
        />
      </template>
    </UCard>
  </form>
</template>
