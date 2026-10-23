<script setup lang="ts">
import type { AdminAIPurposeOverride, AdminAIToolAvailability, AdminAIValidatedPurpose } from '~/composables/admin/useAdminAI'
import { ADMIN_AI_DRAFT_KEY } from '~/composables/admin/useAdminAISettingsForm'
import { toSettingsOverride } from '~/components/admin/ai/toolOverrides'
import { MAX_TOOLS_PER_REQUEST, isValidPurposeToolName, selectedPurposeTools } from '~/utils/admin/adminAI'
import { referencedButUnregistered, setPurposeTool } from '~/utils/admin/adminAITools'

// 引用区是「白名单比工具活得久」的唯一处理入口：插件停用或卸载后，用途白名单里的
// 名字仍然存在，服务端会静默跳过，模型永远看不到它。这里同时提供两件事——把这类
// 死名字逐条清掉，以及在一个工具还没注册时就先把它点名（后端保存时刻意不校验
// 存在性，支持「先配策略、后装插件」）。
//
// 它只读写注入的草稿，不自带标题/工具栏，也不重新拉取数据：与其它 AI 页签共享
// 页面壳的同一份 form 与同一次保存。
const { t, locale } = useI18n()
const toast = useToast()
const draft = inject(ADMIN_AI_DRAFT_KEY)!
const { form, ai, readOnly } = draft

const catalog = computed<AdminAIValidatedPurpose[]>(() => ai.purposes.value ?? [])
const tools = computed<AdminAIToolAvailability[]>(() => ai.availableTools.value ?? [])

function localized(text: { default: string, 'en-US'?: string } | undefined) {
  if (!text) return ''
  return locale.value?.toLowerCase().startsWith('en') && text['en-US'] ? text['en-US'] : text.default
}

function purposeById(id: string) {
  return catalog.value.find(purpose => purpose.id === id)
}

// 未注册名字只带着用途 id 回来，界面要显示运营者认得的用途名。
function purposeLabel(id: string) {
  const purpose = purposeById(id)
  return purpose ? localized(purpose.label) : id
}

const unregistered = computed(() =>
  referencedButUnregistered(catalog.value, form.value?.purposes, tools.value.map(tool => tool.name))
)
const registeredNames = computed(() => new Set(tools.value.map(tool => tool.name)))

// 用途下拉沿用矩阵的列顺序：回复用途最常被讨论，判定次之，其余按 id 稳定排序。
const purposeOptions = computed(() => {
  const rank = (id: string) => id === 'forum.reply' ? 0 : id === 'forum.reply.triage' ? 1 : 2
  return [...catalog.value]
    .sort((left, right) => {
      const delta = rank(left.id) - rank(right.id)
      return delta !== 0 ? delta : left.id.localeCompare(right.id)
    })
    .map(purpose => ({ label: localized(purpose.label), value: purpose.id }))
})

const manualPurpose = ref('')
const manualName = ref('')
const manualError = ref('')

// 目录到位后给一个默认用途。它只是界面的初始选择，不进配置文档；两边渲染的目录
// 来自同一份载荷，因此 SSR 与水合会落到同一个值。
watch(purposeOptions, (options) => {
  if (options.length === 0) return
  if (manualPurpose.value && options.some(option => option.value === manualPurpose.value)) return
  manualPurpose.value = options.some(option => option.value === 'forum.reply') ? 'forum.reply' : options[0]!.value
}, { immediate: true })

function overrideFor(id: string): AdminAIPurposeOverride | undefined {
  return form.value?.purposes?.[id]
}

function effectiveTools(purpose: AdminAIValidatedPurpose) {
  return selectedPurposeTools(purpose.declaredTools, overrideFor(purpose.id))
}

function writePurpose(purpose: AdminAIValidatedPurpose, next: ReturnType<typeof setPurposeTool>) {
  const current = form.value
  if (!current) return
  const purposes: Record<string, AdminAIPurposeOverride> = { ...(current.purposes ?? {}) }
  const converted = toSettingsOverride(next)
  if (converted) purposes[purpose.id] = converted
  else delete purposes[purpose.id]
  current.purposes = purposes
}

// 手动点名只接受「尚未注册」的名字：已注册工具在矩阵里勾选即可，两个入口同时
// 允许写同一个名字只会让运营者不确定哪一个才算数。
function addManual() {
  const current = form.value
  if (!current || readOnly.value) return
  const purpose = purposeById(manualPurpose.value)
  if (!purpose) {
    manualError.value = t('admin.ai.tools.manualAdd.purposeMissing')
    return
  }
  const name = manualName.value.trim()
  if (!isValidPurposeToolName(name)) {
    manualError.value = t('admin.ai.tools.manualAdd.nameInvalid')
    return
  }
  if (registeredNames.value.has(name)) {
    manualError.value = t('admin.ai.tools.manualAdd.registered')
    return
  }
  const effective = effectiveTools(purpose)
  if (effective.includes(name)) {
    manualError.value = ''
    manualName.value = ''
    toast.add({
      color: 'neutral',
      icon: 'i-lucide-info',
      title: t('admin.ai.tools.manualAdd.alreadyPresent', { name, purpose: localized(purpose.label) }),
      duration: 10000
    })
    return
  }
  // 上限只拦新增：取消/移除永远允许，否则一个已经超限的配置会把运营者锁在界面里。
  if (effective.length >= MAX_TOOLS_PER_REQUEST) {
    manualError.value = t('admin.ai.tools.tooMany', { max: MAX_TOOLS_PER_REQUEST })
    return
  }
  writePurpose(purpose, setPurposeTool(purpose.declaredTools, overrideFor(purpose.id), name, true))
  manualError.value = ''
  manualName.value = ''
  toast.add({
    color: 'success',
    icon: 'i-lucide-check',
    title: t('admin.ai.tools.manualAdd.added', { name, purpose: localized(purpose.label) }),
    duration: 10000
  })
}

