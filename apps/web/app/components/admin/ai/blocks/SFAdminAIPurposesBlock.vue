<script setup lang="ts">
import type { AdminAIPurposeOverride, AdminAIToolAvailability, AdminAIToolOrigin, AdminAIValidatedPurpose } from '~/composables/admin/useAdminAI'
import type { AIPurposeOverride } from '~/utils/admin/adminAI'
import { ADMIN_AI_DRAFT_KEY } from '~/composables/admin/useAdminAISettingsForm'
import { toSettingsOverride } from '~/components/admin/ai/toolOverrides'
import {
  AI_FAILURE_POSTURES,
  MAX_TOOLS_PER_REQUEST,
  isValidPurposeToolName,
  normalizePurposeToolNames,
  purposeToolOptions,
  purposeToolsAreOverridden,
  selectedPurposeTools
} from '~/utils/admin/adminAI'
import { setPurposeTool, toolOriginLabel } from '~/utils/admin/adminAITools'

// 用途清单是「站内有哪些 AI 行为」的唯一可枚举答案。每一行都同时展示声明与生效
// 两份事实：运营者要能看出「这个用途原本是什么」与「我把它改成了什么」，否则一次
// 覆盖会让声明的默认值从界面上消失，之后没人知道恢复默认该恢复成什么。
const { t, locale } = useI18n()
const toast = useToast()
const draft = inject(ADMIN_AI_DRAFT_KEY)!
const { form, ai, canManage, readOnly } = draft

// 每行正在输入的工具名与错误提示。它们是纯界面状态，不进配置文档。
const pendingToolName = ref<Record<string, string>>({})
const toolNameError = ref<Record<string, string>>({})

const catalog = computed<AdminAIValidatedPurpose[]>(() => ai.purposes.value ?? [])
// 站内已注册的工具（Core 内置 + 已启用插件）。旧载荷没有这一项时为空，界面退化
// 成「声明列表 + 手动输入」，与引入工具目录之前一致。
const availableTools = computed<AdminAIToolAvailability[]>(() => ai.availableTools.value ?? [])

function localized(text: { default: string, 'en-US'?: string } | undefined) {
  if (!text) return ''
  return locale.value?.toLowerCase().startsWith('en') && text['en-US'] ? text['en-US'] : text.default
}

function overrideFor(id: string): AdminAIPurposeOverride | undefined {
  return form.value?.purposes?.[id]
}

// 用户在界面上做的任何改动都写回同一份草稿：页面壳持有它，保存是整份文档替换。
// 参数接受工具纯函数的覆盖形状，写入前补成设置文档要求的形状（见 toolOverrides）。
function writeOverride(id: string, override: AIPurposeOverride | undefined) {
  const current = form.value
  if (!current) return
  const purposes = { ...(current.purposes ?? {}) }
  const next = toSettingsOverride(override)
  if (!next) {
    delete purposes[id]
  } else {
    purposes[id] = next
  }
  current.purposes = purposes
}

function toolsOverridden(purpose: AdminAIValidatedPurpose) {
  return purposeToolsAreOverridden(overrideFor(purpose.id))
}

// 行级的「已自定义」覆盖两项：工具白名单或失败姿态任一被站点接管即成立。
function isCustomized(purpose: AdminAIValidatedPurpose) {
  return toolsOverridden(purpose) || postureValue(purpose) !== ''
}

function selectedTools(purpose: AdminAIValidatedPurpose) {
  return selectedPurposeTools(purpose.declaredTools, overrideFor(purpose.id))
}

function toolChoices(purpose: AdminAIValidatedPurpose) {
  // 已注册但未声明的工具（插件贡献）也必须出现在选项里：让运营者手打一个拼错的
  // 名字，再让服务端静默跳过，是这条链路上最容易犯的错。
  //
  // declaredTools 在旧载荷上可能是 null（没有声明任何工具的用途），因此这里必须
  // 用 ?? [] 兜底：直接展开 null 会让整个页签渲染失败。
  return purposeToolOptions(
    [...(purpose.declaredTools ?? []), ...availableTools.value.map(tool => tool.name)],
    overrideFor(purpose.id)?.tools?.allowed
  )
}

function toolOrigin(name: string): AdminAIToolOrigin | undefined {
  return availableTools.value.find(tool => tool.name === name)?.origin
}

