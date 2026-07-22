<script setup lang="ts">
import type { NuxtError } from '#app'
import type { PageResolvePayload } from '~/utils/pageResolve'
import NotFoundEmergencyPage from './SFNotFoundEmergencyPage.vue'

const props = defineProps<{
  error: NuxtError
  resolvedPage?: PageResolvePayload | null
  resolving?: boolean
}>()

const error = computed(() => props.error)

provideNotFoundPageContext(error)

if (import.meta.server) {
  const event = useRequestEvent()
  if (event) {
    setResponseStatus(event, 404)
    setResponseHeader(event, 'cache-control', 'no-store')
    setResponseHeader(event, 'x-robots-tag', 'noindex,nofollow')
    const routeRules = (event.context as { _nitro?: { routeRules?: { cache?: boolean, swr?: boolean } } })._nitro?.routeRules
    if (routeRules) {
      routeRules.cache = false
      routeRules.swr = false
    }
  }
}

</script>

<template>
  <div v-if="resolving" class="min-h-screen" aria-busy="true" />
  <SFPageOutlet v-else-if="resolvedPage" page="system.not_found" :resolved-payload="resolvedPage">
    <NotFoundEmergencyPage />
  </SFPageOutlet>
  <NotFoundEmergencyPage v-else />
</template>
