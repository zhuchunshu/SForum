<script setup lang="ts">
import type { AdminWebOption, WebOption } from '~/composables/useWebOptions'
import { useAdminPage } from '~/composables/useAdminPage'
import { apiErrorMessage } from '~/composables/useApiClient'
import {
  useSiteChromeApi,
  type SiteAnnouncement,
  type SiteAnnouncementStyle,
  type SiteFriendLink,
  type SiteNavItem
} from '~/composables/useSiteChromeApi'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminSiteChrome'
})

type ChromeTab = 'brand' | 'nav' | 'announcements' | 'legal' | 'friendLinks'

const { t, locale } = useI18n()
const toast = useToast()
const { options, fetchAdminEnvelope, saveMany } = useWebOptions()
const chromeApi = useSiteChromeApi()
const adminPage = useAdminPage('/site-chrome')

const activeTab = ref<ChromeTab>('brand')
const savingBrand = ref(false)
const savingLegal = ref(false)
const pendingCatalog = ref(false)

const brandForm = reactive({
  logoUrl: '',
  logoAttachmentId: '',
  faviconUrl: '',
  faviconAttachmentId: '',
  appleTouchIconUrl: '',
  appleTouchIconAttachmentId: ''
})
const brandSnapshot = ref('')

const legalForm = reactive({
  termsZh: '',
  termsEn: '',
  privacyZh: '',
  privacyEn: '',
  guidelinesZh: '',
  guidelinesEn: ''
})
const legalSnapshot = ref('')

const navItems = ref<SiteNavItem[]>([])
const friendLinks = ref<SiteFriendLink[]>([])
const announcements = ref<SiteAnnouncement[]>([])

const navDraft = reactive({
  labelZhCN: '',
  labelEnUS: '',
  href: '/',
  openInNewTab: false,
  position: 0,
  enabled: true
})

const friendDraft = reactive({
  name: '',
  url: 'https://',
  description: '',
  logoUrl: '',
  position: 0,
  enabled: true
})

const announcementDraft = reactive({
  titleZhCN: '',
  titleEnUS: '',
  bodyZhCN: '',
  bodyEnUS: '',
  style: 'info' as SiteAnnouncementStyle,
  href: '',
  dismissible: true,
  position: 0,
  enabled: true
})

const styleChoices = computed(() =>
  (['info', 'success', 'warning', 'danger'] as SiteAnnouncementStyle[]).map((value) => ({
    value,
    label: t(`admin.siteChrome.announcements.styles.${value}`)
  }))
)

const tabs = computed(() => [
  { id: 'brand' as const, label: t('admin.siteChrome.tabs.brand'), icon: 'i-lucide-image' },
  { id: 'nav' as const, label: t('admin.siteChrome.tabs.nav'), icon: 'i-lucide-menu' },
  { id: 'announcements' as const, label: t('admin.siteChrome.tabs.announcements'), icon: 'i-lucide-megaphone' },
  { id: 'legal' as const, label: t('admin.siteChrome.tabs.legal'), icon: 'i-lucide-scale' },
  { id: 'friendLinks' as const, label: t('admin.siteChrome.tabs.friendLinks'), icon: 'i-lucide-external-link' }
])

const brandDirty = computed(() => brandFormSnapshot() !== brandSnapshot.value)
const legalDirty = computed(() => legalFormSnapshot() !== legalSnapshot.value)

const {
  data: adminOptions,
  pending: optionsPending,
  error: optionsError,
  refresh: refreshOptions
} = await useAsyncData('admin-site-chrome-options', async () => {
  const envelope = await fetchAdminEnvelope()
  return envelope.data
})

watch(adminOptions, (items) => {
  if (items) {
    applyBrandAndLegal(items)
  }
}, { immediate: true })

const {
  pending: chromePending,
  error: chromeError,
  refresh: refreshChrome
} = await useAsyncData('admin-site-chrome-catalogs', async () => {
  pendingCatalog.value = true
  try {
    const [nav, friends, banners] = await Promise.all([
      chromeApi.listAdminNavItems(),
      chromeApi.listAdminFriendLinks(),
      chromeApi.listAdminAnnouncements()
    ])
    navItems.value = sortByPosition(nav)
    friendLinks.value = sortByPosition(friends)
    announcements.value = sortByPosition(banners)
    return true
  } finally {
    pendingCatalog.value = false
  }
})

