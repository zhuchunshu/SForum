<script setup lang="ts">
import SFPrivacyPage from '~/components/legal/SFPrivacyPage.vue'
/** site.privacy 路由壳：SEO + outlet；呈现由主题 L1 + SFPrivacyPage 岛拥有。 */
const { t } = useI18n()
const { siteName } = useWebOptions()

useSeoMeta({
  title: () => `${t('legal.privacyTitle')} · ${siteName.value}`
})
</script>

<template>
  <SFPageOutlet page="site.privacy">
    <SFPrivacyPage />
  </SFPageOutlet>
</template>
