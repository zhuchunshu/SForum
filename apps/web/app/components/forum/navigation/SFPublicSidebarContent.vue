<script setup lang="ts">
import { useActiveThemeSettings } from '~/composables/themes/useActiveThemeSettings'
import { forumCategoryPath, type ForumCategory } from '~/utils/forum/forumTaxonomy'
import {
  isCoreDynamicCategories,
  isExternalNavigationItem,
  isInternalNavigationItem,
  type PublicNavigationItem
} from '~/utils/navigation/publicNavigation'
import SFCategoryNavigationBlock from './SFCategoryNavigationBlock.vue'

const props = withDefaults(defineProps<{
  items: PublicNavigationItem[]
  categories?: ForumCategory[]
  selectedCategorySlug?: string
  totalTopics?: number
  pending?: boolean
  canCreateTopic?: boolean
  showCategories?: boolean
  navigationMode?: 'filter' | 'route'
}>(), {
  categories: () => [],
  selectedCategorySlug: '',
  totalTopics: 0,
  pending: false,
  canCreateTopic: false,
  showCategories: true,
  navigationMode: 'filter'
})

const emit = defineEmits<{
  'select-category': [slug: string]
  navigate: []
}>()

const { t } = useI18n()
const localePath = useLocalePath()
const router = useRouter()
const route = useRoute()
const { navShowCompose, navShowCounts } = useActiveThemeSettings()
const { siteName, siteAboutUrl, siteAboutOpenInNewTab } = useWebOptions()

const useRouteLinks = computed(() => props.navigationMode === 'route')
const routePath = computed(() => route.path.replace(/\/+$/, '') || '/')
const homePath = computed(() => localePath('/').replace(/\/+$/, '') || '/')
const allTopicsActive = computed(() => useRouteLinks.value
  ? routePath.value === homePath.value && !props.selectedCategorySlug
  : !props.selectedCategorySlug
)
const resolvedItems = computed(() => props.items.filter((item) => {
  if (isCoreDynamicCategories(item)) return props.showCategories
  return Boolean(item.label.trim()) && (isExternalNavigationItem(item) || isInternalNavigationItem(item))
}))
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
    emit('navigate')
    void router.push(slug ? categoryTo(slug) : allTopicsTo())
    return
  }
  if (slug) {
    emit('navigate')
    void navigateTo(forumCategoryPath(slug))
    return
  }
  emit('select-category', slug)
}
</script>

<template>
  <div class="sf-home-navigation__desktop">
    <template v-if="navShowCompose">
      <NuxtLink
        :to="canCreateTopic ? localePath('/topics/new') : localePath('/login')"
        class="sf-home-navigation__compose"
        @click="emit('navigate')"
      >
        <UIcon :name="canCreateTopic ? 'i-lucide-plus' : 'i-lucide-log-in'" class="size-4" aria-hidden="true" />
        {{ canCreateTopic ? t('home.sidebar.newTopic') : t('home.loginToPost') }}
      </NuxtLink>
    </template>

    <div class="sf-home-navigation__label">{{ t('home.sidebar.navTitle') }}</div>
    <template v-for="item in resolvedItems" :key="item.sourceKey">
      <SFCategoryNavigationBlock
        v-if="isCoreDynamicCategories(item)"
        :categories="categories"
        :selected-category-slug="selectedCategorySlug"
        :label="item.label"
        :max-items="item.maxItems"
        :pending="pending"
        :show-counts="navShowCounts"
        :navigation-mode="navigationMode"
        @select-category="selectCategory"
        @navigate="emit('navigate')"
      />
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
        @click="emit('navigate')"
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
        @click="emit('navigate')"
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
        @click="emit('navigate')"
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
        @click="emit('navigate')"
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
</template>
