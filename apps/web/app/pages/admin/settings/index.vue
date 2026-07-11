<script setup lang="ts">
import type {
  AdminWebOption,
  AltchaWidgetAuto,
  AltchaWidgetDisplay,
  AltchaWidgetType,
  HumanVerificationScenario,
  WebOption
} from '~/composables/useWebOptions'
import {
  altchaWidgetAutoModes,
  altchaWidgetDisplays,
  altchaWidgetTypes,
  enabledOptionValue,
  humanVerificationScenarioOptionName,
  normalizeEnabledOption,
  recommendedPasswordPolicy
} from '~/composables/useWebOptions'
import { useAdminPage } from '~/composables/useAdminPage'
import {
  commonSiteTimezones,
  normalizeSiteDateFormat,
  normalizeSiteStartOfWeek,
  normalizeSiteTimeFormat,
  normalizeSiteTimezone,
  previewSiteDateTime,
  recommendedSiteDateTimeSettings,
  siteDateFormats,
  siteTimeFormats,
  type SiteDateFormat,
  type SiteTimeFormat
} from '~/utils/siteDateTime'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminSettings'
})

type SettingsTab = 'basic' | 'accountSecurity' | 'verification'

const { t } = useI18n()
const toast = useToast()
const { options, fetchAdminEnvelope, saveMany } = useWebOptions()
const adminPage = useAdminPage('/settings')

const activeTab = ref<SettingsTab>('basic')
const savingBasic = ref(false)
const savingAccountSecurity = ref(false)
const savingVerification = ref(false)
const showAltchaSecret = ref(false)

type VerificationScenarioConfig = {
  key: HumanVerificationScenario
  label: string
  description: string
  icon: string
}

const scenarioFallbacks: Record<HumanVerificationScenario, boolean> = {
  register: true,
  password_reset: false,
  login_risk: false,
  post_risk: false
}

const ttlSuggestions = [10, 20, 30, 60]
const costSuggestions = [1000, 3000, 5000]
const workerSuggestions = [1, 2, 4, 8]
const minDurationSuggestions = [0, 500, 1000, 1500]

const localeChoices = computed(() => [
  { label: t('admin.settings.locale.zhCN'), value: 'zh-CN' },
  { label: t('admin.settings.locale.enUS'), value: 'en-US' }
])

const dateFormatChoices = computed(() => siteDateFormats.map((value) => ({
  value,
  label: t(`admin.settings.basic.dateFormatOptions.${value}`)
})))

const timeFormatChoices = computed(() => siteTimeFormats.map((value) => ({
  value,
  label: t(`admin.settings.basic.timeFormatOptions.${value}`)
})))

const startOfWeekChoices = computed(() => [
  { value: 0, label: t('admin.settings.basic.weekdays.0') },
  { value: 1, label: t('admin.settings.basic.weekdays.1') },
  { value: 2, label: t('admin.settings.basic.weekdays.2') },
  { value: 3, label: t('admin.settings.basic.weekdays.3') },
  { value: 4, label: t('admin.settings.basic.weekdays.4') },
  { value: 5, label: t('admin.settings.basic.weekdays.5') },
  { value: 6, label: t('admin.settings.basic.weekdays.6') }
])

const tabs = computed<Array<{ id: SettingsTab, label: string, icon: string }>>(() => [
  { id: 'basic', label: t('admin.settings.tabs.basic'), icon: 'i-lucide-sliders-horizontal' },
  { id: 'accountSecurity', label: t('admin.settings.tabs.accountSecurity'), icon: 'i-lucide-shield' },
  { id: 'verification', label: t('admin.settings.tabs.verification'), icon: 'i-lucide-shield-check' }
])

const verificationScenarios = computed<VerificationScenarioConfig[]>(() => [
  {
    key: 'register',
    label: t('admin.settings.verification.scenarios.register.label'),
    description: t('admin.settings.verification.scenarios.register.description'),
    icon: 'i-lucide-user-plus'
  },
  {
    key: 'password_reset',
    label: t('admin.settings.verification.scenarios.passwordReset.label'),
    description: t('admin.settings.verification.scenarios.passwordReset.description'),
    icon: 'i-lucide-key-round'
  },
  {
    key: 'login_risk',
    label: t('admin.settings.verification.scenarios.loginRisk.label'),
    description: t('admin.settings.verification.scenarios.loginRisk.description'),
    icon: 'i-lucide-radar'
  },
  {
    key: 'post_risk',
    label: t('admin.settings.verification.scenarios.postRisk.label'),
    description: t('admin.settings.verification.scenarios.postRisk.description'),
    icon: 'i-lucide-message-square-warning'
  }
])

const altchaTypeOptions = computed(() => altchaWidgetTypes.map((value) => ({
  value,
  label: t(`admin.settings.verification.widget.typeOptions.${value}`)
})))
const altchaAutoOptions = computed(() => altchaWidgetAutoModes.map((value) => ({
  value,
  label: t(`admin.settings.verification.widget.autoOptions.${value}`)
})))
const altchaDisplayOptions = computed(() => altchaWidgetDisplays.map((value) => ({
  value,
  label: t(`admin.settings.verification.widget.displayOptions.${value}`)
})))

const form = reactive({
  siteName: options.value['site.name'] || 'SForum',
  siteUrl: options.value['site.url'] || 'http://127.0.0.1:3000',
  defaultLocale: options.value['site.default_locale'] || 'zh-CN',
  supportedLocales: parseLocaleList(options.value['site.supported_locales'] || 'zh-CN,en-US'),
  // 站点时区与日期时间展示（库内 UTC，仅影响展示）。
  timezone: normalizeSiteTimezone(options.value['site.timezone']),
  dateFormat: normalizeSiteDateFormat(options.value['site.date_format']) as SiteDateFormat,
  timeFormat: normalizeSiteTimeFormat(options.value['site.time_format']) as SiteTimeFormat,
  startOfWeek: normalizeSiteStartOfWeek(options.value['site.start_of_week']),
  passwordMinLength: recommendedPasswordPolicy.minLength,
  passwordMaxLength: recommendedPasswordPolicy.maxLength,
  passwordRequireLowercase: recommendedPasswordPolicy.requireLowercase,
  passwordRequireUppercase: recommendedPasswordPolicy.requireUppercase,
  passwordRequireNumber: recommendedPasswordPolicy.requireNumber,
  passwordRequireSymbol: recommendedPasswordPolicy.requireSymbol,
  // 最大活跃设备数（identity.sessions.max_devices），默认 5（与后端 RecommendedMaxDevices 对齐）。
  sessionsMaxDevices: 5,
  // 已下线历史会话保留天数（identity.sessions.keep_days），默认 30。
  sessionsKeepDays: 30,
  humanVerificationProvider: normalizeProvider(options.value['human_verification.provider']),
  humanVerificationScenarios: {
    register: normalizeEnabledOption(options.value[humanVerificationScenarioOptionName('register')], true),
    password_reset: normalizeEnabledOption(options.value[humanVerificationScenarioOptionName('password_reset')], false),
    login_risk: normalizeEnabledOption(options.value[humanVerificationScenarioOptionName('login_risk')], false),
    post_risk: normalizeEnabledOption(options.value[humanVerificationScenarioOptionName('post_risk')], false)
  } as Record<HumanVerificationScenario, boolean>,
  altchaSecret: '',
  altchaSecretSet: false,
  altchaChallengeTTLMinutes: 10,
  altchaCost: 1000,
  altchaWidgetType: 'checkbox' as AltchaWidgetType,
  altchaWidgetAuto: 'off' as AltchaWidgetAuto,
  altchaWidgetDisplay: 'standard' as AltchaWidgetDisplay,
  altchaWidgetHideLogo: true,
  altchaWidgetHideFooter: true,
  altchaWidgetWorkers: 2,
  altchaWidgetMinDuration: 500
})

