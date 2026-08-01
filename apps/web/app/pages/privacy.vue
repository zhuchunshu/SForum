<script setup lang="ts">
import SFPrivacyPage from '~/components/legal/SFPrivacyPage.vue'
import { useSForumSeo } from '~/composables/seo/useSForumSeo'
/** site.privacy 路由壳：SEO + outlet；呈现由主题 L1 + SFPrivacyPage 岛拥有。 */
const { t } = useI18n()

useSForumSeo({ title: () => t('legal.privacyTitle'), path: '/privacy', public: true })
</script>

<template>
  <SFPageOutlet page="site.privacy">
    <SFPrivacyPage />
  </SFPageOutlet>
</template>
