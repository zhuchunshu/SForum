<script setup lang="ts">
import type { AdminAIPurposeOverride, AdminAIValidatedPurpose } from '~/composables/admin/useAdminAI'
import type { AIPurposeOverride } from '~/utils/admin/adminAI'
import { ADMIN_AI_DRAFT_KEY } from '~/composables/admin/useAdminAISettingsForm'
import { useAdminRoutes } from '~/composables/admin/useAdminRoutes'
import { toSettingsOverride } from '~/components/admin/ai/toolOverrides'
import {
  AI_FAILURE_POSTURES,
  purposeToolsAreOverridden,
  selectedPurposeTools
} from '~/utils/admin/adminAI'

// 用途清单是「站内有哪些 AI 行为」的唯一可枚举答案。每一行都同时展示声明与生效
// 两份事实：运营者要能看出「这个用途原本是什么」与「我把它改成了什么」，否则一次
// 覆盖会让声明的默认值从界面上消失，之后没人知道恢复默认该恢复成什么。
const { t, locale } = useI18n()
const toast = useToast()
const draft = inject(ADMIN_AI_DRAFT_KEY)!
const { form, ai, canManage, readOnly } = draft

const adminRoutes = useAdminRoutes()

const catalog = computed<AdminAIValidatedPurpose[]>(() => ai.purposes.value ?? [])

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

function restorePurpose(purpose: AdminAIValidatedPurpose) {
  writeOverride(purpose.id, undefined)
  toast.add({
    color: 'success',
    icon: 'i-lucide-rotate-ccw',
    title: t('admin.ai.purposes.restoreDone'),
    duration: 5000
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
          :title="t('admin.ai.purposes.restoreHint')"
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
            <UBadge :color="toolsOverridden(purpose) ? 'primary' : 'neutral'" variant="soft" size="sm">
              {{ t(toolsOverridden(purpose) ? 'admin.ai.purposes.tools.overriddenBadge' : 'admin.ai.purposes.tools.declaredBadge') }}
            </UBadge>
          </div>
          <p class="mt-1 text-sm text-slate-900 dark:text-zinc-100" :title="selectedTools(purpose).join(' · ')">
            {{ t('admin.ai.purposes.tools.summary', { count: selectedTools(purpose).length }) }}
          </p>
          <p class="mt-1 text-xs text-slate-400 dark:text-zinc-500">{{ t('admin.ai.purposes.tools.managedHint') }}</p>
          <NuxtLink
            :to="{ path: adminRoutes.path('ai'), query: { tab: 'tools' } }"
            class="mt-2 inline-flex items-center gap-1 text-sm font-medium text-[var(--sf-accent)] hover:underline dark:text-[var(--sf-accent-dark)]"
          >
            {{ t('admin.ai.purposes.tools.manage') }}
            <UIcon name="i-lucide-arrow-up-right" class="size-3.5" />
          </NuxtLink>
        </div>
      </div>
    </section>

    <p v-if="draft.dirty.value" class="text-sm text-amber-600 dark:text-amber-400">
      {{ t('admin.ai.purposes.dirtyHint') }}
    </p>
  </div>
</template>
