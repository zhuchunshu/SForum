<script setup lang="ts">
import type { NuxtError } from '#app'
import SFErrorPageContent from './SFErrorPageContent.vue'

const context = useNotFoundPageContext()
const error = computed(() => context?.error.value || { statusCode: 404 } as NuxtError)
</script>

<template>
  <SFErrorPageContent :error="error" emergency />
</template>
