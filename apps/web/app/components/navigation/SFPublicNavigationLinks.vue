<script setup lang="ts">
import {
  isCoreDynamicCategories,
  isExternalNavigationItem,
  isInternalNavigationItem,
  type PublicNavigationItem
} from '~/utils/navigation/publicNavigation'

const props = withDefaults(defineProps<{
  items: PublicNavigationItem[]
  mode?: 'topbar' | 'mobile' | 'footer'
  visibleLimit?: number
}>(), {
  mode: 'topbar',
  visibleLimit: 4
})

const emit = defineEmits<{ navigate: [] }>()
const { t } = useI18n()
const route = useRoute()
const localePath = useLocalePath()
const slots = useSlots()

const safeItems = computed(() => props.items.filter(item =>
  isExternalNavigationItem(item)
  || isInternalNavigationItem(item)
  || (Boolean(slots.dynamic) && isCoreDynamicCategories(item))
))
const visibleItems = computed(() => props.mode === 'topbar'
  ? safeItems.value.slice(0, props.visibleLimit)
  : safeItems.value)
const overflowItems = computed(() => props.mode === 'topbar'
  ? safeItems.value.slice(props.visibleLimit)
  : [])

function itemTo(item: PublicNavigationItem) {
  const href = (item.href || '').trim()
  return isExternalNavigationItem(item) ? href : localePath(href || '/')
}

function isActive(item: PublicNavigationItem) {
  if (isExternalNavigationItem(item)) return false
  const target = String(itemTo(item)).split('?')[0]?.replace(/\/$/, '') || '/'
  const current = route.path.replace(/\/$/, '') || '/'
  const home = String(localePath('/')).replace(/\/$/, '') || '/'
  return target === home ? current === home : current === target || current.startsWith(`${target}/`)
}

const overflowMenuItems = computed(() => overflowItems.value.map(item => ({
  label: item.label,
  icon: item.iconHidden ? undefined : item.icon || undefined,
  to: itemTo(item),
  target: item.openInNewTab || isExternalNavigationItem(item) ? '_blank' : undefined,
  rel: item.openInNewTab || isExternalNavigationItem(item) ? 'noopener noreferrer' : undefined,
  active: isActive(item),
  onSelect: () => emit('navigate')
})))

const overflowActive = computed(() => overflowItems.value.some(isActive))
</script>

<template>
  <nav
    class="sf-public-navigation-links"
    :class="`sf-public-navigation-links--${mode}`"
    :aria-label="t('nav.mainNav')"
  >
    <template v-for="item in visibleItems" :key="item.sourceKey">
      <slot v-if="isCoreDynamicCategories(item)" name="dynamic" :item="item" />
      <a
        v-else-if="item.openInNewTab || isExternalNavigationItem(item)"
        :href="itemTo(item)"
        class="sf-public-navigation-links__link"
        :class="{ 'is-active': isActive(item) }"
        target="_blank"
        rel="noopener noreferrer"
        :title="item.label"
        @click="emit('navigate')"
      >
        <UIcon v-if="!item.iconHidden && item.icon" :name="item.icon" class="size-4" aria-hidden="true" />
        <span>{{ item.label }}</span>
      </a>
      <NuxtLink
        v-else
        :to="itemTo(item)"
        class="sf-public-navigation-links__link"
        :class="{ 'is-active': isActive(item) }"
        active-class=""
        exact-active-class=""
        :title="item.label"
        @click="emit('navigate')"
      >
        <UIcon v-if="!item.iconHidden && item.icon" :name="item.icon" class="size-4" aria-hidden="true" />
        <span>{{ item.label }}</span>
      </NuxtLink>
    </template>

    <UDropdownMenu
      v-if="overflowMenuItems.length"
      :items="overflowMenuItems"
      :content="{ align: 'start' }"
    >
      <UButton
        color="neutral"
        variant="ghost"
        class="sf-public-navigation-links__more"
        :class="{ 'is-active': overflowActive }"
        :aria-label="t('nav.more')"
        :title="t('nav.more')"
      >
        <span>{{ t('nav.more') }}</span>
        <UIcon name="i-lucide-chevron-down" class="size-3.5" aria-hidden="true" />
      </UButton>
    </UDropdownMenu>
  </nav>
</template>

<style scoped>
.sf-public-navigation-links {
  display: flex;
  align-items: center;
}

.sf-public-navigation-links--topbar {
  min-width: 0;
  align-self: stretch;
  flex: 0 1 auto;
  gap: 20px;
}

.sf-public-navigation-links__link {
  display: flex;
  position: relative;
  min-width: 0;
  align-items: center;
  gap: 6px;
  color: var(--sf-public-text-secondary, #4f5869);
  font-size: 13px;
  font-weight: 600;
  text-decoration: none;
}

.sf-public-navigation-links--topbar .sf-public-navigation-links__link {
  min-height: 100%;
  max-width: 140px;
}

.sf-public-navigation-links--topbar .sf-public-navigation-links__link span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sf-public-navigation-links__link:hover,
.sf-public-navigation-links__link.is-active,
.sf-public-navigation-links__more.is-active {
  color: var(--sf-public-text, #151922);
}

.sf-public-navigation-links--topbar .sf-public-navigation-links__link.is-active::after,
.sf-public-navigation-links--topbar .sf-public-navigation-links__more.is-active::after {
  content: "";
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  height: 2px;
  background: var(--sf-accent);
}

.sf-public-navigation-links__more {
  position: relative;
  min-height: 100%;
  padding: 0;
  border-radius: 0;
  font-size: 13px;
  font-weight: 600;
}

.sf-public-navigation-links--mobile {
  width: 100%;
  flex-direction: column;
  align-items: stretch;
  gap: 4px;
}

.sf-public-navigation-links--footer {
  flex-wrap: wrap;
  justify-content: center;
  gap: 0.75rem 1.25rem;
}

.sf-public-navigation-links--footer .sf-public-navigation-links__link {
  font-size: 0.8125rem;
  color: var(--text-muted);
}

.sf-public-navigation-links--mobile .sf-public-navigation-links__link {
  min-height: 42px;
  padding: 0 12px;
  border-radius: 7px;
}

.sf-public-navigation-links--mobile .sf-public-navigation-links__link:hover,
.sf-public-navigation-links--mobile .sf-public-navigation-links__link.is-active {
  background: var(--sf-public-surface-muted);
}

.dark .sf-public-navigation-links__link,
.dark .sf-public-navigation-links__more {
  color: #d4d4d8;
}

.dark .sf-public-navigation-links__link:hover,
.dark .sf-public-navigation-links__link.is-active,
.dark .sf-public-navigation-links__more.is-active {
  color: #f4f4f5;
}

.dark .sf-public-navigation-links--topbar .sf-public-navigation-links__link.is-active::after,
.dark .sf-public-navigation-links--topbar .sf-public-navigation-links__more.is-active::after {
  background: var(--sf-accent-dark);
}
</style>
