<script setup lang="ts">
/**
 * 徽章墙：等级卡 + 徽章网格。
 * 只用宿主通用的展示语义（图标 + 名称 + 稀有度），不引入插件专属样式。
 */
type BadgeWallLevel = { title: string, icon: string }

type BadgeWallItem = {
  slug: string
  name: string
  description: string
  rarity: string
  icon: string
  grantedAt: string
}

type BadgeWall = {
  userId: number
  points: number
  level: BadgeWallLevel
  badges: BadgeWallItem[]
}

const props = defineProps<{
  wall: BadgeWall | null
  pending: boolean
  displayName: string
}>()

const { t, locale } = useI18n()

const rarityColors: Record<string, 'neutral' | 'info' | 'warning' | 'error'> = {
  common: 'neutral',
  rare: 'info',
  epic: 'warning',
  legendary: 'error'
}

const sortedBadges = computed(() => {
  const items = [...(props.wall?.badges ?? [])]
  const order = ['legendary', 'epic', 'rare', 'common']
  return items.sort((a, b) => {
    const rank = order.indexOf(a.rarity) - order.indexOf(b.rarity)
    return rank !== 0 ? rank : a.name.localeCompare(b.name)
  })
})

function formatDate(value: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleDateString(String(locale.value || 'zh-CN'))
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <!-- 等级与成长值 -->
    <UCard
      v-if="wall"
      class="border-slate-200 bg-white text-slate-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100"
    >
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex min-w-0 items-center gap-3">
          <UIcon
            :name="wall.level.icon || 'i-lucide-sparkles'"
            class="size-6 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]"
            aria-hidden="true"
          />
          <div class="min-w-0">
            <p class="text-xs text-muted">{{ t('badges.level') }}</p>
            <p class="truncate text-base font-semibold">{{ wall.level.title || t('badges.noLevel') }}</p>
          </div>
        </div>
        <div class="text-right">
          <p class="text-xs text-muted">{{ t('badges.points') }}</p>
          <p class="text-base font-semibold tabular-nums">{{ wall.points }}</p>
        </div>
      </div>
    </UCard>

    <UCard
      v-else-if="!pending"
      class="border-slate-200 bg-white text-slate-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100"
    >
      <p class="py-4 text-center text-sm text-muted">{{ t('badges.empty') }}</p>
    </UCard>

    <!-- 徽章网格 -->
    <UCard class="border-slate-200 bg-white text-slate-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <h2 class="text-base font-bold">{{ t('badges.earned') }}</h2>
          <UBadge color="neutral" variant="soft" size="sm">{{ sortedBadges.length }}</UBadge>
        </div>
      </template>

      <div v-if="pending" class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <USkeleton v-for="index in 3" :key="index" class="h-20 w-full" />
      </div>

      <ul v-else-if="sortedBadges.length" class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <li
          v-for="badge in sortedBadges"
          :key="badge.slug"
          class="flex items-start gap-3 rounded-lg border border-slate-200 p-3 dark:border-zinc-800"
        >
          <UIcon
            :name="badge.icon || 'i-lucide-award'"
            class="mt-0.5 size-5 shrink-0 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]"
            aria-hidden="true"
          />
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <span class="truncate font-medium">{{ badge.name }}</span>
              <UBadge :color="rarityColors[badge.rarity] || 'neutral'" variant="soft" size="sm">
                {{ t(`badges.rarity.${badge.rarity}`, badge.rarity) }}
              </UBadge>
            </div>
            <p v-if="badge.description" class="mt-0.5 text-xs text-muted">{{ badge.description }}</p>
            <p v-if="badge.grantedAt" class="mt-1 text-xs text-muted">
              {{ t('badges.grantedAt', { date: formatDate(badge.grantedAt) }) }}
            </p>
          </div>
        </li>
      </ul>

      <p v-else class="py-4 text-center text-sm text-muted">
        {{ t('badges.emptyBadges', { name: displayName }) }}
      </p>
    </UCard>
  </div>
</template>
