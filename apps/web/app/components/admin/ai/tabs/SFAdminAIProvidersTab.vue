<script setup lang="ts">
import SFAdminAIAutoActionBlock from '~/components/admin/ai/blocks/SFAdminAIAutoActionBlock.vue'
import SFAdminAIModelField from '~/components/admin/ai/blocks/SFAdminAIModelField.vue'
import SFAdminAIRedactionBlock from '~/components/admin/ai/blocks/SFAdminAIRedactionBlock.vue'
import SFAdminAIMultimodalBlock from '~/components/admin/ai/blocks/SFAdminAIMultimodalBlock.vue'
import SFAdminAIProviderPicker from '~/components/admin/ai/blocks/SFAdminAIProviderPicker.vue'
import { ADMIN_AI_KEY, MICRO_PER_UNIT } from '~/composables/admin/useAdminAI'
import { ADMIN_AI_DRAFT_KEY } from '~/composables/admin/useAdminAISettingsForm'
import { coerceInteger, coerceNumber, toolCallsPerReplyUsesDefault } from '~/utils/admin/adminAI'

const { t, te } = useI18n()
// 数据由页面壳统一持有并注入；页签只渲染，不重新拉取。
const ai = inject(ADMIN_AI_KEY)!
// 草稿同样由页面壳持有：回复提示词在另一个页签里编辑，两者必须落在同一份表单上，
// 保存也只有一个（页面壳底部的操作条）。
const draft = inject(ADMIN_AI_DRAFT_KEY)!
const { form, selectedProviderId, selectedProfile, isCustom, profileIncomplete, providerOptions, canManage, readOnly } = draft
// 上下限来自服务端（GET /admin/ai/settings 的 limits），前端不再硬编码一份数字。
const limits = ai.limits

// 数字输入框被清空时，模型仍持有上一个合法值（数据是对的），但输入框的 DOM 可能
// 停在空串上——那会让人以为「这一项是空的」却保存了旧值。每次拒绝输入就换一个
// key 让该输入框重新挂载，界面与模型因此不会长期不一致。
const fieldEpoch = reactive<Record<string, number>>({})
function fieldKey(name: string) {
  return `${name}-${fieldEpoch[name] ?? 0}`
}
function resyncField(name: string) {
  fieldEpoch[name] = (fieldEpoch[name] ?? 0) + 1
}

// 密钥输入按 profile 暂存；它只在下一次保存密钥之前存在。
const credentials = reactive<Record<string, string>>({})
const advancedOpen = ref(false)

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

async function submitCredential(profileId: string) {
  const apiKey = (credentials[profileId] || '').trim()
  if (!apiKey) return
  // 后端同样会拒绝，这里挡住是为了不让运营者填完密钥之后才看到失败。
  if (!ai.encryptionEnabled.value) return
  if (await ai.saveCredential(profileId, apiKey)) credentials[profileId] = ''
}

// 每个闸门都有服务端天花板。这里用 coerceInteger 一条路径收敛：空值回退到
// 上一次合法值，负数与分析器产出的小数/科学计数法都被夹到 [0, 上限]。
function setNumber(path: 'rateLimitPerMinute' | 'extensionDailyQuota' | 'userDailyQuota', value: string | number) {
  if (!form.value) return
  const max = path === 'rateLimitPerMinute'
    ? limits.value.maxRateLimitPerMinute
    : path === 'extensionDailyQuota'
      ? limits.value.maxExtensionDailyQuota
      : limits.value.maxUserDailyQuota
  const next = coerceInteger(value, form.value.gates[path], 0, max)
  if (next === form.value.gates[path]) {
    resyncField(path)
    return
  }
  form.value.gates[path] = next
}

