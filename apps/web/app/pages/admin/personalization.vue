<script setup lang="ts">
import {
  appearanceThemes,
  buildCustomAppearanceThemeValue,
  cloneFooterLink,
  cloneFooterLinks,
  customColorFromAppearanceTheme,
  defaultCustomThemeColor,
  normalizeAppearanceThemeValue,
  normalizeHexColor,
  recommendedAppearanceTheme,
  recommendedFooterCopyright,
  recommendedFooterLinks,
  resolveAppearanceTheme,
  type AdminWebOption,
  type AppearanceTheme,
  type AppearanceThemePreset,
  type FooterLinkKey,
  type FooterLinkOption,
  type WebOption
} from '~/composables/useWebOptions'
import { useAdminPage } from '~/composables/useAdminPage'
import { apiErrorMessage } from '~/composables/useApiClient'
import SFAdminSiteChromePanel from '~/components/admin/SFAdminSiteChromePanel.vue'

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

// 个性化页合并外观与前台壳；tab 与权限一一对应。
type SiteChromeSection = 'brand' | 'nav' | 'announcements' | 'legal' | 'friendLinks'
type PersonalizationTab = 'appearance' | SiteChromeSection

const { t } = useI18n()
const toast = useToast()
const route = useRoute()
const router = useRouter()
const { options, fetchAdminEnvelope, saveMany } = useWebOptions()
const adminPage = useAdminPage('/personalization')
const { can } = usePermissions()
const chromePanel = ref<{ refresh: () => Promise<void>, loading: boolean } | null>(null)

const canManageAppearance = computed(() => can('settings.appearance.manage'))
const canManageSiteChrome = computed(() => can('settings.site.manage'))

const themePreviews: Record<AppearanceThemePreset, ThemePreview> = {
  pine_teal: { accent: '#0f766e', hover: '#0b5f59', soft: '#e6f4f1' },
  ocean_blue: { accent: '#2563eb', hover: '#1d4ed8', soft: '#eff6ff' },
  violet: { accent: '#7c3aed', hover: '#6d28d9', soft: '#f3e8ff' },
  rose: { accent: '#e11d48', hover: '#be123c', soft: '#fff1f2' },
  amber: { accent: '#d97706', hover: '#b45309', soft: '#fffbeb' }
}

const form = reactive({
  themeMode: recommendedAppearanceTheme as AppearanceThemePreset | 'custom',
  customColor: defaultCustomThemeColor,
  footerCopyrightZHCN: recommendedFooterCopyright['zh-CN'],
  footerCopyrightENUS: recommendedFooterCopyright['en-US'],
  footerLinks: cloneFooterLinks(recommendedFooterLinks)
})

const saving = ref(false)
const savedSnapshot = ref('')

const allTabs: Array<{
  id: PersonalizationTab
  labelKey: string
  icon: string
  requires: 'appearance' | 'site'
}> = [
  { id: 'appearance', labelKey: 'admin.personalization.tabs.appearance', icon: 'i-lucide-palette', requires: 'appearance' },
  { id: 'brand', labelKey: 'admin.personalization.tabs.brand', icon: 'i-lucide-image', requires: 'site' },
  { id: 'nav', labelKey: 'admin.personalization.tabs.nav', icon: 'i-lucide-menu', requires: 'site' },
  { id: 'announcements', labelKey: 'admin.personalization.tabs.announcements', icon: 'i-lucide-megaphone', requires: 'site' },
  { id: 'legal', labelKey: 'admin.personalization.tabs.legal', icon: 'i-lucide-scale', requires: 'site' },
  { id: 'friendLinks', labelKey: 'admin.personalization.tabs.friendLinks', icon: 'i-lucide-external-link', requires: 'site' }
]

const tabs = computed(() =>
  allTabs
    .filter((tab) => (tab.requires === 'appearance' ? canManageAppearance.value : canManageSiteChrome.value))
    .map((tab) => ({
      id: tab.id,
      label: t(tab.labelKey),
      icon: tab.icon
    }))
)

function normalizeTab(value: unknown): PersonalizationTab {
  const raw = Array.isArray(value) ? value[0] : value
  const candidate = typeof raw === 'string' ? raw : ''
  // 兼容旧 /site-chrome 深链与独立页习惯的 tab 名。
  const aliases: Record<string, PersonalizationTab> = {
    theme: 'appearance',
    footer: 'appearance',
    chrome: 'brand',
    'friend-links': 'friendLinks',
    friends: 'friendLinks'
  }
  const resolved = (aliases[candidate] || candidate) as PersonalizationTab
  const allowed = tabs.value.map((tab) => tab.id)
  if (allowed.includes(resolved)) {
    return resolved
  }
  return allowed[0] || 'appearance'
}

const activeTab = ref<PersonalizationTab>(normalizeTab(route.query.tab))

watch(tabs, (available) => {
  if (!available.some((tab) => tab.id === activeTab.value) && available[0]) {
    setActiveTab(available[0].id)
  }
}, { immediate: true })

watch(() => route.query.tab, (value) => {
  activeTab.value = normalizeTab(value)
})

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

const isChromeTab = computed(() => activeTab.value !== 'appearance')

