<script setup lang="ts">
const props = defineProps<{
  value: string
  max?: string
  percent: number
  label: string
  icon?: string
  color?: string
}>()

const { t } = useI18n()

const radius = 30
const circumference = 2 * Math.PI * radius
const offset = computed(() => {
  const percent = Math.min(100, Math.max(0, props.percent))
  return circumference * (1 - percent / 100)
})

const strokeColor = computed(() => props.color || 'var(--sf-accent)')
</script>

<template>
  <div class="relative w-20 h-20 flex-shrink-0">
    <svg class="w-full h-full -rotate-90" viewBox="0 0 72 72">
      <!-- background track -->
      <circle
        cx="36"
        cy="36"
        r="30"
        fill="none"
        stroke="#e5e7eb"
        stroke-width="4"
      />
      <!-- progress ring -->
      <circle
        cx="36"
        cy="36"
        r="30"
        fill="none"
        stroke="currentColor"
        stroke-width="4"
        stroke-dasharray="100 100"
        :style="{
          stroke: strokeColor,
          'stroke-dasharray': `${circumference} ${circumference}`,
          'stroke-dashoffset': offset
        }"
      />
    </svg>
    <div class="absolute inset-0 flex flex-col items-center justify-center text-center">
      <div v-if="icon" class="mb-1 text-[var(--sf-accent)]">
        <UIcon :name="icon" class="size-4" />
      </div>
      <div class="text-xl font-mono font-bold leading-none">{{ value }}</div>
      <div v-if="max" class="text-[10px] text-slate-400">{{ max }}</div>
      <div class="text-[10px] text-slate-500">{{ label }}</div>
    </div>
  </div>
</template>
