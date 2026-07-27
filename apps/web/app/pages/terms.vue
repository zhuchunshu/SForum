<script setup lang="ts">
import SFTermsPage from '~/components/legal/SFTermsPage.vue'
/** site.terms 路由壳：SEO + outlet；呈现由主题 L1 + SFTermsPage 岛拥有。 */
const { t } = useI18n()
const { siteName } = useWebOptions()

useSeoMeta({
  title: () => `${t('legal.termsTitle')} · ${siteName.value}`
})
</script>

<template>
  <SFPageOutlet page="site.terms">
    <SFTermsPage />
  </SFPageOutlet>
</template>
