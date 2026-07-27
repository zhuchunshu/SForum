<script setup lang="ts">
import { useSForumSeo } from '~/composables/seo/useSForumSeo'
import SFTagIndexPage from '~/components/forum/SFTagIndexPage.vue'
import { forumTagsIndexPath } from '~/utils/forum/forumTaxonomy'
/** forum.tag.index 路由壳：SEO + outlet；呈现由主题 L1 + SFTagIndexPage 岛拥有。 */
const { t } = useI18n()
const { seoSettings, siteName } = useWebOptions()

useSForumSeo(computed(() => ({
  type: 'static' as const,
  path: forumTagsIndexPath(),
  title: t('taxonomy.tags.title'),
  description: t('taxonomy.tags.description'),
  public: true,
  breadcrumbs: [
    { name: seoSettings.value.seoSiteName || siteName.value, path: '/' },
    { name: t('taxonomy.tags.title'), path: forumTagsIndexPath() }
  ]
})))
</script>

<template>
  <SFPageOutlet page="forum.tag.index">
    <SFTagIndexPage />
  </SFPageOutlet>
</template>
