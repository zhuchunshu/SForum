<script setup lang="ts">
import type { AdminWebOption, WebOption } from '~/composables/useWebOptions'
import { useAdminTabs } from '~/composables/useAdminTabs'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminSettings'
})

type SettingsTab = 'basic' | 'verification'

const { t } = useI18n()
const toast = useToast()
const { options, fetchAdminEnvelope, saveMany } = useWebOptions()
const adminTabs = useAdminTabs()

onMounted(() => {
  adminTabs.openTab('/settings', 'admin.nav.settings', 'i-lucide-settings-2', 'AdminSettings')
})

const activeTab = ref<SettingsTab>('basic')
const savingBasic = ref(false)
const savingVerification = ref(false)

const localeChoices = computed(() => [
  { label: t('admin.settings.locale.zhCN'), value: 'zh-CN' },
  { label: t('admin.settings.locale.enUS'), value: 'en-US' }
])

const tabs = computed<Array<{ id: SettingsTab, label: string, icon: string }>>(() => [
  { id: 'basic', label: t('admin.settings.tabs.basic'), icon: 'i-lucide-sliders-horizontal' },
  { id: 'verification', label: t('admin.settings.tabs.verification'), icon: 'i-lucide-shield-check' }
])

const form = reactive({
  siteName: options.value['site.name'] || 'SForum',
  siteUrl: options.value['site.url'] || 'http://127.0.0.1:3000',
  defaultLocale: options.value['site.default_locale'] || 'zh-CN',
  supportedLocales: parseLocaleList(options.value['site.supported_locales'] || 'zh-CN,en-US'),
  humanVerificationProvider: normalizeProvider(options.value['human_verification.provider']),
  altchaSecret: '',
  altchaSecretSet: false,
  altchaChallengeTTL: '10m',
  altchaCost: 1000
})

const altchaSecretPlaceholder = computed(() => {
  return form.altchaSecretSet
    ? t('admin.settings.verification.keepSecretPlaceholder')
    : t('admin.settings.verification.secretPlaceholder')
})

const { pending, error, refresh } = await useAsyncData('admin-web-options', async () => {
  const envelope = await fetchAdminEnvelope()
  applyAdminOptions(envelope.data)
  return envelope.data
})

useSeoMeta({
  title: t('admin.settings.metaTitle')
})

function applyAdminOptions(items: AdminWebOption[]) {
  const publicOptions = items.filter((item) => item.public && !item.secret)
  options.value = {
    ...options.value,
    ...Object.fromEntries(publicOptions.map((item) => [item.name, item.value]))
  }

  const map = Object.fromEntries(items.map((item) => [item.name, item]))
  form.siteName = map['site.name']?.value || 'SForum'
  form.siteUrl = map['site.url']?.value || 'http://127.0.0.1:3000'
  form.defaultLocale = map['site.default_locale']?.value || 'zh-CN'
  form.supportedLocales = parseLocaleList(map['site.supported_locales']?.value || 'zh-CN,en-US')
  if (!form.supportedLocales.includes(form.defaultLocale)) {
    form.defaultLocale = form.supportedLocales[0] || 'zh-CN'
  }
  form.humanVerificationProvider = normalizeProvider(map['human_verification.provider']?.value)
  form.altchaSecret = ''
  form.altchaSecretSet = map['human_verification.altcha.secret']?.secretSet === true
  form.altchaChallengeTTL = map['human_verification.altcha.challenge_ttl']?.value || '10m'
  form.altchaCost = Number(map['human_verification.altcha.cost']?.value || 1000)
}

async function saveBasicSettings() {
  savingBasic.value = true
  try {
    await saveAndApply([
      { name: 'site.name', value: form.siteName },
      { name: 'site.url', value: form.siteUrl },
      { name: 'site.default_locale', value: form.defaultLocale },
      { name: 'site.supported_locales', value: form.supportedLocales.join(',') }
    ])
    toast.add({
      color: 'success',
      icon: 'i-lucide-check',
      title: t('admin.settings.saved')
    })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: apiErrorMessage(error) || t('admin.settings.saveFailed')
    })
  } finally {
    savingBasic.value = false
  }
}

