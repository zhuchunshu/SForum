<script setup lang="ts">
import SFOutboundInterstitial from '~/components/outbound/SFOutboundInterstitial.vue'
import { useSForumSeo } from '~/composables/seo/useSForumSeo'

/**
 * site.outbound 路由壳：SEO + Page Registry outlet。
 * 该页是 Host 安全面（不可替换 ⇒ 不可主题化），呈现由 SFOutboundInterstitial 拥有；
 * 站点导航/页脚由 SFHostPublicChrome 回退壳提供，与公开页几何一致。
 */
definePageMeta({ public: true })

const { t } = useI18n()

useSForumSeo(computed(() => ({
  title: t('outbound.metaTitle'),
  description: t('outbound.metaDescription'),
  type: 'static' as const,
  path: '/outbound',
  public: true,
  // 中转页是纯跳转中间页：永不索引、永不 follow。
  noindex: true,
  nofollow: true
})))
</script>

<template>
  <SFPageOutlet page="site.outbound">
    <SFOutboundInterstitial />
  </SFPageOutlet>
</template>
