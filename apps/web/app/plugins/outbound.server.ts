/**
 * v-outbound 的服务端占位注册。
 *
 * 链接形态补齐只在客户端执行（服务端净化后已由 apps/api 的 Support/ContentLinks 写入
 * 目标形态）；SSR 仍必须注册同名指令，否则 Vue 服务端渲染带 v-outbound 的模板会拿到
 * undefined 指令。
 */
export default defineNuxtPlugin((nuxtApp) => {
  nuxtApp.vueApp.directive('outbound', {
    getSSRProps() {
      return {}
    }
  })
})