// maxTokens / timeoutMs 以前直接用 v-model.number：输入框被清空时模型里留下空串，
// PUT 出去就是 `"maxTokens":""`，Go 侧 int 解码失败 → 422。改走显式转换后，
// 空值只是"恢复成上一次的合法值"，不会污染载荷。
function setProfileInteger(field: 'maxTokens' | 'timeoutMs', value: string | number) {
  const profile = selectedProfile.value
  if (!profile) return
  const next = field === 'maxTokens'
    ? coerceInteger(value, profile.defaults.maxTokens, 1, limits.value.maxCompletionTokens)
    : coerceInteger(value, profile.defaults.timeoutMs, limits.value.minTimeoutMs, limits.value.maxTimeoutMs)
  if (next === profile.defaults[field]) {
    // 输入被拒绝（空串/非法字符）：模型不变，但 DOM 需要回到真实值。
    resyncField(field)
    return
  }
  profile.defaults[field] = next
}

// 只允许数字键：默认允许小数会让 timeoutMs 这类 int 字段产生无法解码的载荷。
function blockNonIntegerKey(event: KeyboardEvent) {
  const allowed = ['Backspace', 'Delete', 'Tab', 'Escape', 'Enter', 'ArrowLeft', 'ArrowRight', 'Home', 'End']
  if (!allowed.includes(event.key) && !event.metaKey && !event.ctrlKey && !/^\d$/.test(event.key)) event.preventDefault()
}

// toolCallsPerReply 用指针语义：清空表示「未设置，按默认 3 生效」，0 表示停用
// 工具调用。两者对服务端是不同的状态，界面必须都能表达。
function setToolCallsPerReply(value: string | number) {
  if (!form.value) return
  const raw = typeof value === 'string' ? value.trim() : value
  if (raw === '' || raw === null || raw === undefined || Number.isNaN(Number(raw))) {
    delete form.value.gates.toolCallsPerReply
    return
  }
  const next = coerceInteger(raw, 0, 0, limits.value.maxToolCallsPerReply)
  if (next === form.value.gates.toolCallsPerReply) {
    resyncField('toolCallsPerReply')
    return
  }
  form.value.gates.toolCallsPerReply = next
}

// 恢复「未设置」：让站点以后能跟随后端默认值的调整，而不是被一次点击固化。
function restoreDefaultToolCalls() {
  if (!form.value) return
  delete form.value.gates.toolCallsPerReply
}

// 输入框里显示的是「当前生效值」：未设置时显示默认，但那不是已保存的显式值。
function toolCallsUsesDefault() {
  return toolCallsPerReplyUsesDefault(form.value?.gates.toolCallsPerReply)
}

function toolCallsValue() {
  return toolCallsUsesDefault() ? limits.value.defaultToolCallsPerReply : form.value?.gates.toolCallsPerReply
}

// 提示里的「推荐 / 上限」必须来自服务端 limits，不能写死在文案里：硬顶与默认值
// 在 B1 已经抬过一次（3→5、5→12），写死的副本只会继续骗运营者。
const toolCallsHint = computed(() => t('admin.ai.toolCallsHint', {
  default: limits.value.defaultToolCallsPerReply,
  max: limits.value.maxToolCallsPerReply
}))

function setBudgetYuan(value: string | number) {
  if (!form.value) return
  const maxUnits = limits.value.maxMonthlyBudgetMicros / MICRO_PER_UNIT
  const current = form.value.gates.monthlyBudgetMicros / MICRO_PER_UNIT
  const units = coerceNumber(value, current, 0, maxUnits)
  if (units === current) {
    resyncField('monthlyBudgetMicros')
    return
  }
  form.value.gates.monthlyBudgetMicros = Math.round(units * MICRO_PER_UNIT)
}

function budgetYuan() {
  return form.value ? form.value.gates.monthlyBudgetMicros / MICRO_PER_UNIT : 0
}

function toggleAdvanced() {
  advancedOpen.value = !advancedOpen.value
}

