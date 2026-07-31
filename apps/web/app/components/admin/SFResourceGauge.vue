<script setup lang="ts">
const props = withDefaults(defineProps<{
  value: string
  max?: string
  percent: number
  label: string
  icon?: string
  color?: string
  size?: 'md' | 'lg'
}>(), {
  size: 'md'
})

const radius = 30
const circumference = 2 * Math.PI * radius
const offset = computed(() => {
  const percent = Math.min(100, Math.max(0, props.percent))
  return circumference * (1 - percent / 100)
})

const strokeColor = computed(() => props.color || 'var(--sf-accent)')
const rootSizeClass = computed(() => props.size === 'lg' ? 'size-36 sm:size-40' : 'size-20')
const valueSizeClass = computed(() => props.size === 'lg' ? 'text-4xl' : 'text-xl')
</script>

<template>
  <div class="relative shrink-0" :class="rootSizeClass">
    <svg class="size-full -rotate-90" viewBox="0 0 72 72" aria-hidden="true">
      <circle
        class="stroke-slate-200 dark:stroke-zinc-700"
        cx="36"
        cy="36"
        r="30"
        fill="none"
        stroke-width="4.5"
      />
      <circle
        class="transition-[stroke-dashoffset] duration-500 ease-out motion-reduce:transition-none"
        cx="36"
        cy="36"
        r="30"
        fill="none"
        stroke="currentColor"
        stroke-linecap="round"
        stroke-width="4.5"
        :style="{
          stroke: strokeColor,
          'stroke-dasharray': `${circumference} ${circumference}`,
          'stroke-dashoffset': offset
        }"
      />
    </svg>
    <div class="absolute inset-0 flex flex-col items-center justify-center text-center">
      <div v-if="icon" class="mb-1 text-[var(--sf-accent)]">
        <UIcon :name="icon" :class="props.size === 'lg' ? 'size-5' : 'size-4'" />
      </div>
      <div
        class="font-mono font-bold leading-none tabular-nums text-slate-900 dark:text-white"
        :class="valueSizeClass"
      >
        {{ value }}
      </div>
      <div v-if="max" :class="props.size === 'lg' ? 'mt-1 text-xs' : 'text-[10px]'" class="text-slate-400">
        {{ max }}
      </div>
      <div
        class="font-semibold text-slate-500 dark:text-zinc-400"
        :class="props.size === 'lg' ? 'mt-2 text-xs' : 'text-[10px]'"
      >
        {{ label }}
      </div>
    </div>
  </div>
</template>
