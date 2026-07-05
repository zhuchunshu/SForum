<script setup lang="ts">
import {
  enabledOptionValue,
  isLocalSiteUrl,
  normalizeEnabledOption,
  normalizeSEOTwitterCard,
  parseSEORobotsPathList,
  type AdminWebOption,
  type SEOTwitterCard,
  type WebOption
} from '~/composables/useWebOptions'
import { useAdminPage } from '~/composables/useAdminPage'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminSeo'
})

type SeoTab = 'overview' | 'meta' | 'robots' | 'sitemap' | 'schema' | 'verification'

const { t } = useI18n()
const toast = useToast()
const { options, siteUrl, fetchAdminEnvelope, saveMany } = useWebOptions()
const adminPage = useAdminPage('/seo')

const activeTab = ref<SeoTab>('overview')
const saving = ref(false)
const lastAdminItems = ref<AdminWebOption[]>([])
const savedSnapshot = ref('')

const form = reactive({
  metaTitleTemplate: '',
  metaDescription: '',
  metaKeywords: '',
  ogImageUrl: '',
  twitterCard: 'summary_large_image' as SEOTwitterCard,
  twitterSite: '',
  allowIndexing: true,
  googleVerification: '',
  bingVerification: '',
  baiduVerification: '',
  yandexVerification: '',
  robotsExtraAllow: '',
  robotsExtraDisallow: '',
  blockAiBots: false,
  blockNonSeoBots: false,
  sitemapEnabled: true,
  sitemapIncludeStaticPages: true,
  sitemapIncludeForumContent: false,
  schemaOrgEnabled: true,
  schemaOrgSearchActionEnabled: true,
  schemaOrgDiscussionEnabled: true,
  schemaOrgOrganizationLogoUrl: ''
})

const tabs = computed<Array<{ id: SeoTab, label: string, icon: string }>>(() => [
  { id: 'overview', label: t('admin.seo.tabs.overview'), icon: 'i-lucide-gauge' },
  { id: 'meta', label: t('admin.seo.tabs.meta'), icon: 'i-lucide-file-text' },
  { id: 'robots', label: t('admin.seo.tabs.robots'), icon: 'i-lucide-bot' },
  { id: 'sitemap', label: t('admin.seo.tabs.sitemap'), icon: 'i-lucide-map' },
  { id: 'schema', label: t('admin.seo.tabs.schema'), icon: 'i-lucide-braces' },
  { id: 'verification', label: t('admin.seo.tabs.verification'), icon: 'i-lucide-badge-check' }
])

const twitterCardChoices = computed(() => [
  { label: t('admin.seo.twitterCardLarge'), value: 'summary_large_image' },
  { label: t('admin.seo.twitterCardSummary'), value: 'summary' }
])

const { data: adminSeoItems, pending, error, refresh } = await useAsyncData('admin-seo-options', async () => {
  const envelope = await fetchAdminEnvelope()
  return envelope.data
})

watch(adminSeoItems, (items) => {
  if (items) {
    applyAdminOptions(items)
  }
}, { immediate: true })

useSeoMeta({
  title: t('admin.seo.metaTitle')
})

const hasChanges = computed(() => formSnapshot() !== savedSnapshot.value)
const localIndexingProtected = computed(() => isLocalSiteUrl(siteUrl.value))
const effectiveIndexing = computed(() => form.allowIndexing && !localIndexingProtected.value)
const canonicalPreview = computed(() => absoluteUrl('/'))
const sitemapPreview = computed(() => absoluteUrl('/sitemap.xml'))
const robotsPreview = computed(() => {
  if (!effectiveIndexing.value) {
    return ['User-agent: *', 'Disallow: /'].join('\n')
  }

  const lines = [
    'User-agent: *',
    'Disallow: /api/',
    'Disallow: /login',
    'Disallow: /register',
    'Disallow: /control-panel/'
  ]
  for (const path of parseSEORobotsPathList(form.robotsExtraAllow)) {
    lines.push(`Allow: ${path}`)
  }
  for (const path of parseSEORobotsPathList(form.robotsExtraDisallow)) {
    lines.push(`Disallow: ${path}`)
  }
  lines.push(`Sitemap: ${sitemapPreview.value}`)
  return lines.join('\n')
})