// 依赖 form 的计算属性放在 form 定义之后。
const timezoneChoices = computed(() => {
  const current = form.timezone
  const base = [...commonSiteTimezones] as string[]
  if (current && !base.includes(current)) {
    base.unshift(current)
  }
  return base.map((value) => ({
    value,
    label: value === 'UTC' ? t('admin.settings.basic.timezoneUtc') : value
  }))
})

const datetimePreview = computed(() => previewSiteDateTime({
  timezone: form.timezone,
  dateFormat: form.dateFormat,
  timeFormat: form.timeFormat,
  startOfWeek: form.startOfWeek
}, form.defaultLocale || 'zh-CN'))

const altchaSecretPlaceholder = computed(() => {
  return form.altchaSecretSet
    ? t('admin.settings.verification.keepSecretPlaceholder')
    : t('admin.settings.verification.secretPlaceholder')
})

const altchaConfigRows = computed(() => [
  { label: t('admin.settings.verification.config.algorithm'), value: 'PBKDF2/SHA-256' },
  { label: t('admin.settings.verification.config.signature'), value: 'HMAC-SHA-256' },
  { label: t('admin.settings.verification.config.challengeEndpoint'), value: '/api/v1/human-verification/challenge?purpose={purpose}' },
  { label: t('admin.settings.verification.config.widgetType'), value: form.altchaWidgetType },
  { label: t('admin.settings.verification.config.widgetAuto'), value: form.altchaWidgetAuto },
  { label: t('admin.settings.verification.config.widgetDisplay'), value: form.altchaWidgetDisplay },
  { label: t('admin.settings.verification.config.replayProtection'), value: t('admin.settings.verification.config.replayProtectionValue') },
  { label: t('admin.settings.verification.config.rateLimit'), value: t('admin.settings.verification.config.rateLimitValue') },
  { label: t('admin.settings.verification.config.clientWidget'), value: 'ALTCHA widget v3' }
])

const adminOptionsMap = ref<Record<string, AdminWebOption>>({})

const { pending, error, refresh } = await useAsyncData('admin-web-options', async () => {
  const envelope = await fetchAdminEnvelope()
  applyAdminOptions(envelope.data)
  return envelope.data
})

useSeoMeta({
  title: t('admin.settings.metaTitle')
})

// 基础配置对比与重置
const initialSiteName = computed(() => adminOptionsMap.value['site.name']?.value || 'SForum')
const initialSiteUrl = computed(() => adminOptionsMap.value['site.url']?.value || 'http://127.0.0.1:3000')
const initialDefaultLocale = computed(() => adminOptionsMap.value['site.default_locale']?.value || 'zh-CN')
const initialSupportedLocales = computed(() => parseLocaleList(adminOptionsMap.value['site.supported_locales']?.value || 'zh-CN,en-US'))
const initialTimezone = computed(() => normalizeSiteTimezone(adminOptionsMap.value['site.timezone']?.value))
const initialDateFormat = computed(() => normalizeSiteDateFormat(adminOptionsMap.value['site.date_format']?.value))
const initialTimeFormat = computed(() => normalizeSiteTimeFormat(adminOptionsMap.value['site.time_format']?.value))
const initialStartOfWeek = computed(() => normalizeSiteStartOfWeek(adminOptionsMap.value['site.start_of_week']?.value))
const initialPasswordMinLength = computed(() => boundedInteger(adminOptionsMap.value['identity.password.min_length']?.value, recommendedPasswordPolicy.minLength, 8, 128))
const initialPasswordMaxLength = computed(() => boundedInteger(adminOptionsMap.value['identity.password.max_length']?.value, recommendedPasswordPolicy.maxLength, 64, 512))
const initialPasswordRequireLowercase = computed(() => normalizeEnabledOption(adminOptionsMap.value['identity.password.require_lowercase']?.value, recommendedPasswordPolicy.requireLowercase))
const initialPasswordRequireUppercase = computed(() => normalizeEnabledOption(adminOptionsMap.value['identity.password.require_uppercase']?.value, recommendedPasswordPolicy.requireUppercase))
const initialPasswordRequireNumber = computed(() => normalizeEnabledOption(adminOptionsMap.value['identity.password.require_number']?.value, recommendedPasswordPolicy.requireNumber))
const initialPasswordRequireSymbol = computed(() => normalizeEnabledOption(adminOptionsMap.value['identity.password.require_symbol']?.value, recommendedPasswordPolicy.requireSymbol))
const initialSessionsMaxDevices = computed(() => boundedInteger(adminOptionsMap.value['identity.sessions.max_devices']?.value, 5, 1, 20))
const initialSessionsKeepDays = computed(() => boundedInteger(adminOptionsMap.value['identity.sessions.keep_days']?.value, 30, 1, 365))

const hasBasicChanges = computed(() => {
  return form.siteName !== initialSiteName.value ||
         form.siteUrl !== initialSiteUrl.value ||
         form.defaultLocale !== initialDefaultLocale.value ||
         JSON.stringify(form.supportedLocales) !== JSON.stringify(initialSupportedLocales.value) ||
         form.timezone !== initialTimezone.value ||
         form.dateFormat !== initialDateFormat.value ||
         form.timeFormat !== initialTimeFormat.value ||
         form.startOfWeek !== initialStartOfWeek.value
})

// 账号安全(密码策略)变更检测,独立于基础信息 tab
const hasAccountSecurityChanges = computed(() => {
  return form.passwordMinLength !== initialPasswordMinLength.value ||
         form.passwordMaxLength !== initialPasswordMaxLength.value ||
         form.passwordRequireLowercase !== initialPasswordRequireLowercase.value ||
         form.passwordRequireUppercase !== initialPasswordRequireUppercase.value ||
         form.passwordRequireNumber !== initialPasswordRequireNumber.value ||
         form.passwordRequireSymbol !== initialPasswordRequireSymbol.value ||
         form.sessionsMaxDevices !== initialSessionsMaxDevices.value ||
         form.sessionsKeepDays !== initialSessionsKeepDays.value
})

