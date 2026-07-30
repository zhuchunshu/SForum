<script setup lang="ts">
import { FORUM_PERMISSIONS, usePermissions } from '~/composables/identity/usePermissions'
import { useForumApi } from '~/composables/forum/useForumApi'
import SFSettingsAccountNav from '~/components/settings/SFSettingsAccountNav.vue'
import SFHomeNavigation from '~/components/forum/SFHomeNavigation.vue'
import SFContentColumnFooter from '~/components/forum/SFContentColumnFooter.vue'
import SFPublicPageHeader from '~/components/public/SFPublicPageHeader.vue'
/**
 * 账号设置页共享 chrome：三栏布局 + 左侧导航 + 页头 + 移动端抽屉。
 * 页面通过插槽提供主内容 (default)、右栏 (rail，桌面 aside 与移动抽屉复用同一份)、
 * 页头附加操作 (head-actions)。根节点为 main，class / data-* 属性自动透传。
 */
const props = withDefaults(defineProps<{
  /** 左栏账号导航高亮项 */
  active: 'profile' | 'loginMethods' | 'password' | 'security' | 'tokens' | 'notifications'
  /** 页头 h1 的 id，供 aria-labelledby 使用 */
  titleId: string
  title: string
  description: string
  /** 右栏 aria-label（同时用作移动端右抽屉标题） */
  railLabel: string
  /** 打开右抽屉按钮的 aria-label */
  railOpenLabel: string
  /** 传给账号导航的公开主页链接（可选） */
  publicProfilePath?: string
  /** 是否渲染右栏与右抽屉（如数据未就绪时可关闭） */
  showRail?: boolean
}>(), {
  showRail: true
})

const { t } = useI18n()
const forumApi = useForumApi()
const { can } = usePermissions()

const { data: categoryGroups, pending: categoriesPending } = await useAsyncData(
  'settings-categories',
  () => forumApi.listCategoryGroups(),
  { default: () => [] }
)

const categories = computed(() => categoryGroups.value.flatMap(group => group.categories || []))
const categoryTopicTotal = computed(() => categories.value.reduce((sum, category) => sum + (category.topicCount || 0), 0))
const canCreateTopic = computed(() => can(FORUM_PERMISSIONS.topicCreate))

const mobileMenuOpen = useState<boolean>('forum-mobile-menu-open', () => false)
const mobileInfoOpen = useState<boolean>('forum-mobile-info-open', () => false)

function closeMobileDrawers() {
  mobileMenuOpen.value = false
  mobileInfoOpen.value = false
}
</script>

<template>
  <main class="sforum-settings" data-layout="fullwidth-3col">
    <div class="sforum-settings__layout">
      <div class="sforum-settings__sidebar sforum-home__sidebar">
        <SFHomeNavigation
          desktop-only
          navigation-mode="route"
          :categories="categories"
          :total-topics="categoryTopicTotal"
          :pending="categoriesPending"
          :can-create-topic="canCreateTopic"
          :show-categories="false"
        >
          <template #after-navigation>
            <SFSettingsAccountNav
              :active="props.active"
              :public-profile-path="props.publicProfilePath"
            />
          </template>
        </SFHomeNavigation>
      </div>

      <section class="sforum-settings__main sforum-content-column" :aria-labelledby="props.titleId">
        <div class="sforum-settings__mobile-nav">
          <SFHomeNavigation
            mobile-only
            navigation-mode="route"
            :categories="categories"
            :total-topics="categoryTopicTotal"
            :pending="categoriesPending"
            :can-create-topic="canCreateTopic"
          />
        </div>

        <SFPublicPageHeader
          class="sforum-settings__head"
          :title-id="props.titleId"
          :title="props.title"
          :subtitle="props.description"
        >
          <template #aside>
            <div class="sforum-settings__head-actions">
              <slot name="head-actions" />
              <button
                type="button"
                class="sforum-settings__icon-button sforum-settings__desktop-hidden"
                :aria-label="props.railOpenLabel"
                @click="mobileInfoOpen = true"
              >
                <UIcon name="i-lucide-panel-right" class="size-[18px]" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="sforum-settings__icon-button sforum-settings__desktop-hidden"
                :aria-label="t('home.sidebar.drawerTitle')"
                @click="mobileMenuOpen = true"
              >
                <UIcon name="i-lucide-menu" class="size-[18px]" aria-hidden="true" />
              </button>
            </div>
          </template>
        </SFPublicPageHeader>

        <slot />

        <SFContentColumnFooter />
      </section>

      <aside
        v-if="props.showRail"
        class="sforum-settings__right"
        :aria-label="props.railLabel"
      >
        <slot name="rail" />
      </aside>
    </div>

    <button
      v-if="mobileMenuOpen || mobileInfoOpen"
      type="button"
      class="sforum-mobile-drawer__backdrop"
      :aria-label="t('common.close')"
      @click="closeMobileDrawers"
    />

    <aside v-if="mobileMenuOpen" class="sforum-mobile-drawer sforum-mobile-drawer--left">
      <header class="sforum-mobile-drawer__head">
        <strong>{{ t('home.sidebar.drawerTitle') }}</strong>
        <button type="button" :aria-label="t('common.close')" @click="closeMobileDrawers">
          <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
        </button>
      </header>
      <SFHomeNavigation
        desktop-only
        navigation-mode="route"
        :categories="categories"
        :total-topics="categoryTopicTotal"
        :pending="categoriesPending"
        :can-create-topic="canCreateTopic"
        :show-categories="false"
      >
        <template #after-navigation>
          <SFSettingsAccountNav
            :active="props.active"
            :public-profile-path="props.publicProfilePath"
            @navigate="closeMobileDrawers"
          />
        </template>
      </SFHomeNavigation>
    </aside>

    <aside v-if="props.showRail && mobileInfoOpen" class="sforum-mobile-drawer sforum-mobile-drawer--right">
      <header class="sforum-mobile-drawer__head">
        <strong>{{ props.railLabel }}</strong>
        <button type="button" :aria-label="t('common.close')" @click="closeMobileDrawers">
          <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
        </button>
      </header>
      <div class="sforum-settings__right sforum-settings__right--drawer" :aria-label="props.railLabel">
        <slot name="rail" />
      </div>
    </aside>
  </main>
</template>

<style src="~/assets/css/sforum-settings.css" lang="css"></style>