useSeoMeta({
  title: t('admin.siteChrome.metaTitle')
})

async function refreshAll() {
  await Promise.all([refreshOptions(), refreshChrome()])
}

function applyBrandAndLegal(items: AdminWebOption[]) {
  const publicOptions = items.filter((item) => item.public && !item.secret)
  options.value = {
    ...options.value,
    ...Object.fromEntries(publicOptions.map((item) => [item.name, item.value]))
  }
  const map = Object.fromEntries(items.map((item) => [item.name, item]))
  brandForm.logoUrl = map['site.logo_url']?.value ?? ''
  brandForm.logoAttachmentId = map['site.logo_attachment_id']?.value ?? ''
  brandForm.faviconUrl = map['site.favicon_url']?.value ?? ''
  brandForm.faviconAttachmentId = map['site.favicon_attachment_id']?.value ?? ''
  brandForm.appleTouchIconUrl = map['site.apple_touch_icon_url']?.value ?? ''
  brandForm.appleTouchIconAttachmentId = map['site.apple_touch_icon_attachment_id']?.value ?? ''
  legalForm.termsZh = map['legal.terms.body.zh-CN']?.value ?? ''
  legalForm.termsEn = map['legal.terms.body.en-US']?.value ?? ''
  legalForm.privacyZh = map['legal.privacy.body.zh-CN']?.value ?? ''
  legalForm.privacyEn = map['legal.privacy.body.en-US']?.value ?? ''
  legalForm.guidelinesZh = map['legal.guidelines.body.zh-CN']?.value ?? ''
  legalForm.guidelinesEn = map['legal.guidelines.body.en-US']?.value ?? ''
  brandSnapshot.value = brandFormSnapshot()
  legalSnapshot.value = legalFormSnapshot()
}

function brandFormSnapshot() {
  return JSON.stringify({ ...brandForm })
}

function legalFormSnapshot() {
  return JSON.stringify({ ...legalForm })
}

function sortByPosition<T extends { position: number, id: number }>(items: T[]) {
  return [...items].sort((a, b) => a.position - b.position || a.id - b.id)
}

function resetBrand() {
  brandForm.logoUrl = ''
  brandForm.logoAttachmentId = ''
  brandForm.faviconUrl = ''
  brandForm.faviconAttachmentId = ''
  brandForm.appleTouchIconUrl = ''
  brandForm.appleTouchIconAttachmentId = ''
}

async function saveBrand() {
  savingBrand.value = true
  try {
    const items: WebOption[] = [
      { name: 'site.logo_url', value: brandForm.logoUrl.trim() },
      { name: 'site.logo_attachment_id', value: brandForm.logoAttachmentId.trim() },
      { name: 'site.favicon_url', value: brandForm.faviconUrl.trim() },
      { name: 'site.favicon_attachment_id', value: brandForm.faviconAttachmentId.trim() },
      { name: 'site.apple_touch_icon_url', value: brandForm.appleTouchIconUrl.trim() },
      { name: 'site.apple_touch_icon_attachment_id', value: brandForm.appleTouchIconAttachmentId.trim() }
    ]
    const updated = await saveMany(items)
    applyBrandAndLegal(updated)
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('admin.siteChrome.brand.saved') })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: apiErrorMessage(error) || t('admin.siteChrome.brand.saveFailed')
    })
  } finally {
    savingBrand.value = false
  }
}

async function saveLegal() {
  savingLegal.value = true
  try {
    const items: WebOption[] = [
      { name: 'legal.terms.body.zh-CN', value: legalForm.termsZh },
      { name: 'legal.terms.body.en-US', value: legalForm.termsEn },
      { name: 'legal.privacy.body.zh-CN', value: legalForm.privacyZh },
      { name: 'legal.privacy.body.en-US', value: legalForm.privacyEn },
      { name: 'legal.guidelines.body.zh-CN', value: legalForm.guidelinesZh },
      { name: 'legal.guidelines.body.en-US', value: legalForm.guidelinesEn }
    ]
    const updated = await saveMany(items)
    applyBrandAndLegal(updated)
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('admin.siteChrome.legal.saved') })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: apiErrorMessage(error) || t('admin.siteChrome.legal.saveFailed')
    })
  } finally {
    savingLegal.value = false
  }
}

