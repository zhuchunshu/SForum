<script setup lang="ts">
import SFPublicMobileRightDrawerHeader from '~/components/navigation/SFPublicMobileRightDrawerHeader.vue'
import SFPublicMobileUserMenu from '~/components/navigation/SFPublicMobileUserMenu.vue'
import SFTopicSideCard from '~/components/forum/SFTopicSideCard.vue'
import type { ForumComment, ForumTopicDetail } from '~/utils/forum/forumTaxonomy'

/**
 * 移动端「主题信息」右侧抽屉。
 *
 * 与桌面右栏共用 SFTopicSideCard：两处曾各自重复一遍同一组 props，抽出这一层后
 * 主题页只负责传主题/作者/标签等数据，抽屉自身拥有移动端 chrome。
 */
const props = defineProps<{
  topic: ForumTopicDetail
  authorName: string
  authorTo?: string
  tags: { id: number, name: string, to: string }[]
  categoryTo: string
  comments: ForumComment[]
}>()

defineEmits<{ close: [] }>()

const { t } = useI18n()
</script>

<template>
  <aside class="sforum-mobile-drawer sforum-mobile-drawer--right">
    <SFPublicMobileRightDrawerHeader
      :title="t('topicDetail.side.title')"
      :close-label="t('topicDetail.cancel')"
      @close="$emit('close')"
    />
    <SFPublicMobileUserMenu />
    <SFTopicSideCard
      :topic="props.topic"
      :author-name="props.authorName"
      :author-to="props.authorTo"
      :tags="props.tags"
      :category-to="props.categoryTo"
      :first-comment-id="props.comments[0]?.id"
      :latest-comment-id="props.comments[props.comments.length - 1]?.id"
      :extension-sidebar="props.topic.extensionSidebar || []"
    />
  </aside>
</template>
