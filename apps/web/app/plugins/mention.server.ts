/**
 * v-mention 的服务端占位注册。
 *
 * 真正的提及链接化只在客户端执行（装饰后端渲染的 HTML）；SSR 仍必须注册同名指令，
 * 否则 Vue 服务端渲染带 v-mention 的模板时会拿到 undefined 指令。
 */
export default defineNuxtPlugin((nuxtApp) => {
  nuxtApp.vueApp.directive('mention', {
    getSSRProps() {
      return {}
    }
  })
})