async function addNavItem() {
  try {
    const created = await chromeApi.createNavItem({
      labelZhCN: navDraft.labelZhCN.trim(),
      labelEnUS: navDraft.labelEnUS.trim(),
      href: navDraft.href.trim(),
      openInNewTab: navDraft.openInNewTab,
      position: navDraft.position,
      enabled: navDraft.enabled
    })
    navItems.value = sortByPosition([...navItems.value, created])
    navDraft.labelZhCN = ''
    navDraft.labelEnUS = ''
    navDraft.href = '/'
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('admin.siteChrome.nav.created') })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: apiErrorMessage(error) || t('admin.siteChrome.nav.saveFailed')
    })
  }
}

async function toggleNavEnabled(item: SiteNavItem) {
  try {
    const updated = await chromeApi.updateNavItem(item.id, { enabled: !item.enabled })
    navItems.value = sortByPosition(navItems.value.map((row) => (row.id === updated.id ? updated : row)))
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('admin.siteChrome.nav.updated') })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: apiErrorMessage(error) || t('admin.siteChrome.nav.saveFailed')
    })
  }
}

async function removeNavItem(item: SiteNavItem) {
  try {
    await chromeApi.deleteNavItem(item.id)
    navItems.value = navItems.value.filter((row) => row.id !== item.id)
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('admin.siteChrome.nav.deleted') })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: apiErrorMessage(error) || t('admin.siteChrome.nav.saveFailed')
    })
  }
}

async function addFriendLink() {
  try {
    const created = await chromeApi.createFriendLink({
      name: friendDraft.name.trim(),
      url: friendDraft.url.trim(),
      description: friendDraft.description.trim(),
      logoUrl: friendDraft.logoUrl.trim(),
      position: friendDraft.position,
      enabled: friendDraft.enabled
    })
    friendLinks.value = sortByPosition([...friendLinks.value, created])
    friendDraft.name = ''
    friendDraft.url = 'https://'
    friendDraft.description = ''
    friendDraft.logoUrl = ''
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('admin.siteChrome.friendLinks.created') })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: apiErrorMessage(error) || t('admin.siteChrome.friendLinks.saveFailed')
    })
  }
}

async function toggleFriendEnabled(item: SiteFriendLink) {
  try {
    const updated = await chromeApi.updateFriendLink(item.id, { enabled: !item.enabled })
    friendLinks.value = sortByPosition(friendLinks.value.map((row) => (row.id === updated.id ? updated : row)))
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('admin.siteChrome.friendLinks.updated') })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: apiErrorMessage(error) || t('admin.siteChrome.friendLinks.saveFailed')
    })
  }
}

async function removeFriendLink(item: SiteFriendLink) {
  try {
    await chromeApi.deleteFriendLink(item.id)
    friendLinks.value = friendLinks.value.filter((row) => row.id !== item.id)
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('admin.siteChrome.friendLinks.deleted') })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: apiErrorMessage(error) || t('admin.siteChrome.friendLinks.saveFailed')
    })
  }
}

async function addAnnouncement() {
  try {
    const created = await chromeApi.createAnnouncement({
      titleZhCN: announcementDraft.titleZhCN.trim(),
      titleEnUS: announcementDraft.titleEnUS.trim(),
      bodyZhCN: announcementDraft.bodyZhCN.trim(),
      bodyEnUS: announcementDraft.bodyEnUS.trim(),
      style: announcementDraft.style,
      href: announcementDraft.href.trim(),
      dismissible: announcementDraft.dismissible,
      position: announcementDraft.position,
      enabled: announcementDraft.enabled
    })
    announcements.value = sortByPosition([...announcements.value, created])
    announcementDraft.titleZhCN = ''
    announcementDraft.titleEnUS = ''
    announcementDraft.bodyZhCN = ''
    announcementDraft.bodyEnUS = ''
    announcementDraft.href = ''
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('admin.siteChrome.announcements.created') })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: apiErrorMessage(error) || t('admin.siteChrome.announcements.saveFailed')
    })
  }
}