// 验证配置对比与重置
const initialProvider = computed(() => normalizeProvider(adminOptionsMap.value['human_verification.provider']?.value))
const initialScenarioSettings = computed(() => readScenarioSettings(adminOptionsMap.value))
const initialChallengeTTLMinutes = computed(() => durationToMinutes(adminOptionsMap.value['human_verification.altcha.challenge_ttl']?.value || '10m'))
const initialCost = computed(() => Number(adminOptionsMap.value['human_verification.altcha.cost']?.value || 1000))
const initialAltchaWidgetType = computed(() => normalizeAltchaWidgetType(adminOptionsMap.value['human_verification.altcha.widget.type']?.value))
const initialAltchaWidgetAuto = computed(() => normalizeAltchaWidgetAuto(adminOptionsMap.value['human_verification.altcha.widget.auto']?.value))
const initialAltchaWidgetDisplay = computed(() => normalizeAltchaWidgetDisplay(adminOptionsMap.value['human_verification.altcha.widget.display']?.value))
const initialAltchaWidgetHideLogo = computed(() => normalizeEnabledOption(adminOptionsMap.value['human_verification.altcha.widget.hide_logo']?.value, true))
const initialAltchaWidgetHideFooter = computed(() => normalizeEnabledOption(adminOptionsMap.value['human_verification.altcha.widget.hide_footer']?.value, true))
const initialAltchaWidgetWorkers = computed(() => boundedInteger(adminOptionsMap.value['human_verification.altcha.widget.workers']?.value, 2, 1, 16))
const initialAltchaWidgetMinDuration = computed(() => boundedInteger(adminOptionsMap.value['human_verification.altcha.widget.min_duration_ms']?.value, 500, 0, 10000))

const hasVerificationChanges = computed(() => {
  return form.humanVerificationProvider !== initialProvider.value ||
         JSON.stringify(form.humanVerificationScenarios) !== JSON.stringify(initialScenarioSettings.value) ||
         form.altchaChallengeTTLMinutes !== initialChallengeTTLMinutes.value ||
         form.altchaCost !== initialCost.value ||
         form.altchaWidgetType !== initialAltchaWidgetType.value ||
         form.altchaWidgetAuto !== initialAltchaWidgetAuto.value ||
         form.altchaWidgetDisplay !== initialAltchaWidgetDisplay.value ||
         form.altchaWidgetHideLogo !== initialAltchaWidgetHideLogo.value ||
         form.altchaWidgetHideFooter !== initialAltchaWidgetHideFooter.value ||
         form.altchaWidgetWorkers !== initialAltchaWidgetWorkers.value ||
         form.altchaWidgetMinDuration !== initialAltchaWidgetMinDuration.value ||
         form.altchaSecret.trim() !== ''
})

function applyAdminOptions(items: AdminWebOption[]) {
  const publicOptions = items.filter((item) => item.public && !item.secret)
  options.value = {
    ...options.value,
    ...Object.fromEntries(publicOptions.map((item) => [item.name, item.value]))
  }

  const map = Object.fromEntries(items.map((item) => [item.name, item]))
  adminOptionsMap.value = map

  form.siteName = map['site.name']?.value || 'SForum'
  form.siteUrl = map['site.url']?.value || 'http://127.0.0.1:3000'
  form.defaultLocale = map['site.default_locale']?.value || 'zh-CN'
  form.supportedLocales = parseLocaleList(map['site.supported_locales']?.value || 'zh-CN,en-US')
  if (!form.supportedLocales.includes(form.defaultLocale)) {
    form.defaultLocale = form.supportedLocales[0] || 'zh-CN'
  }
  form.timezone = normalizeSiteTimezone(map['site.timezone']?.value)
  form.dateFormat = normalizeSiteDateFormat(map['site.date_format']?.value)
  form.timeFormat = normalizeSiteTimeFormat(map['site.time_format']?.value)
  form.startOfWeek = normalizeSiteStartOfWeek(map['site.start_of_week']?.value)
  form.passwordMinLength = boundedInteger(map['identity.password.min_length']?.value, recommendedPasswordPolicy.minLength, 8, 128)
  form.passwordMaxLength = boundedInteger(map['identity.password.max_length']?.value, recommendedPasswordPolicy.maxLength, 64, 512)
  form.passwordRequireLowercase = normalizeEnabledOption(map['identity.password.require_lowercase']?.value, recommendedPasswordPolicy.requireLowercase)
  form.passwordRequireUppercase = normalizeEnabledOption(map['identity.password.require_uppercase']?.value, recommendedPasswordPolicy.requireUppercase)
  form.passwordRequireNumber = normalizeEnabledOption(map['identity.password.require_number']?.value, recommendedPasswordPolicy.requireNumber)
  form.passwordRequireSymbol = normalizeEnabledOption(map['identity.password.require_symbol']?.value, recommendedPasswordPolicy.requireSymbol)
  form.sessionsMaxDevices = boundedInteger(map['identity.sessions.max_devices']?.value, 5, 1, 20)
  form.sessionsKeepDays = boundedInteger(map['identity.sessions.keep_days']?.value, 30, 1, 365)
  form.humanVerificationProvider = normalizeProvider(map['human_verification.provider']?.value)
  form.humanVerificationScenarios = readScenarioSettings(map)
  form.altchaSecret = ''
  form.altchaSecretSet = map['human_verification.altcha.secret']?.secretSet === true
  form.altchaChallengeTTLMinutes = durationToMinutes(map['human_verification.altcha.challenge_ttl']?.value || '10m')
  form.altchaCost = Number(map['human_verification.altcha.cost']?.value || 1000)
  form.altchaWidgetType = normalizeAltchaWidgetType(map['human_verification.altcha.widget.type']?.value)
  form.altchaWidgetAuto = normalizeAltchaWidgetAuto(map['human_verification.altcha.widget.auto']?.value)
  form.altchaWidgetDisplay = normalizeAltchaWidgetDisplay(map['human_verification.altcha.widget.display']?.value)
  form.altchaWidgetHideLogo = normalizeEnabledOption(map['human_verification.altcha.widget.hide_logo']?.value, true)
  form.altchaWidgetHideFooter = normalizeEnabledOption(map['human_verification.altcha.widget.hide_footer']?.value, true)
  form.altchaWidgetWorkers = boundedInteger(map['human_verification.altcha.widget.workers']?.value, 2, 1, 16)
  form.altchaWidgetMinDuration = boundedInteger(map['human_verification.altcha.widget.min_duration_ms']?.value, 500, 0, 10000)
}

