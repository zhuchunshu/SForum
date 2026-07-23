<script setup lang="ts">
import type { NuxtError } from '#app'
import type { PageResolvePayload } from '~/utils/pageResolve'
import { normalizeErrorStatus, resolveErrorPageContent } from '~/utils/errorPage'

const props = defineProps<{
  error: NuxtError
  resolvedPage?: PageResolvePayload | null
}>()

const error = computed(() => props.error)
const page = computed(() => resolveErrorPageContent(props.error?.statusCode))
const statusCode = computed(() => normalizeErrorStatus(props.error?.statusCode))

provideSystemErrorPageContext(error)
provideNotFoundPageContext(error)

if (import.meta.server) {
  const event = useRequestEvent()
  if (event) {
    setResponseStatus(event, statusCode.value)
  }
}
</script>

<template>
  <div
    class="sf-page-outlet"
    :data-page="page.pageId"
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
    <SFSystemErrorEmergencyPage v-else :error="error" />
    <p v-if="resolvedPage?.fallback" class="sr-only">
      page registry fallback to core
    </p>
  </div>
</template>