async function saveVerificationSettings() {
  savingVerification.value = true
  try {
    const payload: WebOption[] = [
      { name: 'human_verification.provider', value: form.humanVerificationProvider },
      { name: 'human_verification.altcha.challenge_ttl', value: form.altchaChallengeTTL },
      { name: 'human_verification.altcha.cost', value: String(form.altchaCost) }
    ]
    if (form.altchaSecret.trim() !== '') {
      payload.push({ name: 'human_verification.altcha.secret', value: form.altchaSecret })
    }
    await saveAndApply(payload)
    toast.add({
      color: 'success',
      icon: 'i-lucide-check',
      title: t('admin.settings.saved')
    })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: apiErrorMessage(error) || t('admin.settings.saveFailed')
    })
  } finally {
    savingVerification.value = false
  }
}

async function saveAndApply(items: WebOption[]) {
  const updated = await saveMany(items)
  applyAdminOptions(updated)
}

function parseLocaleList(value: string) {
  const locales = value.split(',').map((item) => item.trim()).filter(Boolean)
  return locales.length > 0 ? locales : ['zh-CN', 'en-US']
}

function normalizeProvider(value: string | undefined) {
  return value?.trim().toLowerCase() === 'altcha' ? 'altcha' : 'disabled'
}

function setActiveTab(tab: SettingsTab) {
  activeTab.value = tab
}

function onLocaleToggle(locale: string, event: Event) {
  const checked = event.target instanceof HTMLInputElement && event.target.checked
  const selected = new Set(form.supportedLocales)
  if (checked) {
    selected.add(locale)
  } else if (selected.size > 1) {
    selected.delete(locale)
  }

  form.supportedLocales = localeChoices.value
    .map((choice) => choice.value)
    .filter((value) => selected.has(value))
  if (!form.supportedLocales.includes(form.defaultLocale)) {
    form.defaultLocale = form.supportedLocales[0] || 'zh-CN'
  }
}
</script>

