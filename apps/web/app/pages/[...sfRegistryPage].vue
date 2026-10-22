<script setup lang="ts">
import { useSForumSeo } from '~/composables/seo/useSForumSeo'
import { useRegistryPageResolve } from '~/composables/pages/useRegistryPageResolve'
import { extractRegistryPageHeading } from '~/utils/pageResolve'
// 子目录组件必须显式导入：Nuxt 自动导入名会带路径前缀，静态契约测试按名字匹配。
import SFRegistryPageBody from '~/components/extensions/public/SFRegistryPageBody.vue'
import SFRegistryPageShell from '~/components/extensions/public/SFRegistryPageShell.vue'

/**
 * 动态公开路由宿主：匹配 Page Registry 的 action=add 贡献。
 * 仅在无其它 Nuxt 文件路由命中时生效（catch-all）。
 * 权威 access/permission 由 API resolve-path 返回 401/403/404，解析与映射见
 * `useRegistryPageResolve`；外壳选择见 `SFRegistryPageShell`。
 */
definePageMeta({
  public: true
})

const route = useRoute()

const requestPath = computed(() => {
  const raw = route.params.sfRegistryPage
  const parts = Array.isArray(raw) ? raw.map(String) : (raw ? [String(raw)] : [])
  // 去掉可选 locale 前缀由 API strip；此处保留完整 path
  return '/' + parts.join('/')
})

const { data, error, status, pending } = await useRegistryPageResolve(requestPath, 'page-resolve-path')

const templateHtml = computed(() => (data.value?.templateHtml || '').trim())

// Page Registry 贡献没有 label 字段；页面可见标题优先从正文首个 h1 提取，
// 没有标题的页面退回路径本身，避免 document.title 出现 page id 噪声。
// 贡献声明的 label（theme.json）优先于 h1 提取：模板为规避净化器剥 data-*
// 的巨字号闪帧而刻意不输出 h1 时（如 sforum.oauth2-provider），label 是唯一即时标题来源。
const pageTitle = computed(() => {
  const declared = (data.value?.label || '').trim()
  if (declared) return declared
  return extractRegistryPageHeading(templateHtml.value) || requestPath.value
})

useSForumSeo({
  title: () => pageTitle.value,
  type: 'website',
  noindex: true
})
</script>

<template>
  <!-- 客户端换页的解析窗口：与最终「site 布局 + 主题壳」同构的三栏骨架，
       消除 pending→最终渲染的 chrome 重排（原先窄容器骨架会闪一帧整页变窄）。 -->
  <div v-if="pending && !data" class="sf-registry-page-pending" aria-busy="true">
    <main class="sforum-settings" data-layout="fullwidth-3col">
      <div class="sforum-settings__layout">
        <div class="sforum-settings__sidebar sforum-home__sidebar" aria-hidden="true">
          <div class="sf-skeleton sf-registry-page-pending__nav">
            <div class="sf-skeleton__content">
              <span class="sf-skeleton__line" style="width: 58%" />
              <span class="sf-skeleton__line" style="width: 82%" />
              <span class="sf-skeleton__line" style="width: 70%" />
              <span class="sf-skeleton__line" style="width: 76%" />
            </div>
          </div>
        </div>
        <section class="sforum-settings__main sforum-content-column">
          <div class="sf-skeleton">
            <div class="sf-skeleton__content">
              <span class="sf-skeleton__line" style="width: 40%" />
              <span class="sf-skeleton__line" style="width: 62%" />
            </div>
          </div>
          <div class="sf-skeleton mt-6">
            <div class="sf-skeleton__content">
              <span class="sf-skeleton__line" style="width: 30%" />
              <span class="sf-skeleton__line" style="width: 55%" />
              <span class="sf-skeleton__line" style="width: 88%" />
            </div>
          </div>
        </section>
        <aside class="sforum-settings__right" aria-hidden="true">
          <div class="sf-skeleton">
            <div class="sf-skeleton__content">
              <span class="sf-skeleton__line" style="width: 45%" />
              <span class="sf-skeleton__line" style="width: 90%" />
            </div>
          </div>
        </aside>
      </div>
    </main>
  </div>
  <!-- 只在解析真的产出结果时渲染外壳：404/403 由 resolve 抛出后，Vue SSR 会吞掉
       async setup 的拒绝并带着空 setupState 继续渲染本组件，此时 data/error 均不存在，
       外壳绑定会是 undefined（Invalid vnode type + Anonymous 两条警告）。错误文档由
       Nuxt 错误页负责，这里不渲染半成品。 -->
  <SFRegistryPageShell
    v-else-if="data || error"
    :layout="data?.layout"
    :shell-html="data?.shellHtml"
    :shell-provider="data?.shellProvider"
    :body="{
      loadFailed: Boolean(error && status === 'error'),
      templateHtml,
      renderOutput: data?.renderOutput,
      view: data?.view,
      component: data?.component,
      schemaView: data?.schemaView,
      extensionId: data?.extensionId || data?.provider || '',
      dataSource: data?.dataSource,
      dataRoute: data?.dataRoute,
      loaderData: data?.loaderData,
      loaderError: data?.loaderError,
      fallbackDescription: requestPath
    }"
  />
</template>