// 插件工具在标签里带上扩展标识：运营者要能看出这个名字是谁提供的，而不是靠工具名
// 的形状去猜。Core 内置仍然只显示名字，保持这一列不被 @Core 噪音填满。
function toolLabel(name: string) {
  const label = toolOriginLabel(toolOrigin(name))
  return label === 'Core' ? name : `${name} · ${label}`
}

// 真正「未注册」的选中项：既不在用途声明里，也不在站内工具目录里。服务端会跳过
// 这些名字，因此必须让运营者看见它们。
function unknownTools(purpose: AdminAIValidatedPurpose) {
  const known = new Set([...(purpose.declaredTools ?? []), ...availableTools.value.map(tool => tool.name)])
  return selectedTools(purpose).filter(name => !known.has(name))
}

function postureValue(purpose: AdminAIValidatedPurpose): string {
  return overrideFor(purpose.id)?.failurePosture ?? ''
}

// 失败姿态下拉的选项。关键约束：不能把「跟随声明」编码成空字符串值——reka-ui 的
// Select 对空 value 的处理是未定义行为（选项渲染为空、选中态无法成立，就是控制台
// 那条 SelectItem missing render function 的来源）。用一个显式哨兵值表示「未覆盖」，
// 在写入侧把哨兵翻译回「删除字段」。
const POSTURE_DECLARED = '__declared__'

function postureOptions(purpose: AdminAIValidatedPurpose) {
  return [
    {
      label: t('admin.ai.purposes.posture.followDeclared', {
        value: t(`admin.ai.purposes.postureValue.${purpose.failurePosture}`)
      }),
      value: POSTURE_DECLARED
    },
    ...AI_FAILURE_POSTURES.map(value => ({
      label: t(`admin.ai.purposes.posture.${value}`),
      value
    }))
  ]
}

function onPostureChange(purpose: AdminAIValidatedPurpose, value: string) {
  const current = overrideFor(purpose.id)
  const declared = value === POSTURE_DECLARED ? '' : value
  const next: AdminAIPurposeOverride = {
    ...(current?.tools ? { tools: current.tools } : {}),
    ...(declared ? { failurePosture: declared } : {})
  }
  writeOverride(purpose.id, Object.keys(next).length > 0 ? next : undefined)
}

// 「跟随声明」与「自定义」是两种不同的事实：前者删除覆盖字段（声明即生效值），后者
// 显式写下允许列表（哪怕是空列表）。合并成一种状态会让「恢复默认」变成不可能。
function useDeclaredTools(purpose: AdminAIValidatedPurpose) {
  const current = overrideFor(purpose.id)
  const next: AdminAIPurposeOverride = {}
  if (current?.failurePosture) next.failurePosture = current.failurePosture
  writeOverride(purpose.id, Object.keys(next).length > 0 ? next : undefined)
}

function useCustomTools(purpose: AdminAIValidatedPurpose) {
  const current = overrideFor(purpose.id)
  const next: AdminAIPurposeOverride = {
    tools: { allowed: normalizePurposeToolNames(selectedTools(purpose)) }
  }
  if (current?.failurePosture) next.failurePosture = current.failurePosture
  writeOverride(purpose.id, next)
}

// 勾选与取消都走 setPurposeTool：它把生效集合物化后增删名字，再与用途声明比较——
// 一旦结果与声明一致就把整个 tools 字段删掉，于是「勾上再取消」会回到「未覆盖」，
// 而不是留下一个与声明同值、却永远显示「已自定义」的覆盖。
function toggleTool(purpose: AdminAIValidatedPurpose, name: string, checked: boolean) {
  const current = normalizePurposeToolNames(selectedTools(purpose))
  // 上限只在「新增」时拦：取消勾选永远允许，否则超限的配置会把运营者锁在界面里。
  if (checked && !current.includes(name) && current.length >= MAX_TOOLS_PER_REQUEST) return
  writeOverride(purpose.id, setPurposeTool(purpose.declaredTools, overrideFor(purpose.id), name, checked))
}

// 每行的输入框是纯界面状态：它不进配置文档，也不参与脏判定。
function setPendingToolName(purpose: AdminAIValidatedPurpose, value: string) {
  pendingToolName.value = { ...pendingToolName.value, [purpose.id]: value }
}

