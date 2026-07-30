<script setup lang="ts">
import { useActiveThemeSettings } from '~/composables/themes/useActiveThemeSettings'
import { usePublicNavigation } from '~/composables/navigation/usePublicNavigation'
import { forumCategoryPath, type ForumCategory } from '~/utils/forum/forumTaxonomy'
import {
  isCoreDynamicCategories,
  isExternalNavigationItem,
  isInternalNavigationItem,
  limitDynamicNavigationItems,
  type PublicNavigationItem
} from '~/utils/navigation/publicNavigation'

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
const { siteName, siteAboutUrl, siteAboutOpenInNewTab } = useWebOptions()
const { sidebarItems } = usePublicNavigation()

const useRouteLinks = computed(() => props.navigationMode === 'route')
const routePath = computed(() => route.path.replace(/\/+$/, '') || '/')
const homePath = computed(() => localePath('/').replace(/\/+$/, '') || '/')
const allTopicsActive = computed(() => useRouteLinks.value
  ? routePath.value === homePath.value && !props.selectedCategorySlug
  : !props.selectedCategorySlug
)
const showCategorySkeleton = computed(() => props.showCategories && props.pending && props.categories.length === 0)
const resolvedSidebarItems = computed(() => sidebarItems.value.filter((item) => {
  if (isCoreDynamicCategories(item)) return props.showCategories
  return Boolean(item.label.trim()) && (isExternalNavigationItem(item) || isInternalNavigationItem(item))
}))
const dynamicCategoryItem = computed(() => sidebarItems.value.find(isCoreDynamicCategories))
const hasDynamicCategories = computed(() => props.showCategories && Boolean(dynamicCategoryItem.value))
const visibleCategories = computed(() => limitDynamicNavigationItems(props.categories, dynamicCategoryItem.value?.maxItems, props.selectedCategorySlug))
const hasHiddenCategories = computed(() => visibleCategories.value.length < props.categories.length)
const aboutSiteLabel = computed(() => t('home.sidebar.aboutSite', { siteName: siteName.value }))
const aboutSiteURL = computed(() => siteAboutUrl.value.trim())
const aboutSiteExternal = computed(() => /^https?:\/\//i.test(aboutSiteURL.value))
const aboutSiteTo = computed(() => aboutSiteURL.value.startsWith('/') && !aboutSiteURL.value.startsWith('//') ? localePath(aboutSiteURL.value) : aboutSiteURL.value)
const aboutSiteTarget = computed(() => siteAboutOpenInNewTab.value ? '_blank' : undefined)
const aboutSiteRel = computed(() => siteAboutOpenInNewTab.value ? 'noopener noreferrer' : undefined)

function allTopicsTo() {
  return localePath('/')
}

function categoryTo(slug: string) {
  return localePath(forumCategoryPath(slug))
}

function navigationItemTo(item: PublicNavigationItem) {
  const href = (item.href || '').trim()
  return isExternalNavigationItem(item) ? href : localePath(href || '/')
}

function navigationItemActive(item: PublicNavigationItem) {
  if (isExternalNavigationItem(item)) return false
  if (item.sourceKey === 'core.home') return allTopicsActive.value
  const target = String(navigationItemTo(item)).split('?')[0]?.replace(/\/+$/, '') || '/'
  return routePath.value === target || routePath.value.startsWith(`${target}/`)
}

function isHomeFilterControl(item: PublicNavigationItem) {
  return !useRouteLinks.value && item.sourceKey === 'core.home' && isInternalNavigationItem(item)
}

function navigationItemCount(item: PublicNavigationItem) {
  return item.sourceKey === 'core.home' ? props.totalTopics : undefined
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
  <aside class="sf-home-navigation" data-navigation-location="public.sidebar.primary" :aria-busy="pending">
    <div v-if="!desktopOnly && hasDynamicCategories" class="sf-home-navigation__mobile">
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
            <option v-for="category in visibleCategories" :key="category.slug" :value="category.slug">
              {{ category.name }} ({{ category.topicCount }})
            </option>
          </select>
          <UIcon name="i-lucide-chevron-down" class="size-4" aria-hidden="true" />
        </span>
      </label>
      <NuxtLink v-if="hasHiddenCategories" :to="localePath('/categories')" class="sf-home-navigation__more">
        {{ t('home.sidebar.viewAllCategories') }}
        <UIcon name="i-lucide-arrow-right" class="size-4" aria-hidden="true" />
      </NuxtLink>
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
      <template v-for="item in resolvedSidebarItems" :key="item.sourceKey">
        <template v-if="isCoreDynamicCategories(item)">
          <div class="sf-home-navigation__label">{{ item.label }}</div>
          <div v-if="showCategorySkeleton" class="sf-home-navigation__pending">
            <SFSkeleton v-for="skeleton in 4" :key="skeleton" :lines="1" />
          </div>
          <template v-if="useRouteLinks">
            <NuxtLink
              v-for="category in visibleCategories"
              :key="category.slug"
              :to="categoryTo(category.slug)"
              class="sf-home-navigation__link"
              :class="{ 'is-active': selectedCategorySlug === category.slug }"
            >
              <span class="sf-home-navigation__link-main">
                <span class="sf-home-navigation__cat-icon" :style="{ color: categoryIconColor(category) }" aria-hidden="true">
                  <UIcon :name="categoryIconName(category)" class="size-[18px]" />
                </span>
                {{ category.name }}
              </span>
              <span v-if="navShowCounts" class="sf-home-navigation__count">{{ category.topicCount }}</span>
            </NuxtLink>
          </template>
          <template v-else>
            <button
              v-for="category in visibleCategories"
              :key="category.slug"
              type="button"
              class="sf-home-navigation__link"
              :class="{ 'is-active': selectedCategorySlug === category.slug }"
              :aria-pressed="selectedCategorySlug === category.slug"
              @click="selectCategory(category.slug)"
            >
              <span class="sf-home-navigation__link-main">
                <span class="sf-home-navigation__cat-icon" :style="{ color: categoryIconColor(category) }" aria-hidden="true">
                  <UIcon :name="categoryIconName(category)" class="size-[18px]" />
                </span>
                {{ category.name }}
              </span>
              <span v-if="navShowCounts" class="sf-home-navigation__count">{{ category.topicCount }}</span>
            </button>
          </template>
          <NuxtLink v-if="hasHiddenCategories" :to="localePath('/categories')" class="sf-home-navigation__more">
            {{ t('home.sidebar.viewAllCategories') }}
            <UIcon name="i-lucide-arrow-right" class="size-4" aria-hidden="true" />
          </NuxtLink>
        </template>
        <button
          v-else-if="isHomeFilterControl(item)"
          type="button"
          class="sf-home-navigation__link"
          :class="{ 'is-active': navigationItemActive(item) }"
          :aria-pressed="navigationItemActive(item)"
          @click="selectCategory('')"
        >
          <span class="sf-home-navigation__link-main">
            <UIcon v-if="!item.iconHidden && item.icon" :name="item.icon" class="size-[18px]" aria-hidden="true" />
            {{ item.label }}
          </span>
          <span v-if="navShowCounts" class="sf-home-navigation__count">{{ navigationItemCount(item) }}</span>
        </button>
        <a
          v-else-if="item.openInNewTab || isExternalNavigationItem(item)"
          :href="navigationItemTo(item)"
          class="sf-home-navigation__link"
          target="_blank"
          rel="noopener noreferrer"
          :title="item.label"
        >
          <span class="sf-home-navigation__link-main">
            <UIcon v-if="!item.iconHidden && item.icon" :name="item.icon" class="size-[18px]" aria-hidden="true" />
            {{ item.label }}
          </span>
          <span v-if="navShowCounts && navigationItemCount(item) !== undefined" class="sf-home-navigation__count">{{ navigationItemCount(item) }}</span>
        </a>
        <NuxtLink
          v-else
          :to="navigationItemTo(item)"
          class="sf-home-navigation__link"
          :class="{ 'is-active': navigationItemActive(item) }"
          active-class=""
          exact-active-class=""
          :title="item.label"
        >
          <span class="sf-home-navigation__link-main">
            <UIcon v-if="!item.iconHidden && item.icon" :name="item.icon" class="size-[18px]" aria-hidden="true" />
            {{ item.label }}
          </span>
          <span v-if="navShowCounts && navigationItemCount(item) !== undefined" class="sf-home-navigation__count">{{ navigationItemCount(item) }}</span>
        </NuxtLink>
      </template>

      <slot name="after-navigation" />

      <div class="sf-home-navigation__foot">
        <a
          v-if="aboutSiteURL && aboutSiteExternal"
          :href="aboutSiteURL"
          class="sf-home-navigation__foot-item"
          :target="aboutSiteTarget"
          :rel="aboutSiteRel"
        >
          <UIcon name="i-lucide-info" class="size-4" aria-hidden="true" />
          {{ aboutSiteLabel }}
        </a>
        <NuxtLink
          v-else-if="aboutSiteURL"
          :to="aboutSiteTo"
          class="sf-home-navigation__foot-item"
          :target="aboutSiteTarget"
          :rel="aboutSiteRel"
        >
          <UIcon name="i-lucide-info" class="size-4" aria-hidden="true" />
          {{ aboutSiteLabel }}
        </NuxtLink>
        <span v-else class="sf-home-navigation__foot-item">
          <UIcon name="i-lucide-info" class="size-4" aria-hidden="true" />
          {{ aboutSiteLabel }}
        </span>
      </div>
    </div>
  </aside>
</template>
