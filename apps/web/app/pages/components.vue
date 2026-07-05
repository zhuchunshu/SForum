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
const categoryName = ref('架构设计')
const tagName = ref('nuxt-ssr')
const profileName = ref('林知夏')
const profileTitle = ref('社区维护者 / 全栈工程师')
const reviewReason = ref('标题需要更具体，建议补充运行环境和复现步骤。')
const selectedFilter = ref('all')
const selectedTab = ref('latest')
const composerTab = ref('write')
const moderationTab = ref('queue')
const profileTab = ref('overview')
const currentPage = ref(2)
const selectedIcon = ref('i-tabler-message-circle')
const digestEnabled = ref(true)
const mentionEnabled = ref(true)
const anonymousDraft = ref(false)
const hideActivity = ref(false)
const autoCloseQueue = ref(true)
const replyDraft = ref('我建议把版块权限先抽象成角色能力矩阵，这样后续加审核流会更稳。')
const longReplyDraft = ref(`## 编辑器组件验证

这是 **SForum Tiptap** 的第一版编辑器，用来验证自定义 toolbar、表情节点和三格式内容输出。[emoji name="sparkles" label="灵感" native="✨"]

> 客户端输出只用于草稿、预览和提交载荷，最终入库 HTML 必须由 API 重新生成并净化。

- Markdown 会保留给编辑和审计
- HTML 会在服务端净化后用于 SSR 展示
- Tiptap JSON 会作为原生结构保存，方便后续支持附件、@提及和自定义节点
`)

type EditorPreviewPayload = {
  html: string
  markdown: string
  native: unknown
  text: string
  characterCount: number
  wordCount: number
  isEmpty: boolean
}

const editorPreviewPayload = ref<EditorPreviewPayload | null>(null)

const editorHtmlOutput = computed(() => editorPreviewPayload.value?.html || '')
const editorMarkdownOutput = computed(() => editorPreviewPayload.value?.markdown || longReplyDraft.value)
const editorNativeOutput = computed(() => {
  if (!editorPreviewPayload.value) {
    return '{}'
  }

  return JSON.stringify(editorPreviewPayload.value.native, null, 2)
})

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

const composerTabs = [
  { label: '撰写', value: 'write' },
  { label: '预览', value: 'preview' },
  { label: '发布设置', value: 'settings', badge: 2 }
]

const moderationTabs = [
  { label: '待处理', value: 'queue', badge: 8 },
  { label: '已通过', value: 'approved' },
  { label: '已退回', value: 'rejected' }
]

const profileTabs = [
  { label: '概览', value: 'overview' },
  { label: '主题', value: 'threads', badge: 24 },
  { label: '回复', value: 'replies', badge: 128 }
]

const feedBadges = [
  { label: '公告', variant: 'primary' as const },
  { label: '已解决', variant: 'success' as const }
]

const questionBadges = [
  { label: '问答', variant: 'info' as const },
  { label: '高质量回复', variant: 'success' as const }
]

const reviewBadges = [
  { label: '待审核', variant: 'warning' as const },
  { label: '含外链', variant: 'neutral' as const }
]

const navItems = [
  { href: '#foundations', label: '基础' },
  { href: '#icons', label: '图标' },
  { href: '#feedback', label: '反馈' },
  { href: '#forum', label: '论坛' },
  { href: '#composer', label: '发布' },
  { href: '#moderation', label: '审核' },
  { href: '#profile', label: '成员' },
  { href: '#states', label: '状态' }
]

