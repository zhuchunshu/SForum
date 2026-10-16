<script setup lang="ts">
import SFSettingsAccountSidebar from '~/components/settings/SFSettingsAccountSidebar.vue'
import SFContentColumnFooter from '~/components/forum/SFContentColumnFooter.vue'
import SFPublicPageHeader from '~/components/public/SFPublicPageHeader.vue'
import SFPublicMobileUserMenu from '~/components/navigation/SFPublicMobileUserMenu.vue'
import SFPublicMobileRightDrawerHeader from '~/components/navigation/SFPublicMobileRightDrawerHeader.vue'
import { usePublicSidebarDrawer } from '~/composables/navigation/usePublicSidebarDrawer'
/**
 * 账号设置页共享 chrome：三栏布局 + 左侧导航 + 页头 + 移动端抽屉。
 * 页面通过插槽提供主内容 (default)、右栏 (rail，桌面 aside 与移动抽屉复用同一份)、
 * 页头附加操作 (head-actions)。根节点为 main，class / data-* 属性自动透传。
 *
 * 左栏由 `SFSettingsAccountSidebar` 拥有（与插件公开页的 `sf-account-settings-sidebar`
 * 岛共用同一份实现），本组件只负责栏位与右栏抽屉。
 */
const props = withDefaults(defineProps<{
  /** 左栏账号导航高亮项 */
  active: string
  /** 页头 h1 的 id，供 aria-labelledby 使用 */
  titleId: string
  title: string
  description: string
  /** 右栏 aria-label（同时用作移动端右抽屉标题） */
  railLabel: string
  /** 传给账号导航的公开主页链接（可选） */
  publicProfilePath?: string
  /** 是否渲染右栏与右抽屉（如数据未就绪时可关闭） */
  showRail?: boolean
}>(), {
  showRail: true
})

const { t } = useI18n()

const mobileInfoOpen = useState<boolean>('forum-mobile-info-open', () => false)
const { closeDrawer: closeMobileMenu } = usePublicSidebarDrawer()

function closeMobileDrawers() {
  closeMobileMenu()
  mobileInfoOpen.value = false
}
</script>

<template>
  <main class="sforum-settings" data-layout="fullwidth-3col">
    <div class="sforum-settings__layout">
      <SFSettingsAccountSidebar
        :active="props.active"
        :public-profile-path="props.publicProfilePath"
        @navigate="closeMobileDrawers"
      />

      <section class="sforum-settings__main sforum-content-column" :aria-labelledby="props.titleId">
        <SFPublicPageHeader
          class="sforum-settings__head"
          :title-id="props.titleId"
          :title="props.title"
          :subtitle="props.description"
        >
          <template #aside>
            <div class="sforum-settings__head-actions">
              <slot name="head-actions" />
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
      v-if="mobileInfoOpen"
      type="button"
      class="sforum-mobile-drawer__backdrop"
      :aria-label="t('common.close')"
      @click="closeMobileDrawers"
    />

    <aside v-if="props.showRail && mobileInfoOpen" class="sforum-mobile-drawer sforum-mobile-drawer--right">
      <SFPublicMobileRightDrawerHeader
        :title="props.railLabel"
        :close-label="t('common.close')"
        @close="closeMobileDrawers"
      />
      <SFPublicMobileUserMenu />
      <div class="sforum-settings__right sforum-settings__right--drawer" :aria-label="props.railLabel">
        <slot name="rail" />
      </div>
    </aside>
  </main>
</template>

<style src="~/assets/css/sforum-settings.css" lang="css"></style>
