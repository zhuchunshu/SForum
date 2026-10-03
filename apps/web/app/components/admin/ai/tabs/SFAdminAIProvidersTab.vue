<script setup lang="ts">
import type { AdminAIProfile, AdminAISettings } from '~/composables/admin/useAdminAI'
import { ADMIN_AI_KEY, MICRO_PER_UNIT } from '~/composables/admin/useAdminAI'
import { useAuthSession } from '~/composables/identity/useAuthSession'
import { normalizedAISettingsForComparison, promptUsesDefault } from '~/utils/admin/adminAI'

const { t, te } = useI18n()
const { can } = useAuthSession()
// 数据由页面壳统一持有并注入；页签只渲染，不重新拉取。
const ai = inject(ADMIN_AI_KEY)!
const canManage = computed(() => can('ai.manage'))

const CUSTOM_PROFILE_ID = 'custom'

const form = ref<AdminAISettings | null>(null)
// selectedProviderId 是界面选择，profile.enabled 才是实际生效：两者分开，
// 是为了让「选了自定义但还没填完」这种中间状态可以保存而不被拒绝。
const selectedProviderId = ref('')
const credentials = reactive<Record<string, string>>({})
const advancedOpen = ref(false)

watch(ai.settings, (next) => {
  form.value = JSON.parse(JSON.stringify(next)) as AdminAISettings
  // 已保存的文档可能来自引入 reply 之前的版本，那时这个字段不存在。
  // 先补全再绑定，否则任何 .trim() 都会炸在 undefined 上。
  if (form.value && (typeof form.value.reply?.systemPrompt !== 'string')) {
    form.value.reply = { systemPrompt: '' }
  }
  // 没有自定义时在编辑器里预填内置默认：运营者要能看见默认值是什么，
  // 而不是面对一个空框去猜。「使用默认」的语义由保存时的归一化维持。
  if (form.value && !form.value.reply.systemPrompt.trim()) {
    form.value.reply.systemPrompt = ai.defaultSystemPrompt.value
  }
  const active = next.profiles.find(profile => profile.enabled)
  selectedProviderId.value = next.enabled ? (active?.id ?? '') : ''
}, { immediate: true })

const providerOptions = computed(() => {
  const builtins = (form.value?.profiles ?? [])
    .filter(profile => profile.id !== CUSTOM_PROFILE_ID)
    .map(profile => ({
      id: profile.id,
      label: profile.label,
      hint: `${profile.protocol} · ${profile.model || t('admin.ai.modelMissing')}`
    }))
  return [
    { id: '', label: t('admin.ai.providerNone'), hint: t('admin.ai.providerNoneHint') },
    ...builtins,
    { id: CUSTOM_PROFILE_ID, label: t('admin.ai.providerCustom'), hint: t('admin.ai.providerCustomHint') }
  ]
})

const selectedProfile = computed<AdminAIProfile | null>(() =>
  form.value?.profiles.find(profile => profile.id === selectedProviderId.value) ?? null
)
const isCustom = computed(() => selectedProviderId.value === CUSTOM_PROFILE_ID)
const profileIncomplete = computed(() =>
  selectedProfile.value ? !isProfileComplete(selectedProfile.value) : false
)

// 与内置默认相同的文本不算自定义：空白差异（编辑器会重排 Markdown）与「空串 /
// 字段被省略」两种写法都视为「使用默认」。比较与保存都走这里，避免「预填了默认
// 就被当成已修改」——那会让保存按钮一直亮着，也会把默认值固化成自定义内容。
function normalizedForComparison(settings: AdminAISettings) {
  return normalizedAISettingsForComparison(settings, ai.defaultSystemPrompt.value)
}

const dirty = computed(() => {
  if (!form.value) return false
  return JSON.stringify(normalizedForComparison(form.value)) !== JSON.stringify(normalizedForComparison(ai.settings.value))
})

const localizedWarnings = computed(() => ai.warnings.value.map((warning) => {
  const [code, detail] = warning.split(':')
  const key = `admin.ai.warnings.${code}`
  if (te(key)) return t(key)
  return detail ? `${code} (${detail})` : code
}))

const protocolOptions = computed(() => [
  { label: 'OpenAI 兼容', value: 'openai-chat' },
  { label: 'Claude', value: 'anthropic-messages' }
])

function isProfileComplete(profile: AdminAIProfile) {
  return Boolean(profile.label.trim() && profile.baseUrl.trim() && profile.model.trim())
}

