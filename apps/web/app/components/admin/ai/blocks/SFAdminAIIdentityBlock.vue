<script setup lang="ts">
import type { AdminAISettings } from '~/composables/admin/useAdminAI'

const { t } = useI18n()
const form = defineModel<AdminAISettings | null>('form', { required: true })
const disabled = defineModel<boolean>('disabled', { default: false })

// 身份声明由网关注入，因为模型名只在运行期解析（成本等级 → 供应商绑定），
// 提示词作者写不出它。开关的缺省是「公开」：字段缺失（旧文档）与显式 true 等价，
// 因此 getter 用 ?? true 而不是 === true。
//
// 用 getter/setter 而不是直接 v-model 到 form.identity：直接绑定会在挂载时写入
// 一个显式值，面板一打开就显示「有未保存的改动」。
const discloseModel = computed<boolean>({
  get: () => form.value?.identity?.discloseModel ?? true,
  set: (value: boolean) => {
    const current = form.value
    if (!current) return
    current.identity = { ...(current.identity ?? {}), discloseModel: value }
  }
})
</script>

<template>
  <section v-if="form" class="min-w-0 rounded-lg border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
    <p class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.ai.identity') }}</p>
    <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.ai.identityIntro') }}</p>

    <UFormField class="mt-3">
      <UCheckbox
        v-model="discloseModel"
        :label="t('admin.ai.identityDisclose')"
        :disabled="disabled"
      />
    </UFormField>

    <p class="mt-2 text-xs text-slate-500 dark:text-zinc-400">
      {{ t('admin.ai.identityHint') }}
    </p>
  </section>
</template>
