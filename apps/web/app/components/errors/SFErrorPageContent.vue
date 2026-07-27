<script setup lang="ts">
import { useSForumSeo } from '~/composables/seo/useSForumSeo'
import type { NuxtError } from '#app'
import { normalizeErrorStatus, resolveErrorPageContent } from '~/utils/errors/errorPage'
import SFFooter from '../SFFooter.vue'
import SFErrorPagePanel from './SFErrorPagePanel.vue'
import SFNavbar from '../SFNavbar.vue'

const props = withDefaults(defineProps<{
  error: NuxtError
  /** 仅供 Page Registry/主题不可用时的本地 404 应急路径使用。 */
  emergency?: boolean
}>(), {
  emergency: false
})

const { t } = useI18n()
const { siteName } = useWebOptions()
const route = useRoute()
const page = computed(() => resolveErrorPageContent(props.error?.statusCode))
const title = computed(() => t(page.value.titleKey, { siteName: siteName.value }))
const description = computed(() => t(page.value.descriptionKey, { siteName: siteName.value }))

// 系统错误 document head 由根 error.vue 收口，Core 紧急页不能重新注入 canonical/JSON-LD。
if (!props.emergency && normalizeErrorStatus(props.error?.statusCode) !== 404) {
  useSForumSeo({
    title,
    description,
    path: () => route.path,
    noindex: true,
    schema: { type: 'WebPage' }
  })
}

</script>

<template>
  <div class="sforum-error-page">
    <SFNavbar :fetch-remote-chrome="!emergency" />

    <main class="sforum-error-page__main">
      <SFErrorPagePanel :error="error" />
    </main>

    <SFFooter :fetch-remote-chrome="!emergency" />
  </div>
</template>

<style scoped>
.sforum-error-page {
  display: flex;
  min-height: 100vh;
  flex-direction: column;
  background: var(--sf-surface);
  color: var(--sf-fg);
}

.sforum-error-page__main {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  padding: 48px 16px;
}

@media (max-width: 640px) {
  .sforum-error-page__main {
    align-items: stretch;
    padding: 28px 14px;
  }
}
</style>