function setMicro(field: 'inputPerMillionMicros' | 'outputPerMillionMicros', value: string | number) {
  const profile = selectedProfile.value
  if (!profile) return
  // 单价只要求非负，没有上限；负值会被后端拒绝，因此在这里就夹住。
  const current = profile.price[field] / MICRO_PER_UNIT
  const units = coerceNumber(value, current, 0, Number.MAX_SAFE_INTEGER / MICRO_PER_UNIT)
  profile.price[field] = Math.round(units * MICRO_PER_UNIT)
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

    <SFAdminAIProviderPicker
      :options="providerOptions"
      :selected-id="selectedProviderId"
      :credential-status="ai.credentialStatus.value"
      :disabled="!canManage"
      @select="draft.selectProvider"
    />
    <section v-if="selectedProfile" class="min-w-0 rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <div class="border-b border-slate-200 px-4 py-3 dark:border-zinc-800">
        <p class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.ai.providerConfig') }}</p>
        <p v-if="profileIncomplete" class="mt-1 text-xs text-amber-600 dark:text-amber-400">{{ t('admin.ai.profileIncomplete') }}</p>
      </div>
      <div class="flex flex-col gap-3 px-4 py-4">
        <UFormField v-if="isCustom" :label="t('admin.ai.customName')">
          <UInput v-model="selectedProfile.label" :disabled="!canManage" class="w-full" />
        </UFormField>

        <UFormField
          :label="t('admin.ai.credential')"
          :help="ai.encryptionEnabled.value ? t('admin.ai.credentialHint') : t('admin.ai.credentialBlockedHint')"
        >
          <div class="flex min-w-0 flex-col gap-2 sm:flex-row">
            <UInput
              v-model="credentials[selectedProfile.id]"
              type="password"
              autocomplete="off"
              :placeholder="ai.credentialStatus.value[selectedProfile.id] ? t('admin.ai.credentialPlaceholderKeep') : t('admin.ai.credentialPlaceholder')"
              :disabled="!canManage || !ai.encryptionEnabled.value"
              class="min-w-0 flex-1"
            />
            <UButton
              icon="i-lucide-key-round"
              color="neutral"
              variant="subtle"
              :loading="ai.busy.value === `credential:${selectedProfile.id}`"
              :disabled="!canManage || !ai.encryptionEnabled.value || !(credentials[selectedProfile.id] || '').trim()"
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
          <UFormField :label="t('admin.ai.baseUrl')" :error="draft.fieldError('baseUrl')">
            <UInput v-model="selectedProfile.baseUrl" :disabled="!canManage" class="w-full" />
          </UFormField>
          <UFormField :label="t('admin.ai.model')">
            <!-- 模型标识既可以手填，也可以一键从供应商读取列表后选择。探测用当前
                 表单里的地址与密钥（未保存也能用），因此不会逼运营者先存一份没有
                 模型的配置。 -->
            <SFAdminAIModelField
              v-model="selectedProfile.model"
              :profile-id="selectedProfile.id"
              :protocol="selectedProfile.protocol"
              :base-url="selectedProfile.baseUrl"
              :api-key="credentials[selectedProfile.id] || ''"
              :credential-configured="ai.credentialStatus.value[selectedProfile.id] === true"
              :disabled="!canManage"
            />
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
          <UFormField :label="t('admin.ai.maxTokens')" :error="draft.fieldError('defaults.maxTokens')">
            <UInput
              :key="fieldKey('maxTokens')"
              :model-value="selectedProfile.defaults.maxTokens"
              type="number"
              min="1"
              :max="limits.maxCompletionTokens"
              step="1"
              :disabled="!canManage"
              class="w-full"
              @keydown="blockNonIntegerKey"
              @update:model-value="value => setProfileInteger('maxTokens', value)"
            />
          </UFormField>
          <UFormField :label="t('admin.ai.timeoutMs')" :error="draft.fieldError('defaults.timeoutMs')">
            <UInput
              :key="fieldKey('timeoutMs')"
              :model-value="selectedProfile.defaults.timeoutMs"
              type="number"
              :min="limits.minTimeoutMs"
              :max="limits.maxTimeoutMs"
              step="1"
              :disabled="!canManage"
              class="w-full"
              @keydown="blockNonIntegerKey"
              @update:model-value="value => setProfileInteger('timeoutMs', value)"
            />
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
        <UFormField :label="t('admin.ai.rateLimit')" :error="draft.fieldError('gates.rateLimitPerMinute')">
          <UInput :key="fieldKey('rateLimitPerMinute')" :model-value="form.gates.rateLimitPerMinute" type="number" min="0" :max="limits.maxRateLimitPerMinute"
            step="1" :disabled="!canManage" class="w-full" @keydown="blockNonIntegerKey"
            @update:model-value="value => setNumber('rateLimitPerMinute', value)" />
        </UFormField>
        <UFormField :label="t('admin.ai.extensionQuota')" :error="draft.fieldError('gates.extensionDailyQuota')">
          <UInput :key="fieldKey('extensionDailyQuota')" :model-value="form.gates.extensionDailyQuota" type="number" min="0" :max="limits.maxExtensionDailyQuota"
            step="1" :disabled="!canManage" class="w-full" @keydown="blockNonIntegerKey"
            @update:model-value="value => setNumber('extensionDailyQuota', value)" />
        </UFormField>
        <UFormField :label="t('admin.ai.userQuota')" :error="draft.fieldError('gates.userDailyQuota')">
          <UInput :key="fieldKey('userDailyQuota')" :model-value="form.gates.userDailyQuota" type="number" min="0" :max="limits.maxUserDailyQuota"
            step="1" :disabled="!canManage" class="w-full" @keydown="blockNonIntegerKey"
            @update:model-value="value => setNumber('userDailyQuota', value)" />
        </UFormField>
        <UFormField
          :label="t('admin.ai.monthlyBudget')"
          :help="t('admin.ai.monthlyBudgetHint', { max: limits.maxMonthlyBudgetMicros / MICRO_PER_UNIT })"
          :error="draft.fieldError('gates.monthlyBudgetMicros')"
        >
          <UInput :key="fieldKey('monthlyBudgetMicros')" :model-value="budgetYuan()" type="number" min="0" step="0.01" :disabled="!canManage" class="w-full"
            @update:model-value="setBudgetYuan" />
        </UFormField>
        <UFormField :label="t('admin.ai.toolCalls')" :help="toolCallsHint">
          <div class="flex min-w-0 items-center gap-2">
            <UInput
              :key="fieldKey('toolCallsPerReply')"
              :model-value="toolCallsValue()"
              type="number"
              min="0"
              :max="limits.maxToolCallsPerReply"
              :placeholder="String(limits.defaultToolCallsPerReply)"
              :disabled="!canManage"
              class="min-w-0 flex-1"
              @update:model-value="setToolCallsPerReply"
            />
            <UButton
              size="xs"
              color="neutral"
              variant="ghost"
              icon="i-lucide-rotate-ccw"
              :disabled="!canManage || toolCallsUsesDefault()"
              :aria-label="t('admin.ai.toolCallsRestore')"
              :title="t('admin.ai.toolCallsRestore')"
              @click="restoreDefaultToolCalls"
            />
          </div>
          <template #help>
            <span class="text-xs">{{ toolCallsHint }}</span>
            <span v-if="toolCallsUsesDefault()" class="mt-0.5 block text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.ai.toolCallsDefaultState') }}
            </span>
            <span v-else class="mt-0.5 block text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.ai.toolCallsExplicitState') }}
            </span>
          </template>
        </UFormField>
      </div>
    </section>

    <SFAdminAIAutoActionBlock v-model:form="form" v-model:disabled="readOnly" />

    <SFAdminAIRedactionBlock v-model:form="form" v-model:disabled="readOnly" />

    <SFAdminAIMultimodalBlock v-model:form="form" v-model:disabled="readOnly" />
  </div>
</template>
