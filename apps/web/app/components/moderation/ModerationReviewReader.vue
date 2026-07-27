<script setup lang="ts">
import { sanitizeHtml } from '~/utils/sfSanitize'
import type { ModerationReviewContext } from '~/composables/moderation/useModerationApi'
import type { ModerationWorkbenchTab } from '~/utils/moderation/moderationWorkbench'

defineProps<{ context: ModerationReviewContext | null; loading?: boolean; tab: ModerationWorkbenchTab }>()
defineEmits<{ back: [] }>()

const { t } = useI18n()
const { format: formatDate } = useSiteDateTime()
</script>

<template>
  <!-- 主列壳由 SFModerationReviewPage 提供，阅读器只负责正文 -->
  <div class="sforum-moderation-reader" aria-live="polite">
    <div v-if="loading" class="sforum-moderation-reader__loading">
      <SFSkeleton width="100%" height="28px" />
      <SFSkeleton width="70%" height="18px" />
      <SFSkeleton width="100%" height="340px" />
    </div>

    <SFEmptyState
      v-else-if="!context"
      icon-label="MOD"
      :title="t('moderation.workbench.contextUnavailableTitle')"
      :description="t('moderation.workbench.contextUnavailableDescription')"
    />

    <article v-else class="sforum-moderation-reader__article" aria-labelledby="moderation-reading-title">
      <nav class="sforum-moderation-reader__breadcrumb" :aria-label="t('moderation.workbench.reviewBreadcrumb')">
        <span>{{ t('moderation.workbench.title') }}</span>
        <UIcon name="i-lucide-chevron-right" class="size-3.5" aria-hidden="true" />
        <span>{{ tab === 'history' ? t('moderation.workbench.history') : context.source === 'report' ? t('moderation.workbench.reports') : t('moderation.workbench.pending') }}</span>
        <UIcon name="i-lucide-chevron-right" class="size-3.5" aria-hidden="true" />
        <span>{{ t(`admin.moderation.type.${context.targetType}`) }} #{{ context.targetId }}</span>
      </nav>

      <p class="sforum-moderation-reader__kicker">
        {{ context.category }}
        <span aria-hidden="true">/</span>
        {{ t(`admin.moderation.type.${context.targetType}`) }}
      </p>
      <h1 id="moderation-reading-title" class="sforum-moderation-reader__title">
        {{ context.title }}
      </h1>
      <div class="sforum-moderation-reader__meta">
        <SFAvatar :name="context.authorName" size="sm" />
        <strong>{{ context.authorName }}</strong>
        <span>{{ formatDate(context.createdAt) }}</span>
        <span>{{ context.status }}</span>
        <span v-if="context.ipAddress" class="font-mono">{{ t('moderation.workbench.createIp') }} {{ context.ipAddress }}</span>
      </div>

      <div v-if="context.triggers.length" class="sforum-moderation-reader__notice">
        <UIcon name="i-lucide-shield-alert" class="size-4" aria-hidden="true" />
        <span>
          <strong>{{ t('moderation.workbench.triggerNoticeTitle') }}</strong>
          {{ context.triggers.map(trigger => t(`moderation.trigger.${trigger}`)).join(' / ') }}
        </span>
      </div>

      <p v-if="context.parentTopic" class="sforum-moderation-reader__parent">
        {{ t('moderation.workbench.parentTopic') }}: {{ context.parentTopic }}
      </p>

      <div class="sf-prose sforum-moderation-reader__prose overflow-wrap-anywhere" v-highlight v-html="sanitizeHtml(context.html)" />
    </article>
  </div>
</template>