async function toggleAnnouncementEnabled(item: SiteAnnouncement) {
  try {
    const updated = await chromeApi.updateAnnouncement(item.id, { enabled: !item.enabled })
    announcements.value = sortByPosition(announcements.value.map((row) => (row.id === updated.id ? updated : row)))
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('admin.siteChrome.announcements.updated') })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: apiErrorMessage(error) || t('admin.siteChrome.announcements.saveFailed')
    })
  }
}

async function removeAnnouncement(item: SiteAnnouncement) {
  try {
    await chromeApi.deleteAnnouncement(item.id)
    announcements.value = announcements.value.filter((row) => row.id !== item.id)
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('admin.siteChrome.announcements.deleted') })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: apiErrorMessage(error) || t('admin.siteChrome.announcements.saveFailed')
    })
  }
}

function isEnglishLocale() {
  // i18n locale 可能是 en / en-US；按前缀判断即可。
  return String(locale.value).toLowerCase().startsWith('en')
}

function navLabel(item: SiteNavItem) {
  return isEnglishLocale() ? item.labelEnUS : item.labelZhCN
}

function setActiveTab(tab: ChromeTab) {
  activeTab.value = tab
}
</script>

<template>
  <div class="mb-4 flex flex-col gap-1">
    <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ t('admin.siteChrome.title') }}
    </h2>
    <p class="text-sm text-slate-500 dark:text-zinc-400">
      {{ t('admin.siteChrome.intro') }}
    </p>
  </div>

  <UDashboardToolbar class="mb-6 rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-500 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm">
        <UIcon name="i-lucide-panel-top" class="size-4" />
        <span class="truncate">{{ t('admin.siteChrome.toolbar') }}</span>
      </div>
    </template>
    <template #right>
      <UButton
        color="neutral"
        variant="outline"
        leading-icon="i-lucide-refresh-cw"
        :loading="optionsPending || chromePending || pendingCatalog"
        class="border-slate-200 dark:border-zinc-700"
        @click="refreshAll()"
      >
        {{ t('admin.siteChrome.refresh') }}
      </UButton>
    </template>
  </UDashboardToolbar>

  <UAlert
    v-if="optionsError || chromeError"
    color="error"
    variant="soft"
    icon="i-lucide-triangle-alert"
    class="mb-4"
    :title="t('admin.siteChrome.loadFailed')"
  />

  <UAlert
    color="primary"
    variant="soft"
    icon="i-lucide-sparkles"
    class="mb-4"
    :title="t('admin.siteChrome.recommendedTitle')"
    :description="t('admin.siteChrome.recommendedBody')"
  />

  <div class="mb-5 flex flex-wrap gap-1" role="tablist" :aria-label="t('admin.siteChrome.title')">
    <UButton
      v-for="tab in tabs"
      :key="tab.id"
      size="sm"
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

  <!-- 品牌资源 -->
  <form v-if="activeTab === 'brand'" class="flex flex-col gap-5" @submit.prevent="saveBrand">
    <UCard class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <template #header>
        <div>
          <h3 class="text-base font-bold text-slate-900 dark:text-white">
            {{ t('admin.siteChrome.brand.title') }}
          </h3>
          <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.siteChrome.brand.description') }}
          </p>
        </div>
      </template>

      <div class="grid max-w-4xl gap-4 md:grid-cols-2">
        <UFormField :label="t('admin.siteChrome.brand.logoUrl')" name="logo-url">
          <UInput v-model="brandForm.logoUrl" icon="i-lucide-image" class="w-full" :placeholder="t('admin.siteChrome.brand.urlPlaceholder')" />
        </UFormField>
        <UFormField :label="t('admin.siteChrome.brand.logoAttachmentId')" name="logo-attachment">
          <UInput v-model="brandForm.logoAttachmentId" class="w-full font-mono" placeholder="42" />
        </UFormField>
        <UFormField :label="t('admin.siteChrome.brand.faviconUrl')" name="favicon-url">
          <UInput v-model="brandForm.faviconUrl" icon="i-lucide-bookmark" class="w-full" :placeholder="t('admin.siteChrome.brand.urlPlaceholder')" />
        </UFormField>
        <UFormField :label="t('admin.siteChrome.brand.faviconAttachmentId')" name="favicon-attachment">
          <UInput v-model="brandForm.faviconAttachmentId" class="w-full font-mono" placeholder="43" />
        </UFormField>
        <UFormField :label="t('admin.siteChrome.brand.appleTouchUrl')" name="apple-touch-url">
          <UInput v-model="brandForm.appleTouchIconUrl" icon="i-lucide-smartphone" class="w-full" :placeholder="t('admin.siteChrome.brand.urlPlaceholder')" />
        </UFormField>
        <UFormField :label="t('admin.siteChrome.brand.appleTouchAttachmentId')" name="apple-touch-attachment">
          <UInput v-model="brandForm.appleTouchIconAttachmentId" class="w-full font-mono" placeholder="44" />
        </UFormField>
      </div>

      <template #footer>
        <SFAdminFormFooter
          :saving="savingBrand"
          :show-unsaved-alert="brandDirty"
          :submit-text="t('admin.siteChrome.brand.save')"
          :reset-text="t('admin.siteChrome.brand.restoreEmpty')"
          @reset="resetBrand"
        />
      </template>
    </UCard>
  </form>

  <!-- 导航 -->
  <div v-else-if="activeTab === 'nav'" class="flex flex-col gap-5">
    <UCard class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <template #header>
        <div>
          <h3 class="text-base font-bold text-slate-900 dark:text-white">
            {{ t('admin.siteChrome.nav.title') }}
          </h3>
          <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.siteChrome.nav.description') }}
          </p>
        </div>
      </template>

      <div class="mb-4 grid gap-3 rounded-lg border border-slate-200 bg-slate-50 p-3 dark:border-zinc-800 dark:bg-zinc-950/60 lg:grid-cols-[1fr_1fr_1.2fr_auto_auto]">
        <UInput v-model="navDraft.labelZhCN" :placeholder="t('admin.siteChrome.nav.labelZh')" maxlength="40" />
        <UInput v-model="navDraft.labelEnUS" :placeholder="t('admin.siteChrome.nav.labelEn')" maxlength="40" />
        <UInput v-model="navDraft.href" icon="i-lucide-link" :placeholder="t('admin.siteChrome.nav.hrefPlaceholder')" />
        <UInput v-model.number="navDraft.position" type="number" class="w-24" :placeholder="t('admin.siteChrome.position')" />
        <UButton color="primary" leading-icon="i-lucide-plus" @click="addNavItem">
          {{ t('admin.siteChrome.nav.add') }}
        </UButton>
      </div>

      <div v-if="navItems.length === 0" class="py-8 text-center text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.siteChrome.nav.empty') }}
      </div>
      <ul v-else class="divide-y divide-slate-200 dark:divide-zinc-800">
        <li
          v-for="item in navItems"
          :key="item.id"
          class="flex flex-col gap-2 py-3 sm:flex-row sm:items-center sm:justify-between"
        >
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <span class="font-semibold text-slate-900 dark:text-zinc-100">{{ navLabel(item) }}</span>
              <UBadge color="neutral" variant="soft" class="font-mono text-xs">
                #{{ item.position }}
              </UBadge>
              <UBadge :color="item.enabled ? 'success' : 'neutral'" variant="soft">
                {{ item.enabled ? t('admin.siteChrome.enabled') : t('admin.siteChrome.disabled') }}
              </UBadge>
            </div>
            <p class="mt-0.5 truncate font-mono text-xs text-slate-500 dark:text-zinc-400">
              {{ item.href }}
            </p>
          </div>
          <div class="flex shrink-0 gap-2">
            <UButton size="sm" color="neutral" variant="outline" @click="toggleNavEnabled(item)">
              {{ item.enabled ? t('admin.siteChrome.disable') : t('admin.siteChrome.enable') }}
            </UButton>
            <UButton size="sm" color="error" variant="soft" leading-icon="i-lucide-trash-2" @click="removeNavItem(item)">
              {{ t('admin.siteChrome.delete') }}
            </UButton>
          </div>
        </li>
      </ul>
    </UCard>
  </div>

  <!-- 公告 -->
  <div v-else-if="activeTab === 'announcements'" class="flex flex-col gap-5">
    <UCard class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <template #header>
        <div>
          <h3 class="text-base font-bold text-slate-900 dark:text-white">
            {{ t('admin.siteChrome.announcements.title') }}
          </h3>
          <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.siteChrome.announcements.description') }}
          </p>
        </div>
      </template>

      <div class="mb-4 grid gap-3 rounded-lg border border-slate-200 bg-slate-50 p-3 dark:border-zinc-800 dark:bg-zinc-950/60 md:grid-cols-2">
        <UInput v-model="announcementDraft.titleZhCN" :placeholder="t('admin.siteChrome.announcements.titleZh')" maxlength="120" />
        <UInput v-model="announcementDraft.titleEnUS" :placeholder="t('admin.siteChrome.announcements.titleEn')" maxlength="120" />
        <UTextarea v-model="announcementDraft.bodyZhCN" :rows="2" :placeholder="t('admin.siteChrome.announcements.bodyZh')" maxlength="2000" class="md:col-span-1" />
        <UTextarea v-model="announcementDraft.bodyEnUS" :rows="2" :placeholder="t('admin.siteChrome.announcements.bodyEn')" maxlength="2000" class="md:col-span-1" />
        <USelect v-model="announcementDraft.style" :items="styleChoices" value-key="value" label-key="label" class="w-full" />
        <UInput v-model="announcementDraft.href" icon="i-lucide-link" :placeholder="t('admin.siteChrome.announcements.hrefPlaceholder')" />
        <div class="flex flex-wrap items-end gap-3 md:col-span-2">
          <UInput v-model.number="announcementDraft.position" type="number" class="w-28" :placeholder="t('admin.siteChrome.position')" />
          <UButton color="primary" leading-icon="i-lucide-plus" @click="addAnnouncement">
            {{ t('admin.siteChrome.announcements.add') }}
          </UButton>
        </div>
      </div>

      <div v-if="announcements.length === 0" class="py-8 text-center text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.siteChrome.announcements.empty') }}
      </div>
      <ul v-else class="divide-y divide-slate-200 dark:divide-zinc-800">
        <li
          v-for="item in announcements"
          :key="item.id"
          class="flex flex-col gap-2 py-3 sm:flex-row sm:items-start sm:justify-between"
        >
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <span class="font-semibold text-slate-900 dark:text-zinc-100">
                {{ isEnglishLocale() ? (item.titleEnUS || item.titleZhCN) : (item.titleZhCN || item.titleEnUS) }}
              </span>
              <UBadge color="neutral" variant="soft">
                {{ t(`admin.siteChrome.announcements.styles.${item.style}`) }}
              </UBadge>
              <UBadge :color="item.enabled ? 'success' : 'neutral'" variant="soft">
                {{ item.enabled ? t('admin.siteChrome.enabled') : t('admin.siteChrome.disabled') }}
              </UBadge>
            </div>
            <p class="mt-1 text-sm text-slate-600 dark:text-zinc-400">
              {{ isEnglishLocale() ? (item.bodyEnUS || item.bodyZhCN) : (item.bodyZhCN || item.bodyEnUS) }}
            </p>
          </div>
          <div class="flex shrink-0 gap-2">
            <UButton size="sm" color="neutral" variant="outline" @click="toggleAnnouncementEnabled(item)">
              {{ item.enabled ? t('admin.siteChrome.disable') : t('admin.siteChrome.enable') }}
            </UButton>
            <UButton size="sm" color="error" variant="soft" leading-icon="i-lucide-trash-2" @click="removeAnnouncement(item)">
              {{ t('admin.siteChrome.delete') }}
            </UButton>
          </div>
        </li>
      </ul>
    </UCard>
  </div>

  <!-- 法律正文 -->
  <form v-else-if="activeTab === 'legal'" class="flex flex-col gap-5" @submit.prevent="saveLegal">
    <UCard class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <template #header>
        <div>
          <h3 class="text-base font-bold text-slate-900 dark:text-white">
            {{ t('admin.siteChrome.legal.title') }}
          </h3>
          <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.siteChrome.legal.description') }}
          </p>
        </div>
      </template>

      <div class="grid max-w-5xl gap-5">
        <div class="grid gap-3 lg:grid-cols-2">
          <UFormField :label="t('admin.siteChrome.legal.termsZh')" name="terms-zh">
            <UTextarea v-model="legalForm.termsZh" :rows="8" class="w-full font-mono text-sm" />
          </UFormField>
          <UFormField :label="t('admin.siteChrome.legal.termsEn')" name="terms-en">
            <UTextarea v-model="legalForm.termsEn" :rows="8" class="w-full font-mono text-sm" />
          </UFormField>
        </div>
        <div class="grid gap-3 lg:grid-cols-2">
          <UFormField :label="t('admin.siteChrome.legal.privacyZh')" name="privacy-zh">
            <UTextarea v-model="legalForm.privacyZh" :rows="8" class="w-full font-mono text-sm" />
          </UFormField>
          <UFormField :label="t('admin.siteChrome.legal.privacyEn')" name="privacy-en">
            <UTextarea v-model="legalForm.privacyEn" :rows="8" class="w-full font-mono text-sm" />
          </UFormField>
        </div>
        <div class="grid gap-3 lg:grid-cols-2">
          <UFormField :label="t('admin.siteChrome.legal.guidelinesZh')" name="guidelines-zh">
            <UTextarea v-model="legalForm.guidelinesZh" :rows="8" class="w-full font-mono text-sm" />
          </UFormField>
          <UFormField :label="t('admin.siteChrome.legal.guidelinesEn')" name="guidelines-en">
            <UTextarea v-model="legalForm.guidelinesEn" :rows="8" class="w-full font-mono text-sm" />
          </UFormField>
        </div>
      </div>

      <template #footer>
        <div class="flex w-full items-center justify-between">
          <p v-if="legalDirty" class="text-xs font-medium text-amber-600 dark:text-amber-400">
            {{ t('admin.form.unsavedChanges') }}
          </p>
          <span v-else />
          <UButton type="submit" color="primary" leading-icon="i-lucide-save" :loading="savingLegal">
            {{ t('admin.siteChrome.legal.save') }}
          </UButton>
        </div>
      </template>
    </UCard>
  </form>

  <!-- 友情链接 -->
  <div v-else class="flex flex-col gap-5">
    <UCard class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <template #header>
        <div>
          <h3 class="text-base font-bold text-slate-900 dark:text-white">
            {{ t('admin.siteChrome.friendLinks.title') }}
          </h3>
          <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.siteChrome.friendLinks.description') }}
          </p>
        </div>
      </template>

      <div class="mb-4 grid gap-3 rounded-lg border border-slate-200 bg-slate-50 p-3 dark:border-zinc-800 dark:bg-zinc-950/60 lg:grid-cols-[1fr_1.2fr_1fr_auto]">
        <UInput v-model="friendDraft.name" :placeholder="t('admin.siteChrome.friendLinks.name')" maxlength="80" />
        <UInput v-model="friendDraft.url" icon="i-lucide-link" :placeholder="t('admin.siteChrome.friendLinks.urlPlaceholder')" />
        <UInput v-model="friendDraft.description" :placeholder="t('admin.siteChrome.friendLinks.descriptionField')" maxlength="200" />
        <UButton color="primary" leading-icon="i-lucide-plus" @click="addFriendLink">
          {{ t('admin.siteChrome.friendLinks.add') }}
        </UButton>
      </div>

      <div v-if="friendLinks.length === 0" class="py-8 text-center text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.siteChrome.friendLinks.empty') }}
      </div>
      <ul v-else class="divide-y divide-slate-200 dark:divide-zinc-800">
        <li
          v-for="item in friendLinks"
          :key="item.id"
          class="flex flex-col gap-2 py-3 sm:flex-row sm:items-center sm:justify-between"
        >
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <span class="font-semibold text-slate-900 dark:text-zinc-100">{{ item.name }}</span>
              <UBadge :color="item.enabled ? 'success' : 'neutral'" variant="soft">
                {{ item.enabled ? t('admin.siteChrome.enabled') : t('admin.siteChrome.disabled') }}
              </UBadge>
            </div>
            <p class="mt-0.5 truncate font-mono text-xs text-slate-500 dark:text-zinc-400">
              {{ item.url }}
            </p>
          </div>
          <div class="flex shrink-0 gap-2">
            <UButton size="sm" color="neutral" variant="outline" @click="toggleFriendEnabled(item)">
              {{ item.enabled ? t('admin.siteChrome.disable') : t('admin.siteChrome.enable') }}
            </UButton>
            <UButton size="sm" color="error" variant="soft" leading-icon="i-lucide-trash-2" @click="removeFriendLink(item)">
              {{ t('admin.siteChrome.delete') }}
            </UButton>
          </div>
        </li>
      </ul>
    </UCard>
  </div>
</template>