async function saveBasicSettings() {
  savingBasic.value = true
  try {
    await saveAndApply([
      { name: 'site.name', value: form.siteName },
      { name: 'site.url', value: form.siteUrl },
      { name: 'site.default_locale', value: form.defaultLocale },
      { name: 'site.supported_locales', value: form.supportedLocales.join(',') },
      { name: 'site.timezone', value: form.timezone },
      { name: 'site.date_format', value: form.dateFormat },
      { name: 'site.time_format', value: form.timeFormat },
      { name: 'site.start_of_week', value: String(form.startOfWeek) }
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

// 账号安全(密码策略)独立保存,提交 identity.password.* 选项
async function saveAccountSecuritySettings() {
  form.passwordMinLength = boundedInteger(form.passwordMinLength, recommendedPasswordPolicy.minLength, 8, 128)
  form.passwordMaxLength = boundedInteger(form.passwordMaxLength, recommendedPasswordPolicy.maxLength, 64, 512)
  if (form.passwordMaxLength < form.passwordMinLength) {
    form.passwordMaxLength = form.passwordMinLength
  }
  // 最大活跃设备数 clamp 到 1-20。
  form.sessionsMaxDevices = boundedInteger(form.sessionsMaxDevices, 5, 1, 20)
  form.sessionsKeepDays = boundedInteger(form.sessionsKeepDays, 30, 1, 365)
  savingAccountSecurity.value = true
  try {
    await saveAndApply([
      { name: 'identity.password.min_length', value: String(form.passwordMinLength) },
      { name: 'identity.password.max_length', value: String(form.passwordMaxLength) },
      { name: 'identity.password.require_lowercase', value: enabledOptionValue(form.passwordRequireLowercase) },
      { name: 'identity.password.require_uppercase', value: enabledOptionValue(form.passwordRequireUppercase) },
      { name: 'identity.password.require_number', value: enabledOptionValue(form.passwordRequireNumber) },
      { name: 'identity.password.require_symbol', value: enabledOptionValue(form.passwordRequireSymbol) },
      { name: 'identity.sessions.max_devices', value: String(form.sessionsMaxDevices) },
      { name: 'identity.sessions.keep_days', value: String(form.sessionsKeepDays) }
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
    savingAccountSecurity.value = false
  }
}

async function saveVerificationSettings() {
  form.altchaChallengeTTLMinutes = positiveInteger(form.altchaChallengeTTLMinutes, 10)
  form.altchaCost = positiveInteger(form.altchaCost, 1000)
  form.altchaWidgetWorkers = boundedInteger(form.altchaWidgetWorkers, 2, 1, 16)
  form.altchaWidgetMinDuration = boundedInteger(form.altchaWidgetMinDuration, 500, 0, 10000)
  savingVerification.value = true
  try {
    const payload: WebOption[] = [
      { name: 'human_verification.provider', value: form.humanVerificationProvider },
      ...verificationScenarios.value.map((scenario) => ({
        name: humanVerificationScenarioOptionName(scenario.key),
        value: enabledOptionValue(form.humanVerificationScenarios[scenario.key])
      })),
      { name: 'human_verification.altcha.challenge_ttl', value: `${form.altchaChallengeTTLMinutes}m` },
      { name: 'human_verification.altcha.cost', value: String(form.altchaCost) },
      { name: 'human_verification.altcha.widget.type', value: form.altchaWidgetType },
      { name: 'human_verification.altcha.widget.auto', value: form.altchaWidgetAuto },
      { name: 'human_verification.altcha.widget.display', value: form.altchaWidgetDisplay },
      { name: 'human_verification.altcha.widget.hide_logo', value: enabledOptionValue(form.altchaWidgetHideLogo) },
      { name: 'human_verification.altcha.widget.hide_footer', value: enabledOptionValue(form.altchaWidgetHideFooter) },
      { name: 'human_verification.altcha.widget.workers', value: String(form.altchaWidgetWorkers) },
      { name: 'human_verification.altcha.widget.min_duration_ms', value: String(form.altchaWidgetMinDuration) }
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

function resetBasicForm() {
  form.siteName = initialSiteName.value
  form.siteUrl = initialSiteUrl.value
  form.defaultLocale = initialDefaultLocale.value
  form.supportedLocales = [...initialSupportedLocales.value]
  form.timezone = initialTimezone.value
  form.dateFormat = initialDateFormat.value
  form.timeFormat = initialTimeFormat.value
  form.startOfWeek = initialStartOfWeek.value
  toast.add({
    color: 'neutral',
    icon: 'i-lucide-rotate-ccw',
    title: t('admin.settings.basic.resetChanges')
  })
}

// 一键恢复时区/日期时间推荐默认值（不改站点名与语言）。
function restoreRecommendedDateTimeSettings() {
  form.timezone = recommendedSiteDateTimeSettings.timezone
  form.dateFormat = recommendedSiteDateTimeSettings.dateFormat
  form.timeFormat = recommendedSiteDateTimeSettings.timeFormat
  form.startOfWeek = recommendedSiteDateTimeSettings.startOfWeek
  toast.add({
    color: 'success',
    icon: 'i-lucide-rotate-ccw',
    title: t('admin.settings.basic.restoreDateTimeDefaults')
  })
}

// 账号安全(密码策略)独立重置
function resetAccountSecurityForm() {
  form.passwordMinLength = initialPasswordMinLength.value
  form.passwordMaxLength = initialPasswordMaxLength.value
  form.passwordRequireLowercase = initialPasswordRequireLowercase.value
  form.passwordRequireUppercase = initialPasswordRequireUppercase.value
  form.passwordRequireNumber = initialPasswordRequireNumber.value
  form.passwordRequireSymbol = initialPasswordRequireSymbol.value
  form.sessionsMaxDevices = initialSessionsMaxDevices.value
  form.sessionsKeepDays = initialSessionsKeepDays.value
  toast.add({
    color: 'neutral',
    icon: 'i-lucide-rotate-ccw',
    title: '已重置账号安全更改'
  })
}

function restoreRecommendedPasswordPolicy() {
  form.passwordMinLength = recommendedPasswordPolicy.minLength
  form.passwordMaxLength = recommendedPasswordPolicy.maxLength
  form.passwordRequireLowercase = recommendedPasswordPolicy.requireLowercase
  form.passwordRequireUppercase = recommendedPasswordPolicy.requireUppercase
  form.passwordRequireNumber = recommendedPasswordPolicy.requireNumber
  form.passwordRequireSymbol = recommendedPasswordPolicy.requireSymbol
  // 同时恢复最大活跃设备数到推荐默认。
  form.sessionsMaxDevices = 5
  form.sessionsKeepDays = 30
  toast.add({
    color: 'neutral',
    icon: 'i-lucide-rotate-ccw',
    title: t('admin.settings.basic.restorePasswordDefaults')
  })
}

function resetVerificationForm() {
  form.humanVerificationProvider = initialProvider.value
  form.humanVerificationScenarios = { ...initialScenarioSettings.value }
  form.altchaChallengeTTLMinutes = initialChallengeTTLMinutes.value
  form.altchaCost = initialCost.value
  form.altchaWidgetType = initialAltchaWidgetType.value
  form.altchaWidgetAuto = initialAltchaWidgetAuto.value
  form.altchaWidgetDisplay = initialAltchaWidgetDisplay.value
  form.altchaWidgetHideLogo = initialAltchaWidgetHideLogo.value
  form.altchaWidgetHideFooter = initialAltchaWidgetHideFooter.value
  form.altchaWidgetWorkers = initialAltchaWidgetWorkers.value
  form.altchaWidgetMinDuration = initialAltchaWidgetMinDuration.value
  form.altchaSecret = ''
  toast.add({
    color: 'neutral',
    icon: 'i-lucide-rotate-ccw',
    title: '已重置验证设置更改'
  })
}

function parseLocaleList(value: string) {
  const locales = value.split(',').map((item) => item.trim()).filter(Boolean)
  return locales.length > 0 ? locales : ['zh-CN', 'en-US']
}

function readScenarioSettings(map: Record<string, AdminWebOption>) {
  return Object.fromEntries(
    (['register', 'password_reset', 'login_risk', 'post_risk'] as HumanVerificationScenario[]).map((scenario) => [
      scenario,
      normalizeEnabledOption(
        map[humanVerificationScenarioOptionName(scenario)]?.value,
        scenarioFallbacks[scenario]
      )
    ])
  ) as Record<HumanVerificationScenario, boolean>
}

function durationToMinutes(value: string) {
  const raw = value.trim().toLowerCase()
  if (/^\d+$/.test(raw)) {
    return positiveInteger(Number(raw), 10)
  }

  const hours = Number(raw.match(/(\d+)h/)?.[1] || 0)
  const minutes = Number(raw.match(/(\d+)m/)?.[1] || 0)
  const seconds = Number(raw.match(/(\d+)s/)?.[1] || 0)
  const totalSeconds = hours * 3600 + minutes * 60 + seconds
  if (totalSeconds > 0) {
    return Math.max(1, Math.ceil(totalSeconds / 60))
  }
  return 10
}

function positiveInteger(value: unknown, fallback: number) {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed > 0 ? Math.trunc(parsed) : fallback
}

function boundedInteger(value: unknown, fallback: number, min: number, max: number) {
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) {
    return fallback
  }
  const normalized = Math.trunc(parsed)
  return normalized >= min && normalized <= max ? normalized : fallback
}

function normalizeProvider(value: string | undefined) {
  return value?.trim().toLowerCase() === 'altcha' ? 'altcha' : 'disabled'
}

function normalizeAltchaWidgetType(value: string | undefined): AltchaWidgetType {
  return normalizeChoice(value, altchaWidgetTypes, 'checkbox')
}

function normalizeAltchaWidgetAuto(value: string | undefined): AltchaWidgetAuto {
  return normalizeChoice(value, altchaWidgetAutoModes, 'off')
}

function normalizeAltchaWidgetDisplay(value: string | undefined): AltchaWidgetDisplay {
  return normalizeChoice(value, altchaWidgetDisplays, 'standard')
}

function normalizeChoice<T extends string>(value: string | undefined, choices: readonly T[], fallback: T): T {
  const normalized = value?.trim().toLowerCase()
  return choices.find((choice) => choice === normalized) || fallback
}

function setChallengeTTL(minutes: number) {
  form.altchaChallengeTTLMinutes = minutes
}

function setAltchaCost(cost: number) {
  form.altchaCost = cost
}

function setAltchaWidgetWorkers(workers: number) {
  form.altchaWidgetWorkers = workers
}

function setAltchaWidgetMinDuration(duration: number) {
  form.altchaWidgetMinDuration = duration
}

function toggleAltchaSecretVisibility() {
  showAltchaSecret.value = !showAltchaSecret.value
}

function toggleVerificationScenario(scenario: HumanVerificationScenario) {
  form.humanVerificationScenarios[scenario] = !form.humanVerificationScenarios[scenario]
}

function blockNonIntegerKey(event: KeyboardEvent) {
  const allowedKeys = ['Backspace', 'Delete', 'Tab', 'Escape', 'Enter', 'ArrowLeft', 'ArrowRight', 'Home', 'End']
  if (allowedKeys.includes(event.key) || event.metaKey || event.ctrlKey) {
    return
  }
  if (!/^\d$/.test(event.key)) {
    event.preventDefault()
  }
}

function generateAltchaSecret() {
  if (!globalThis.crypto?.getRandomValues) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: t('admin.settings.verification.secretGenerateUnavailable')
    })
    return
  }

  const bytes = new Uint8Array(32)
  globalThis.crypto.getRandomValues(bytes)
  form.altchaSecret = Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('')
  toast.add({
    color: 'success',
    icon: 'i-lucide-key-round',
    title: t('admin.settings.verification.secretGenerated')
  })
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
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
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

    <!-- 基础配置 Tab -->
    <form v-if="activeTab === 'basic'" class="flex flex-col" @submit.prevent="saveBasicSettings">
      <UCard
        class="border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100"
        :ui="{ footer: 'sticky bottom-0 z-20 bg-white/95 dark:bg-zinc-900/95 backdrop-blur-sm border-t border-slate-200 dark:border-zinc-800 p-4 sm:px-6' }"
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

        <div class="grid max-w-3xl gap-4">
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
              class="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-900 outline-none transition focus:border-[var(--sf-accent)] focus:ring-2 focus:ring-[var(--sf-accent-focus)] dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100"
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
                  class="size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]"
                  :checked="form.supportedLocales.includes(choice.value)"
                  @change="onLocaleToggle(choice.value, $event)"
                />
                <span>{{ choice.label }}</span>
              </label>
            </div>
          </UFormField>

          <div class="border-t border-slate-200 pt-4 dark:border-zinc-800">
            <div class="mb-3 flex flex-wrap items-start justify-between gap-3">
              <div>
                <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                  {{ t('admin.settings.basic.datetimeTitle') }}
                </h3>
                <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
                  {{ t('admin.settings.basic.datetimeDescription') }}
                </p>
              </div>
              <UButton
                type="button"
                color="neutral"
                variant="outline"
                leading-icon="i-lucide-rotate-ccw"
                class="shrink-0"
                @click="restoreRecommendedDateTimeSettings"
              >
                {{ t('admin.settings.basic.restoreDateTimeDefaults') }}
              </UButton>
            </div>

            <div class="grid gap-4 md:grid-cols-2">
              <UFormField :label="t('admin.settings.basic.timezone')" name="site-timezone" class="md:col-span-2">
                <select
                  v-model="form.timezone"
                  class="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-900 outline-none transition focus:border-[var(--sf-accent)] focus:ring-2 focus:ring-[var(--sf-accent-focus)] dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100"
                >
                  <option
                    v-for="choice in timezoneChoices"
                    :key="choice.value"
                    :value="choice.value"
                  >
                    {{ choice.label }}
                  </option>
                </select>
                <template #hint>
                  {{ t('admin.settings.basic.timezoneHint') }}
                </template>
              </UFormField>

              <UFormField :label="t('admin.settings.basic.dateFormat')" name="site-date-format">
                <select
                  v-model="form.dateFormat"
                  class="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-900 outline-none transition focus:border-[var(--sf-accent)] focus:ring-2 focus:ring-[var(--sf-accent-focus)] dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100"
                >
                  <option
                    v-for="choice in dateFormatChoices"
                    :key="choice.value"
                    :value="choice.value"
                  >
                    {{ choice.label }}
                  </option>
                </select>
              </UFormField>

              <UFormField :label="t('admin.settings.basic.timeFormat')" name="site-time-format">
                <select
                  v-model="form.timeFormat"
                  class="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-900 outline-none transition focus:border-[var(--sf-accent)] focus:ring-2 focus:ring-[var(--sf-accent-focus)] dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100"
                >
                  <option
                    v-for="choice in timeFormatChoices"
                    :key="choice.value"
                    :value="choice.value"
                  >
                    {{ choice.label }}
                  </option>
                </select>
              </UFormField>

              <UFormField :label="t('admin.settings.basic.startOfWeek')" name="site-start-of-week" class="md:col-span-2">
                <select
                  v-model.number="form.startOfWeek"
                  class="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-900 outline-none transition focus:border-[var(--sf-accent)] focus:ring-2 focus:ring-[var(--sf-accent-focus)] dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100"
                >
                  <option
                    v-for="choice in startOfWeekChoices"
                    :key="choice.value"
                    :value="choice.value"
                  >
                    {{ choice.label }}
                  </option>
                </select>
                <template #hint>
                  {{ t('admin.settings.basic.startOfWeekHint') }}
                </template>
              </UFormField>
            </div>

            <UAlert
              class="mt-4"
              color="neutral"
              variant="soft"
              icon="i-lucide-clock-3"
              :title="t('admin.settings.basic.datetimePreviewTitle')"
              :description="datetimePreview"
            />
          </div>
        </div>

        <template #footer>
          <SFAdminFormFooter
            :saving="savingBasic"
            :show-unsaved-alert="hasBasicChanges"
            :submit-text="t('admin.settings.save')"
            @reset="resetBasicForm"
          />
        </template>
      </UCard>
    </form>

    <!-- 账号安全(密码策略)Tab -->
    <form v-else-if="activeTab === 'accountSecurity'" class="flex flex-col" @submit.prevent="saveAccountSecuritySettings">
      <UCard
        class="border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100"
        :ui="{ footer: 'sticky bottom-0 z-20 bg-white/95 dark:bg-zinc-900/95 backdrop-blur-sm border-t border-slate-200 dark:border-zinc-800 p-4 sm:px-6' }"
      >
        <template #header>
          <div class="flex items-center justify-between gap-3">
            <div>
              <h2 class="text-base font-bold text-slate-900 dark:text-white">
                {{ t('admin.settings.basic.accountSecurityTitle') }}
              </h2>
              <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
                {{ t('admin.settings.basic.accountSecurityDescription') }}
              </p>
            </div>
            <UBadge color="neutral" variant="soft" class="border border-slate-200 dark:border-zinc-800 font-mono">
              identity.password.*
            </UBadge>
          </div>
        </template>

        <div class="space-y-4">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <UAlert
              color="neutral"
              variant="soft"
              icon="i-lucide-info"
              :title="t('admin.settings.basic.passwordRecommended')"
              class="flex-1"
            />
            <UButton
              type="button"
              color="neutral"
              variant="outline"
              leading-icon="i-lucide-rotate-ccw"
              class="shrink-0"
              @click="restoreRecommendedPasswordPolicy"
            >
              {{ t('admin.settings.basic.restorePasswordDefaults') }}
            </UButton>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="t('admin.settings.basic.passwordMinLength')" name="password-min-length">
              <UInput
                v-model.number="form.passwordMinLength"
                icon="i-lucide-ruler"
                type="number"
                inputmode="numeric"
                min="8"
                max="128"
                step="1"
                required
                class="w-full"
                @keydown="blockNonIntegerKey"
              />
            </UFormField>

            <UFormField :label="t('admin.settings.basic.passwordMaxLength')" name="password-max-length">
              <UInput
                v-model.number="form.passwordMaxLength"
                icon="i-lucide-ruler"
                type="number"
                inputmode="numeric"
                min="64"
                max="512"
                step="1"
                required
                class="w-full"
                @keydown="blockNonIntegerKey"
              />
            </UFormField>
          </div>

          <div class="grid gap-3 md:grid-cols-2">
            <label class="flex cursor-pointer items-start gap-3 rounded-md border border-slate-200 bg-slate-50 px-3 py-3 text-sm transition hover:border-[var(--sf-accent-soft-border)] dark:border-zinc-800 dark:bg-zinc-950/60">
              <input
                v-model="form.passwordRequireLowercase"
                type="checkbox"
                class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]"
              />
              <span class="font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.settings.basic.passwordRequireLowercase') }}
              </span>
            </label>

            <label class="flex cursor-pointer items-start gap-3 rounded-md border border-slate-200 bg-slate-50 px-3 py-3 text-sm transition hover:border-[var(--sf-accent-soft-border)] dark:border-zinc-800 dark:bg-zinc-950/60">
              <input
                v-model="form.passwordRequireUppercase"
                type="checkbox"
                class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]"
              />
              <span class="font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.settings.basic.passwordRequireUppercase') }}
              </span>
            </label>

            <label class="flex cursor-pointer items-start gap-3 rounded-md border border-slate-200 bg-slate-50 px-3 py-3 text-sm transition hover:border-[var(--sf-accent-soft-border)] dark:border-zinc-800 dark:bg-zinc-950/60">
              <input
                v-model="form.passwordRequireNumber"
                type="checkbox"
                class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]"
              />
              <span class="font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.settings.basic.passwordRequireNumber') }}
              </span>
            </label>

            <label class="flex cursor-pointer items-start gap-3 rounded-md border border-slate-200 bg-slate-50 px-3 py-3 text-sm transition hover:border-[var(--sf-accent-soft-border)] dark:border-zinc-800 dark:bg-zinc-950/60">
              <input
                v-model="form.passwordRequireSymbol"
                type="checkbox"
                class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]"
              />
              <span class="font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.settings.basic.passwordRequireSymbol') }}
              </span>
            </label>
          </div>

          <!-- 最大活跃设备数：超出时登录会自动下线最旧设备 -->
          <UFormField
            :label="t('admin.settings.basic.sessionsMaxDevices')"
            :description="t('admin.settings.basic.sessionsMaxDevicesHint')"
            name="sessions-max-devices"
            class="pt-2"
          >
            <UInput
              v-model.number="form.sessionsMaxDevices"
              icon="i-lucide-devices"
              type="number"
              inputmode="numeric"
              min="1"
              max="20"
              step="1"
              required
              class="w-full max-w-xs"
              @keydown="blockNonIntegerKey"
            />
          </UFormField>

          <!-- 历史会话保留天数：超过后由后台定期任务清理 -->
          <UFormField
            :label="t('admin.settings.basic.sessionsKeepDays')"
            :description="t('admin.settings.basic.sessionsKeepDaysHint')"
            name="sessions-keep-days"
          >
            <UInput
              v-model.number="form.sessionsKeepDays"
              icon="i-lucide-calendar-clock"
              type="number"
              inputmode="numeric"
              min="1"
              max="365"
              step="1"
              required
              class="w-full max-w-xs"
              @keydown="blockNonIntegerKey"
            />
          </UFormField>
        </div>

        <template #footer>
          <SFAdminFormFooter
            :saving="savingAccountSecurity"
            :show-unsaved-alert="hasAccountSecurityChanges"
            :submit-text="t('admin.settings.save')"
            @reset="resetAccountSecurityForm"
          />
        </template>
      </UCard>
    </form>

    <!-- 人机验证 Tab -->
    <form v-else class="flex flex-col" @submit.prevent="saveVerificationSettings">
      <UCard
        class="border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100"
        :ui="{ footer: 'sticky bottom-0 z-20 bg-white/95 dark:bg-zinc-900/95 backdrop-blur-sm border-t border-slate-200 dark:border-zinc-800 p-4 sm:px-6' }"
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

        <div class="grid max-w-5xl gap-6">
          <UFormField :label="t('admin.settings.verification.provider')" name="verification-provider">
            <select
              v-model="form.humanVerificationProvider"
              class="h-10 w-full max-w-xl rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-900 outline-none transition focus:border-[var(--sf-accent)] focus:ring-2 focus:ring-[var(--sf-accent-focus)] dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100"
            >
              <option value="disabled">{{ t('admin.settings.verification.disabled') }}</option>
              <option value="altcha">{{ t('admin.settings.verification.altcha') }}</option>
            </select>
          </UFormField>

          <section class="space-y-3 border-t border-slate-200 pt-4 dark:border-zinc-800">
            <div>
              <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.settings.verification.scenarios.title') }}
              </h3>
              <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
                {{ t('admin.settings.verification.scenarios.description') }}
              </p>
            </div>
            <div class="grid gap-3 md:grid-cols-2">
              <label
                v-for="scenario in verificationScenarios"
                :key="scenario.key"
                class="flex cursor-pointer gap-3 rounded-md border border-slate-200 bg-slate-50 px-3 py-3 text-sm transition hover:border-[var(--sf-accent-soft-border)] dark:border-zinc-800 dark:bg-zinc-950/60"
              >
                <input
                  type="checkbox"
                  class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]"
                  :checked="form.humanVerificationScenarios[scenario.key]"
                  @change="toggleVerificationScenario(scenario.key)"
                />
                <span class="min-w-0">
                  <span class="flex items-center gap-2 font-semibold text-slate-900 dark:text-zinc-100">
                    <UIcon :name="scenario.icon" class="size-4 text-[var(--sf-accent)]" />
                    {{ scenario.label }}
                  </span>
                  <span class="mt-1 block text-xs leading-5 text-slate-500 dark:text-zinc-400">
                    {{ scenario.description }}
                  </span>
                </span>
              </label>
            </div>
          </section>

          <section class="border-t border-slate-200 pt-4 dark:border-zinc-800">
            <UFormField :label="t('admin.settings.verification.altchaSecret')" name="altcha-secret">
              <div class="flex flex-wrap items-center gap-2">
                <UInput
                  v-model="form.altchaSecret"
                  icon="i-lucide-key-round"
                  :type="showAltchaSecret ? 'text' : 'password'"
                  :placeholder="altchaSecretPlaceholder"
                  class="flex-1 min-w-[200px]"
                />
                <UButton
                  type="button"
                  color="neutral"
                  variant="outline"
                  :icon="showAltchaSecret ? 'i-lucide-eye-off' : 'i-lucide-eye'"
                  :aria-label="showAltchaSecret ? t('admin.settings.verification.hideSecret') : t('admin.settings.verification.showSecret')"
                  @click="() => { toggleAltchaSecretVisibility() }"
                />
                <UButton
                  type="button"
                  color="neutral"
                  variant="outline"
                  leading-icon="i-lucide-key-round"
                  @click="generateAltchaSecret"
                >
                  {{ t('admin.settings.verification.generateSecret') }}
                </UButton>
                <UBadge
                  :color="form.altchaSecretSet ? 'success' : 'neutral'"
                  variant="soft"
                  class="h-9 border border-slate-200 px-3 dark:border-zinc-800"
                >
                  {{ form.altchaSecretSet ? t('admin.settings.verification.secretConfigured') : t('admin.settings.verification.secretMissing') }}
                </UBadge>
              </div>
              <p class="mt-2 text-xs leading-5 text-slate-500 dark:text-zinc-400">
                {{ t('admin.settings.verification.secretHint') }}
              </p>
            </UFormField>
          </section>

          <section class="space-y-4 border-t border-slate-200 pt-4 dark:border-zinc-800">
            <div>
              <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.settings.verification.widget.title') }}
              </h3>
              <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
                {{ t('admin.settings.verification.widget.description') }}
              </p>
            </div>

            <div class="grid gap-4 md:grid-cols-3">
              <UFormField :label="t('admin.settings.verification.widget.type')" name="altcha-widget-type">
                <select
                  v-model="form.altchaWidgetType"
                  class="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-900 outline-none transition focus:border-[var(--sf-accent)] focus:ring-2 focus:ring-[var(--sf-accent-focus)] dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100"
                >
                  <option v-for="item in altchaTypeOptions" :key="item.value" :value="item.value">
                    {{ item.label }}
                  </option>
                </select>
                <p class="mt-2 text-xs leading-5 text-slate-500 dark:text-zinc-400">
                  {{ t('admin.settings.verification.widget.typeHint') }}
                </p>
              </UFormField>

              <UFormField :label="t('admin.settings.verification.widget.auto')" name="altcha-widget-auto">
                <select
                  v-model="form.altchaWidgetAuto"
                  class="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-900 outline-none transition focus:border-[var(--sf-accent)] focus:ring-2 focus:ring-[var(--sf-accent-focus)] dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100"
                >
                  <option v-for="item in altchaAutoOptions" :key="item.value" :value="item.value">
                    {{ item.label }}
                  </option>
                </select>
                <p class="mt-2 text-xs leading-5 text-slate-500 dark:text-zinc-400">
                  {{ t('admin.settings.verification.widget.autoHint') }}
                </p>
              </UFormField>

              <UFormField :label="t('admin.settings.verification.widget.display')" name="altcha-widget-display">
                <select
                  v-model="form.altchaWidgetDisplay"
                  class="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-900 outline-none transition focus:border-[var(--sf-accent)] focus:ring-2 focus:ring-[var(--sf-accent-focus)] dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100"
                >
                  <option v-for="item in altchaDisplayOptions" :key="item.value" :value="item.value">
                    {{ item.label }}
                  </option>
                </select>
                <p class="mt-2 text-xs leading-5 text-slate-500 dark:text-zinc-400">
                  {{ t('admin.settings.verification.widget.displayHint') }}
                </p>
              </UFormField>
            </div>

            <div class="grid gap-3 md:grid-cols-2">
              <label class="flex cursor-pointer items-start gap-3 rounded-md border border-slate-200 bg-slate-50 px-3 py-3 text-sm transition hover:border-[var(--sf-accent-soft-border)] dark:border-zinc-800 dark:bg-zinc-950/60">
                <input
                  v-model="form.altchaWidgetHideLogo"
                  type="checkbox"
                  class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]"
                />
                <span>
                  <span class="font-semibold text-slate-900 dark:text-zinc-100">
                    {{ t('admin.settings.verification.widget.hideLogo') }}
                  </span>
                  <span class="mt-1 block text-xs leading-5 text-slate-500 dark:text-zinc-400">
                    {{ t('admin.settings.verification.widget.hideLogoHint') }}
                  </span>
                </span>
              </label>

              <label class="flex cursor-pointer items-start gap-3 rounded-md border border-slate-200 bg-slate-50 px-3 py-3 text-sm transition hover:border-[var(--sf-accent-soft-border)] dark:border-zinc-800 dark:bg-zinc-950/60">
                <input
                  v-model="form.altchaWidgetHideFooter"
                  type="checkbox"
                  class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]"
                />
                <span>
                  <span class="font-semibold text-slate-900 dark:text-zinc-100">
                    {{ t('admin.settings.verification.widget.hideFooter') }}
                  </span>
                  <span class="mt-1 block text-xs leading-5 text-slate-500 dark:text-zinc-400">
                    {{ t('admin.settings.verification.widget.hideFooterHint') }}
                  </span>
                </span>
              </label>
            </div>

            <div class="grid gap-4 md:grid-cols-2">
              <UFormField :label="t('admin.settings.verification.widget.workers')" name="altcha-widget-workers">
                <UInput
                  v-model.number="form.altchaWidgetWorkers"
                  icon="i-lucide-cpu"
                  type="number"
                  inputmode="numeric"
                  min="1"
                  max="16"
                  step="1"
                  required
                  class="w-full"
                  @keydown="blockNonIntegerKey"
                />
                <p class="mt-2 text-xs leading-5 text-slate-500 dark:text-zinc-400">
                  {{ t('admin.settings.verification.widget.workersHint') }}
                </p>
                <div class="mt-2 flex flex-wrap gap-2">
                  <UButton
                    v-for="workers in workerSuggestions"
                    :key="workers"
                    type="button"
                    size="xs"
                    color="neutral"
                    :variant="form.altchaWidgetWorkers === workers ? 'solid' : 'outline'"
                    @click="setAltchaWidgetWorkers(workers)"
                  >
                    {{ workers }}
                  </UButton>
                </div>
              </UFormField>

              <UFormField :label="t('admin.settings.verification.widget.minDuration')" name="altcha-widget-min-duration">
                <UInput
                  v-model.number="form.altchaWidgetMinDuration"
                  icon="i-lucide-timer"
                  type="number"
                  inputmode="numeric"
                  min="0"
                  max="10000"
                  step="100"
                  required
                  class="w-full"
                  @keydown="blockNonIntegerKey"
                />
                <p class="mt-2 text-xs leading-5 text-slate-500 dark:text-zinc-400">
                  {{ t('admin.settings.verification.widget.minDurationHint') }}
                </p>
                <div class="mt-2 flex flex-wrap gap-2">
                  <UButton
                    v-for="duration in minDurationSuggestions"
                    :key="duration"
                    type="button"
                    size="xs"
                    color="neutral"
                    :variant="form.altchaWidgetMinDuration === duration ? 'solid' : 'outline'"
                    @click="setAltchaWidgetMinDuration(duration)"
                  >
                    {{ t('admin.settings.verification.widget.msOption', { count: duration }) }}
                  </UButton>
                </div>
              </UFormField>
            </div>
          </section>

          <section class="grid gap-4 border-t border-slate-200 pt-4 dark:border-zinc-800 md:grid-cols-2">
            <UFormField :label="t('admin.settings.verification.challengeTTL')" name="altcha-ttl">
              <UInput
                v-model.number="form.altchaChallengeTTLMinutes"
                icon="i-lucide-clock-3"
                type="number"
                inputmode="numeric"
                min="1"
                step="1"
                placeholder="20"
                required
                class="w-full"
                @keydown="blockNonIntegerKey"
              />
              <p class="mt-2 text-xs leading-5 text-slate-500 dark:text-zinc-400">
                {{ t('admin.settings.verification.challengeTTLHint') }}
              </p>
              <div class="mt-2 flex flex-wrap gap-2">
                <UButton
                  v-for="minutes in ttlSuggestions"
                  :key="minutes"
                  type="button"
                  size="xs"
                  color="neutral"
                  :variant="form.altchaChallengeTTLMinutes === minutes ? 'solid' : 'outline'"
                  @click="setChallengeTTL(minutes)"
                >
                  {{ t('admin.settings.verification.minutesOption', { count: minutes }) }}
                </UButton>
              </div>
            </UFormField>

            <UFormField :label="t('admin.settings.verification.cost')" name="altcha-cost">
              <UInput
                v-model.number="form.altchaCost"
                icon="i-lucide-cpu"
                type="number"
                inputmode="numeric"
                min="1"
                step="100"
                required
                class="w-full"
                @keydown="blockNonIntegerKey"
              />
              <p class="mt-2 text-xs leading-5 text-slate-500 dark:text-zinc-400">
                {{ t('admin.settings.verification.costHint') }}
              </p>
              <div class="mt-2 flex flex-wrap gap-2">
                <UButton
                  v-for="cost in costSuggestions"
                  :key="cost"
                  type="button"
                  size="xs"
                  color="neutral"
                  :variant="form.altchaCost === cost ? 'solid' : 'outline'"
                  @click="setAltchaCost(cost)"
                >
                  {{ t(`admin.settings.verification.costOptions.${cost}`) }}
                </UButton>
              </div>
            </UFormField>
          </section>

          <section class="space-y-3 border-t border-slate-200 pt-4 dark:border-zinc-800">
            <div>
              <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.settings.verification.config.title') }}
              </h3>
              <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
                {{ t('admin.settings.verification.config.description') }}
              </p>
            </div>
            <dl class="grid gap-x-6 gap-y-3 text-sm md:grid-cols-2">
              <div
                v-for="row in altchaConfigRows"
                :key="row.label"
                class="grid gap-1 border-b border-slate-100 pb-2 dark:border-zinc-800"
              >
                <dt class="text-xs font-medium text-slate-500 dark:text-zinc-400">
                  {{ row.label }}
                </dt>
                <dd class="break-words font-mono text-xs text-slate-800 dark:text-zinc-200">
                  {{ row.value }}
                </dd>
              </div>
            </dl>
          </section>
        </div>

        <template #footer>
          <SFAdminFormFooter
            :saving="savingVerification"
            :show-unsaved-alert="hasVerificationChanges"
            :submit-text="t('admin.settings.save')"
            @reset="resetVerificationForm"
          />
        </template>
      </UCard>
    </form>
  </div>
</template>
