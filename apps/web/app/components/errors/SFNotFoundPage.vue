<script setup lang="ts">
import SFNotFoundEmergencyPage from '~/components/errors/SFNotFoundEmergencyPage.vue'
import { provideNotFoundPageContext } from '~/composables/errors/useNotFoundPageContext'
import type { NuxtError } from '#app'
import type { PageResolvePayload } from '~/utils/pageResolve'

const props = defineProps<{
  error: NuxtError
  resolvedPage?: PageResolvePayload | null
}>()

const error = computed(() => props.error)

provideNotFoundPageContext(error)

if (import.meta.server) {
  const event = useRequestEvent()
  if (event) {
    setResponseStatus(event, 404)
    // 响应头和 Nitro cache/SWR 由 not-found-document-policy 统一收口。
  }
}
</script>

<template>
  <div
    class="sf-page-outlet"
    data-page="system.not_found"
    :data-provider="resolvedPage?.provider || 'core'"
    :data-template="resolvedPage?.provider !== 'core' && !resolvedPage?.fallback && Boolean(resolvedPage?.renderOutput || resolvedPage?.templateHtml?.trim()) ? '1' : '0'"
    data-host-chrome="0"
  >
    <SFSystemThemeTemplate
      v-if="resolvedPage?.provider !== 'core' && !resolvedPage?.fallback && Boolean(resolvedPage?.renderOutput || resolvedPage?.templateHtml?.trim())"
      :html="resolvedPage?.templateHtml?.trim() || ''"
      :render-output="resolvedPage?.renderOutput"
      :extension-id="resolvedPage?.extensionId || resolvedPage?.provider || 'core'"
    />
    <!-- 未解析或明确失败时只展示完整本地 Core，不请求 Page Registry 或远程 chrome。 -->
    <SFNotFoundEmergencyPage v-else />
    <p v-if="resolvedPage?.fallback" class="sr-only">
      page registry fallback to core
    </p>
  </div>
</template>
