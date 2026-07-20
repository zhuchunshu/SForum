<script setup lang="ts">
/** site.guidelines 路由壳：SEO + outlet；呈现由主题 L1 + SFGuidelinesPage 岛拥有。 */
const { t } = useI18n()
const { siteName } = useWebOptions()

useSeoMeta({
  title: () => `${t('legal.guidelinesTitle')} · ${siteName.value}`
})
</script>

<template>
  <SFPageOutlet page="site.guidelines">
    <SFGuidelinesPage />
  </SFPageOutlet>
</template>
