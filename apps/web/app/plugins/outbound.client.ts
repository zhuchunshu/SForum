import { useContentLinkPolicy } from '~/composables/forum/useContentLinkPolicy'
import { decorateContentLinks } from '~/utils/forum/forumOutboundLinks'

/**
 * v-outbound：为正文容器补齐「新标签页 + 站外中转」形态。
 *
 * 为什么还需要客户端指令：服务端已经把新渲染的正文写成目标形态，但
 *   - 历史内容与插件的 SSR 片段可能仍是旧的 <a href>（无 target、无中转）；
 *   - 运营改开关/白名单不需要重渲染历史内容。
 * 因此这里按当前配置就地补齐，服务端形态与它互为纵深。
 *
 * 与 v-mention 同形：只在客户端改 DOM，SSR 输出保持后端原文；服务端用
 * outbound.server.ts 注册同名占位指令，避免模板出现未注册指令。
 */
export default defineNuxtPlugin((nuxtApp) => {
  const { contentLinkPolicy } = useContentLinkPolicy()

  const decorate = (el: Element) => {
    decorateContentLinks(el, contentLinkPolicy.value)
  }

  nuxtApp.vueApp.directive('outbound', {
    mounted: decorate,
    updated: decorate,
    getSSRProps() {
      // 服务端不输出额外属性，改写完全在客户端进行。
      return {}
    }
  })
})
