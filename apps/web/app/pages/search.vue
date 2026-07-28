<script setup lang="ts">
import { useSForumSeo } from '~/composables/seo/useSForumSeo'
import SFHomePage from '~/components/forum/SFHomePage.vue'
import { buildForumHomeQuery, parseForumHomeQuery } from '~/utils/forum/forumHome'

/** forum.search 路由壳：SEO + Page Registry outlet；结果交互由搜索页 Host 岛拥有。 */
const { t } = useI18n()
const route = useRoute()
const committedFilters = computed(() => parseForumHomeQuery(route.query))
const currentPage = computed(() => parsePublicPage(route.query.page))

useSForumSeo(computed(() => ({
  title: t('search.metaTitle'),
  type: 'website',
  path: publicPagePath('/search', currentPage.value, buildForumHomeQuery(committedFilters.value)),
  description: t('search.metaDescription'),
  public: true,
  noindex: true
})))
</script>

<template>
  <SFPageOutlet page="forum.search">
    <SFHomePage />
  </SFPageOutlet>
</template>
