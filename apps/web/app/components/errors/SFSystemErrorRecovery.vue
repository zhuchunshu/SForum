<script setup lang="ts">
import { useSystemErrorRecoveryData } from '~/composables/errors/useSystemErrorRecoveryData'
import SFSystemErrorSidebar from '~/components/errors/SFSystemErrorSidebar.vue'
import SFSystemErrorRail from '~/components/errors/SFSystemErrorRail.vue'
const { t } = useI18n()
const localePath = useLocalePath()
const { activeTags, categories } = useSystemErrorRecoveryData()
const searchAction = computed(() => localePath('/'))
const mobileMenuOpen = useState<boolean>('system-error-mobile-menu-open', () => false)
const mobileInfoOpen = useState<boolean>('system-error-mobile-info-open', () => false)

const recoveryLinks = computed(() => [
  { key: 'home', label: t('errors.page.recovery.home'), to: localePath('/'), icon: 'i-lucide-house' },
  { key: 'categories', label: t('errors.page.recovery.categories'), to: localePath('/categories'), icon: 'i-lucide-folder-tree' },
  { key: 'tags', label: t('errors.page.recovery.tags'), to: localePath('/tags'), icon: 'i-lucide-tags' },
  { key: 'guidelines', label: t('errors.page.recovery.guidelines'), to: localePath('/guidelines'), icon: 'i-lucide-book-open' }
])

function closeMobileDrawers() {
  mobileMenuOpen.value = false
  mobileInfoOpen.value = false
}
</script>

<template>
  <section class="sforum-system-error__recovery" data-system-error-region="recovery">
    <div class="sforum-system-error__mobile-tools" role="group" :aria-label="t('errors.page.recovery.mobileTools')">
      <button type="button" class="sforum-system-error__mobile-tool" @click="mobileMenuOpen = true">
        <UIcon name="i-lucide-menu" aria-hidden="true" />
        {{ t('home.sidebar.drawerTitle') }}
      </button>
      <button type="button" class="sforum-system-error__mobile-tool" @click="mobileInfoOpen = true">
        <UIcon name="i-lucide-panel-right-open" aria-hidden="true" />
        {{ t('home.rightRail.drawerTitle') }}
      </button>
    </div>

    <form class="sforum-system-error__search" role="search" method="get" :action="searchAction">
      <label class="sforum-system-error__search-box">
        <UIcon name="i-lucide-search" class="sforum-system-error__search-icon" aria-hidden="true" />
        <input
          class="sforum-system-error__search-input"
          type="search"
          name="q"
          :placeholder="t('errors.page.recovery.searchPlaceholder')"
          :aria-label="t('errors.page.recovery.searchPlaceholder')"
        >
        <button type="submit" class="sforum-system-error__search-submit">
          {{ t('errors.page.recovery.searchSubmit') }}
        </button>
      </label>
    </form>

    <div class="sforum-system-error__continue">
      <h2 class="sforum-system-error__section-title">{{ t('errors.page.recovery.continueTitle') }}</h2>
      <div class="sforum-system-error__continue-grid">
        <NuxtLink
          v-for="link in recoveryLinks"
          :key="link.key"
          class="sforum-system-error__continue-link"
          :to="link.to"
        >
          <UIcon :name="link.icon" aria-hidden="true" />
          <span>{{ link.label }}</span>
        </NuxtLink>
      </div>
      <div v-if="activeTags.length || categories.length" class="sforum-system-error__quick-links">
        <NuxtLink
          v-for="category in categories.slice(0, 5)"
          :key="`category-${category.slug}`"
          :to="localePath(`/c/${category.slug}`)"
        >
          {{ category.name }}
        </NuxtLink>
        <NuxtLink
          v-for="tag in activeTags.slice(0, 5)"
          :key="`tag-${tag.slug}`"
          :to="localePath(`/tags/${tag.slug}`)"
        >
          #{{ tag.name }}
        </NuxtLink>
      </div>
    </div>

    <button
      v-if="mobileMenuOpen || mobileInfoOpen"
      type="button"
      class="sforum-mobile-drawer__backdrop"
      :aria-label="t('topicDetail.cancel')"
      @click="closeMobileDrawers"
    />
    <aside v-if="mobileMenuOpen" class="sforum-mobile-drawer sforum-mobile-drawer--left">
      <header class="sforum-mobile-drawer__head">
        <strong>{{ t('home.sidebar.drawerTitle') }}</strong>
        <button type="button" :aria-label="t('topicDetail.cancel')" @click="closeMobileDrawers">
          <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
        </button>
      </header>
      <SFSystemErrorSidebar />
    </aside>
    <aside v-if="mobileInfoOpen" class="sforum-mobile-drawer sforum-mobile-drawer--right">
      <header class="sforum-mobile-drawer__head">
        <strong>{{ t('home.rightRail.drawerTitle') }}</strong>
        <button type="button" :aria-label="t('topicDetail.cancel')" @click="closeMobileDrawers">
          <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
        </button>
      </header>
      <SFSystemErrorRail />
    </aside>
  </section>
</template>
