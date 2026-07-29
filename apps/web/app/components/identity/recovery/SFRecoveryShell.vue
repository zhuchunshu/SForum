<script setup lang="ts">
import SFAuthShell from '~/components/identity/auth/SFAuthShell.vue'

defineProps<{
  phase: 1 | 2 | 3
}>()
</script>

<template>
  <SFAuthShell :recovery-phase="phase">
    <slot />
  </SFAuthShell>
</template>

<style src="./SFRecoveryShell.css" lang="css"></style>