<template>
  <div class="mb-4">
    <h2 class="text-xl font-bold flex items-center gap-2 text-slate-900 dark:text-zinc-100">
      <UIcon name="i-lucide-settings-2" class="size-5 text-teal-600 dark:text-teal-400" />
      {{ t('admin.settings.title') }}
    </h2>
  </div>

  <UDashboardToolbar class="border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 rounded-lg px-4 py-2.5 mb-6 text-slate-500 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm text-slate-500 dark:text-zinc-400">
        <UIcon name="i-lucide-database" class="size-4" />
        <span class="truncate">{{ t('admin.settings.intro') }}</span>
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
        {{ t('admin.settings.refresh') }}
      </UButton>
    </template>
  </UDashboardToolbar>

  <div class="flex flex-col gap-4">
    <UAlert
      v-if="error"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="t('admin.settings.loadFailed')"
    />

    <div
      role="tablist"
      :aria-label="t('admin.settings.tabs.label')"
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

    <UCard
      v-if="activeTab === 'basic'"
      class="border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100"
    >
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div>
            <h2 class="text-base font-bold text-slate-900 dark:text-white">
              {{ t('admin.settings.basic.title') }}
            </h2>
            <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.settings.basic.description') }}
            </p>
          </div>
          <UBadge color="neutral" variant="soft" class="border border-slate-200 dark:border-zinc-800 font-mono">
            site.*
          </UBadge>
        </div>
      </template>

      <form class="grid max-w-3xl gap-4" @submit.prevent="saveBasicSettings">
        <UFormField :label="t('admin.settings.siteName')" name="site-name">
          <UInput
            v-model="form.siteName"
            icon="i-lucide-message-square-text"
            :placeholder="t('admin.settings.siteNamePlaceholder')"
            maxlength="80"
            required
            class="w-full"
          />
        </UFormField>

        <UFormField :label="t('admin.settings.siteUrl')" name="site-url">
          <UInput
            v-model="form.siteUrl"
            icon="i-lucide-link"
            type="url"
            :placeholder="t('admin.settings.siteUrlPlaceholder')"
            required
            class="w-full"
          />
        </UFormField>

        <UFormField :label="t('admin.settings.defaultLocale')" name="default-locale">
          <select
            v-model="form.defaultLocale"
            class="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-900 outline-none transition focus:border-teal-500 focus:ring-2 focus:ring-teal-500/20 dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100"
          >
            <option
              v-for="choice in localeChoices.filter((choice) => form.supportedLocales.includes(choice.value))"
              :key="choice.value"
              :value="choice.value"
            >
              {{ choice.label }}
            </option>
          </select>
        </UFormField>

        <UFormField :label="t('admin.settings.supportedLocales')" name="supported-locales">
          <div class="flex flex-wrap gap-3">
            <label
              v-for="choice in localeChoices"
              :key="choice.value"
              class="flex h-10 items-center gap-2 rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-700 dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-200"
            >
              <input
                type="checkbox"
                class="size-4 rounded border-slate-300 text-teal-600 focus:ring-teal-500"
                :checked="form.supportedLocales.includes(choice.value)"
                @change="onLocaleToggle(choice.value, $event)"
              />
              <span>{{ choice.label }}</span>
            </label>
          </div>
        </UFormField>

        <div class="flex justify-end pt-2">
          <UButton
            type="submit"
            leading-icon="i-lucide-save"
            :loading="savingBasic"
            class="bg-teal-600 hover:bg-teal-700 dark:bg-teal-500 dark:hover:bg-teal-600 text-white font-medium"
          >
            {{ t('admin.settings.save') }}
          </UButton>
        </div>
      </form>
    </UCard>

    <UCard
      v-else
      class="border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100"
    >
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div>
            <h2 class="text-base font-bold text-slate-900 dark:text-white">
              {{ t('admin.settings.verification.title') }}
            </h2>
            <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.settings.verification.description') }}
            </p>
          </div>
          <UBadge color="neutral" variant="soft" class="border border-slate-200 dark:border-zinc-800 font-mono">
            ALTCHA
          </UBadge>
        </div>
      </template>

      <form class="grid max-w-3xl gap-4" @submit.prevent="saveVerificationSettings">
        <UFormField :label="t('admin.settings.verification.provider')" name="verification-provider">
          <select
            v-model="form.humanVerificationProvider"
            class="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-900 outline-none transition focus:border-teal-500 focus:ring-2 focus:ring-teal-500/20 dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100"
          >
            <option value="disabled">{{ t('admin.settings.verification.disabled') }}</option>
            <option value="altcha">{{ t('admin.settings.verification.altcha') }}</option>
          </select>
        </UFormField>

        <div class="grid gap-4 md:grid-cols-2">
          <UFormField :label="t('admin.settings.verification.altchaSecret')" name="altcha-secret">
            <UInput
              v-model="form.altchaSecret"
              icon="i-lucide-key-round"
              type="password"
              :placeholder="altchaSecretPlaceholder"
              class="w-full"
            />
          </UFormField>

          <div class="flex items-end">
            <UBadge
              :color="form.altchaSecretSet ? 'success' : 'neutral'"
              variant="soft"
              class="h-9 border border-slate-200 px-3 dark:border-zinc-800"
            >
              {{ form.altchaSecretSet ? t('admin.settings.verification.secretConfigured') : t('admin.settings.verification.secretMissing') }}
            </UBadge>
          </div>
        </div>

        <div class="grid gap-4 md:grid-cols-2">
          <UFormField :label="t('admin.settings.verification.challengeTTL')" name="altcha-ttl">
            <UInput
              v-model="form.altchaChallengeTTL"
              icon="i-lucide-clock-3"
              placeholder="10m"
              required
              class="w-full"
            />
          </UFormField>

          <UFormField :label="t('admin.settings.verification.cost')" name="altcha-cost">
            <UInput
              v-model.number="form.altchaCost"
              icon="i-lucide-cpu"
              type="number"
              min="1"
              required
              class="w-full"
            />
          </UFormField>
        </div>

        <div class="flex justify-end pt-2">
          <UButton
            type="submit"
            leading-icon="i-lucide-save"
            :loading="savingVerification"
            class="bg-teal-600 hover:bg-teal-700 dark:bg-teal-500 dark:hover:bg-teal-600 text-white font-medium"
          >
            {{ t('admin.settings.save') }}
          </UButton>
        </div>
      </form>
    </UCard>
  </div>
</template>
