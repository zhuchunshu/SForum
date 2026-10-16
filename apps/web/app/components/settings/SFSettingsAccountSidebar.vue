<script setup lang="ts">
import { useAuthSession } from '~/composables/identity/useAuthSession'
import { FORUM_PERMISSIONS, usePermissions } from '~/composables/identity/usePermissions'
import { useForumApi } from '~/composables/forum/useForumApi'
import { useAccountSettingsNavigation, type AccountSettingsNavItem } from '~/composables/settings/useAccountSettingsNavigation'
import SFSettingsAccountNav from '~/components/settings/SFSettingsAccountNav.vue'
import SFHomeNavigation from '~/components/forum/SFHomeNavigation.vue'
import SFResponsivePublicSidebar from '~/components/forum/navigation/SFResponsivePublicSidebar.vue'

/**
 * 账号设置页左栏：站点导航 + 账号设置分组 + 扩展分组 + 我的空间。
 *
 * 这是账号设置左栏的**唯一**实现：`SFSettingsShell`（主题岛页）与插件公开页的
 * `sf-account-settings-sidebar` 岛都渲染它，因此插件页的左栏与 /settings/profile
 * 始终是同构的——核心 7 项、扩展项、计数徽标、移动端抽屉所有者声明只有一个来源。
 * 复制一份标记或另接一份数据都会重新引入漂移，不要在别处再写一段左栏。
 */
const props = defineProps<{
  /** 账号导航高亮键：核心项为 profile/appearance/…，扩展项为 registry 节点 id。 */
  active: string
  /** 公开主页链接；缺省时按当前会话用户名推导，避免插件页少一组「我的空间」。 */
  publicProfilePath?: string
}>()

const emit = defineEmits<{
  navigate: []
}>()

const { t } = useI18n()
const localePath = useLocalePath()
const forumApi = useForumApi()
const accountSettingsApi = useAccountSettingsNavigation()
const { can } = usePermissions()
const { user } = useAuthSession()

const { data: categoryGroups, pending: categoriesPending } = await useAsyncData(
  'settings-categories',
  () => forumApi.listCategoryGroups(),
  { default: () => [] }
)
const { data: accountSettingsNavigation } = await useAsyncData(
  'settings-account-navigation',
  async () => {
    try {
      const response = await accountSettingsApi.list()
      return Array.isArray(response?.items) ? response.items : []
    } catch {
      return [] as AccountSettingsNavItem[]
    }
  },
  { default: () => [] as AccountSettingsNavItem[] }
)

const categories = computed(() => categoryGroups.value.flatMap(group => group.categories || []))
const categoryTopicTotal = computed(() => categories.value.reduce((sum, category) => sum + (category.topicCount || 0), 0))
const canCreateTopic = computed(() => can(FORUM_PERMISSIONS.topicCreate))
const resolvedPublicProfilePath = computed(() => props.publicProfilePath
  || (user.value?.username ? localePath(`/u/${user.value.username}`) : undefined))
</script>

<template>
  <SFResponsivePublicSidebar
    owner-id="forum.settings"
    :title="t('home.sidebar.drawerTitle')"
    class="sforum-settings__sidebar sforum-home__sidebar"
  >
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
          :extension-items="accountSettingsNavigation"
          :public-profile-path="resolvedPublicProfilePath"
          @navigate="emit('navigate')"
        />
      </template>
    </SFHomeNavigation>
  </SFResponsivePublicSidebar>
</template>

<!-- 三栏 chrome 的样式表挂在这里：本组件是「账号设置左栏」的唯一实现，
     SFSettingsShell 与插件公开页的 sf-account-settings-sidebar 岛都渲染它，
     因此无论走哪条渲染路径，sforum-settings.css 都在场。
     只挂在 Shell 上会让插件页丢掉 .sforum-settings__layout 的网格。 -->
<style src="~/assets/css/sforum-settings.css" lang="css"></style>
