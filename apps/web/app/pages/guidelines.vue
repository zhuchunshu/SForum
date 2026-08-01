<script setup lang="ts">
import SFGuidelinesPage from '~/components/legal/SFGuidelinesPage.vue'
import { useSForumSeo } from '~/composables/seo/useSForumSeo'
/** site.guidelines 路由壳：SEO + outlet；呈现由主题 L1 + SFGuidelinesPage 岛拥有。 */
const { t } = useI18n()

useSForumSeo({ title: () => t('legal.guidelinesTitle'), path: '/guidelines', public: true })
</script>

<template>
  <SFPageOutlet page="site.guidelines">
    <SFGuidelinesPage />
  </SFPageOutlet>
</template>
