<script setup lang="ts">
if (import.meta.env.PROD) {
  showError({
    statusCode: 404,
    statusMessage: 'Page not found',
    fatal: true
  })
}

useSeoMeta({
  title: 'SF 组件库',
  description: 'SForum 可复用 Vue 组件库预览。'
})

const searchQuery = ref('')
const threadTitle = ref('')
const selectedFilter = ref('all')
const selectedTab = ref('latest')
const currentPage = ref(2)
const digestEnabled = ref(true)
const replyDraft = ref('我建议把版块权限先抽象成角色能力矩阵，这样后续加审核流会更稳。')

const searchFilters = [
  { label: '全部', value: 'all' },
  { label: '主题', value: 'threads' },
  { label: '用户', value: 'users' },
  { label: '标签', value: 'tags' }
]

const tabs = [
  { label: '最新', value: 'latest', badge: 12 },
  { label: '热门', value: 'hot' },
  { label: '精华', value: 'featured' },
  { label: '待审核', value: 'review', badge: 3 }
]

const feedBadges = [
  { label: '公告', variant: 'primary' as const },
  { label: '已解决', variant: 'success' as const }
]

const navItems = [
  { href: '#foundations', label: '基础' },
  { href: '#feedback', label: '反馈' },
  { href: '#forum', label: '论坛' },
  { href: '#states', label: '状态' }
]
</script>

<template>
  <main class="sf-component-page">
    <div class="sf-component-page__shell">
      <header class="sf-component-page__header">
        <div>
          <p class="sf-component-page__eyebrow">
            SF Components
          </p>
          <h1 class="sf-component-page__title">
            SForum 组件库
          </h1>
          <p class="sf-component-page__intro">
            从静态 demo 中拆出的论坛组件层，统一使用 SF 前缀、Pine Teal 视觉令牌和小半径实用界面。
          </p>
        </div>
        <SFBadge variant="primary" dot>
          dev only
        </SFBadge>
      </header>

      <div class="sf-component-page__grid">
        <aside class="sf-component-page__sidebar">
          <SFCard title="组件目录" subtitle="按论坛页面常用场景整理">
            <nav class="sf-preview-stack" aria-label="组件分类">
              <a
                v-for="item in navItems"
                :key="item.href"
                class="design-nav-item"
                :href="item.href"
              >
                {{ item.label }}
              </a>
            </nav>
          </SFCard>
        </aside>

        <div>
          <section id="foundations" class="sf-component-section">
            <SFCard title="基础组件" subtitle="按钮、输入、头像、卡片、开关">
              <div class="sf-preview-grid">
                <div class="sf-preview-stack">
                  <div class="variant-grid">
                    <SFButton>发布主题</SFButton>
                    <SFButton variant="secondary">保存草稿</SFButton>
                    <SFButton variant="ghost">取消</SFButton>
                    <SFButton variant="danger">锁定</SFButton>
                  </div>
                  <SFInput
                    v-model="threadTitle"
                    label="主题标题"
                    placeholder="输入一个清晰的问题标题"
                    hint="标题会优先展示在信息流和搜索结果中。"
                  />
                </div>
                <div class="sf-preview-stack">
                  <div class="variant-grid">
                    <SFAvatar name="林知夏" status="online" />
                    <SFAvatar name="Open Source" shape="square" status="idle" />
                    <SFAvatar name="SF" size="lg" />
                  </div>
                  <SFToggle
                    v-model="digestEnabled"
                    label="接收本帖摘要"
                    description="当主题有高质量回复时发送一次站内通知。"
                  />
                </div>
              </div>
            </SFCard>
          </section>

          <section id="feedback" class="sf-component-section">
            <SFCard title="Alerts 与 Badges" subtitle="页面提示、状态标签、通知反馈">
              <div class="sf-preview-stack">
                <div class="variant-grid">
                  <SFBadge variant="primary" dot>官方公告</SFBadge>
                  <SFBadge variant="neutral">草稿</SFBadge>
                  <SFBadge variant="info">问答</SFBadge>
                  <SFBadge variant="success">已解决</SFBadge>
                  <SFBadge variant="warning">待审核</SFBadge>
                  <SFBadge variant="danger">已锁定</SFBadge>
                </div>
                <SFAlert
                  title="社区规范已更新"
                  description="发布前请确认标题清晰、分类准确，避免重复主题。"
                  variant="primary"
                />
                <SFAlert
                  title="内容等待审核"
                  description="包含外链或敏感关键词时，帖子会进入人工审核队列。"
                  variant="warning"
                  closable
                />
                <SFToast
                  title="回复已保存"
                  description="草稿会保留在当前设备，下次打开主题时继续编辑。"
                  action-label="查看"
                  closable
                />
              </div>
            </SFCard>
          </section>

          <section id="forum" class="sf-component-section">
            <SFCard title="论坛组合组件" subtitle="搜索、信息流、评论、编辑器">
              <div class="sf-preview-stack">
                <SFSearch
                  v-model="searchQuery"
                  v-model:selected-filter="selectedFilter"
                  :filters="searchFilters"
                />
                <SFTabs v-model="selectedTab" :items="tabs" />
                <SFFeedRow
                  title="Nuxt SSR 下如何设计论坛主题详情页缓存？"
                  excerpt="希望列表页能快，详情页也能稳定拿到最新回复，目前在 API cache 和页面级 cache 之间摇摆。"
                  author="林知夏"
                  meta="12 分钟前 · architecture"
                  :score="42"
                  :replies="18"
                  :views="1240"
                  :badges="feedBadges"
                />
                <SFComment
                  author="陈大文"
                  meta="刚刚"
                  content="先拆读模型，再把权限和可见性作为查询条件下沉到 API，前端只负责呈现状态会清爽很多。"
                />
                <SFEditor v-model="replyDraft" />
              </div>
            </SFCard>
          </section>

          <section id="states" class="sf-component-section">
            <SFCard title="加载与空状态" subtitle="分页、进度、骨架屏、空列表">
              <div class="sf-preview-grid">
                <div class="sf-preview-stack">
                  <SFProgress label="资料完整度" :value="68" show-value />
                  <SFPagination v-model:page="currentPage" :total-pages="8" />
                  <SFSkeleton avatar :lines="3" />
                </div>
                <SFEmptyState
                  title="暂无匹配主题"
                  description="调整筛选条件，或新建一个更聚焦的讨论。"
                  action-label="发布主题"
                />
              </div>
            </SFCard>
          </section>
        </div>
      </div>
    </div>
  </main>
</template>
