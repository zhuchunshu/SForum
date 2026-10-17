<script setup lang="ts">
import { useSForumSeo } from '~/composables/seo/useSForumSeo'
import { useRegistryPageResolve } from '~/composables/pages/useRegistryPageResolve'
import SFRegistryPageBody from '~/components/extensions/public/SFRegistryPageBody.vue'
import SFRegistryPageShell from '~/components/extensions/public/SFRegistryPageShell.vue'

/**
 * 兼容路径 /x/*：转发到统一 Registry 解析（与根 catch-all 共用同一实现）。
 * 正式 add 路径应使用 manifest 声明的真实 path（由 [...sfRegistryPage] 承接）。
 */
definePageMeta({ public: true })

const route = useRoute()

const requestPath = computed(() => {
  const raw = route.params.path
  const parts = Array.isArray(raw) ? raw.map(String) : (raw ? [String(raw)] : [])
  return '/x/' + parts.join('/')
})

const { data, error, status, pending } = await useRegistryPageResolve(requestPath, 'page-resolve-path-x')

const templateHtml = computed(() => (data.value?.templateHtml || '').trim())

useSForumSeo({
  title: () => data.value?.page?.id || requestPath.value,
  type: 'website',
  noindex: true
})
</script>

<template>
  <!-- 客户端换页的解析窗口：与最终「site 布局 + 主题壳」同构的三栏骨架，
       消除 pending→最终渲染的 chrome 重排。 -->
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
  <SFRegistryPageShell
    v-else
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
