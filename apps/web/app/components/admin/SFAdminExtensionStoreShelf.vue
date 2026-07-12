<script setup lang="ts">
/**
 * 应用商城货架（01C 粘性筛选）。
 * kind 区分主题 / 插件目录；当前为本地占位数据，不接远端 API。
 */
type StoreKind = 'theme' | 'plugin'
type StoreCategory = 'all' | 'official' | 'payment' | 'notify' | 'storage' | 'community'
type StoreSort = 'recommended' | 'updated' | 'name'

type StoreCatalogItem = {
  id: string
  kind: StoreKind
  nameKey: string
  summaryKey: string
  authorKey: string
  version: string
  categories: StoreCategory[]
  tags: string[]
  icon: string
  accent: string
}

const props = defineProps<{
  kind: StoreKind
  pageId: string
}>()

const { t } = useI18n()
const adminPage = useAdminPage(props.pageId)
const adminRoutes = useAdminRoutes()

// 占位目录：仅用于展示货架布局，上线后由远端目录替换
const fullCatalog: StoreCatalogItem[] = [
  {
    id: 'smtp-mail',
    kind: 'plugin',
    nameKey: 'admin.extensions.store.demo.smtpName',
    summaryKey: 'admin.extensions.store.demo.smtpSummary',
    authorKey: 'admin.extensions.store.demo.officialAuthor',
    version: 'v1.2.0',
    categories: ['official', 'notify'],
    tags: ['mail', 'provider'],
    icon: 'i-lucide-mail',
    accent: 'bg-teal-500/15 text-teal-700 dark:text-teal-300'
  },
  {
    id: 's3-storage',
    kind: 'plugin',
    nameKey: 'admin.extensions.store.demo.s3Name',
    summaryKey: 'admin.extensions.store.demo.s3Summary',
    authorKey: 'admin.extensions.store.demo.officialAuthor',
    version: 'v0.9.1',
    categories: ['official', 'storage'],
    tags: ['storage', 's3'],
    icon: 'i-lucide-hard-drive',
    accent: 'bg-indigo-500/15 text-indigo-700 dark:text-indigo-300'
  },
  {
    id: 'wechat-notify',
    kind: 'plugin',
    nameKey: 'admin.extensions.store.demo.wechatName',
    summaryKey: 'admin.extensions.store.demo.wechatSummary',
    authorKey: 'admin.extensions.store.demo.communityAuthor',
    version: 'v0.4.0',
    categories: ['community', 'notify'],
    tags: ['notify', 'wechat'],
    icon: 'i-lucide-message-circle',
    accent: 'bg-amber-500/15 text-amber-700 dark:text-amber-300'
  },
  {
    id: 'alipay',
    kind: 'plugin',
    nameKey: 'admin.extensions.store.demo.alipayName',
    summaryKey: 'admin.extensions.store.demo.alipaySummary',
    authorKey: 'admin.extensions.store.demo.officialAuthor',
    version: '—',
    categories: ['official', 'payment'],
    tags: ['payment'],
    icon: 'i-lucide-wallet',
    accent: 'bg-pink-500/15 text-pink-700 dark:text-pink-300'
  },
  {
    id: 'analytics',
    kind: 'plugin',
    nameKey: 'admin.extensions.store.demo.analyticsName',
    summaryKey: 'admin.extensions.store.demo.analyticsSummary',
    authorKey: 'admin.extensions.store.demo.communityAuthor',
    version: '—',
    categories: ['community'],
    tags: ['analytics'],
    icon: 'i-lucide-chart-column',
    accent: 'bg-sky-500/15 text-sky-700 dark:text-sky-300'
  },
  {
    id: 'night-theme',
    kind: 'theme',
    nameKey: 'admin.extensions.store.demo.themeName',
    summaryKey: 'admin.extensions.store.demo.themeSummary',
    authorKey: 'admin.extensions.store.demo.communityAuthor',
    version: 'v2.0.0',
    categories: ['community'],
    tags: ['theme', 'dark'],
    icon: 'i-lucide-palette',
    accent: 'bg-violet-500/15 text-violet-700 dark:text-violet-300'
  },
  {
    id: 'aurora-theme',
    kind: 'theme',
    nameKey: 'admin.extensions.store.demo.themeAuroraName',
    summaryKey: 'admin.extensions.store.demo.themeAuroraSummary',
    authorKey: 'admin.extensions.store.demo.officialAuthor',
    version: 'v1.1.0',
    categories: ['official'],
    tags: ['theme', 'light'],
    icon: 'i-lucide-sun',
    accent: 'bg-orange-500/15 text-orange-700 dark:text-orange-300'
  },
  {
    id: 'telegram',
    kind: 'plugin',
    nameKey: 'admin.extensions.store.demo.telegramName',
    summaryKey: 'admin.extensions.store.demo.telegramSummary',
    authorKey: 'admin.extensions.store.demo.communityAuthor',
    version: 'v0.3.2',
    categories: ['community', 'notify'],
    tags: ['notify', 'telegram'],
    icon: 'i-lucide-send',
    accent: 'bg-cyan-500/15 text-cyan-700 dark:text-cyan-300'
  },
  {
    id: 'webhook',
    kind: 'plugin',
    nameKey: 'admin.extensions.store.demo.webhookName',
    summaryKey: 'admin.extensions.store.demo.webhookSummary',
    authorKey: 'admin.extensions.store.demo.officialAuthor',
    version: 'v1.0.0',
    categories: ['official'],
    tags: ['hooks'],
    icon: 'i-lucide-webhook',
    accent: 'bg-rose-500/15 text-rose-700 dark:text-rose-300'
  },
  {
    id: 'content-ai',
    kind: 'plugin',
    nameKey: 'admin.extensions.store.demo.aiName',
    summaryKey: 'admin.extensions.store.demo.aiSummary',
    authorKey: 'admin.extensions.store.demo.communityAuthor',
    version: '—',
    categories: ['community'],
    tags: ['ai'],
    icon: 'i-lucide-sparkles',
    accent: 'bg-purple-500/15 text-purple-700 dark:text-purple-300'
  }
]

