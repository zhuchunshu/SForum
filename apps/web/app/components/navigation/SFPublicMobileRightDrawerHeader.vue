<script setup lang="ts">
import { useAuthSession } from '~/composables/identity/useAuthSession'

defineProps<{
  title: string
  closeLabel: string
}>()

defineEmits<{
  close: []
}>()

const { t } = useI18n()
const { user } = useAuthSession()
</script>

<template>
  <header class="sforum-mobile-drawer__head">
    <strong>{{ user ? t('nav.personalCenter') : title }}</strong>
    <button type="button" :aria-label="closeLabel" @click="$emit('close')">
      <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
    </button>
  </header>
</template>
