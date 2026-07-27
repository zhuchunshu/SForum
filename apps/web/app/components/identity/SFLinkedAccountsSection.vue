<script setup lang="ts">
import { useAuthProviders } from '~/composables/identity/useAuthProviders'
import { useAccountSecurityApi } from '~/composables/identity/useAccountSecurityApi'
/**
 * 账号安全：外部登录方式（redacted 列表 / link / unlink / inert）+ external-only 密码设置。
 * 展示名/图标只消费 Host catalog；Core 不硬编码供应商品牌。
 * 敏感操作依赖 Host session-bound recent-auth；失败时引导重新登录（step-up）。
 */

import type { ExternalIdentityItem } from '~/composables/identity/useAccountSecurityApi'
import type { PublicAuthProvider } from '~/composables/identity/useAuthProviders'
import { authProviderDisplayMeta } from '~/composables/identity/useAuthProviders'
import { apiErrorMessage, apiErrorReason } from '~/composables/useApiClient'
import {
  passwordPolicyProgress,
  passwordPolicyProgressLevel,
  passwordPolicyRequirements
} from '~/composables/useWebOptions'
import { buildAuthPageLink } from '~/utils/identity/authReturn'

const SECURITY_RETURN_PATH = '/settings/security'

const { t } = useI18n()
const toast = useToast()
const localePath = useLocalePath()
const securityApi = useAccountSecurityApi()
const {
  linkProviders,
  providers: catalogProviders,
  pending: catalogPending,
  redirectToProvider,
  refresh: refreshCatalog
} = useAuthProviders()
const { passwordPolicy } = useWebOptions()
const { format: formatSiteDateTime } = useSiteDateTime()

const identities = ref<ExternalIdentityItem[]>([])
const identitiesPending = ref(true)
const identitiesError = ref('')
const linkStartingId = ref('')
const unlinkingId = ref<number | null>(null)

// session-bound recent-auth / step-up 门控：Host 拒绝后展示，引导重新登录。
const recentAuthRequired = ref(false)
const surfaceError = ref('')

// 本地密码设置：对 external-only 用户创建 credential；有密码时等同更新。
const passwordForm = reactive({
  password: '',
  confirm: ''
})
const passwordSubmitting = ref(false)
const passwordFieldError = ref('')
// 本页成功设置后视为已有密码，用于 last-method 提示（Host 仍为权威）。
const passwordReadyLocally = ref(false)

const activeIdentities = computed(() =>
  identities.value.filter(item => item.status === 'active' || item.status === 'inert')
)

const activeProviderIds = computed(() => {
  const set = new Set<string>()
  for (const item of identities.value) {
    if (item.status === 'active') {
      set.add(item.providerId)
    }
  }
  return set
})

/** Host 已开放 link 且当前账号尚未 active 绑定的提供方。 */
const availableLinkProviders = computed(() =>
  linkProviders.value.filter(provider => !activeProviderIds.value.has(provider.id))
)

const activeLinkCount = computed(() =>
  identities.value.filter(item => item.status === 'active').length
)

const passwordsMatch = computed(() => passwordForm.password === passwordForm.confirm)
const passwordProgress = computed(() =>
  passwordPolicyProgress(passwordForm.password, passwordPolicy.value)
)
const passwordProgressLevel = computed(() =>
  passwordPolicyProgressLevel(passwordProgress.value)
)
const passwordRequirementRows = computed(() =>
  passwordPolicyRequirements(passwordForm.password, passwordPolicy.value).map(item => ({
    ...item,
    label: passwordRequirementLabel(item.key)
  }))
)
const newPasswordMeetsPolicy = computed(() =>
  passwordRequirementRows.value.every(item => item.met)
)
const canSubmitPassword = computed(() =>
  newPasswordMeetsPolicy.value
  && passwordsMatch.value
  && passwordForm.password.length > 0
  && !passwordSubmitting.value
)

const reauthLoginLink = computed(() =>
  buildAuthPageLink(localePath('/login'), SECURITY_RETURN_PATH)
)

function passwordRequirementLabel(key: string) {
  switch (key) {
    case 'lowercase':
      return t('auth.passwordRequirementLowercase')
    case 'uppercase':
      return t('auth.passwordRequirementUppercase')
    case 'number':
      return t('auth.passwordRequirementNumber')
    case 'symbol':
      return t('auth.passwordRequirementSymbol')
    default:
      return t('auth.passwordRequirementLength', {
        min: passwordPolicy.value.minLength,
        max: passwordPolicy.value.maxLength
      })
  }
}

