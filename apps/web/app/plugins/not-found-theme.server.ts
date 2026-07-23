import { isThemeableSystemErrorStatus, systemErrorPageIdForStatus } from '~/utils/errorPage'

/** 服务端内部错误请求在 Vue 首轮渲染前准备系统错误页主题快照。 */
export default defineNuxtPlugin({
  name: 'sforum-system-error-theme',
  async setup(nuxtApp) {
    const statusCode = Number(nuxtApp.payload.error?.statusCode)
    if (!isThemeableSystemErrorStatus(statusCode)) {
      return
    }
    const pageId = systemErrorPageIdForStatus(statusCode)
    await useSystemErrorPagePresentation(pageId).prepare()
  }
})