function updateEditorPreview(payload: EditorPreviewPayload) {
  editorPreviewPayload.value = payload
}
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
        <div class="variant-grid">
          <SFBadge variant="primary" dot>
            dev only
          </SFBadge>
          <SFBadge variant="neutral">
            18 components
          </SFBadge>
        </div>
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
                  <div class="variant-grid">
                    <SFButton size="sm">小按钮</SFButton>
                    <SFButton size="md" loading>提交中</SFButton>
                    <SFButton size="lg" variant="secondary">大按钮</SFButton>
                    <SFButton icon-only aria-label="新建主题">+</SFButton>
                    <SFButton disabled variant="ghost">不可用</SFButton>
                  </div>
                  <SFInput
                    v-model="threadTitle"
                    label="主题标题"
                    placeholder="输入一个清晰的问题标题"
                    hint="标题会优先展示在信息流和搜索结果中。"
                  />
                  <SFInput
                    v-model="categoryName"
                    label="默认版块"
                    placeholder="选择主题所属版块"
                    size="sm"
                  />
                  <SFInput
                    v-model="tagName"
                    label="标签"
                    placeholder="例如 nuxt-ssr"
                    error="标签只能包含小写字母、数字和连字符。"
                  />
                </div>
                <div class="sf-preview-stack">
                  <div class="variant-grid">
                    <SFAvatar name="林知夏" status="online" />
                    <SFAvatar name="Open Source" shape="square" status="idle" />
                    <SFAvatar name="SF" size="lg" />
                    <SFAvatar name="归档用户" status="offline" />
                  </div>
                  <SFToggle
                    v-model="digestEnabled"
                    label="接收本帖摘要"
                    description="当主题有高质量回复时发送一次站内通知。"
                  />
                  <SFToggle
                    v-model="mentionEnabled"
                    label="被提及时通知"
                    description="用于评论、引用和版主处理结果。"
                  />
                  <SFToggle
                    :model-value="false"
                    label="关闭的全局开关"
                    description="禁用状态也应保持文字清晰可读。"
                    disabled
                  />
                </div>
              </div>
            </SFCard>
          </section>

          <section id="icons" class="sf-component-section">
            <SFCard title="Icons 选择器" subtitle="为后台设置、导航、版块和用户配置预留的图标选择控件">
              <div class="sf-preview-grid">
                <SFIconPicker v-model="selectedIcon" />
                <div class="sf-preview-stack">
                  <div class="sf-icon-picker-demo">
                    <span class="sf-icon-picker-demo__icon">
                      <UIcon :name="selectedIcon" class="sf-icon-picker-demo__svg" />
                    </span>
                    <div class="sf-icon-picker-demo__content">
                      <p class="sf-icon-picker-demo__title">
                        后台字段预览
                      </p>
                      <code class="sf-icon-picker-demo__value">{{ selectedIcon }}</code>
                    </div>
                  </div>
                  <SFAlert
                    title="保存值保持简单"
                    description="后台配置只需要保存 i-tabler-* 或 i-lucide-* 字符串，渲染处直接交给 Nuxt Icon。"
                    variant="info"
                    compact
                  />
                  <div class="variant-grid">
                    <SFButton>
                      <template #leading>
                        <UIcon :name="selectedIcon" class="size-4" />
                      </template>
                      保存图标
                    </SFButton>
                    <SFButton variant="ghost">
                      <template #leading>
                        <UIcon name="i-lucide-refresh-cw" class="size-4" />
                      </template>
                      重置
                    </SFButton>
                  </div>
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
                <div class="sf-preview-grid">
                  <SFAlert
                    title="权限不足"
                    description="你需要完成邮箱验证后才能发布链接。"
                    variant="danger"
                    compact
                  />
                  <SFAlert
                    title="资料已同步"
                    description="用户名、头像和公开简介会出现在主题详情页。"
                    variant="success"
                    compact
                  />
                </div>
                <SFToast
                  title="回复已保存"
                  description="草稿会保留在当前设备，下次打开主题时继续编辑。"
                  action-label="查看"
                  closable
                />
                <SFToast
                  title="审核已通过"
                  description="主题已经重新出现在列表和搜索结果中。"
                  variant="success"
                  action-label="打开主题"
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
                <SFFeedRow
                  title="Go Fiber 模块边界怎么拆才不会越来越厚？"
                  excerpt="现在 API、权限、查询和通知都还在早期规划阶段，想先确定一个不会互相缠住的目录结构。"
                  author="沈柏舟"
                  meta="48 分钟前 · backend"
                  :score="27"
                  :replies="9"
                  :views="620"
                  :badges="questionBadges"
                />
                <SFFeedRow
                  title="新用户发帖是否需要冷启动审核？"
                  excerpt="希望减少垃圾内容，但不想让认真提问的人感觉被拦在门外。"
                  author="版务组"
                  meta="今天 09:12 · moderation"
                  :score="15"
                  :replies="6"
                  :views="384"
                  :badges="reviewBadges"
                />
                <SFComment
                  author="陈大文"
                  meta="刚刚"
                  content="先拆读模型，再把权限和可见性作为查询条件下沉到 API，前端只负责呈现状态会清爽很多。"
                />
                <SFComment
                  author="林知夏"
                  meta="2 分钟前"
                  content="这个方向靠谱，我会把 topic summary 和 permission scope 分开建模。"
                  :depth="1"
                />
                <SFEditor v-model="replyDraft" />
              </div>
            </SFCard>
          </section>

          <section id="composer" class="sf-component-section">
            <SFCard title="发布工作流" subtitle="主题标题、分类标签、富编辑器状态和发布前检查">
              <div class="sf-preview-stack">
                <SFTabs v-model="composerTab" :items="composerTabs" aria-label="发布流程" />
                <div class="sf-preview-grid">
                  <div class="sf-preview-stack">
                    <SFInput
                      v-model="threadTitle"
                      label="主题标题"
                      placeholder="一句话说明你想讨论的问题"
                      required
                    />
                    <SFInput
                      v-model="categoryName"
                      label="发布版块"
                      hint="建议选择最具体的版块，便于后续沉淀知识。"
                    />
                    <SFInput
                      v-model="tagName"
                      label="标签 slug"
                      placeholder="architecture"
                      size="sm"
                    />
                  </div>
                  <div class="sf-preview-stack">
                    <SFAlert
                      title="富编辑器状态"
                      description="正文会自动保存草稿；当内容包含外链时，发布前会展示审核提示。"
                      variant="info"
                    />
                    <SFProgress label="发布资料完整度" :value="82" variant="success" show-value />
                    <SFToggle
                      v-model="anonymousDraft"
                      label="匿名发布"
                      description="隐藏个人主页入口，但仍保留版主管理记录。"
                    />
                  </div>
                </div>
                <SFEditor
                  v-model="longReplyDraft"
                  placeholder="补充背景、已尝试方案和期望结果..."
                  :rows="8"
                  hint="草稿每 30 秒自动保存"
                  submit-label="发布主题"
                  @content-change="updateEditorPreview"
                />
                <SFAlert
                  title="内容安全边界"
                  description="数据库保存 sanitized_html、markdown_source 和 native_json；客户端生成的 HTML 不直接信任，API 写入前必须按服务端规则重新渲染并净化。"
                  variant="warning"
                  compact
                />
                <div class="sf-editor-contract">
                  <div class="sf-editor-contract__panel">
                    <p class="sf-editor-contract__label">
                      sanitized_html
                    </p>
                    <pre class="sf-editor-contract__code">{{ editorHtmlOutput }}</pre>
                  </div>
                  <div class="sf-editor-contract__panel">
                    <p class="sf-editor-contract__label">
                      markdown_source
                    </p>
                    <pre class="sf-editor-contract__code">{{ editorMarkdownOutput }}</pre>
                  </div>
                  <div class="sf-editor-contract__panel">
                    <p class="sf-editor-contract__label">
                      native_json
                    </p>
                    <pre class="sf-editor-contract__code">{{ editorNativeOutput }}</pre>
                  </div>
                </div>
                <div class="variant-grid">
                  <SFButton>发布主题</SFButton>
                  <SFButton variant="secondary">保存草稿</SFButton>
                  <SFButton variant="ghost">预览</SFButton>
                </div>
              </div>
            </SFCard>
          </section>

          <section id="moderation" class="sf-component-section">
            <SFCard title="审核与管理" subtitle="待处理队列、退回原因、锁定和批量处理状态">
              <div class="sf-preview-stack">
                <SFTabs v-model="moderationTab" :items="moderationTabs" aria-label="审核队列" />
                <SFAlert
                  title="队列压力偏高"
                  description="过去 1 小时新增 8 条待审主题，其中 3 条来自新注册用户。"
                  variant="warning"
                />
                <SFFeedRow
                  title="如何给生产环境数据库做零停机迁移？"
                  excerpt="帖子包含外部迁移脚本链接，建议人工确认链接安全后再放行。"
                  author="新成员"
                  meta="待审 6 分钟 · database"
                  :score="0"
                  :replies="0"
                  :views="12"
                  :badges="reviewBadges"
                />
                <div class="sf-preview-grid">
                  <SFInput
                    v-model="reviewReason"
                    label="退回原因"
                    placeholder="告诉作者需要补充什么"
                  />
                  <div class="sf-preview-stack">
                    <SFProgress label="审核队列完成度" :value="46" variant="warning" show-value />
                    <SFToggle
                      v-model="autoCloseQueue"
                      label="处理后自动打开下一条"
                      description="适合连续审核列表内容。"
                    />
                  </div>
                </div>
                <div class="variant-grid">
                  <SFButton>通过</SFButton>
                  <SFButton variant="secondary">请求修改</SFButton>
                  <SFButton variant="danger">锁定主题</SFButton>
                  <SFButton variant="ghost">跳过</SFButton>
                </div>
              </div>
            </SFCard>
          </section>

          <section id="profile" class="sf-component-section">
            <SFCard title="成员资料" subtitle="公开身份、贡献状态、隐私设置和个人内容索引">
              <div class="sf-preview-stack">
                <div class="variant-grid">
                  <SFAvatar name="林知夏" size="lg" status="online" />
                  <SFBadge variant="primary" dot>核心成员</SFBadge>
                  <SFBadge variant="success">回答采纳率 82%</SFBadge>
                  <SFBadge variant="neutral">加入 18 个月</SFBadge>
                </div>
                <SFTabs v-model="profileTab" :items="profileTabs" aria-label="成员资料视图" />
                <div class="sf-preview-grid">
                  <div class="sf-preview-stack">
                    <SFInput v-model="profileName" label="显示名称" />
                    <SFInput v-model="profileTitle" label="个人头衔" />
                    <SFProgress label="资料完整度" :value="74" show-value />
                  </div>
                  <div class="sf-preview-stack">
                    <SFToggle
                      v-model="hideActivity"
                      label="隐私设置：隐藏在线状态"
                      description="公开资料仍可展示头像、昵称和历史内容。"
                    />
                    <SFComment
                      author="林知夏"
                      meta="代表性回复"
                      content="把复杂权限拆成能力点，再用角色组合能力，会比在页面里写条件判断更耐用。"
                      :actions="[
                        { label: '查看主题', value: 'open' },
                        { label: '收藏回复', value: 'save' }
                      ]"
                    />
                  </div>
                </div>
              </div>
            </SFCard>
          </section>

          <section id="states" class="sf-component-section">
            <SFCard title="加载与空状态" subtitle="分页、进度、骨架屏、空列表">
              <div class="sf-preview-grid">
                <div class="sf-preview-stack">
                  <SFProgress label="资料完整度" :value="68" show-value />
                  <SFProgress label="索引同步" :value="34" variant="info" show-value />
                  <SFProgress label="风险评分" :value="18" variant="danger" show-value />
                  <SFPagination v-model:page="currentPage" :total-pages="8" />
                  <SFSkeleton avatar :lines="3" />
                  <SFSkeleton :lines="5" />
                </div>
                <div class="sf-preview-stack">
                  <SFEmptyState
                    title="暂无匹配主题"
                    description="调整筛选条件，或新建一个更聚焦的讨论。"
                    action-label="发布主题"
                  />
                  <SFEmptyState
                    title="还没有收藏"
                    description="收藏高质量主题后，它们会显示在这里，方便稍后继续阅读。"
                    icon-label="★"
                  />
                </div>
              </div>
            </SFCard>
          </section>
        </div>
      </div>
    </div>
  </main>
</template>