function catalogFor(providerId: string): PublicAuthProvider | undefined {
  return catalogProviders.value.find(item => item.id === providerId)
}

function displayFor(providerId: string) {
  const catalog = catalogFor(providerId)
  if (catalog) {
    return authProviderDisplayMeta(catalog, t('auth.providers.genericName'))
  }
  // 插件禁用/不在公共 catalog 时：Host 通用回退，不按 id 猜品牌。
  return {
    id: providerId,
    label: t('auth.providers.genericName'),
    icon: 'i-lucide-key-round',
    activatedOperations: [] as const
  }
}

function statusLabel(status: string) {
  if (status === 'active') {
    return t('accountSecurity.linkedAccounts.statusActive')
  }
  if (status === 'inert') {
    return t('accountSecurity.linkedAccounts.statusInert')
  }
  return t('accountSecurity.linkedAccounts.statusOther')
}

function formatLinkedAt(iso?: string | null) {
  if (!iso) {
    return t('accountSecurity.unknown')
  }
  return formatSiteDateTime(iso)
}

/** 仅 1 条 active 时提示可能为最后登录方式（Host 仍校验是否有密码）。 */
function mayBeLastLoginMethod(item: ExternalIdentityItem) {
  return item.status === 'active' && activeLinkCount.value <= 1 && !passwordReadyLocally.value
}

async function loadIdentities() {
  identitiesPending.value = true
  identitiesError.value = ''
  try {
    identities.value = await securityApi.listExternalIdentities()
  } catch (error) {
    identitiesError.value = apiErrorMessage(error) || t('accountSecurity.linkedAccounts.loadFailed')
    // 401/会话拒绝：不伪造列表。
    if (apiErrorReason(error) === 'auth.required' || apiErrorReason(error) === 'auth.recent_auth_required') {
      recentAuthRequired.value = apiErrorReason(error) === 'auth.recent_auth_required'
    }
  } finally {
    identitiesPending.value = false
  }
}

function markRecentAuthRequired(message?: string) {
  recentAuthRequired.value = true
  surfaceError.value = message || t('accountSecurity.linkedAccounts.recentAuthRequired')
}

function clearSurfaceError() {
  surfaceError.value = ''
}

function handleSensitiveError(error: unknown, fallback: string) {
  const reason = apiErrorReason(error)
  if (reason === 'auth.recent_auth_required') {
    markRecentAuthRequired()
    return
  }
  if (reason === 'auth.required') {
    markRecentAuthRequired(t('accountSecurity.linkedAccounts.sessionRequired'))
    return
  }
  if (reason === 'auth.last_login_method_required') {
    surfaceError.value = t('accountSecurity.linkedAccounts.lastMethodBlocked')
    toast.add({
      color: 'error',
      icon: 'i-lucide-shield-alert',
      title: t('accountSecurity.linkedAccounts.lastMethodBlocked')
    })
    return
  }
  const message = apiErrorMessage(error) || fallback
  surfaceError.value = message
  toast.add({ color: 'error', icon: 'i-lucide-alert-triangle', title: message })
}

async function startLink(provider: PublicAuthProvider) {
  if (linkStartingId.value) {
    return
  }
  clearSurfaceError()
  recentAuthRequired.value = false
  linkStartingId.value = provider.id
  try {
    // 回跳账号安全页；成功 Toast 由根布局 ext_auth 消费。
    await redirectToProvider(provider.id, 'link', {
      redirectHint: SECURITY_RETURN_PATH
    })
  } catch (error) {
    handleSensitiveError(error, t('accountSecurity.linkedAccounts.linkFailed'))
  } finally {
    linkStartingId.value = ''
  }
}

async function unlinkIdentity(item: ExternalIdentityItem) {
  if (item.status !== 'active' || unlinkingId.value !== null) {
    return
  }
  clearSurfaceError()
  recentAuthRequired.value = false
  unlinkingId.value = item.linkId
  try {
    await securityApi.unlinkExternalIdentity(item.linkId)
    toast.add({
      color: 'success',
      icon: 'i-lucide-check',
      title: t('accountSecurity.linkedAccounts.unlinked'),
      duration: 10000
    })
    await loadIdentities()
  } catch (error) {
    handleSensitiveError(error, t('accountSecurity.linkedAccounts.unlinkFailed'))
  } finally {
    unlinkingId.value = null
  }
}

