<script setup lang="ts">
import { usePublicNavigation } from '~/composables/navigation/usePublicNavigation'
import SFPublicSidebarContent from '~/components/forum/navigation/SFPublicSidebarContent.vue'
import type { ForumCategory } from '~/utils/forum/forumTaxonomy'

const props = withDefaults(defineProps<{
  categories?: ForumCategory[]
  selectedCategorySlug?: string
  totalTopics?: number
  pending?: boolean
  canCreateTopic?: boolean
  /** 旧主题兼容：正文移动分类导航已下线，启用时不渲染。 */
  mobileOnly?: boolean
  /** 仅渲染桌面左栏 */
  desktopOnly?: boolean
  /** 导航模式下是否显示分类列表（默认为 true） */
  showCategories?: boolean
  /**
   * filter：首页 URL query 筛选（emit select-category）
   * route：跳转 / 与 /c/:slug（类别页/标签页）
   */
  navigationMode?: 'filter' | 'route'
}>(), {
  categories: () => [],
  selectedCategorySlug: '',
  totalTopics: 0,
  pending: false,
  canCreateTopic: false,
  mobileOnly: false,
  desktopOnly: false,
  showCategories: true,
  navigationMode: 'filter'
})

const emit = defineEmits<{
  'select-category': [slug: string]
}>()

const { sidebarItems } = usePublicNavigation()
</script>

<template>
  <aside v-if="!mobileOnly" class="sf-home-navigation" data-navigation-location="public.sidebar.primary" :aria-busy="pending">
    <SFPublicSidebarContent
      :items="sidebarItems"
      :categories="categories"
      :selected-category-slug="selectedCategorySlug"
      :total-topics="totalTopics"
      :pending="pending"
      :can-create-topic="canCreateTopic"
      :show-categories="showCategories"
      :navigation-mode="navigationMode"
      @select-category="emit('select-category', $event)"
    >
      <template #after-navigation><slot name="after-navigation" /></template>
    </SFPublicSidebarContent>
  </aside>
</template>
