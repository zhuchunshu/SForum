<script setup lang="ts">
import type { AdminAIProtocol } from '~/composables/admin/useAdminAI'
import { ADMIN_AI_KEY } from '~/composables/admin/useAdminAI'
import { modelOptionsForSelect } from '~/utils/admin/adminAI'

const props = defineProps<{
  profileId: string
  protocol: AdminAIProtocol
  baseUrl: string
  apiKey: string
  credentialConfigured: boolean
  disabled: boolean
}>()

// 模型标识本身是唯一需要回写父表单的值；地址、协议与密钥都只是探测的输入。
const model = defineModel<string>({ required: true })

const { t, te } = useI18n()
const toast = useToast()
const ai = inject(ADMIN_AI_KEY)!

// 探测结果按 profile 记录：切换供应商后旧列表必须消失，否则运营者会在 A 的地址
// 下选中 B 的模型，而保存后的配置会指向一个该端点根本不存在的模型。
const fetched = ref<{ profileId: string, models: string[], truncated: boolean } | null>(null)
const manual = ref(false)
const failure = ref<{ reason: string, detail: string } | null>(null)

const currentFetch = computed(() => (fetched.value && fetched.value.profileId === props.profileId ? fetched.value : null))
const models = computed(() => currentFetch.value?.models ?? [])
const truncated = computed(() => Boolean(currentFetch.value?.truncated))
const busy = computed(() => ai.busy.value === `models:${props.profileId}`)
const hasCredential = computed(() => props.apiKey.trim() !== '' || props.credentialConfigured)
// 探测需要地址与密钥。缺哪一样就说明哪一样，而不是让按钮点下去才失败。
const missing = computed(() => {
  if (props.disabled) return ''
  if (!props.baseUrl.trim()) return 'baseUrl'
  if (!hasCredential.value) return 'credential'
  return ''
})
// 有列表时默认用下拉框；「手动输入」让运营者仍能填一个列表里没有的模型标识
// （自建端点常常不实现 /models）。
const useSelect = computed(() => !manual.value && models.value.length > 0)
const selection = computed(() => modelOptionsForSelect(models.value, model.value))
const canFetch = computed(() => !props.disabled && missing.value === '')

// 失败原因优先用翻译；缺翻译时退回原始原因码，至少可诊断。
const reasonText = computed(() => {
  const reason = failure.value?.reason
  if (!reason) return ''
  const key = `admin.ai.modelFetchReasons.${reason}`
  return te(key) ? t(key) : reason
})

// 手动输入与使用列表之间切换。列表里没有的模型（自建端点常常不实现 /models）必须
// 仍然可以填写，所以这个开关是必要的，而不是可选的便利。
function useManualEntry(value: boolean) {
  manual.value = value
}

async function fetchModels() {
  if (!canFetch.value) return
  failure.value = null
  const result = await ai.fetchModels({
    profileId: props.profileId,
    baseUrl: props.baseUrl.trim(),
    protocol: props.protocol,
    apiKey: props.apiKey.trim()
  })
  if (!result) {
    // HTTP 层失败已经弹过提示；这里留下字段旁的一句，避免提示消失后没有痕迹。
    failure.value = { reason: '', detail: '' }
    return
  }
  if (result.error) {
    failure.value = { reason: result.reason || 'provider_failed', detail: result.error }
    return
  }
  const listed = result.models ?? []
  fetched.value = { profileId: props.profileId, models: listed, truncated: Boolean(result.truncated) }
  manual.value = false
  toast.add({
    color: 'success',
    icon: 'i-lucide-list-checks',
    title: t('admin.ai.modelFetchDone', { count: listed.length }),
    duration: 5000
  })
}
</script>

<template>
  <div class="flex min-w-0 flex-col gap-2">
    <div class="flex min-w-0 flex-col gap-2 sm:flex-row">
      <USelect
        v-if="useSelect"
        v-model="model"
        :items="selection.options"
        :placeholder="t('admin.ai.modelSelectPlaceholder')"
        :disabled="disabled"
        class="min-w-0 flex-1"
      />
      <UInput
        v-else
        v-model="model"
        :disabled="disabled"
        :placeholder="t('admin.ai.modelSelectPlaceholder')"
        class="min-w-0 flex-1"
      />
      <UButton
        icon="i-lucide-list"
        color="neutral"
        variant="subtle"
        :loading="busy"
        :disabled="!canFetch"
        :title="t('admin.ai.modelFetchHint')"
        :aria-label="t('admin.ai.modelFetch')"
        @click="fetchModels"
      >
        {{ t('admin.ai.modelFetch') }}
      </UButton>
    </div>

    <div v-if="models.length" class="flex flex-wrap items-center gap-2 text-xs text-slate-500 dark:text-zinc-400">
      <UBadge color="neutral" variant="soft" size="sm">
        {{ t('admin.ai.modelFetched', { count: models.length }) }}
      </UBadge>
      <span v-if="truncated">{{ t('admin.ai.modelTruncated') }}</span>
      <UButton
        v-if="useSelect"
        size="xs"
        color="neutral"
        variant="ghost"
        icon="i-lucide-pencil"
        :aria-label="t('admin.ai.modelManual')"
        :title="t('admin.ai.modelManual')"
        @click="useManualEntry(true)"
      >
        {{ t('admin.ai.modelManual') }}
      </UButton>
      <UButton
        v-else
        size="xs"
        color="neutral"
        variant="ghost"
        icon="i-lucide-list"
        :aria-label="t('admin.ai.modelUseList')"
        :title="t('admin.ai.modelUseList')"
        @click="useManualEntry(false)"
      >
        {{ t('admin.ai.modelUseList') }}
      </UButton>
      <span v-if="model.trim() && !selection.currentListed" class="text-amber-600 dark:text-amber-400">
        {{ t('admin.ai.modelNotInList') }}
      </span>
    </div>

    <p v-if="missing" class="text-xs text-slate-500 dark:text-zinc-400">
      {{ missing === 'baseUrl' ? t('admin.ai.modelFetchNeedsBaseUrl') : t('admin.ai.modelFetchNeedsCredential') }}
    </p>
    <p v-if="failure" class="text-xs text-rose-600 dark:text-rose-400">
      {{ reasonText || t('admin.ai.modelFetchFailed') }}
    </p>
    <p v-if="failure?.detail" class="break-all font-mono text-xs text-slate-400 dark:text-zinc-500">
      {{ failure.detail }}
    </p>
  </div>
</template>
