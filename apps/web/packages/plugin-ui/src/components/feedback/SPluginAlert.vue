<script setup lang="ts">
import SPluginIcon from '../icons/SPluginIcon.vue'

withDefaults(defineProps<{
  title?: string
  tone?: 'info' | 'success' | 'warning' | 'danger'
  /** 宿主图标名，例如 `i-lucide-info`；解析失败时静默隐藏。 */
  icon?: string
}>(), {
  title: '',
  tone: 'info',
  icon: ''
})
</script>

<template>
  <div class="splugin-alert" :class="`splugin-alert--${tone}`" :role="tone === 'danger' ? 'alert' : 'status'">
    <p v-if="title" class="splugin-alert__title">
      <SPluginIcon v-if="icon" :name="icon" size="md" class="splugin-alert__title-icon" />
      {{ title }}
    </p>
    <div class="splugin-alert__body"><slot /></div>
  </div>
</template>
