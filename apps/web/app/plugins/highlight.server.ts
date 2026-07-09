/**
 * v-highlight 的服务端占位注册。
 *
 * 真正的 highlight.js 扫描只在客户端执行；SSR 仍必须注册同名指令，
 * 否则 Vue 服务端渲染带 v-highlight 的模板时会拿到 undefined 指令。
 */
export default defineNuxtPlugin((nuxtApp) => {
  nuxtApp.vueApp.directive('highlight', {
    getSSRProps() {
      return {}
    },
  })
})
