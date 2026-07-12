import { fileURLToPath } from 'node:url'

const themeCssPath = fileURLToPath(
  new URL('./app/assets/css/sforum-theme.css', import.meta.url)
)
const homeCssPath = fileURLToPath(
  new URL('./app/assets/css/sforum-home.css', import.meta.url)
)
const topicCssPath = fileURLToPath(
  new URL('./app/assets/css/sforum-topic.css', import.meta.url)
)
const taxonomyCssPath = fileURLToPath(
  new URL('./app/assets/css/sforum-taxonomy.css', import.meta.url)
)
const profileCssPath = fileURLToPath(
  new URL('./app/assets/css/sforum-profile.css', import.meta.url)
)

export default defineNuxtConfig({
  // Nuxt layer 中的 ~ 会按宿主 app 解析，主题资源必须固定到 layer 自身目录。
  css: [themeCssPath, homeCssPath, topicCssPath, taxonomyCssPath, profileCssPath]
})