const catalog = computed(() => fullCatalog.filter(item => item.kind === props.kind))

const searchQuery = ref('')
const category = ref<StoreCategory>('all')
const sort = ref<StoreSort>('recommended')

const categoryOptions = computed(() => {
  const rows = catalog.value
  const counts: Partial<Record<StoreCategory, number>> = { all: rows.length }

  for (const item of rows) {
    for (const key of item.categories) {
      counts[key] = (counts[key] || 0) + 1
    }
  }

  // 主题货架只展示全部 / 官方 / 社区；插件货架额外展示能力类筛选
  const keys: Array<[StoreCategory, string]> = props.kind === 'theme'
    ? [
        ['all', 'admin.extensions.store.categoryAll'],
        ['official', 'admin.extensions.store.categoryOfficial'],
        ['community', 'admin.extensions.store.categoryCommunity']
      ]
    : [
        ['all', 'admin.extensions.store.categoryAll'],
        ['official', 'admin.extensions.store.categoryOfficial'],
        ['payment', 'admin.extensions.store.categoryPayment'],
        ['notify', 'admin.extensions.store.categoryNotify'],
        ['storage', 'admin.extensions.store.categoryStorage'],
        ['community', 'admin.extensions.store.categoryCommunity']
      ]

  return keys.map(([id, labelKey]) => ({
    id,
    labelKey,
    count: counts[id] || 0
  }))
})

const sortItems = computed(() => [
  { label: t('admin.extensions.store.sortRecommended'), value: 'recommended' as const },
  { label: t('admin.extensions.store.sortUpdated'), value: 'updated' as const },
  { label: t('admin.extensions.store.sortName'), value: 'name' as const }
])

const titleKey = computed(() =>
  props.kind === 'theme'
    ? 'admin.extensions.store.themesTitle'
    : 'admin.extensions.store.pluginsTitle'
)