function addTool(purpose: AdminAIValidatedPurpose) {
  const raw = (pendingToolName.value[purpose.id] ?? '').trim()
  if (!raw) return
  if (!isValidPurposeToolName(raw)) {
    toolNameError.value = { ...toolNameError.value, [purpose.id]: t('admin.ai.purposes.tools.nameInvalid') }
    return
  }
  const current = normalizePurposeToolNames(selectedTools(purpose))
  if (current.includes(raw)) {
    pendingToolName.value = { ...pendingToolName.value, [purpose.id]: '' }
    return
  }
  if (current.length >= MAX_TOOLS_PER_REQUEST) {
    toolNameError.value = {
      ...toolNameError.value,
      [purpose.id]: t('admin.ai.purposes.tools.tooMany', { max: MAX_TOOLS_PER_REQUEST })
    }
    return
  }
  const existing = overrideFor(purpose.id)
  const override: AdminAIPurposeOverride = {
    tools: { allowed: normalizePurposeToolNames([...current, raw]) }
  }
  if (existing?.failurePosture) override.failurePosture = existing.failurePosture
  writeOverride(purpose.id, override)
  const errors = { ...toolNameError.value }
  delete errors[purpose.id]
  toolNameError.value = errors
  pendingToolName.value = { ...pendingToolName.value, [purpose.id]: '' }
}

function removeTool(purpose: AdminAIValidatedPurpose, name: string) {
  toggleTool(purpose, name, false)
}

function restorePurpose(purpose: AdminAIValidatedPurpose) {
  writeOverride(purpose.id, undefined)
  const errors = { ...toolNameError.value }
  delete errors[purpose.id]
  toolNameError.value = errors
  pendingToolName.value = { ...pendingToolName.value, [purpose.id]: '' }
  toast.add({
    color: 'success',
    icon: 'i-lucide-rotate-ccw',
    title: t('admin.ai.purposes.restoreDone'),
    duration: 10000
  })
}
</script>

