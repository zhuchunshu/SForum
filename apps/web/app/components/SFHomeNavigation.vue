<script setup lang="ts">
import { forumCategoryPath, type ForumCategory } from '~/utils/forumTaxonomy'

const props = withDefaults(defineProps<{
  categories?: ForumCategory[]
  selectedCategorySlug?: string
  totalTopics?: number
  pending?: boolean
  canCreateTopic?: boolean
  /** 仅渲染移动端选择器（用于主列顶部） */
  mobileOnly?: boolean
  /** 仅渲染桌面左栏 */
  desktopOnly?: boolean
  /** 导航模式下是否显示分类列表（默认为 true） */
  showCategories?: boolean
  /**
   * filter：首页 URL query 筛选（emit select-category）
   * route：跳转 / 与 /c/:slug（类别页/标签页）
   */
  navigationMode?: 'filter' | 'route'
}>(), {
  categories: () => [],
  selectedCategorySlug: '',
  totalTopics: 0,
  pending: false,
  canCreateTopic: false,
  mobileOnly: false,
  desktopOnly: false,
  showCategories: true,
  navigationMode: 'filter'
})

const emit = defineEmits<{
  'select-category': [slug: string]
}>()

const { t } = useI18n()
const localePath = useLocalePath()
const router = useRouter()
const route = useRoute()
const { navShowCompose, navShowCounts } = useActiveThemeSettings()
const { siteName } = useWebOptions()

const useRouteLinks = computed(() => props.navigationMode === 'route')
const routePath = computed(() => route.path.replace(/\/+$/, '') || '/')
const homePath = computed(() => localePath('/').replace(/\/+$/, '') || '/')
const categoriesPath = computed(() => localePath('/categories').replace(/\/+$/, ''))
const tagsPath = computed(() => localePath('/tags').replace(/\/+$/, ''))
const activeTopLevel = computed(() => {
  if (!useRouteLinks.value) {
    return ''
  }
  if (routePath.value === tagsPath.value || routePath.value.startsWith(`${tagsPath.value}/`)) {
    return 'tags'
  }
  if (routePath.value === categoriesPath.value) {
    return 'categories'
  }
  return routePath.value === homePath.value ? 'home' : ''
})
const allTopicsActive = computed(() => useRouteLinks.value
  ? activeTopLevel.value === 'home' && !props.selectedCategorySlug
  : !props.selectedCategorySlug
)

function allTopicsTo() {
  return localePath('/')
}

function categoryTo(slug: string) {
  return localePath(forumCategoryPath(slug))
}

function selectCategory(slug: string) {
  if (useRouteLinks.value) {
    void router.push(slug ? categoryTo(slug) : allTopicsTo())
    return
  }
  // filter 模式下，点击分类直接跳转到干净的 /c/slug（与首页 ?category= 的统一逻辑一致）
  if (slug) {
    void navigateTo(forumCategoryPath(slug))
    return
  }
  emit('select-category', slug)
}

function selectFromMenu(event: Event) {
  selectCategory((event.target as HTMLSelectElement).value)
}

/** 左栏分类图标颜色：优先管理端 iconColor，否则主色。 */
function categoryIconColor(category: ForumCategory) {
  return category.iconColor?.trim() || 'var(--sf-accent)'
}

/** 管理端配置的 Iconify 名（i-lucide-* / i-tabler-*）；否则回退 folder。 */
function categoryIconName(category: ForumCategory) {
  const icon = category.icon?.trim() || ''
  if (icon.startsWith('i-')) {
    return icon
  }
  return 'i-lucide-folder'
}
</script>

