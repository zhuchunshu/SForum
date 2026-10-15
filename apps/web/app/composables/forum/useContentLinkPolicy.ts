import type { ContentLinkPolicy } from '~/utils/forum/forumOutboundLinks'
import {
  contentLinkInternalHosts,
  parseTrustedDomains
} from '~/utils/forum/forumOutboundLinks'
import { normalizeEnabledOption } from '~/composables/useWebOptions'

/**
 * 正文外链呈现策略：把运营开关与站点域名归一成一份判定输入。
 * 指令（历史内容）与中转页（点击时）共用它，避免两处判定漂移。
 */
export function useContentLinkPolicy() {
  const { webOption, siteUrl, siteDomain } = useWebOptions()

  const contentLinkPolicy = computed<ContentLinkPolicy>(() => ({
    interstitial: normalizeEnabledOption(webOption('forum.links.external_interstitial', 'enabled'), true),
    trustedDomains: parseTrustedDomains(webOption('forum.links.trusted_domains', '')),
    internalHosts: contentLinkInternalHosts({
      siteUrl: siteUrl.value,
      siteDomain: siteDomain.value,
      // SSR 阶段没有 window：服务端渲染的正文链接已由后端写好中转，无需当前 host。
      currentHost: import.meta.client ? window.location.host : ''
    })
  }))

  return { contentLinkPolicy }
}