async function submitPassword() {
  if (!canSubmitPassword.value) {
    return
  }
  passwordFieldError.value = ''
  clearSurfaceError()
  recentAuthRequired.value = false
  passwordSubmitting.value = true
  try {
    await securityApi.setupPassword(passwordForm.password)
    passwordReadyLocally.value = true
    passwordForm.password = ''
    passwordForm.confirm = ''
    toast.add({
      color: 'success',
      icon: 'i-lucide-check',
      title: t('accountSecurity.passwordSetup.success'),
      duration: 10000
    })
  } catch (error) {
    const reason = apiErrorReason(error)
    if (reason === 'auth.recent_auth_required' || reason === 'auth.required') {
      handleSensitiveError(error, t('accountSecurity.passwordSetup.failed'))
      return
    }
    // 字段级密码策略错误优先贴在表单旁。
    passwordFieldError.value = apiErrorMessage(error) || t('accountSecurity.passwordSetup.failed')
  } finally {
    passwordSubmitting.value = false
  }
}

async function refreshAll() {
  await Promise.all([loadIdentities(), refreshCatalog()])
}

onMounted(() => {
  void loadIdentities()
})

// 绑定成功回跳后刷新列表（根布局已 Toast）。
const route = useRoute()
watch(
  () => route.query.ext_auth,
  (reason) => {
    if (typeof reason === 'string' && reason.includes('link')) {
      void loadIdentities()
    }
  }
)

defineExpose({ refreshAll, loadIdentities })
</script>

