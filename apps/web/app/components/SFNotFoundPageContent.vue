<script setup lang="ts">
import type { ForumCategoryGroup } from '~/utils/forumTaxonomy'
import type { NuxtError } from '#app'

const context = useNotFoundPageContext()
const forumApi = useForumApi()
const { can } = usePermissions()
// 404 岛处于 ThemeRenderTree 的动态 VNode 内，不能以顶层 await 依赖 Suspense。
// 先同步输出错误正文和导航骨架，分类数据抵达后再填充。
const { data: categoryGroups, pending } = useAsyncData(
  'system-not-found-category-groups',
  async () => {
    try {
      return await forumApi.listCategoryGroups()
    } catch {
      // 404 仍须可用；导航数据不可用时只省略分类链接。
      return [] as ForumCategoryGroup[]
    }
  },
  { default: () => [] as ForumCategoryGroup[] }
)

const error = computed(() => context?.error.value || { statusCode: 404 } as NuxtError)
const categories = computed(() => categoryGroups.value.flatMap(group => group.categories || []))
const totalTopics = computed(() => categories.value.reduce((total, category) => total + category.topicCount, 0))
const canCreateTopic = computed(() => can(FORUM_PERMISSIONS.topicCreate))
</script>

<template>
  <main class="sforum-not-found-page" data-not-found-body="host">
    <div class="sforum-not-found-page__layout sforum-home__layout">
      <aside class="sforum-not-found-page__sidebar sforum-home__sidebar">
        <SFHomeNavigation
          desktop-only
          navigation-mode="route"
          :categories="categories"
          :total-topics="totalTopics"
          :pending="pending"
          :can-create-topic="canCreateTopic"
        />
      </aside>
      <div class="sforum-not-found-page__main sforum-home__main">
        <SFErrorPagePanel :error="error" />
      </div>
    </div>
  </main>
</template>

<style scoped>
.sforum-not-found-page__main {
  align-items: center;
}
</style>
