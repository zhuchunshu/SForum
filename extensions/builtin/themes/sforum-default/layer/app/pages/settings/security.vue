<script setup lang="ts">
definePageMeta({ requiresAuth: true })

const { t } = useI18n()
const toast = useToast()
const { siteName } = useWebOptions()
const sessionsApi = useAccountSecurityApi()

useSForumSeo({
  title: () => `${t('accountSecurity.metaTitle')} - ${siteName.value}`,
  description: () => t('accountSecurity.metaDescription'),
  type: 'website'
})

const emptySessionList = (): LoginSessionList => ({
  items: [],
  total: 0,
  page: 1,
  perPage: 20
})

// SSR 首屏加载活跃设备列表。
const { data: sessions, pending, refresh } = await useAsyncData(
  'account-security-sessions',
  () => sessionsApi.listSessions(),
  { default: emptySessionList }
)
const activeSessions = computed(() => sessions.value?.items || [])

// 历史记录折叠区。
const showHistory = ref(false)
const historySessions = ref<LoginSession[]>([])
const historyLoading = ref(false)

async function loadHistory() {
  if (showHistory.value) {
    showHistory.value = false
    return
  }
  historyLoading.value = true
  try {
    const result = await sessionsApi.listSessions({ includeHistory: true, perPage: 50 })
    historySessions.value = result.items.filter(s => s.revokedAt)
    showHistory.value = true
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-alert-triangle', title: apiErrorMessage(error) || t('accountSecurity.loadFailed') })
  } finally {
    historyLoading.value = false
  }
}

// 下线单个设备。
async function revokeDevice(session: LoginSession) {
  if (session.isCurrent) {
    return // 当前设备不在此处下线（用「退出登录」即可）
  }
  try {
    await sessionsApi.revokeSession(session.id)
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('accountSecurity.deviceRevoked') })
    await refresh()
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-alert-triangle', title: apiErrorMessage(error) || t('accountSecurity.revokeFailed') })
  }
}

// 下线除当前外的所有设备。
const revokingOthers = ref(false)
async function revokeOthers() {
  if (revokingOthers.value) {
    return
  }
  revokingOthers.value = true
  try {
    const result = await sessionsApi.revokeOtherSessions()
    toast.add({
      color: 'success',
      icon: 'i-lucide-check',
      title: t('accountSecurity.othersRevoked', { count: result.revoked })
    })
    await refresh()
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-alert-triangle', title: apiErrorMessage(error) || t('accountSecurity.revokeFailed') })
  } finally {
    revokingOthers.value = false
  }
}

const { format: formatSiteDateTime } = useSiteDateTime()

// 按站点时区与日期时间格式展示登录/最后活跃时间。
function formatTime(iso: string): string {
  return formatSiteDateTime(iso)
}

// —— 个人访问令牌（F3.4）——
const tokens = ref<APIToken[]>([])
const tokensLoading = ref(false)
const tokenForm = reactive({
  name: '',
  scopesText: 'topic.create,post.create'
})
const createdPlaintext = ref('')
const tokenBusy = ref(false)

async function loadTokens() {
  tokensLoading.value = true
  try {
    const result = await sessionsApi.listAPITokens()
    tokens.value = result.items || []
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-alert-triangle', title: apiErrorMessage(error) || t('accountSecurity.tokensLoadFailed') })
  } finally {
    tokensLoading.value = false
  }
}

async function createToken() {
  if (tokenBusy.value) return
  tokenBusy.value = true
  createdPlaintext.value = ''
  try {
    const scopes = tokenForm.scopesText.split(/[,\s]+/).map(s => s.trim()).filter(Boolean)
    const created = await sessionsApi.createAPIToken({ name: tokenForm.name, scopes })
    createdPlaintext.value = created.token
    tokenForm.name = ''
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('accountSecurity.tokenCreated') })
    await loadTokens()
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-alert-triangle', title: apiErrorMessage(error) || t('accountSecurity.tokenSaveFailed') })
  } finally {
    tokenBusy.value = false
  }
}

async function revokeToken(token: APIToken) {
  try {
    await sessionsApi.revokeAPIToken(token.id)
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('accountSecurity.tokenRevoked') })
    await loadTokens()
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-alert-triangle', title: apiErrorMessage(error) || t('accountSecurity.tokenSaveFailed') })
  }
}

async function rotateToken(token: APIToken) {
  try {
    const created = await sessionsApi.rotateAPIToken(token.id)
    createdPlaintext.value = created.token
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('accountSecurity.tokenRotated') })
    await loadTokens()
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-alert-triangle', title: apiErrorMessage(error) || t('accountSecurity.tokenSaveFailed') })
  }
}

async function copyPlaintext() {
  if (!createdPlaintext.value) return
  try {
    await navigator.clipboard.writeText(createdPlaintext.value)
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('accountSecurity.tokenCopied') })
  } catch {
    toast.add({ color: 'error', icon: 'i-lucide-alert-triangle', title: t('accountSecurity.tokenCopyFailed') })
  }
}