<template>
  <section class="sf-linked-accounts" data-testid="linked-accounts-section">
    <h2 class="text-lg font-semibold text-slate-900 dark:text-zinc-50">
      {{ t('accountSecurity.linkedAccounts.title') }}
    </h2>
    <p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">
      {{ t('accountSecurity.linkedAccounts.intro') }}
    </p>

    <SFAlert
      v-if="recentAuthRequired"
      variant="warning"
      class="mt-4"
      :title="t('accountSecurity.linkedAccounts.recentAuthTitle')"
      :description="t('accountSecurity.linkedAccounts.recentAuthRequired')"
      closable
      @close="recentAuthRequired = false"
    >
      <NuxtLink
        :to="reauthLoginLink"
        class="mt-2 inline-flex items-center gap-1 text-sm font-semibold text-amber-800 underline dark:text-amber-200"
      >
        <UIcon name="i-lucide-log-in" class="size-4" aria-hidden="true" />
        {{ t('accountSecurity.linkedAccounts.reauthenticate') }}
      </NuxtLink>
    </SFAlert>

    <SFAlert
      v-else-if="surfaceError"
      variant="danger"
      class="mt-4"
      :title="surfaceError"
      closable
      @close="clearSurfaceError"
    />

    <SFCard class="mt-4 p-0 overflow-hidden">
      <div v-if="identitiesPending && activeIdentities.length === 0" class="divide-y divide-slate-100 dark:divide-zinc-800">
        <div v-for="i in 2" :key="i" class="p-4">
          <SFSkeleton class="h-4 w-1/3 mb-2" />
          <SFSkeleton class="h-3 w-1/2" />
        </div>
      </div>

      <p v-else-if="identitiesError" class="p-4 text-sm text-red-600 dark:text-red-400">
        {{ identitiesError }}
      </p>

      <ul
        v-else-if="activeIdentities.length > 0"
        class="divide-y divide-slate-100 dark:divide-zinc-800"
        data-testid="linked-accounts-list"
      >
        <li
          v-for="item in activeIdentities"
          :key="item.linkId"
          class="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between"
          :data-provider-id="item.providerId"
          :data-link-status="item.status"
        >
          <div class="min-w-0 flex items-start gap-3">
            <UIcon
              :name="displayFor(item.providerId).icon"
              class="mt-0.5 size-5 shrink-0 text-slate-500 dark:text-zinc-400"
              aria-hidden="true"
            />
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <span class="font-medium text-slate-900 dark:text-zinc-100">
                  {{ displayFor(item.providerId).label }}
                </span>
                <span
                  class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium"
                  :class="item.status === 'active'
                    ? 'bg-teal-50 text-teal-700 dark:bg-teal-900/30 dark:text-teal-300'
                    : 'bg-slate-100 text-slate-600 dark:bg-zinc-800 dark:text-zinc-300'"
                >
                  {{ statusLabel(item.status) }}
                </span>
              </div>
              <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
                {{ t('accountSecurity.linkedAccounts.linkedAt') }}: {{ formatLinkedAt(item.linkedAt) }}
              </p>
              <p
                v-if="item.status === 'inert'"
                class="mt-1 text-xs text-amber-700 dark:text-amber-300"
                data-testid="linked-account-inert-hint"
              >
                {{ t('accountSecurity.linkedAccounts.inertHint') }}
              </p>
              <p
                v-else-if="mayBeLastLoginMethod(item)"
                class="mt-1 text-xs text-slate-500 dark:text-zinc-400"
              >
                {{ t('accountSecurity.linkedAccounts.lastMethodHint') }}
              </p>
            </div>
          </div>
          <div class="shrink-0">
            <SFButton
              v-if="item.status === 'active'"
              variant="ghost"
              size="sm"
              :disabled="unlinkingId !== null"
              :aria-busy="unlinkingId === item.linkId ? 'true' : undefined"
              data-testid="unlink-external-identity"
              @click="unlinkIdentity(item)"
            >
              <UIcon name="i-lucide-unlink" class="mr-1" aria-hidden="true" />
              {{ unlinkingId === item.linkId
                ? t('accountSecurity.linkedAccounts.unlinking')
                : t('accountSecurity.linkedAccounts.unlink') }}
            </SFButton>
            <span
              v-else
              class="inline-flex items-center text-xs font-medium text-slate-400 dark:text-zinc-500"
            >
              {{ t('accountSecurity.linkedAccounts.inertAction') }}
            </span>
          </div>
        </li>
      </ul>

      <p v-else class="p-4 text-sm text-slate-500 dark:text-zinc-400" data-testid="linked-accounts-empty">
        {{ t('accountSecurity.linkedAccounts.empty') }}
      </p>
    </SFCard>

    <!-- link 入口：仅 Host 有效开放 link 且会话可用时展示 -->
    <div
      v-if="availableLinkProviders.length > 0"
      class="mt-4"
      data-testid="link-provider-entry"
    >
      <p class="mb-2 text-sm font-medium text-slate-700 dark:text-zinc-300">
        {{ t('accountSecurity.linkedAccounts.linkTitle') }}
      </p>
      <p class="mb-3 text-xs text-slate-500 dark:text-zinc-400">
        {{ t('accountSecurity.linkedAccounts.linkHelp') }}
      </p>
      <div class="flex flex-col gap-2 sm:flex-row sm:flex-wrap">
        <SFButton
          v-for="provider in availableLinkProviders"
          :key="provider.id"
          variant="secondary"
          size="sm"
          class="justify-center sm:justify-start"
          :disabled="Boolean(linkStartingId) || catalogPending"
          :data-provider-id="provider.id"
          data-testid="link-provider-button"
          @click="startLink(provider)"
        >
          <UIcon
            :name="authProviderDisplayMeta(provider, t('auth.providers.genericName')).icon"
            class="mr-1.5 size-4"
            aria-hidden="true"
          />
          {{ linkStartingId === provider.id
            ? t('accountSecurity.linkedAccounts.linking')
            : t('accountSecurity.linkedAccounts.linkWith', {
              name: authProviderDisplayMeta(provider, t('auth.providers.genericName')).label
            }) }}
        </SFButton>
      </div>
    </div>
    <p
      v-else-if="!catalogPending && linkProviders.length === 0"
      class="mt-3 text-xs text-slate-500 dark:text-zinc-400"
      data-testid="link-entry-closed"
    >
      {{ t('accountSecurity.linkedAccounts.linkClosed') }}
    </p>

    <!-- external-only / 通用本地密码设置 -->
    <section class="mt-10" data-testid="password-setup-section">
      <h2 class="text-lg font-semibold text-slate-900 dark:text-zinc-50">
        {{ t('accountSecurity.passwordSetup.title') }}
      </h2>
      <p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">
        {{ t('accountSecurity.passwordSetup.intro') }}
      </p>
      <p
        v-if="activeLinkCount > 0 && !passwordReadyLocally"
        class="mt-2 text-xs text-slate-500 dark:text-zinc-400"
      >
        {{ t('accountSecurity.passwordSetup.externalOnlyHint') }}
      </p>

      <SFCard class="mt-4 p-4">
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="block text-sm sm:col-span-2">
            <span class="mb-1 block font-medium text-slate-600 dark:text-zinc-300">
              {{ t('accountSecurity.passwordSetup.password') }}
            </span>
            <input
              v-model="passwordForm.password"
              type="password"
              autocomplete="new-password"
              class="w-full rounded-md border border-slate-200 bg-white px-3 py-2 text-sm dark:border-zinc-700 dark:bg-zinc-950"
              data-testid="password-setup-input"
            >
            <p class="mt-1 text-xs text-slate-400 dark:text-zinc-500">
              {{ t('auth.passwordPolicySummary', {
                min: passwordPolicy.minLength,
                max: passwordPolicy.maxLength
              }) }}
            </p>
            <div class="sf-password-policy mt-2">
              <div class="sf-password-policy__header">
                <span>{{ t('auth.passwordStrength') }}</span>
                <span :class="['sf-password-policy__value', `is-${passwordProgressLevel}`]">
                  {{ passwordProgress }}%
                </span>
              </div>
              <div
                class="sf-password-policy__bar"
                :class="[`is-${passwordProgressLevel}`]"
                aria-hidden="true"
              >
                <span :style="{ width: `${passwordProgress}%` }" />
              </div>
              <ul class="sf-password-policy__list">
                <li
                  v-for="row in passwordRequirementRows"
                  :key="row.key"
                  :class="{ 'is-met': row.met }"
                >
                  <UIcon
                    :name="row.met ? 'i-lucide-check' : 'i-lucide-circle'"
                    class="sf-password-policy__icon"
                    aria-hidden="true"
                  />
                  <span>{{ row.label }}</span>
                </li>
              </ul>
            </div>
          </label>

          <label class="block text-sm sm:col-span-2">
            <span class="mb-1 block font-medium text-slate-600 dark:text-zinc-300">
              {{ t('accountSecurity.passwordSetup.confirm') }}
            </span>
            <input
              v-model="passwordForm.confirm"
              type="password"
              autocomplete="new-password"
              class="w-full rounded-md border border-slate-200 bg-white px-3 py-2 text-sm dark:border-zinc-700 dark:bg-zinc-950"
              data-testid="password-setup-confirm"
            >
            <p
              v-if="passwordForm.confirm && !passwordsMatch"
              class="mt-1 text-sm text-red-600 dark:text-red-400"
            >
              {{ t('auth.passwordsDoNotMatch') }}
            </p>
          </label>
        </div>

        <p
          v-if="passwordFieldError"
          class="mt-3 text-sm text-red-600 dark:text-red-400"
          data-testid="password-setup-error"
        >
          {{ passwordFieldError }}
        </p>

        <div class="mt-4 flex flex-col gap-2 sm:flex-row sm:items-center">
          <SFButton
            variant="primary"
            size="sm"
            :disabled="!canSubmitPassword"
            data-testid="password-setup-submit"
            @click="submitPassword"
          >
            <UIcon name="i-lucide-key-round" class="mr-1" aria-hidden="true" />
            {{ passwordSubmitting
              ? t('accountSecurity.passwordSetup.submitting')
              : t('accountSecurity.passwordSetup.submit') }}
          </SFButton>
          <p class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('accountSecurity.passwordSetup.recentAuthHint') }}
          </p>
        </div>
      </SFCard>
    </section>
  </section>
