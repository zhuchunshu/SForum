<script setup lang="ts">
import { FORUM_PERMISSIONS, usePermissions } from '~/composables/identity/usePermissions'
import { useForumApi } from '~/composables/forum/useForumApi'
import type { ForumCategoryGroup } from '~/utils/forum/forumTaxonomy'
import type { PublicNavigationItem } from '~/utils/navigation/publicNavigation'
import SFPublicSidebarContent from './SFPublicSidebarContent.vue'

const props = defineProps<{
  items: PublicNavigationItem[]
}>()

const emit = defineEmits<{ navigate: [] }>()
const { locale } = useI18n()
const route = useRoute()
const forumApi = useForumApi()
const { can } = usePermissions()
const categoryDataKey = computed(() => `public-sidebar-navigation-categories:${locale.value}`)

const { data: categoryGroups, pending } = useAsyncData(
  categoryDataKey,
  () => forumApi.listCategoryGroups({ serverInternal: false }),
  { default: () => [] as ForumCategoryGroup[] }
)

const categories = computed(() => categoryGroups.value.flatMap(group => group.categories || []))
const totalTopics = computed(() => categories.value.reduce((sum, category) => sum + (category.topicCount || 0), 0))
const selectedCategorySlug = computed(() => {
  const routeSlug = route.params.categorySlug
  if (typeof routeSlug === 'string') return routeSlug
  const querySlug = route.query.category
  return typeof querySlug === 'string' ? querySlug : ''
})
</script>

<template>
  <SFPublicSidebarContent
    :items="props.items"
    :categories="categories"
    :selected-category-slug="selectedCategorySlug"
    :total-topics="totalTopics"
    :pending="pending"
    :can-create-topic="can(FORUM_PERMISSIONS.topicCreate)"
    navigation-mode="route"
    @navigate="emit('navigate')"
  />
</template>
