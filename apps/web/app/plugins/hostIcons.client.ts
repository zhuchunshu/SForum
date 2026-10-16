import { hydrateHostIconPlaceholders } from '~/utils/icons/hostIcons'

/**
 * 填充主题模板 `{{ icon "i-lucide-*" }}` 产生的宿主图标占位元素。
 *
 * 这些占位元素来自服务端渲染的静态模板 HTML，不是 Vue 模板，所以这里在挂载
 * 与每次导航结束后各扫描一次，而不是用指令。图标是装饰性的：没有占位元素时
 * 只做一次 querySelectorAll，解析失败也只是留下空元素。
 */
export default defineNuxtPlugin((nuxtApp) => {
  const fill = () => {
    void hydrateHostIconPlaceholders().catch(() => {})
  }

  nuxtApp.hook('app:mounted', () => {
    fill()
    // 主题岛屿可能在挂载后随页面数据异步插入，再补一次。
    void nextTick(fill)
  })
  nuxtApp.hook('page:finish', fill)
})