const overviewItems = computed(() => [
  {
    label: t('admin.seo.overview.indexing'),
    value: effectiveIndexing.value ? t('admin.seo.statusEnabled') : t('admin.seo.statusDisabled'),
    icon: effectiveIndexing.value ? 'i-lucide-check-circle-2' : 'i-lucide-circle-slash'
  },
  {
    label: t('admin.seo.overview.canonical'),
    value: canonicalPreview.value,
    icon: 'i-lucide-link'
  },
  {
    label: t('admin.seo.overview.sitemap'),
    value: form.sitemapEnabled ? sitemapPreview.value : t('admin.seo.statusDisabled'),
    icon: 'i-lucide-map'
  },
  {
    label: t('admin.seo.overview.schema'),
    value: form.schemaOrgEnabled ? t('admin.seo.statusEnabled') : t('admin.seo.statusDisabled'),
    icon: 'i-lucide-braces'
  }
])

function applyAdminOptions(items: AdminWebOption[]) {
  lastAdminItems.value = items
  const publicOptions = items.filter((item) => item.public && !item.secret)
  options.value = {
    ...options.value,
    ...Object.fromEntries(publicOptions.map((item) => [item.name, item.value]))
  }

  const map = Object.fromEntries(items.map((item) => [item.name, item]))
  form.metaTitleTemplate = read(map, 'seo.meta_title_template')
  form.metaDescription = read(map, 'seo.meta_description')
  form.metaKeywords = read(map, 'seo.meta_keywords')
  form.ogImageUrl = read(map, 'seo.og_image_url')
  form.twitterCard = normalizeSEOTwitterCard(read(map, 'seo.twitter_card', 'summary_large_image'))
  form.twitterSite = read(map, 'seo.twitter_site')
  form.allowIndexing = enabled(map, 'seo.allow_indexing', true)
  form.googleVerification = read(map, 'seo.google_verification')
  form.bingVerification = read(map, 'seo.bing_verification')
  form.baiduVerification = read(map, 'seo.baidu_verification')
  form.yandexVerification = read(map, 'seo.yandex_verification')
  form.robotsExtraAllow = read(map, 'seo.robots.extra_allow')
  form.robotsExtraDisallow = read(map, 'seo.robots.extra_disallow')
  form.blockAiBots = enabled(map, 'seo.robots.block_ai_bots')
  form.blockNonSeoBots = enabled(map, 'seo.robots.block_non_seo_bots')
  form.sitemapEnabled = enabled(map, 'seo.sitemap.enabled', true)
  form.sitemapIncludeStaticPages = enabled(map, 'seo.sitemap.include_static_pages', true)
  form.sitemapIncludeForumContent = enabled(map, 'seo.sitemap.include_forum_content')
  form.schemaOrgEnabled = enabled(map, 'seo.schema_org.enabled', true)
  form.schemaOrgSearchActionEnabled = enabled(map, 'seo.schema_org.search_action_enabled', true)
  form.schemaOrgDiscussionEnabled = enabled(map, 'seo.schema_org.discussion_enabled', true)
  form.schemaOrgOrganizationLogoUrl = read(map, 'seo.schema_org.organization_logo_url')
  savedSnapshot.value = formSnapshot()
}

