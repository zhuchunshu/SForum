<script setup lang="ts">
import {
  forumCategoriesIndexPath,
  forumCategoryPath,
  type ForumCategory,
  type ForumCategoryGroup
} from '~/utils/forumTaxonomy'

type CategorySort = 'default' | 'active' | 'name'

const { t } = useI18n()
const localePath = useLocalePath()
const { seoSettings, siteName } = useWebOptions()
const forumApi = useForumApi()

const sort = ref<CategorySort>('default')

const { data: groups, pending } = await useAsyncData(
  'forum-categories-index',
  () => forumApi.listCategoryGroups(),
  { default: () => [] as ForumCategoryGroup[] }
)

// API 已过滤 public；前端再防御一层 hidden。
const visibleGroups = computed(() => {
  const source = groups.value || []
  return source
    .filter((group) => group.visibility !== 'hidden')
    .map((group) => ({
      ...group,
      categories: (group.categories || []).filter((category) => category.visibility !== 'hidden')
    }))
    .filter((group) => group.categories.length > 0)
})

const displayGroups = computed(() => {
  return visibleGroups.value.map((group) => {
    const categories = sortCategories(group.categories, sort.value)
    return { ...group, categories }
  })
})

useSForumSeo(computed(() => ({
  type: 'static' as const,
  path: forumCategoriesIndexPath(),
  title: t('taxonomy.categories.title'),
  description: t('taxonomy.categories.description'),
  public: true,
  breadcrumbs: [
    { name: seoSettings.value.seoSiteName || siteName.value, path: '/' },
    { name: t('taxonomy.categories.title'), path: forumCategoriesIndexPath() }
  ]
})))

function sortCategories(categories: ForumCategory[], mode: CategorySort) {
  const list = categories.slice()
  switch (mode) {
    case 'active':
      return list.sort((a, b) =>
        (b.topicCount || 0) - (a.topicCount || 0)
        || a.name.localeCompare(b.name, undefined, { sensitivity: 'base' })
      )
    case 'name':
      return list.sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: 'base' }))
    default:
      return list
  }
}

function setSort(next: CategorySort) {
  sort.value = next
}

function tileAccent(category: ForumCategory) {
  return category.iconColor?.trim() || 'var(--sf-accent)'
}

// 管理端 icon 多为 Iconify 名（i-lucide-*）；否则回退首字。
function tileIconName(category: ForumCategory) {
  const icon = category.icon?.trim() || ''
  if (icon.startsWith('i-')) {
    return icon
  }
  return ''
}

function tileIconFallback(category: ForumCategory) {
  const name = category.name?.trim() || category.slug || '?'
  return name.slice(0, 1)
}

function formatCount(value: number) {
  return new Intl.NumberFormat(undefined).format(value)
}
</script>

<template>
  <SFPageOutlet page="forum.category.index">
  <main class="sforum-taxonomy">
    <div class="sforum-taxonomy__shell">
      <header class="sforum-taxonomy__head">
        <div>
          <h1>{{ t('taxonomy.categories.title') }}</h1>
          <p>{{ t('taxonomy.categories.description') }}</p>
        </div>
        <div class="sforum-taxonomy__tools">
          <button
            type="button"
            class="sforum-taxonomy__chip"
            :class="{ 'is-active': sort === 'default' }"
            @click="setSort('default')"
          >
            {{ t('taxonomy.categories.sorts.all') }}
          </button>
          <button
            type="button"
            class="sforum-taxonomy__chip"
            :class="{ 'is-active': sort === 'active' }"
            @click="setSort('active')"
          >
            {{ t('taxonomy.categories.sorts.active') }}
          </button>
          <button
            type="button"
            class="sforum-taxonomy__chip"
            :class="{ 'is-active': sort === 'name' }"
            @click="setSort('name')"
          >
            {{ t('taxonomy.categories.sorts.name') }}
          </button>
        </div>
      </header>

      <div v-if="pending" class="sforum-taxonomy__pending" aria-busy="true">
        <SFSkeleton :lines="3" />
        <SFSkeleton :lines="3" />
      </div>

      <template v-else-if="displayGroups.length">
        <section
          v-for="group in displayGroups"
          :key="group.id"
          class="sforum-taxonomy__group"
        >
          <div class="sforum-taxonomy__group-label">{{ group.name }}</div>
          <div class="sforum-taxonomy__grid">
            <NuxtLink
              v-for="category in group.categories"
              :key="category.id"
              :to="localePath(forumCategoryPath(category.slug))"
              class="sforum-taxonomy__tile"
              :style="{ '--tile-accent': tileAccent(category) }"
            >
              <div class="sforum-taxonomy__tile-top">
                <div class="sforum-taxonomy__tile-icon" aria-hidden="true">
                  <UIcon
                    v-if="tileIconName(category)"
                    :name="tileIconName(category)"
                    class="size-5"
                  />
                  <template v-else>
                    {{ tileIconFallback(category) }}
                  </template>
                </div>
              </div>
              <h2>{{ category.name }}</h2>
              <p>{{ category.description || t('taxonomy.categories.noDescription') }}</p>
              <div class="sforum-taxonomy__tile-meta">
                <div class="sforum-taxonomy__tile-nums">
                  {{ t('taxonomy.categories.topicCount', { count: formatCount(category.topicCount || 0) }) }}
                  <span>·</span>
                  {{ t('taxonomy.categories.replyCount', { count: formatCount(category.commentCount || 0) }) }}
                </div>
                <span class="sforum-taxonomy__tile-go">{{ t('taxonomy.categories.enter') }}</span>
              </div>
            </NuxtLink>
          </div>
        </section>
      </template>

      <div v-else class="sforum-taxonomy__empty">
        <SFEmptyState
          :title="t('taxonomy.categories.emptyTitle')"
          :description="t('taxonomy.categories.emptyDescription')"
        />
      </div>
    </div>
  </main>

  </SFPageOutlet>
</template>