// 逐用途移除仍然走 setPurposeTool：每个用途各自自动收敛，某个用途因此回到声明值时
// 它的 tools 字段会被整个删掉，而不是留下一个与声明同值的覆盖。
function removeUnregistered(entry: { name: string, purposes: string[] }) {
  const current = form.value
  if (!current || readOnly.value) return
  const purposes: Record<string, AdminAIPurposeOverride> = { ...(current.purposes ?? {}) }
  for (const id of entry.purposes) {
    const purpose = purposeById(id)
    if (!purpose) continue
    const converted = toSettingsOverride(setPurposeTool(purpose.declaredTools, purposes[id], entry.name, false))
    if (converted) purposes[id] = converted
    else delete purposes[id]
  }
  current.purposes = purposes
  manualError.value = ''
  toast.add({
    color: 'success',
    icon: 'i-lucide-check',
    title: t('admin.ai.tools.removeEverywhereDone', { name: entry.name }),
    duration: 10000
  })
}
</script>

<template>
  <section class="min-w-0 rounded-lg border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
    <div class="flex flex-wrap items-start justify-between gap-2">
      <div class="min-w-0">
        <p class="text-base font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.ai.tools.manualAdd.title') }}</p>
        <p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">{{ t('admin.ai.tools.manualAdd.hint') }}</p>
      </div>
      <UBadge color="neutral" variant="soft" size="sm">
        {{ t('admin.ai.tools.unregisteredCount', { count: unregistered.length }) }}
      </UBadge>
    </div>

    <div class="mt-3 grid min-w-0 gap-2 sm:grid-cols-[minmax(0,14rem)_minmax(0,1fr)_auto]">
      <USelect
        v-model="manualPurpose"
        :items="purposeOptions"
        value-key="value"
        label-key="label"
        size="md"
        class="w-full"
        :disabled="readOnly"
        :aria-label="t('admin.ai.tools.manualAdd.purpose')"
      />
      <UInput
        v-model="manualName"
        size="md"
        class="w-full font-mono"
        maxlength="64"
        :disabled="readOnly"
        :placeholder="t('admin.ai.tools.manualAdd.namePlaceholder')"
        :aria-label="t('admin.ai.tools.manualAdd.namePlaceholder')"
        @keydown.enter.prevent="addManual"
      />
      <UButton
        size="md"
        color="neutral"
        variant="outline"
        icon="i-lucide-plus"
        :disabled="readOnly"
        :aria-label="t('admin.ai.tools.manualAdd.add')"
        @click="addManual"
      >
        {{ t('admin.ai.tools.manualAdd.add') }}
      </UButton>
    </div>
    <p v-if="manualError" class="mt-1 text-xs text-red-600 dark:text-red-400">{{ manualError }}</p>

    <div v-if="unregistered.length > 0" class="mt-3 border-t border-slate-200 pt-3 dark:border-zinc-800">
      <p class="text-sm font-semibold text-amber-800 dark:text-amber-200">{{ t('admin.ai.tools.unregisteredTitle') }}</p>
      <p class="mt-1 text-xs text-amber-700 dark:text-amber-300">{{ t('admin.ai.tools.unregisteredHint') }}</p>
      <ul class="mt-2 flex flex-col gap-1.5">
        <li
          v-for="entry in unregistered"
          :key="entry.name"
          class="flex flex-wrap items-center justify-between gap-2"
        >
          <div class="flex min-w-0 flex-wrap items-center gap-2">
            <span class="truncate font-mono text-sm text-slate-900 dark:text-zinc-100">{{ entry.name }}</span>
            <UBadge v-for="id in entry.purposes" :key="id" color="neutral" variant="soft" size="sm">
              {{ purposeLabel(id) }}
            </UBadge>
          </div>
          <UButton
            size="xs"
            color="warning"
            variant="soft"
            icon="i-lucide-x"
            :disabled="readOnly"
            :aria-label="t('admin.ai.tools.unregisteredRemove', { name: entry.name })"
            :title="t('admin.ai.tools.unregisteredRemove', { name: entry.name })"
            @click="removeUnregistered(entry)"
          >
            {{ t('admin.ai.tools.removeEverywhere') }}
          </UButton>
        </li>
      </ul>
    </div>
    <p v-else class="mt-3 text-sm text-slate-500 dark:text-zinc-400">{{ t('admin.ai.tools.manualAdd.empty') }}</p>
  </section>
</template>
