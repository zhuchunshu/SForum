/** 服务端内部错误请求在 Vue 首轮渲染前准备 404 主题快照。 */
export default defineNuxtPlugin({
  name: 'sforum-not-found-theme',
  async setup(nuxtApp) {
    if (Number(nuxtApp.payload.error?.statusCode) !== 404) {
      return
    }
    await useNotFoundPagePresentation().prepare()
  }
})