onMounted(() => {
  loadTokens()
})
</script>

<template>
  <main class="sf-public-page min-h-screen py-8">
    <div class="sf-public-page__container sf-public-page__container--narrow mx-auto px-4 sm:px-6">
      <div class="flex items-center justify-between mb-6">
        <h1 class="text-2xl font-bold text-slate-900 dark:text-zinc-50">
          {{ t('accountSecurity.title') }}
        </h1>
        <SFButton
          variant="secondary"
          size="sm"
          :disabled="revokingOthers || activeSessions.length <= 1"
          @click="revokeOthers"
        >
          <UIcon name="i-lucide-log-out" class="mr-1" />
          {{ t('accountSecurity.revokeOthers') }}
        </SFButton>
      </div>

      <!-- 账号设置子导航：在资料设置与账号安全间切换 -->
      <div class="flex gap-2 mb-6">
        <NuxtLink
          to="/settings/profile"
          class="inline-flex items-center gap-1.5 rounded-md border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-600 hover:bg-slate-50 dark:border-zinc-800 dark:text-zinc-400 dark:hover:bg-zinc-900"
        >
          <UIcon name="i-lucide-user" />
          {{ t('profileSettings.title') }}
        </NuxtLink>
        <NuxtLink
          to="/settings/security"
          class="inline-flex items-center gap-1.5 rounded-md border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-600 hover:bg-slate-50 dark:border-zinc-800 dark:text-zinc-400 dark:hover:bg-zinc-900"
        >
          <UIcon name="i-lucide-shield-check" />
          {{ t('accountSecurity.title') }}
        </NuxtLink>
      </div>

      <p class="text-sm text-slate-500 dark:text-zinc-400 mb-6">
        {{ t('accountSecurity.intro') }}
      </p>

      <SFCard class="p-0 overflow-hidden">
        <!-- 加载骨架 -->
        <div v-if="pending && activeSessions.length === 0" class="divide-y divide-slate-100 dark:divide-zinc-800">
          <div v-for="i in 3" :key="i" class="p-4">
            <SFSkeleton class="h-4 w-1/3 mb-2" />
            <SFSkeleton class="h-3 w-1/2" />
          </div>
        </div>

        <!-- 活跃设备列表 -->
        <ul v-else-if="activeSessions.length > 0" class="divide-y divide-slate-100 dark:divide-zinc-800">
          <li
            v-for="session in activeSessions"
            :key="session.id"
            class="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between"
          >
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <UIcon
                  :name="session.isCurrent ? 'i-lucide-monitor-check' : 'i-lucide-monitor'"
                  class="text-slate-400 shrink-0"
                />
                <span class="font-medium text-slate-900 dark:text-zinc-100 truncate">
                  {{ session.deviceName || t('accountSecurity.unknownDevice') }}
                </span>
                <span
                  v-if="session.isCurrent"
                  class="inline-flex items-center rounded-full bg-teal-50 px-2 py-0.5 text-xs font-medium text-teal-700 dark:bg-teal-900/30 dark:text-teal-300"
                >
                  {{ t('accountSecurity.currentDevice') }}
                </span>
              </div>
              <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
                {{ t('accountSecurity.ip') }}: {{ session.ipPrefix || t('accountSecurity.unknown') }}
                · {{ t('accountSecurity.loggedIn') }}: {{ formatTime(session.createdAt) }}
                · {{ t('accountSecurity.lastActive') }}: {{ formatTime(session.lastSeenAt) }}
              </p>
            </div>
            <div class="shrink-0">
              <SFButton
                variant="ghost"
                size="sm"
                :disabled="session.isCurrent"
                @click="revokeDevice(session)"
              >
                <UIcon name="i-lucide-x" class="mr-1" />
                {{ t('accountSecurity.revoke') }}
              </SFButton>
            </div>
          </li>
        </ul>

        <!-- 空状态 -->
        <SFEmptyState
          v-else
          :title="t('accountSecurity.emptyTitle')"
          :description="t('accountSecurity.emptyDescription')"
        />
      </SFCard>

      <!-- 登录历史折叠区 -->
      <div class="mt-6">
        <button
          class="flex items-center gap-1 text-sm font-medium text-slate-600 hover:text-slate-900 dark:text-zinc-400 dark:hover:text-zinc-200"
          @click="loadHistory"
        >
          <UIcon :name="showHistory ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'" />
          {{ t('accountSecurity.showHistory') }}
        </button>

        <SFCard v-if="showHistory" class="p-0 mt-3 overflow-hidden">
          <div v-if="historyLoading" class="p-4">
            <SFSkeleton class="h-4 w-1/2 mb-2" />
            <SFSkeleton class="h-3 w-1/3" />
          </div>
          <ul v-else-if="historySessions.length > 0" class="divide-y divide-slate-100 dark:divide-zinc-800">
            <li
              v-for="session in historySessions"
              :key="session.id"
              class="p-4"
            >
              <div class="flex items-center gap-2">
                <UIcon name="i-lucide-monitor-off" class="text-slate-400 shrink-0" />
                <span class="font-medium text-slate-700 dark:text-zinc-300 truncate">
                  {{ session.deviceName || t('accountSecurity.unknownDevice') }}
                </span>
              </div>
              <p class="mt-1 text-xs text-slate-400 dark:text-zinc-500">
                {{ t('accountSecurity.loggedIn') }}: {{ formatTime(session.createdAt) }}
                · {{ t('accountSecurity.revokedAt') }}: {{ formatTime(session.revokedAt || '') }}
              </p>
            </li>
          </ul>
          <p v-else class="p-4 text-sm text-slate-400 dark:text-zinc-500">
            {{ t('accountSecurity.noHistory') }}
          </p>
        </SFCard>
      </div>

      <!-- 个人访问令牌 -->
      <section class="mt-10">
        <h2 class="text-lg font-semibold text-slate-900 dark:text-zinc-50">
          {{ t('accountSecurity.tokensTitle') }}
        </h2>
        <p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">
          {{ t('accountSecurity.tokensIntro') }}
        </p>

        <SFCard class="mt-4 p-4">
          <div class="grid gap-3 sm:grid-cols-2">
            <label class="block text-sm">
              <span class="mb-1 block text-slate-600 dark:text-zinc-300">{{ t('accountSecurity.tokenName') }}</span>
              <input
                v-model="tokenForm.name"
                class="w-full rounded-md border border-slate-200 bg-white px-3 py-2 text-sm dark:border-zinc-700 dark:bg-zinc-950"
                type="text"
              >
            </label>
            <label class="block text-sm">
              <span class="mb-1 block text-slate-600 dark:text-zinc-300">{{ t('accountSecurity.tokenScopes') }}</span>
              <input
                v-model="tokenForm.scopesText"
                class="w-full rounded-md border border-slate-200 bg-white px-3 py-2 text-sm font-mono dark:border-zinc-700 dark:bg-zinc-950"
                type="text"
              >
            </label>
          </div>
          <p class="mt-2 text-xs text-slate-500">
            {{ t('accountSecurity.tokenScopesHint') }}
          </p>
          <div class="mt-3">
            <SFButton
              variant="primary"
              size="sm"
              :disabled="tokenBusy || !tokenForm.name"
              @click="createToken"
            >
              <UIcon name="i-lucide-key-round" class="mr-1" />
              {{ t('accountSecurity.tokenCreate') }}
            </SFButton>
          </div>

          <div
            v-if="createdPlaintext"
            class="mt-4 rounded-md border border-amber-200 bg-amber-50 p-3 dark:border-amber-900/50 dark:bg-amber-950/30"
          >
            <p class="text-xs font-medium text-amber-800 dark:text-amber-200">
              {{ t('accountSecurity.tokenOnceHint') }}
            </p>
            <p class="mt-2 break-all font-mono text-sm text-slate-900 dark:text-zinc-100">
              {{ createdPlaintext }}
            </p>
            <SFButton class="mt-2" variant="secondary" size="sm" @click="copyPlaintext">
              <UIcon name="i-lucide-copy" class="mr-1" />
              {{ t('accountSecurity.tokenCopy') }}
            </SFButton>
          </div>
        </SFCard>

        <SFCard class="mt-4 p-0 overflow-hidden">
          <div v-if="tokensLoading" class="p-4 text-sm text-slate-500">
            {{ t('accountSecurity.tokensLoading') }}
          </div>
          <ul v-else-if="tokens.length" class="divide-y divide-slate-100 dark:divide-zinc-800">
            <li
              v-for="token in tokens"
              :key="token.id"
              class="flex flex-col gap-2 p-4 sm:flex-row sm:items-center sm:justify-between"
            >
              <div class="min-w-0">
                <p class="font-medium text-slate-900 dark:text-zinc-100">
                  {{ token.name }}
                  <span class="ml-2 font-mono text-xs text-slate-400">{{ token.prefix }}…</span>
                </p>
                <p class="mt-1 text-xs text-slate-500">
                  {{ token.scopes.join(', ') }}
                  · {{ t('accountSecurity.loggedIn') }}: {{ formatTime(token.createdAt) }}
                </p>
              </div>
              <div class="flex shrink-0 gap-2">
                <SFButton variant="ghost" size="sm" @click="rotateToken(token)">
                  {{ t('accountSecurity.tokenRotate') }}
                </SFButton>
                <SFButton variant="ghost" size="sm" @click="revokeToken(token)">
                  {{ t('accountSecurity.tokenRevoke') }}
                </SFButton>
              </div>
            </li>
          </ul>
          <p v-else class="p-4 text-sm text-slate-500">
            {{ t('accountSecurity.tokensEmpty') }}
          </p>
        </SFCard>
      </section>
    </div>
  </main>
</template>