// 只有一个提供商生效。选中项配置不完整时保持未启用——启用一份不完整的配置
// 只会让每次调用都失败，而那正是后端拒绝 422 的条件。
function syncEnabledProfiles() {
  const current = form.value
  if (!current) return
  for (const profile of current.profiles) {
    profile.enabled = profile.id === selectedProviderId.value && isProfileComplete(profile)
  }
  current.enabled = selectedProviderId.value !== ''
  // 单选模式下成本等级必然指向唯一启用的提供商。不同步的话，界面选了 A 而
  // 绑定仍写着 B，实际调用会走 B——界面就在撒谎。
  if (selectedProviderId.value) {
    for (const costClass of ['economy', 'standard', 'premium']) {
      current.costClassProfiles[costClass] = selectedProviderId.value
    }
  }
}

function selectProvider(id: string) {
  selectedProviderId.value = id
  if (id === CUSTOM_PROFILE_ID) ensureCustomProfile()
  syncEnabledProfiles()
}

// 自定义提供商在首次选中时创建，之后一直保留：切走再切回来不该丢掉已填内容。
function ensureCustomProfile() {
  const current = form.value
  if (!current || current.profiles.some(profile => profile.id === CUSTOM_PROFILE_ID)) return
  current.profiles.push({
    id: CUSTOM_PROFILE_ID,
    label: t('admin.ai.providerCustom'),
    protocol: 'openai-chat',
    baseUrl: '',
    apiKeyRef: 'sforum.secret://core/ai.custom.api_key',
    model: '',
    costClass: 'standard',
    enabled: false,
    price: { inputPerMillionMicros: 0, outputPerMillionMicros: 0 },
    defaults: { maxTokens: 2048, timeoutMs: 30000 }
  })
}

async function save() {
  if (!form.value) return
  syncEnabledProfiles()
  // 归一化后再提交：与默认逐字相同的内容以空值存库，保持「使用默认」语义，
  // 这样以后升级内置提示词时这部分站点会跟着更新。
  await ai.saveSettings(normalizedForComparison(form.value))
}

async function submitCredential(profileId: string) {
  const apiKey = (credentials[profileId] || '').trim()
  if (!apiKey) return
  if (await ai.saveCredential(profileId, apiKey)) credentials[profileId] = ''
}

function setNumber(path: 'rateLimitPerMinute' | 'extensionDailyQuota' | 'userDailyQuota' | 'toolCallsPerReply', value: string | number) {
  if (!form.value) return
  form.value.gates[path] = Math.max(0, Math.round(Number(value || 0)))
}

function setBudgetYuan(value: string | number) {
  if (!form.value) return
  form.value.gates.monthlyBudgetMicros = Math.round(Number(value || 0) * MICRO_PER_UNIT)
}

function budgetYuan() {
  return form.value ? form.value.gates.monthlyBudgetMicros / MICRO_PER_UNIT : 0
}

// 提示词为空表示使用内置默认；一键恢复就是把自定义内容清空。
const promptCustomized = computed(() => !promptUsesDefault(form.value?.reply?.systemPrompt, ai.defaultSystemPrompt.value))

function restoreDefaultPrompt() {
  if (!form.value) return
  form.value.reply = { systemPrompt: ai.defaultSystemPrompt.value }
}

function toggleAdvanced() {
  advancedOpen.value = !advancedOpen.value
}

function setMicro(field: 'inputPerMillionMicros' | 'outputPerMillionMicros', value: string | number) {
  const profile = selectedProfile.value
  if (!profile) return
  profile.price[field] = Math.round(Number(value || 0) * MICRO_PER_UNIT)
}

// 缺省（undefined）视为支持工具调用；只有明确关闭时才写 false。
function setSupportsTools(value: boolean | 'indeterminate') {
  const profile = selectedProfile.value
  if (!profile) return
  profile.supportsTools = value === true
}
</script>