async function saveSeoSettings() {
  saving.value = true
  try {
    await saveAndApply(payload())
    toast.add({
      color: 'success',
      icon: 'i-lucide-check',
      title: t('admin.seo.saved')
    })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: apiErrorMessage(error) || t('admin.seo.saveFailed')
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

function setActiveTab(tab: SeoTab) {
  activeTab.value = tab
}

function payload(): WebOption[] {
  return [
    { name: 'seo.meta_title_template', value: form.metaTitleTemplate },
    { name: 'seo.meta_description', value: form.metaDescription },
    { name: 'seo.meta_keywords', value: form.metaKeywords },
    { name: 'seo.og_image_url', value: form.ogImageUrl },
    { name: 'seo.twitter_card', value: form.twitterCard },
    { name: 'seo.twitter_site', value: form.twitterSite },
    { name: 'seo.allow_indexing', value: enabledOptionValue(form.allowIndexing) },
    { name: 'seo.google_verification', value: form.googleVerification },
    { name: 'seo.bing_verification', value: form.bingVerification },
    { name: 'seo.baidu_verification', value: form.baiduVerification },
    { name: 'seo.yandex_verification', value: form.yandexVerification },
    { name: 'seo.robots.extra_allow', value: form.robotsExtraAllow },
    { name: 'seo.robots.extra_disallow', value: form.robotsExtraDisallow },
    { name: 'seo.robots.block_ai_bots', value: enabledOptionValue(form.blockAiBots) },
    { name: 'seo.robots.block_non_seo_bots', value: enabledOptionValue(form.blockNonSeoBots) },
    { name: 'seo.sitemap.enabled', value: enabledOptionValue(form.sitemapEnabled) },
    { name: 'seo.sitemap.include_static_pages', value: enabledOptionValue(form.sitemapIncludeStaticPages) },
    { name: 'seo.sitemap.include_forum_content', value: enabledOptionValue(form.sitemapIncludeForumContent) },
    { name: 'seo.schema_org.enabled', value: enabledOptionValue(form.schemaOrgEnabled) },
    { name: 'seo.schema_org.search_action_enabled', value: enabledOptionValue(form.schemaOrgSearchActionEnabled) },
    { name: 'seo.schema_org.discussion_enabled', value: enabledOptionValue(form.schemaOrgDiscussionEnabled) },
    { name: 'seo.schema_org.organization_logo_url', value: form.schemaOrgOrganizationLogoUrl }
  ]
}

function formSnapshot() {
  return JSON.stringify(payload())
}

function read(map: Record<string, AdminWebOption>, name: string, fallback = '') {
  return map[name]?.value ?? options.value[name] ?? fallback
}

function enabled(map: Record<string, AdminWebOption>, name: string, fallback = false) {
  return normalizeEnabledOption(read(map, name), fallback)
}

function absoluteUrl(path: string) {
  const base = siteUrl.value.replace(/\/+$/, '') || 'http://127.0.0.1:3000'
  return `${base}${path}`
}
</script>

<template>
  <div class="mb-4 flex flex-col gap-1">
    <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ t('admin.seo.title') }}
    </h2>
    <p class="text-sm text-slate-500 dark:text-zinc-400">
      {{ t('admin.seo.intro') }}
    </p>
  </div>

  <UDashboardToolbar class="mb-6 rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-500 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm">
        <UIcon name="i-lucide-search-check" class="size-4" />
        <span class="truncate">{{ t('admin.seo.toolbar') }}</span>
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
        {{ t('admin.seo.refresh') }}
      </UButton>
    </template>
  </UDashboardToolbar>

  <div class="flex flex-col gap-4">
    <UAlert
      v-if="error"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="t('admin.seo.loadFailed')"
    />

    <div
      role="tablist"
      :aria-label="t('admin.seo.tabs.label')"
      class="flex flex-wrap gap-2 border-b border-slate-200 pb-3 dark:border-zinc-800"
    >
      <UButton
        v-for="tab in tabs"
        :key="tab.id"
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

    <form class="flex flex-col" @submit.prevent="saveSeoSettings">
      <UCard
        class="border-slate-200 bg-white text-slate-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100"
        :ui="{ footer: 'sticky bottom-0 z-20 bg-white/95 dark:bg-zinc-900/95 backdrop-blur-sm border-t border-slate-200 dark:border-zinc-800 p-4 sm:px-6' }"
      >
        <template #header>
          <div class="flex items-center justify-between gap-3">
            <div>
              <h2 class="text-base font-bold text-slate-900 dark:text-white">
                {{ t(`admin.seo.sections.${activeTab}.title`) }}
              </h2>
              <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
                {{ t(`admin.seo.sections.${activeTab}.description`) }}
              </p>
            </div>
            <UBadge color="neutral" variant="soft" class="border border-slate-200 font-mono dark:border-zinc-800">
              seo.*
            </UBadge>
          </div>
        </template>

        <div v-if="activeTab === 'overview'" class="grid gap-5">
          <UAlert
            v-if="localIndexingProtected"
            color="warning"
            variant="soft"
            icon="i-lucide-shield-alert"
            :title="t('admin.seo.localProtectionTitle')"
            :description="t('admin.seo.localProtectionDescription')"
          />
          <div class="grid gap-3 md:grid-cols-2">
            <div
              v-for="item in overviewItems"
              :key="item.label"
              class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-zinc-800 dark:bg-zinc-950/60"
            >
              <div class="flex items-center gap-2 text-xs font-semibold uppercase text-slate-500 dark:text-zinc-400">
                <UIcon :name="item.icon" class="size-4" />
                <span>{{ item.label }}</span>
              </div>
              <p class="mt-2 break-all text-sm font-medium text-slate-900 dark:text-zinc-100">
                {{ item.value }}
              </p>
            </div>
          </div>
          <div class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_minmax(280px,420px)]">
            <div class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-zinc-800 dark:bg-zinc-950/60">
              <h3 class="text-sm font-bold text-slate-900 dark:text-zinc-100">
                {{ t('admin.seo.robotsPreview') }}
              </h3>
              <pre class="mt-3 max-h-72 overflow-auto whitespace-pre-wrap rounded-md bg-slate-950 p-4 text-xs leading-6 text-slate-100">{{ robotsPreview }}</pre>
            </div>
            <div class="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-zinc-800 dark:bg-zinc-950/60">
              <h3 class="text-sm font-bold text-slate-900 dark:text-zinc-100">
                {{ t('admin.seo.checklist') }}
              </h3>
              <div class="mt-3 space-y-3 text-sm">
                <div class="flex items-center gap-2">
                  <UIcon :name="form.metaDescription ? 'i-lucide-check-circle-2' : 'i-lucide-circle-alert'" class="size-4 text-[var(--sf-accent)]" />
                  <span>{{ t('admin.seo.checkMetaDescription') }}</span>
                </div>
                <div class="flex items-center gap-2">
                  <UIcon :name="form.ogImageUrl ? 'i-lucide-check-circle-2' : 'i-lucide-circle-alert'" class="size-4 text-[var(--sf-accent)]" />
                  <span>{{ t('admin.seo.checkOgImage') }}</span>
                </div>
                <div class="flex items-center gap-2">
                  <UIcon :name="form.sitemapEnabled ? 'i-lucide-check-circle-2' : 'i-lucide-circle-alert'" class="size-4 text-[var(--sf-accent)]" />
                  <span>{{ t('admin.seo.checkSitemap') }}</span>
                </div>
                <div class="flex items-center gap-2">
                  <UIcon :name="form.schemaOrgEnabled ? 'i-lucide-check-circle-2' : 'i-lucide-circle-alert'" class="size-4 text-[var(--sf-accent)]" />
                  <span>{{ t('admin.seo.checkSchema') }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-else-if="activeTab === 'meta'" class="grid max-w-3xl gap-4">
          <UFormField :label="t('admin.seo.metaTitleTemplate')" name="seo-meta-title-template">
            <UInput v-model="form.metaTitleTemplate" icon="i-lucide-file-text" :placeholder="t('admin.seo.metaTitleTemplatePlaceholder')" maxlength="120" class="w-full" />
          </UFormField>
          <UFormField :label="t('admin.seo.metaDescription')" name="seo-meta-description">
            <UTextarea v-model="form.metaDescription" :placeholder="t('admin.seo.metaDescriptionPlaceholder')" :rows="3" maxlength="320" class="w-full" />
          </UFormField>
          <UFormField :label="t('admin.seo.metaKeywords')" name="seo-meta-keywords">
            <UInput v-model="form.metaKeywords" icon="i-lucide-tags" :placeholder="t('admin.seo.metaKeywordsPlaceholder')" maxlength="200" class="w-full" />
          </UFormField>
          <UFormField :label="t('admin.seo.ogImageUrl')" name="seo-og-image-url">
            <UInput v-model="form.ogImageUrl" icon="i-lucide-image" type="url" :placeholder="t('admin.seo.ogImageUrlPlaceholder')" maxlength="500" class="w-full" />
          </UFormField>
          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="t('admin.seo.twitterCard')" name="seo-twitter-card">
              <select v-model="form.twitterCard" class="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-900 outline-none transition focus:border-[var(--sf-accent)] focus:ring-2 focus:ring-[var(--sf-accent-focus)] dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100">
                <option v-for="choice in twitterCardChoices" :key="choice.value" :value="choice.value">{{ choice.label }}</option>
              </select>
            </UFormField>
            <UFormField :label="t('admin.seo.twitterSite')" name="seo-twitter-site">
              <UInput v-model="form.twitterSite" icon="i-lucide-at-sign" placeholder="@sforum" maxlength="80" class="w-full" />
            </UFormField>
          </div>
        </div>

        <div v-else-if="activeTab === 'robots'" class="grid max-w-3xl gap-4">
          <label class="flex items-start gap-3 rounded-lg border border-slate-200 p-4 dark:border-zinc-800">
            <input v-model="form.allowIndexing" type="checkbox" class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]" />
            <span>
              <span class="block text-sm font-semibold">{{ t('admin.seo.allowIndexing') }}</span>
              <span class="mt-1 block text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.seo.allowIndexingDescription') }}</span>
            </span>
          </label>
          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="t('admin.seo.extraAllow')" name="seo-robots-extra-allow">
              <UTextarea v-model="form.robotsExtraAllow" :rows="6" placeholder="/public-preview" class="w-full font-mono text-xs" />
            </UFormField>
            <UFormField :label="t('admin.seo.extraDisallow')" name="seo-robots-extra-disallow">
              <UTextarea v-model="form.robotsExtraDisallow" :rows="6" placeholder="/drafts" class="w-full font-mono text-xs" />
            </UFormField>
          </div>
          <div class="grid gap-3 md:grid-cols-2">
            <label class="flex items-start gap-3 rounded-lg border border-slate-200 p-4 dark:border-zinc-800">
              <input v-model="form.blockAiBots" type="checkbox" class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]" />
              <span class="text-sm font-medium">{{ t('admin.seo.blockAiBots') }}</span>
            </label>
            <label class="flex items-start gap-3 rounded-lg border border-slate-200 p-4 dark:border-zinc-800">
              <input v-model="form.blockNonSeoBots" type="checkbox" class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]" />
              <span class="text-sm font-medium">{{ t('admin.seo.blockNonSeoBots') }}</span>
            </label>
          </div>
        </div>

        <div v-else-if="activeTab === 'sitemap'" class="grid max-w-3xl gap-4">
          <label class="flex items-start gap-3 rounded-lg border border-slate-200 p-4 dark:border-zinc-800">
            <input v-model="form.sitemapEnabled" type="checkbox" class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]" />
            <span>
              <span class="block text-sm font-semibold">{{ t('admin.seo.sitemapEnabled') }}</span>
              <span class="mt-1 block text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.seo.sitemapEnabledDescription') }}</span>
            </span>
          </label>
          <label class="flex items-start gap-3 rounded-lg border border-slate-200 p-4 dark:border-zinc-800">
            <input v-model="form.sitemapIncludeStaticPages" type="checkbox" class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]" />
            <span>
              <span class="block text-sm font-semibold">{{ t('admin.seo.sitemapStaticPages') }}</span>
              <span class="mt-1 block text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.seo.sitemapStaticPagesDescription') }}</span>
            </span>
          </label>
          <label class="flex items-start gap-3 rounded-lg border border-slate-200 p-4 dark:border-zinc-800">
            <input v-model="form.sitemapIncludeForumContent" type="checkbox" class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]" />
            <span>
              <span class="block text-sm font-semibold">{{ t('admin.seo.sitemapForumContent') }}</span>
              <span class="mt-1 block text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.seo.sitemapForumContentDescription') }}</span>
            </span>
          </label>
          <div class="rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm dark:border-zinc-800 dark:bg-zinc-950/60">
            <span class="font-semibold">{{ t('admin.seo.sitemapUrl') }}</span>
            <span class="mt-1 block break-all font-mono text-xs text-slate-600 dark:text-zinc-300">{{ sitemapPreview }}</span>
          </div>
        </div>

        <div v-else-if="activeTab === 'schema'" class="grid max-w-3xl gap-4">
          <label class="flex items-start gap-3 rounded-lg border border-slate-200 p-4 dark:border-zinc-800">
            <input v-model="form.schemaOrgEnabled" type="checkbox" class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]" />
            <span>
              <span class="block text-sm font-semibold">{{ t('admin.seo.schemaEnabled') }}</span>
              <span class="mt-1 block text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.seo.schemaEnabledDescription') }}</span>
            </span>
          </label>
          <label class="flex items-start gap-3 rounded-lg border border-slate-200 p-4 dark:border-zinc-800">
            <input v-model="form.schemaOrgSearchActionEnabled" type="checkbox" class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]" />
            <span>
              <span class="block text-sm font-semibold">{{ t('admin.seo.schemaSearchAction') }}</span>
              <span class="mt-1 block text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.seo.schemaSearchActionDescription') }}</span>
            </span>
          </label>
          <label class="flex items-start gap-3 rounded-lg border border-slate-200 p-4 dark:border-zinc-800">
            <input v-model="form.schemaOrgDiscussionEnabled" type="checkbox" class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]" />
            <span>
              <span class="block text-sm font-semibold">{{ t('admin.seo.schemaDiscussion') }}</span>
              <span class="mt-1 block text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.seo.schemaDiscussionDescription') }}</span>
            </span>
          </label>
          <UFormField :label="t('admin.seo.organizationLogoUrl')" name="seo-schema-logo-url">
            <UInput v-model="form.schemaOrgOrganizationLogoUrl" icon="i-lucide-image-up" type="url" placeholder="https://example.com/logo.png" maxlength="500" class="w-full" />
          </UFormField>
        </div>

        <div v-else class="grid max-w-3xl gap-4">
          <UFormField :label="t('admin.seo.googleVerification')" name="seo-google-verification">
            <UInput v-model="form.googleVerification" icon="i-lucide-badge-check" :placeholder="t('admin.seo.googleVerificationPlaceholder')" maxlength="120" class="w-full" />
          </UFormField>
          <UFormField :label="t('admin.seo.bingVerification')" name="seo-bing-verification">
            <UInput v-model="form.bingVerification" icon="i-lucide-badge-check" placeholder="msvalidate.01" maxlength="120" class="w-full" />
          </UFormField>
          <UFormField :label="t('admin.seo.baiduVerification')" name="seo-baidu-verification">
            <UInput v-model="form.baiduVerification" icon="i-lucide-badge-check" placeholder="baidu-site-verification" maxlength="120" class="w-full" />
          </UFormField>
          <UFormField :label="t('admin.seo.yandexVerification')" name="seo-yandex-verification">
            <UInput v-model="form.yandexVerification" icon="i-lucide-badge-check" placeholder="yandex-verification" maxlength="120" class="w-full" />
          </UFormField>
        </div>

        <template #footer>
          <SFAdminFormFooter
            :saving="saving"
            :show-unsaved-alert="hasChanges"
            :submit-text="t('admin.seo.save')"
            @reset="resetForm"
          />
        </template>
      </UCard>
    </form>
  </div>
</template>
