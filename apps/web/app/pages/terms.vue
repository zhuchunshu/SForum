<script setup lang="ts">
import SFTermsPage from '~/components/legal/SFTermsPage.vue'
import { useSForumSeo } from '~/composables/seo/useSForumSeo'
/** site.terms 路由壳：SEO + outlet；呈现由主题 L1 + SFTermsPage 岛拥有。 */
const { t } = useI18n()

useSForumSeo({ title: () => t('legal.termsTitle'), path: '/terms', public: true })
</script>

<template>
  <SFPageOutlet page="site.terms">
    <SFTermsPage />
  </SFPageOutlet>
</template>