<template>
  <div v-if="form" class="flex min-w-0 flex-col gap-4">
    <div class="min-w-0">
      <p class="text-base font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.ai.purposes.title') }}</p>
      <p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">{{ t('admin.ai.purposes.intro') }}</p>
    </div>

    <UAlert
      v-if="catalog.length === 0"
      color="neutral"
      variant="soft"
      icon="i-lucide-info"
      :description="t('admin.ai.purposes.empty')"
    />

    <section
      v-for="purpose in catalog"
      :key="purpose.id"
      class="min-w-0 rounded-lg border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900"
    >
      <div class="flex flex-wrap items-start justify-between gap-2">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <p class="text-base font-semibold text-slate-900 dark:text-zinc-100">{{ localized(purpose.label) }}</p>
            <UBadge class="font-mono" color="neutral" variant="soft" size="sm">{{ purpose.id }}</UBadge>
            <UBadge color="neutral" variant="soft" size="sm">
              {{ t(`admin.ai.purposes.cost.${purpose.costClass}`) }}
            </UBadge>
            <UBadge
              :color="isCustomized(purpose) ? 'primary' : 'neutral'"
              variant="soft"
              size="sm"
            >
              {{ t(isCustomized(purpose) ? 'admin.ai.purposes.tools.overriddenBadge' : 'admin.ai.purposes.tools.declaredBadge') }}
            </UBadge>
          </div>
          <p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">{{ localized(purpose.description) }}</p>
          <p class="mt-1 flex flex-wrap gap-x-3 text-xs text-slate-400 dark:text-zinc-500">
            <span>{{ t('admin.ai.purposes.blastRadius.title') }}: {{ t(`admin.ai.purposes.blastRadius.${purpose.blastRadius}`) }}</span>
            <span v-if="purpose.resourceType">{{ t('admin.ai.purposes.resourceType', { value: purpose.resourceType }) }}</span>
            <span v-if="purpose.promptVersion">{{ t('admin.ai.purposes.promptVersion', { value: purpose.promptVersion }) }}</span>
          </p>
        </div>
        <UButton
          icon="i-lucide-rotate-ccw"
          color="neutral"
          variant="outline"
          size="md"
          :disabled="!canManage || !isCustomized(purpose)"
          @click="restorePurpose(purpose)"
        >
          {{ t('admin.ai.purposes.restore') }}
        </UButton>
      </div>

      <div class="mt-4 grid min-w-0 gap-4 lg:grid-cols-2">
        <div class="min-w-0">
          <label class="text-sm font-medium text-slate-600 dark:text-zinc-300" :for="`ai-purpose-posture-${purpose.id}`">
            {{ t('admin.ai.purposes.columns.posture') }}
          </label>
          <USelect
            :model-value="postureValue(purpose) || POSTURE_DECLARED"
            class="mt-1 w-full"
            size="md"
            value-key="value"
            label-key="label"
            :disabled="readOnly"
            :aria-label="t('admin.ai.purposes.columns.posture')"
            :items="postureOptions(purpose)"
            @update:model-value="(value: string) => onPostureChange(purpose, value)"
          />
          <p class="mt-1 text-xs text-slate-400 dark:text-zinc-500">{{ t('admin.ai.purposes.posture.economyHint') }}</p>
        </div>

        <div class="min-w-0">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <span class="text-sm font-medium text-slate-600 dark:text-zinc-300">{{ t('admin.ai.purposes.columns.tools') }}</span>
            <div class="flex items-center gap-1">
              <UButton
                size="sm"
                :color="toolsOverridden(purpose) ? 'neutral' : 'primary'"
                :variant="toolsOverridden(purpose) ? 'ghost' : 'soft'"
                :disabled="readOnly"
                @click="useDeclaredTools(purpose)"
              >
                {{ t('admin.ai.purposes.tools.followDeclared') }}
              </UButton>
              <UButton
                size="sm"
                :color="toolsOverridden(purpose) ? 'primary' : 'neutral'"
                :variant="toolsOverridden(purpose) ? 'soft' : 'ghost'"
                :disabled="readOnly"
                @click="useCustomTools(purpose)"
              >
                {{ t('admin.ai.purposes.tools.custom') }}
              </UButton>
            </div>
          </div>
          <p class="mt-1 text-xs text-slate-400 dark:text-zinc-500">
            {{ toolsOverridden(purpose) ? t('admin.ai.purposes.tools.customHint') : t('admin.ai.purposes.tools.followDeclaredHint') }}
          </p>

          <div class="mt-2 flex flex-wrap gap-x-4 gap-y-1.5">
            <UCheckbox
              v-for="name in toolChoices(purpose)"
              :key="name"
              :model-value="selectedTools(purpose).includes(name)"
              :disabled="readOnly"
              :label="toolLabel(name)"
              :aria-label="name"
              @update:model-value="(checked: boolean | 'indeterminate') => toggleTool(purpose, name, checked === true)"
            />
          </div>

          <div class="mt-2">
            <div class="flex items-center gap-2">
              <UInput
                :model-value="pendingToolName[purpose.id] ?? ''"
                :placeholder="t('admin.ai.purposes.tools.addPlaceholder')"
                size="md"
                class="min-w-0 flex-1"
                :disabled="readOnly"
                @update:model-value="(value: string) => setPendingToolName(purpose, value)"
                @keydown.enter.prevent="addTool(purpose)"
              />
              <UButton size="md" color="neutral" variant="outline" :disabled="readOnly" @click="addTool(purpose)">
                {{ t('admin.ai.purposes.tools.add') }}
              </UButton>
            </div>
            <p v-if="toolNameError[purpose.id]" class="mt-1 text-xs text-red-600 dark:text-red-400">
              {{ toolNameError[purpose.id] }}
            </p>
            <div v-if="unknownTools(purpose).length > 0" class="mt-2 flex flex-wrap gap-1">
              <UBadge
                v-for="name in unknownTools(purpose)"
                :key="name"
                color="warning"
                variant="soft"
                size="sm"
                class="font-mono"
              >
                {{ name }}
                <button
                  v-if="!readOnly"
                  type="button"
                  class="ml-1 text-slate-400 hover:text-slate-700 dark:hover:text-zinc-200"
                  :aria-label="t('admin.ai.purposes.tools.remove', { name })"
                  @click="removeTool(purpose, name)"
                >
                  <UIcon name="i-lucide-x" class="size-3" />
                </button>
              </UBadge>
            </div>
            <p class="mt-1 text-xs text-slate-400 dark:text-zinc-500">{{ t('admin.ai.purposes.tools.unknownHint') }}</p>
          </div>
        </div>
      </div>
    </section>

    <p v-if="draft.dirty.value" class="text-sm text-amber-600 dark:text-amber-400">
      {{ t('admin.ai.purposes.dirtyHint') }}
    </p>
  </div>
</template>