</template>

<style scoped>
.sf-password-policy {
  margin-top: 0.5rem;
}

.sf-password-policy__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
  color: var(--sf-fg-tertiary, #64748b);
  margin-bottom: 0.35rem;
}

.sf-password-policy__bar {
  height: 6px;
  border-radius: 999px;
  background: #e2e8f0;
  overflow: hidden;
}

.sf-password-policy__bar span {
  display: block;
  height: 100%;
  border-radius: 999px;
  transition: width 0.15s ease;
  background: #94a3b8;
}

.sf-password-policy__bar.is-weak span {
  background: #f59e0b;
}

.sf-password-policy__bar.is-medium span {
  background: #0d9488;
}

.sf-password-policy__bar.is-strong span {
  background: #0f766e;
}

.sf-password-policy__value.is-weak {
  color: #d97706;
}

.sf-password-policy__value.is-medium {
  color: #0d9488;
}

.sf-password-policy__value.is-strong {
  color: #0f766e;
}

.sf-password-policy__list {
  margin: 0.5rem 0 0;
  padding: 0;
  list-style: none;
  display: grid;
  gap: 0.25rem;
}

.sf-password-policy__list li {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 12px;
  color: #94a3b8;
}

.sf-password-policy__list li.is-met {
  color: #0f766e;
}

.sf-password-policy__icon {
  width: 14px;
  height: 14px;
  flex: 0 0 14px;
}

:global(.dark) .sf-password-policy__bar {
  background: #3f3f46;
}

:global(.dark) .sf-password-policy__list li {
  color: #71717a;
}

:global(.dark) .sf-password-policy__list li.is-met {
  color: #2dd4bf;
}

@media (max-width: 640px) {
  .sf-linked-accounts :deep([data-testid='link-provider-button']) {
    width: 100%;
    min-height: 44px;
  }
}
</style>