const introKey = computed(() =>
  props.kind === 'theme'
    ? 'admin.extensions.store.themesIntro'
    : 'admin.extensions.store.pluginsIntro'
)

const metaTitleKey = computed(() =>
  props.kind === 'theme'
    ? 'admin.extensions.store.themesMetaTitle'
    : 'admin.extensions.store.pluginsMetaTitle'
)

const countKey = computed(() =>
  props.kind === 'theme'
    ? 'admin.extensions.store.themesCount'
    : 'admin.extensions.store.pluginsCount'
)

const emptyTitleKey = computed(() =>
  props.kind === 'theme'
    ? 'admin.extensions.store.themesEmptyTitle'
    : 'admin.extensions.store.pluginsEmptyTitle'
)

const emptyDescriptionKey = computed(() =>
  props.kind === 'theme'
    ? 'admin.extensions.store.themesEmptyDescription'
    : 'admin.extensions.store.pluginsEmptyDescription'
)

const localListPath = computed(() =>
  props.kind === 'theme' ? '/extensions/themes' : '/extensions/plugins'
)

const localListLabelKey = computed(() =>
  props.kind === 'theme'
    ? 'admin.extensions.store.openThemes'
    : 'admin.extensions.store.openPlugins'
)

const localListIcon = computed(() =>
  props.kind === 'theme' ? 'i-lucide-palette' : 'i-lucide-plug'
)

const filteredCatalog = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  let rows = catalog.value.filter((item) => {
    if (category.value !== 'all' && !item.categories.includes(category.value)) {
      return false
    }
    if (!q) {
      return true
    }
    const haystack = [
      t(item.nameKey),
      t(item.summaryKey),
      t(item.authorKey),
      item.id,
      ...item.tags
    ].join(' ').toLowerCase()
    return haystack.includes(q)
  })

  if (sort.value === 'name') {
    rows = [...rows].sort((a, b) => t(a.nameKey).localeCompare(t(b.nameKey), undefined, { sensitivity: 'base' }))
  } else if (sort.value === 'updated') {
    // 占位：无真实更新时间，按 id 倒序模拟
    rows = [...rows].sort((a, b) => b.id.localeCompare(a.id))
  }

  return rows
})

useSeoMeta({
  title: () => t(metaTitleKey.value)
})
</script>

