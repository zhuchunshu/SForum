<script setup lang="ts">
import {
  forumCategoriesIndexPath,
  forumCategoryPath,
  type ForumCategory,
  type ForumCategoryGroup
} from '~/utils/forumTaxonomy'
/** forum.category.index 路由壳：SEO + outlet；呈现由主题 L1 + SFCategoryIndexPage 岛拥有。 */
const { t } = useI18n()
const { seoSettings, siteName } = useWebOptions()

useSForumSeo(computed(() => ({
  type: 'static' as const,
  path: forumCategoriesIndexPath(),
  title: t('taxonomy.categories.title'),
  description: t('taxonomy.categories.description'),
  public: true,
  breadcrumbs: [
    { name: seoSettings.value.seoSiteName || siteName.value, path: '/' },
    { name: t('taxonomy.categories.title'), path: forumCategoriesIndexPath() }
  ]
})))
</script>

<template>
  <SFPageOutlet page="forum.category.index">
    <SFCategoryIndexPage />
  </SFPageOutlet>
</template>
