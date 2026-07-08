<script setup lang="ts">
type AvatarSize = 'sm' | 'md' | 'lg'
type AvatarShape = 'circle' | 'square'
type AvatarStatus = 'online' | 'idle' | 'offline'
type AvatarView = {
  kind: 'uploaded' | 'initials' | 'gravatar' | 'static'
  url: string
  attachmentId?: number | null
  alt: string
}

const props = withDefaults(defineProps<{
  name?: string
  src?: string
  avatar?: AvatarView | null
  alt?: string
  size?: AvatarSize
  shape?: AvatarShape
  status?: AvatarStatus
}>(), {
  name: '匿名用户',
  src: undefined,
  alt: undefined,
  size: 'md',
  shape: 'circle',
  status: undefined
})

// 头像尺寸到像素的映射，用于 NuxtImg 的 width/height 和 sizes 预设。
const sizePixels: Record<AvatarSize, number> = {
  sm: 48,
  md: 96,
  lg: 256
}

const initials = computed(() => {
  const source = props.name.trim()
  if (!source) {
    return 'U'
  }
  return source
    .split(/\s+/)
    .slice(0, 2)
    .map((part) => part.slice(0, 1))
    .join('')
    .toUpperCase()
})

const avatarClass = computed(() => [
  'sf-avatar',
  `sf-avatar--${props.size}`,
  `sf-avatar--${props.shape}`
].join(' '))

const imageSrc = computed(() => props.avatar?.url || props.src || '')
const imageAlt = computed(() => props.alt || props.avatar?.alt || props.name)
const usePlainImage = computed(() => /^https?:\/\//i.test(imageSrc.value))
</script>

<template>
  <span :class="avatarClass">
    <img
      v-if="imageSrc && usePlainImage"
      class="sf-avatar__image"
      :src="imageSrc"
      :alt="imageAlt"
      :width="sizePixels[size]"
      :height="sizePixels[size]"
      loading="lazy"
      decoding="async"
    >
    <NuxtImg
      v-else-if="imageSrc"
      class="sf-avatar__image"
      :src="imageSrc"
      :alt="imageAlt"
      :width="sizePixels[size]"
      :height="sizePixels[size]"
      :sizes="`${sizePixels[size]}px`"
      format="webp"
      loading="lazy"
      decoding="async"
    />
    <span v-else>{{ initials }}</span>
    <span
      v-if="status"
      :class="['sf-avatar__status', `sf-avatar__status--${status}`]"
      aria-hidden="true"
    />
  </span>
</template>
