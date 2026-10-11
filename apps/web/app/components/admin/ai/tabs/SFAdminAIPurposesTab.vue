<script setup lang="ts">
import SFAdminAIPurposesBlock from '~/components/admin/ai/blocks/SFAdminAIPurposesBlock.vue'
import { ADMIN_AI_DRAFT_KEY } from '~/composables/admin/useAdminAISettingsForm'

// 页签只负责把用途区块放进当前页签的几何里：数据、草稿与保存都归页面壳所有，
// 与其它 AI 页签保持一致（页签不重复拉取，也不自带标题/工具栏）。
const { t } = useI18n()
const draft = inject(ADMIN_AI_DRAFT_KEY)!
</script>

<template>
  <div class="flex min-w-0 flex-col gap-4">
    <SFAdminAIPurposesBlock v-if="draft.form.value" />
    <p v-else class="text-sm text-slate-500 dark:text-zinc-400">
      {{ t('admin.ai.loading') }}
    </p>
  </div>
</template>
