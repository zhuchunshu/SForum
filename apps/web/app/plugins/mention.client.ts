import type { DirectiveBinding } from 'vue'
import { linkifyForumMentions } from '~/utils/forum/forumMentions'

export type ForumMentionHrefResolver = (username: string) => string

/**
 * v-mention：把后端已净化正文 HTML 里的 @提及 链接化（高亮 + 可点）。
 *
 * 与 v-highlight 同形：只在客户端改动 DOM，SSR 输出仍是后端原文；服务端用
 * mention.server.ts 注册同名占位指令。可选绑定值传入主页地址解析器（带 localePath
 * 的调用方传入自己的实现），不传时使用站点默认公开资料路径。
 */
export default defineNuxtPlugin((nuxtApp) => {
  const decorate = (el: Element, binding: DirectiveBinding<ForumMentionHrefResolver | undefined>) => {
    linkifyForumMentions(el, {
      hrefFor: typeof binding.value === 'function' ? binding.value : undefined
    })
  }

  nuxtApp.vueApp.directive('mention', {
    mounted: decorate,
    updated: decorate,
    getSSRProps() {
      // 服务端渲染不输出额外属性，提及链接化完全在客户端进行。
      return {}
    }
  })
})