<template>
  <div v-if="form" class="flex min-w-0 flex-col gap-4">
    <UAlert
      v-if="!ai.encryptionEnabled.value"
      color="warning"
      variant="soft"
      icon="i-lucide-shield-alert"
      :title="t('admin.ai.encryptionMissing')"
    />
    <UAlert
      v-for="warning in localizedWarnings"
      :key="warning"
      color="warning"
      variant="soft"
      icon="i-lucide-info"
      :title="warning"
    />

    <section class="min-w-0 rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <div class="border-b border-slate-200 px-4 py-3 dark:border-zinc-800">
        <p class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.ai.providerTitle') }}</p>
        <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.ai.providerIntro') }}</p>
      </div>
      <div class="divide-y divide-slate-200 dark:divide-zinc-800">
        <label
          v-for="option in providerOptions"
          :key="option.id || 'none'"
          class="flex cursor-pointer items-start gap-3 px-4 py-3"
        >
          <input
            type="radio"
            class="mt-1 size-4 accent-[var(--sf-accent)]"
            :value="option.id"
            :checked="selectedProviderId === option.id"
            :disabled="!canManage"
            @change="selectProvider(option.id)"
          >
          <span class="min-w-0 flex-1">
            <span class="flex flex-wrap items-center gap-2">
              <span class="text-sm font-medium text-slate-900 dark:text-zinc-100">{{ option.label }}</span>
              <UBadge v-if="option.id && ai.credentialStatus.value[option.id]" color="success" variant="soft" size="sm">
                {{ t('admin.ai.credentialConfigured') }}
              </UBadge>
            </span>
            <span class="mt-0.5 block text-xs text-slate-500 dark:text-zinc-400">{{ option.hint }}</span>
          </span>
        </label>
      </div>
    </section>

    <section v-if="selectedProfile" class="min-w-0 rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <div class="border-b border-slate-200 px-4 py-3 dark:border-zinc-800">
        <p class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.ai.providerConfig') }}</p>
        <p v-if="profileIncomplete" class="mt-1 text-xs text-amber-600 dark:text-amber-400">{{ t('admin.ai.profileIncomplete') }}</p>
      </div>
      <div class="flex flex-col gap-3 px-4 py-4">
        <UFormField v-if="isCustom" :label="t('admin.ai.customName')">
          <UInput v-model="selectedProfile.label" :disabled="!canManage" class="w-full" />
        </UFormField>

        <UFormField :label="t('admin.ai.credential')" :help="t('admin.ai.credentialHint')">
          <div class="flex min-w-0 flex-col gap-2 sm:flex-row">
            <UInput
              v-model="credentials[selectedProfile.id]"
              type="password"
              autocomplete="off"
              :placeholder="ai.credentialStatus.value[selectedProfile.id] ? t('admin.ai.credentialPlaceholderKeep') : t('admin.ai.credentialPlaceholder')"
              :disabled="!canManage"
              class="min-w-0 flex-1"
            />
            <UButton
              icon="i-lucide-key-round"
              color="neutral"
              variant="subtle"
              :loading="ai.busy.value === `credential:${selectedProfile.id}`"
              :disabled="!canManage || !(credentials[selectedProfile.id] || '').trim()"
              @click="submitCredential(selectedProfile.id)"
            >
              {{ t('admin.ai.credentialSave') }}
            </UButton>
          </div>
        </UFormField>

        <div class="grid grid-cols-1 gap-3 lg:grid-cols-3">
          <UFormField v-if="isCustom" :label="t('admin.ai.protocol')">
            <USelect v-model="selectedProfile.protocol" :items="protocolOptions" :disabled="!canManage" class="w-full" />
          </UFormField>
          <UFormField :label="t('admin.ai.baseUrl')">
            <UInput v-model="selectedProfile.baseUrl" :disabled="!canManage" class="w-full" />
          </UFormField>
          <UFormField :label="t('admin.ai.model')">
            <UInput v-model="selectedProfile.model" :disabled="!canManage" class="w-full" />
          </UFormField>
        </div>

        <div>
          <UButton
            size="xs"
            color="neutral"
            variant="ghost"
            :icon="advancedOpen ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
            @click="toggleAdvanced"
          >
            {{ advancedOpen ? t('admin.ai.advancedHide') : t('admin.ai.advancedShow') }}
          </UButton>
        </div>

        <div v-if="advancedOpen" class="grid grid-cols-1 gap-3 rounded-md bg-slate-50 p-3 dark:bg-zinc-950/60 lg:grid-cols-3">
          <UFormField :label="t('admin.ai.costClass')">
            <USelect v-model="selectedProfile.costClass" :items="[
              { label: t('admin.ai.costEconomy'), value: 'economy' },
              { label: t('admin.ai.costStandard'), value: 'standard' },
              { label: t('admin.ai.costPremium'), value: 'premium' }
            ]" :disabled="!canManage" class="w-full" />
          </UFormField>
          <UFormField :label="t('admin.ai.maxTokens')">
            <UInput v-model.number="selectedProfile.defaults.maxTokens" type="number" :disabled="!canManage" class="w-full" />
          </UFormField>
          <UFormField :label="t('admin.ai.timeoutMs')">
            <UInput v-model.number="selectedProfile.defaults.timeoutMs" type="number" :disabled="!canManage" class="w-full" />
          </UFormField>
          <UFormField :label="t('admin.ai.supportsTools')" :help="t('admin.ai.supportsToolsHint')">
            <UCheckbox
              :model-value="selectedProfile?.supportsTools !== false"
              :disabled="!canManage"
              @update:model-value="setSupportsTools"
            />
          </UFormField>
          <UFormField :label="t('admin.ai.priceInput')" :help="t('admin.ai.priceHint')">
            <UInput
              :model-value="selectedProfile.price.inputPerMillionMicros / MICRO_PER_UNIT"
              type="number"
              step="0.01"
              :disabled="!canManage"
              class="w-full"
              @update:model-value="value => setMicro('inputPerMillionMicros', value)"
            />
          </UFormField>
          <UFormField :label="t('admin.ai.priceOutput')">
            <UInput
              :model-value="selectedProfile.price.outputPerMillionMicros / MICRO_PER_UNIT"
              type="number"
              step="0.01"
              :disabled="!canManage"
              class="w-full"
              @update:model-value="value => setMicro('outputPerMillionMicros', value)"
            />
          </UFormField>
        </div>
      </div>
    </section>

    <section class="min-w-0 rounded-lg border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
      <p class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.ai.gates') }}</p>
      <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.ai.gatesIntro') }}</p>
      <div class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <UFormField :label="t('admin.ai.rateLimit')">
          <UInput :model-value="form.gates.rateLimitPerMinute" type="number" :disabled="!canManage" class="w-full"
            @update:model-value="value => setNumber('rateLimitPerMinute', value)" />
        </UFormField>
        <UFormField :label="t('admin.ai.extensionQuota')">
          <UInput :model-value="form.gates.extensionDailyQuota" type="number" :disabled="!canManage" class="w-full"
            @update:model-value="value => setNumber('extensionDailyQuota', value)" />
        </UFormField>
        <UFormField :label="t('admin.ai.userQuota')">
          <UInput :model-value="form.gates.userDailyQuota" type="number" :disabled="!canManage" class="w-full"
            @update:model-value="value => setNumber('userDailyQuota', value)" />
        </UFormField>
        <UFormField :label="t('admin.ai.monthlyBudget')">
          <UInput :model-value="budgetYuan()" type="number" step="0.01" :disabled="!canManage" class="w-full"
            @update:model-value="setBudgetYuan" />
        </UFormField>
        <UFormField :label="t('admin.ai.toolCalls')" :help="t('admin.ai.toolCallsHint')">
          <UInput :model-value="form.gates.toolCallsPerReply ?? 3" type="number" min="0" max="5" :disabled="!canManage" class="w-full"
            @update:model-value="value => setNumber('toolCallsPerReply', value)" />
        </UFormField>
      </div>
    </section>

    <section class="min-w-0 rounded-lg border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div class="min-w-0">
          <p class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.ai.replyPrompt.title') }}</p>
          <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.ai.replyPrompt.intro') }}</p>
        </div>
        <UBadge :color="promptCustomized ? 'primary' : 'neutral'" variant="soft" size="sm">
          {{ promptCustomized ? t('admin.ai.replyPrompt.customized') : t('admin.ai.replyPrompt.default') }}
        </UBadge>
      </div>
      <div class="mt-3">
        <LazySFEditor
          v-model="form.reply.systemPrompt"
          preset="basic-field"
          :load-trusted-catalog="false"
          :rows="10"
          :max-characters="8000"
          :disabled="!canManage"
          :aria-label="t('admin.ai.replyPrompt.title')"
          :placeholder="t('admin.ai.replyPrompt.placeholder')"
        />
      </div>
      <div class="mt-3 flex flex-wrap items-center gap-2">
        <UButton
          icon="i-lucide-rotate-ccw"
          color="neutral"
          variant="outline"
          size="sm"
          :disabled="!canManage || !promptCustomized"
          @click="restoreDefaultPrompt"
        >
          {{ t('admin.ai.replyPrompt.restore') }}
        </UButton>
        <span class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.ai.replyPrompt.safetyHint') }}</span>
      </div>
      <div v-if="ai.safetyAppendix.value" class="mt-3 rounded-md bg-slate-50 p-3 dark:bg-zinc-950/60">
        <p class="text-xs font-medium text-slate-600 dark:text-zinc-300">{{ t('admin.ai.replyPrompt.safetyTitle') }}</p>
        <pre class="mt-1 whitespace-pre-wrap font-sans text-xs text-slate-500 dark:text-zinc-400">{{ ai.safetyAppendix.value }}</pre>
      </div>
    </section>

    <div v-if="canManage" class="flex flex-wrap items-center gap-2">
      <UButton icon="i-lucide-save" :loading="ai.busy.value === 'settings'" :disabled="!dirty" @click="save">
        {{ t('admin.ai.save') }}
      </UButton>
      <UButton icon="i-lucide-rotate-ccw" color="neutral" variant="outline" :loading="ai.busy.value === 'reset'" @click="ai.resetSettings">
        {{ t('admin.ai.reset') }}
      </UButton>
      <span v-if="dirty" class="text-xs text-amber-600 dark:text-amber-400">{{ t('admin.ai.dirtyHint') }}</span>
    </div>
  </div>
</template>