<template>
  <aside class="sf-home-navigation" :aria-busy="pending">
    <div v-if="!desktopOnly" class="sf-home-navigation__mobile">
      <label class="sf-home-navigation__select-wrap">
        <span class="sf-home-navigation__select-label">{{ t('home.categories') }}</span>
        <span class="sf-home-navigation__select-control">
          <UIcon name="i-lucide-layout-list" class="size-4" aria-hidden="true" />
          <select
            class="sf-home-navigation__select"
            :value="selectedCategorySlug"
            @change="selectFromMenu"
          >
            <option value="">{{ t('home.allTopics') }} ({{ totalTopics }})</option>
            <option v-for="category in categories" :key="category.slug" :value="category.slug">
              {{ category.name }} ({{ category.topicCount }})
            </option>
          </select>
          <UIcon name="i-lucide-chevron-down" class="size-4" aria-hidden="true" />
        </span>
      </label>
    </div>

    <div v-if="!mobileOnly" class="sf-home-navigation__desktop">
      <template v-if="navShowCompose">
        <NuxtLink
          v-if="canCreateTopic"
          :to="localePath('/topics/new')"
          class="sf-home-navigation__compose"
        >
          <UIcon name="i-lucide-plus" class="size-4" aria-hidden="true" />
          {{ t('home.sidebar.newTopic') }}
        </NuxtLink>
        <NuxtLink
          v-else
          :to="localePath('/login')"
          class="sf-home-navigation__compose"
        >
          <UIcon name="i-lucide-log-in" class="size-4" aria-hidden="true" />
          {{ t('home.loginToPost') }}
        </NuxtLink>
      </template>

      <div class="sf-home-navigation__label">{{ t('home.sidebar.navTitle') }}</div>
      <NuxtLink
        v-if="useRouteLinks"
        :to="allTopicsTo()"
        class="sf-home-navigation__link"
        :class="{ 'is-active': allTopicsActive }"
      >
        <span class="sf-home-navigation__link-main">
          <UIcon name="i-lucide-layout-list" class="size-[18px]" aria-hidden="true" />
          {{ t('home.allTopics') }}
        </span>
        <span v-if="navShowCounts" class="sf-home-navigation__count">{{ totalTopics }}</span>
      </NuxtLink>
      <button
        v-else
        type="button"
        class="sf-home-navigation__link"
        :class="{ 'is-active': allTopicsActive }"
        :aria-pressed="allTopicsActive"
        @click="selectCategory('')"
      >
        <span class="sf-home-navigation__link-main">
          <UIcon name="i-lucide-layout-list" class="size-[18px]" aria-hidden="true" />
          {{ t('home.allTopics') }}
        </span>
        <span v-if="navShowCounts" class="sf-home-navigation__count">{{ totalTopics }}</span>
      </button>

      <NuxtLink
        :to="localePath('/categories')"
        class="sf-home-navigation__link"
        :class="{ 'is-active': activeTopLevel === 'categories' }"
      >
        <span class="sf-home-navigation__link-main">
          <UIcon name="i-lucide-layout-grid" class="size-[18px]" aria-hidden="true" />
          {{ t('home.categories') }}
        </span>
      </NuxtLink>
      <NuxtLink
        :to="localePath('/tags')"
        class="sf-home-navigation__link"
        :class="{ 'is-active': activeTopLevel === 'tags' }"
      >
        <span class="sf-home-navigation__link-main">
          <UIcon name="i-lucide-tags" class="size-[18px]" aria-hidden="true" />
          {{ t('home.tags') }}
        </span>
      </NuxtLink>

      <div v-if="props.showCategories" class="sf-home-navigation__label">{{ t('home.categories') }}</div>
      <div v-if="props.showCategories" class="sf-home-navigation__pending">
        <SFSkeleton v-for="item in 4" :key="item" :lines="1" />
      </div>
      <template v-if="props.showCategories">
        <template v-if="useRouteLinks">
          <NuxtLink
            v-for="category in categories"
            :key="category.slug"
            :to="categoryTo(category.slug)"
            class="sf-home-navigation__link"
            :class="{ 'is-active': selectedCategorySlug === category.slug }"
          >
            <span class="sf-home-navigation__link-main">
              <span
                class="sf-home-navigation__cat-icon"
                :style="{ color: categoryIconColor(category) }"
                aria-hidden="true"
              >
                <UIcon :name="categoryIconName(category)" class="size-[18px]" />
              </span>
              {{ category.name }}
            </span>
            <span v-if="navShowCounts" class="sf-home-navigation__count">{{ category.topicCount }}</span>
          </NuxtLink>
        </template>
        <template v-else>
          <button
            v-for="category in categories"
            :key="category.slug"
            type="button"
            class="sf-home-navigation__link"
            :class="{ 'is-active': selectedCategorySlug === category.slug }"
            :aria-pressed="selectedCategorySlug === category.slug"
            @click="selectCategory(category.slug)"
          >
            <span class="sf-home-navigation__link-main">
              <span
                class="sf-home-navigation__cat-icon"
                :style="{ color: categoryIconColor(category) }"
                aria-hidden="true"
              >
                <UIcon :name="categoryIconName(category)" class="size-[18px]" />
              </span>
              {{ category.name }}
            </span>
            <span v-if="navShowCounts" class="sf-home-navigation__count">{{ category.topicCount }}</span>
          </button>
        </template>
      </template>

      <div class="sf-home-navigation__foot">
        <NuxtLink :to="localePath('/guidelines')">
          <UIcon name="i-lucide-book-open" class="size-4" aria-hidden="true" />
          {{ t('home.sidebar.guidelines') }}
        </NuxtLink>
        <span class="sf-home-navigation__foot-item">
          <UIcon name="i-lucide-info" class="size-4" aria-hidden="true" />
          {{ t('home.sidebar.aboutSite', { siteName }) }}
        </span>
      </div>
    </div>
  </aside>
</template>