<template>
  <div
    :data-testid="kind === 'theme' ? 'admin-extension-store-themes-page' : 'admin-extension-store-plugins-page'"
    class="min-w-0 shrink-0"
  >
    <div class="mb-4 flex flex-col gap-1">
      <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
        <UIcon
          :name="adminPage.icon"
          class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]"
        />
        {{ t(titleKey) }}
      </h2>
      <p class="text-sm text-slate-500 dark:text-zinc-400">
        {{ t(introKey) }}
      </p>
    </div>

    <!-- 粘性筛选条：搜索 + 排序 + 分类 Chip -->
    <div class="sticky top-0 z-20 -mx-1 mb-5 bg-gradient-to-b from-slate-50 via-slate-50/95 to-transparent px-1 pb-3 pt-1 dark:from-zinc-950 dark:via-zinc-950/95">
      <div class="rounded-xl border border-slate-200 bg-white p-3 shadow-sm dark:border-zinc-800 dark:bg-zinc-900 sm:p-4">
        <div class="mb-3 flex flex-col gap-2 sm:flex-row sm:items-center">
          <UInput
            v-model="searchQuery"
            icon="i-lucide-search"
            :placeholder="t('admin.extensions.store.searchPlaceholder')"
            class="min-w-0 flex-1"
            size="md"
          />
          <USelect
            v-model="sort"
            :items="sortItems"
            class="w-full sm:w-44"
            size="md"
          />
          <span class="shrink-0 text-xs font-semibold tabular-nums text-slate-500 dark:text-zinc-400 sm:ml-1">
            {{ t(countKey, { visible: filteredCatalog.length, total: catalog.length }) }}
          </span>
        </div>
        <div class="flex flex-wrap gap-2">
          <UButton
            v-for="item in categoryOptions"
            :key="item.id"
            size="sm"
            :color="category === item.id ? 'primary' : 'neutral'"
            :variant="category === item.id ? 'soft' : 'outline'"
            class="rounded-full"
            @click="category = item.id"
          >
            {{ t(item.labelKey) }}
            <span class="ms-1 opacity-70 tabular-nums">{{ item.count }}</span>
          </UButton>
        </div>
      </div>
    </div>

    <div
      v-if="filteredCatalog.length === 0"
      class="mb-6 overflow-hidden rounded-lg border border-slate-200 bg-white p-10 dark:border-zinc-800 dark:bg-zinc-900"
    >
      <SFEmptyState
        icon-label="APP"
        :title="t(emptyTitleKey)"
        :description="t(emptyDescriptionKey)"
      />
    </div>

    <div
      v-else
      class="mb-6 grid gap-3 sm:grid-cols-2 xl:grid-cols-3"
    >
      <article
        v-for="item in filteredCatalog"
        :key="item.id"
        class="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900"
      >
        <div class="flex items-start gap-3">
          <div
            class="flex size-11 shrink-0 items-center justify-center rounded-xl"
            :class="item.accent"
          >
            <UIcon :name="item.icon" class="size-5" />
          </div>
          <div class="min-w-0">
            <h3 class="truncate text-sm font-semibold text-slate-900 dark:text-zinc-100">
              {{ t(item.nameKey) }}
            </h3>
            <p class="mt-0.5 truncate text-xs text-slate-500 dark:text-zinc-400">
              {{ t(item.authorKey) }}
            </p>
          </div>
        </div>
        <p class="line-clamp-2 flex-1 text-sm leading-5 text-slate-600 dark:text-zinc-300">
          {{ t(item.summaryKey) }}
        </p>
        <div class="flex flex-wrap gap-1.5">
          <UBadge
            v-for="tag in item.tags"
            :key="tag"
            color="neutral"
            variant="subtle"
            size="sm"
          >
            {{ tag }}
          </UBadge>
        </div>
        <div class="flex items-center justify-between gap-2 border-t border-dashed border-slate-200 pt-3 dark:border-zinc-800">
          <span class="text-xs font-medium tabular-nums text-slate-500 dark:text-zinc-400">
            {{ item.version }}
          </span>
          <UButton size="sm" color="neutral" variant="subtle" disabled>
            {{ t('admin.extensions.store.installDisabled') }}
          </UButton>
        </div>
      </article>
    </div>

    <!-- 底部即将上线提示 -->
    <div
      class="flex flex-col gap-3 rounded-xl border border-dashed border-[color-mix(in_srgb,var(--sf-accent)_40%,#e2e8f0)] bg-gradient-to-br from-[color-mix(in_srgb,var(--sf-accent)_8%,white)] to-white p-4 dark:border-[color-mix(in_srgb,var(--sf-accent-dark)_40%,#3f3f46)] dark:from-[color-mix(in_srgb,var(--sf-accent-dark)_12%,#18181b)] dark:to-zinc-900 sm:flex-row sm:items-center"
    >
      <UBadge color="primary" variant="solid" class="w-fit shrink-0">
        {{ t('admin.extensions.store.comingSoonBadge') }}
      </UBadge>
      <div class="min-w-0 flex-1">
        <p class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
          {{ t('admin.extensions.store.comingSoonTitle') }}
        </p>
        <p class="mt-0.5 text-sm leading-5 text-slate-500 dark:text-zinc-400">
          {{ t('admin.extensions.store.comingSoonDescription') }}
        </p>
      </div>
      <div class="flex shrink-0 flex-wrap gap-2">
        <UButton
          size="sm"
          color="neutral"
          variant="subtle"
          :icon="localListIcon"
          :to="adminRoutes.path(localListPath)"
        >
          {{ t(localListLabelKey) }}
        </UButton>
      </div>
    </div>
  </div>
</template>