// useAsyncData 在 SSR 水合时不会重跑 handler；表单副作用必须用 watch 同步，
// 否则主题/页脚等控件会卡在 reactive 初始默认值。
const {
  data: adminPersonalizationOptions,
  pending: appearancePending,
  error: appearanceError,
  refresh: refreshAppearance
} = await useAsyncData('admin-personalization-options', async () => {
  if (!canManageAppearance.value) {
    return null
  }
  const envelope = await fetchAdminEnvelope()
  return envelope.data
})

watch(adminPersonalizationOptions, (items) => {
  if (items) {
    applyAdminOptions(items)
  }
}, { immediate: true })

useSeoMeta({
  title: t('admin.personalization.metaTitle')
})

const toolbarPending = computed(() => {
  if (activeTab.value === 'appearance') {
    return appearancePending.value
  }
  return Boolean(chromePanel.value?.loading)
})

async function refreshActive() {
  if (activeTab.value === 'appearance') {
    await refreshAppearance()
    return
  }
  await chromePanel.value?.refresh()
}

function setActiveTab(tab: PersonalizationTab) {
  activeTab.value = tab
  const nextQuery = { ...route.query, tab }
  if (tab === tabs.value[0]?.id) {
    const { tab: _tab, ...rest } = nextQuery
    void router.replace({ query: rest })
    return
  }
  void router.replace({ query: nextQuery })
}

function applyAdminOptions(items: AdminWebOption[]) {
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
  form.themeMode = recommendedAppearanceTheme
  form.customColor = defaultCustomThemeColor
  form.footerCopyrightZHCN = recommendedFooterCopyright['zh-CN']
  form.footerCopyrightENUS = recommendedFooterCopyright['en-US']
  form.footerLinks = cloneFooterLinks(recommendedFooterLinks)
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
    return cloneFooterLinks(recommendedFooterLinks)
  }

  try {
    const parsed = JSON.parse(value)
    if (!Array.isArray(parsed)) {
      return cloneFooterLinks(recommendedFooterLinks)
    }
    const byKey = new Map<FooterLinkKey, FooterLinkOption>()
    for (const item of parsed) {
      const link = normalizeFooterLink(item)
      if (link) {
        byKey.set(link.key, link)
      }
    }
    if (byKey.size !== recommendedFooterLinks.length) {
      return cloneFooterLinks(recommendedFooterLinks)
    }
    return recommendedFooterLinks.map((link) => byKey.get(link.key) || cloneFooterLink(link))
  } catch {
    return cloneFooterLinks(recommendedFooterLinks)
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

function isFooterLinkKey(value: unknown): value is FooterLinkKey {
  return value === 'terms' || value === 'privacy' || value === 'guidelines'
}

function defaultFooterText(locale: 'zh-CN' | 'en-US') {
  return recommendedFooterCopyright[locale]
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
        :loading="toolbarPending"
        class="border-slate-200 dark:border-zinc-700"
        @click="refreshActive()"
      >
        {{ t('admin.personalization.refresh') }}
      </UButton>
    </template>
  </UDashboardToolbar>

  <UAlert
    v-if="canManageSiteChrome"
    color="primary"
    variant="soft"
    icon="i-lucide-sparkles"
    class="mb-4"
    :title="t('admin.siteChrome.recommendedTitle')"
    :description="t('admin.siteChrome.recommendedBody')"
  />

  <!-- 使用 md 尺寸 + 底部分割，避免原先 size=sm 的小按钮观感 -->
  <div
    role="tablist"
    :aria-label="t('admin.personalization.tabs.label')"
    class="mb-5 flex flex-wrap gap-2 border-b border-slate-200 pb-3 dark:border-zinc-800"
  >
    <UButton
      v-for="tab in tabs"
      :key="tab.id"
      size="md"
      class="min-h-10 px-4"
      :color="activeTab === tab.id ? 'primary' : 'neutral'"
      :variant="activeTab === tab.id ? 'solid' : 'ghost'"
      :leading-icon="tab.icon"
      role="tab"
      :aria-selected="activeTab === tab.id"
      @click="setActiveTab(tab.id)"
    >
      {{ tab.label }}
    </UButton>
  </div>

  <!-- 外观：配色 + 页脚 -->
  <form
    v-if="activeTab === 'appearance' && canManageAppearance"
    class="flex flex-col gap-5"
    @submit.prevent="savePersonalizationSettings"
  >
    <UAlert
      v-if="appearanceError"
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
              <UIcon name="i-lucide-corner-down-right" class="size-4 text-slate-400 dark:text-zinc-500" />
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
          :reset-text="t('admin.personalization.restoreRecommended')"
          @reset="resetForm"
        />
      </template>
    </UCard>
  </form>

  <!-- 前台壳：品牌 / 导航 / 公告 / 法律 / 友情链接 -->
  <SFAdminSiteChromePanel
    v-else-if="isChromeTab && canManageSiteChrome"
    ref="chromePanel"
    :section="activeTab as SiteChromeSection"
  />

  <UAlert
    v-else
    color="warning"
    variant="soft"
    icon="i-lucide-lock"
    :title="t('admin.personalization.noAccessTitle')"
    :description="t('admin.personalization.noAccessBody')"
  />
</template>
