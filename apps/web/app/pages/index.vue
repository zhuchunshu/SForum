<script setup lang="ts">
import { useSForumSeo } from '~/composables/seo/useSForumSeo'
import SFHomePage from '~/components/forum/SFHomePage.vue'
/**
 * 论坛首页路由壳：SEO + Page Registry outlet。
 * 默认呈现由主题 L1（home.html → forum.component.home_page 岛）拥有；
 * 本页 slot 仅作 SFPageOutlet fail-closed 紧急回退（永不删除）。
 */
import {
  buildForumHomeQuery,
  parseForumHomeQuery
} from '~/utils/forum/forumHome'
import { forumCategoryPath } from '~/utils/forum/forumTaxonomy'

const { t } = useI18n()
const route = useRoute()
const localePath = useLocalePath()
const { seoSettings } = useWebOptions()
const committedFilters = computed(() => parseForumHomeQuery(route.query))
const currentPage = computed(() => parsePublicPage(route.query.page))
const hasActiveFilters = computed(() => Boolean(
  committedFilters.value.query
  || committedFilters.value.categorySlug
  || committedFilters.value.tagSlug
))

// 统一到独立搜索页；旧首页查询链接保留筛选和分页后永久跳转。
if (committedFilters.value.query) {
  await navigateTo(
    publicPageLocation(
      localePath('/search'),
      currentPage.value,
      buildForumHomeQuery(committedFilters.value)
    ),
    { redirectCode: 301, replace: true }
  )
} else if (route.query.category && !route.path.startsWith('/c/')) {
  await navigateTo(forumCategoryPath(route.query.category as string), { redirectCode: 301, replace: true })
}

useSForumSeo(computed(() => ({
  type: 'home',
  path: publicPagePath('/', currentPage.value),
  description: t('home.metaDescription', { siteName: seoSettings.value.seoSiteName }),
  public: true,
  noindex: hasActiveFilters.value || Object.keys(route.query).some(key => key !== 'page')
})))
</script>

<template>
  <SFPageOutlet page="forum.home">
    <!-- fail-closed 紧急回退：主题 L1 / resolve 失败时仍渲染完整首页岛 -->
    <SFHomePage />
  </SFPageOutlet>
</template>
