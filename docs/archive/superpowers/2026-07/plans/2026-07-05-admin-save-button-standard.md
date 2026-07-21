# 后台统一保存与表单操作栏 (Admin Save & Form Action Footer) 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 规范后台表单操作与保存按钮样式与位置，引入粘性吸底的统一组件 `SFAdminFormFooter.vue`，并将站点设置表单重构为该模式，同时更新后台开发决策规范。

**Architecture:** 
1. 语言包注入统一表单文本；
2. 新建 `<SFAdminFormFooter>` 组件，该组件通过 `sticky bottom-0 z-20` 粘性吸附在页面滚动视口的底部，并且支持 `backdrop-blur-sm bg-white/90` 毛玻璃阴影视觉效果；
3. 将 `apps/web/app/pages/admin/settings/index.vue` 重构，使用新组件并与表单状态联动（如表单已修改的左侧未保存提示，重置按钮联动表单重置等）。

**Tech Stack:** Nuxt 4, Nuxt UI, Tailwind CSS, Nuxt i18n, Bun

---

### Task 1: 注入全局表单操作多语言翻译

**Files:**
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`

- [ ] **Step 1: 在简体中文语言包 `zh-CN.json` 的 `"admin"` 节点下注入 `"form"` 词条**
  
  修改 `apps/web/i18n/locales/zh-CN.json` 的 `"admin"` 节点，加入 `"form"` 翻译。
  ```json
  "form": {
    "save": "保存",
    "saving": "保存中...",
    "reset": "重置",
    "unsavedChanges": "检测到有未保存的更改"
  }
  ```

- [ ] **Step 2: 在英文语言包 `en-US.json` 的 `"admin"` 节点下注入 `"form"` 词条**
  
  修改 `apps/web/i18n/locales/en-US.json` 的 `"admin"` 节点，加入 `"form"` 翻译。
  ```json
  "form": {
    "save": "Save",
    "saving": "Saving...",
    "reset": "Reset",
    "unsavedChanges": "Unsaved changes detected"
  }
  ```

- [ ] **Step 3: 运行验证测试确保翻译文件 JSON 语法正确**
  
  运行: `bun tests/validate-admin-framework.ts`
  预期: PASS (输出 "Admin framework validation passed.")

- [ ] **Step 4: 提交**
  
  ```bash
  git add apps/web/i18n/locales/zh-CN.json apps/web/i18n/locales/en-US.json
  git commit -m "i18n: add unified admin form action translations"
  ```

---

### Task 2: 新建 `SFAdminFormFooter.vue` 统一组件

**Files:**
- Create: `apps/web/app/components/SFAdminFormFooter.vue`

- [ ] **Step 1: 新建并编写 `SFAdminFormFooter.vue`**
  
  新建组件 `/Users/inkedus/Code/SForum/apps/web/app/components/SFAdminFormFooter.vue`，写入完整代码：
  ```html
  <script setup lang="ts">
  const { t } = useI18n()
  
  const props = withDefaults(defineProps<{
    saving?: boolean
    disabled?: boolean
    submitText?: string
    resetText?: string
    submitIcon?: string
    resetIcon?: string
    showUnsavedAlert?: boolean
  }>(), {
    saving: false,
    disabled: false,
    submitText: undefined,
    resetText: undefined,
    submitIcon: 'i-lucide-save',
    resetIcon: 'i-lucide-rotate-ccw',
    showUnsavedAlert: false
  })
  
  const emit = defineEmits<{
    submit: []
    reset: []
  }>()
  </script>
  
  <template>
    <div class="sticky bottom-0 z-20 mt-8 flex items-center justify-between px-6 py-4 rounded-xl border border-slate-200 dark:border-zinc-800 bg-white/90 dark:bg-zinc-900/90 backdrop-blur-sm shadow-lg transition-all">
      <!-- 左侧插槽或提示语 -->
      <div class="flex items-center gap-2 text-xs sm:text-sm text-slate-500 dark:text-zinc-400">
        <slot name="left">
          <template v-if="showUnsavedAlert">
            <UIcon name="i-lucide-circle-alert" class="size-4 text-amber-500 animate-pulse" />
            <span class="text-amber-600 dark:text-amber-400 font-medium">
              {{ t('admin.form.unsavedChanges') }}
            </span>
          </template>
        </slot>
      </div>
  
      <!-- 右侧动作按钮区 -->
      <div class="flex items-center gap-3">
        <slot name="actions">
          <!-- 重置按钮 -->
          <UButton
            color="neutral"
            variant="outline"
            :leading-icon="resetIcon"
            :disabled="disabled || saving"
            class="border-slate-200 dark:border-zinc-700 font-medium"
            @click="emit('reset')"
          >
            {{ resetText || t('admin.form.reset') }}
          </UButton>
  
          <!-- 保存按钮 -->
          <UButton
            type="submit"
            :leading-icon="submitIcon"
            :loading="saving"
            :disabled="disabled"
            class="bg-teal-600 hover:bg-teal-700 dark:bg-teal-500 dark:hover:bg-teal-600 text-white font-semibold"
            @click="emit('submit')"
          >
            {{ submitText || t('admin.form.save') }}
          </UButton>
        </slot>
      </div>
    </div>
  </template>
  ```

- [ ] **Step 2: 运行后台框架自检以确认组件无编译错误**
  
  运行: `bun tests/validate-admin-framework.ts`
  预期: PASS

- [ ] **Step 3: 提交**
  
  ```bash
  git add apps/web/app/components/SFAdminFormFooter.vue
  git commit -m "feat: implement unified SFAdminFormFooter component"
  ```

---

### Task 3: 重构后台“站点设置”页面表单

**Files:**
- Modify: `apps/web/app/pages/admin/settings/index.vue`

- [ ] **Step 1: 修改设置页面逻辑，添加表单修改状态检测与重置逻辑**
  
  修改 `apps/web/app/pages/admin/settings/index.vue`，使其监控输入值 `siteName` 的变化，以启用 `hasUnsavedChanges` 提示，并实现 `resetForm` 方法。
  替换现有 script 部分及 template 部分中的 form 保存按钮。
  
  修改后的完整代码应为：
  ```html
  <script setup lang="ts">
  import { useAdminTabs } from '~/composables/useAdminTabs'
  
  definePageMeta({
    middleware: 'admin',
    layout: 'admin'
  })
  
  defineOptions({
    name: 'AdminSettings'
  })
  
  const { t } = useI18n()
  const toast = useToast()
  const { options, fetchEnvelope, save } = useWebOptions()
  const adminTabs = useAdminTabs()
  
  onMounted(() => {
    adminTabs.openTab('/settings', 'admin.nav.settings', 'i-lucide-settings-2', 'AdminSettings')
  })
  
  const siteName = ref(options.value['site.name'] || 'SForum')
  const saving = ref(false)
  
  const { pending, error, refresh } = await useAsyncData('admin-web-options', async () => {
    const envelope = await fetchEnvelope()
    options.value = {
      ...options.value,
      ...Object.fromEntries(envelope.data.map((item) => [item.name, item.value]))
    }
    siteName.value = options.value['site.name'] || 'SForum'
    return envelope.data
  })
  
  // 检查是否有未保存的更改
  const initialSiteName = computed(() => options.value['site.name'] || 'SForum')
  const hasUnsavedChanges = computed(() => siteName.value !== initialSiteName.value)
  
  useSeoMeta({
    title: t('admin.settings.metaTitle')
  })
  
  async function submit() {
    saving.value = true
    try {
      await save('site.name', siteName.value)
      toast.add({
        color: 'success',
        icon: 'i-lucide-check',
        title: t('admin.settings.saved')
      })
    } catch (error) {
      toast.add({
        color: 'error',
        icon: 'i-lucide-triangle-alert',
        title: apiErrorMessage(error) || t('admin.settings.saveFailed')
      })
    } finally {
      saving.value = false
    }
  }
  
  function resetForm() {
    siteName.value = initialSiteName.value
    toast.add({
      color: 'neutral',
      icon: 'i-lucide-rotate-ccw',
      title: '已重置未保存的更改'
    })
  }
  </script>
  
  <template>
    <!-- 局部标题 -->
    <div class="mb-4">
      <h2 class="text-xl font-bold flex items-center gap-2 text-slate-900 dark:text-zinc-100">
        <UIcon name="i-lucide-settings-2" class="size-5 text-teal-600 dark:text-teal-400" />
        {{ t('admin.settings.title') }}
      </h2>
    </div>
  
    <!-- 整合刷新按钮的统一 Toolbar -->
    <UDashboardToolbar class="border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 rounded-lg px-4 py-2.5 mb-6 text-slate-500 dark:text-zinc-400">
      <template #left>
        <div class="flex min-w-0 items-center gap-2 text-sm text-slate-500 dark:text-zinc-400">
          <UIcon name="i-lucide-database" class="size-4" />
          <span class="truncate">{{ t('admin.settings.intro') }}</span>
        </div>
      </template>
      <template #right>
        <UButton
          color="neutral"
          variant="outline"
          leading-icon="i-lucide-refresh-cw"
          :loading="pending"
          class="border-slate-200 dark:border-zinc-700"
          @click="refresh()"
        >
          {{ t('admin.settings.refresh') }}
        </UButton>
      </template>
    </UDashboardToolbar>
  
    <div class="flex flex-col gap-4">
      <UAlert
        v-if="error"
        color="error"
        variant="soft"
        icon="i-lucide-triangle-alert"
        :title="t('admin.settings.loadFailed')"
      />
  
      <form class="flex flex-col" @submit.prevent="submit">
        <UCard class="border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-900 dark:text-zinc-100">
          <template #header>
            <div class="flex items-center justify-between gap-3">
              <div>
                <h2 class="text-base font-bold text-slate-900 dark:text-white">
                  {{ t('admin.settings.basic.title') }}
                </h2>
                <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
                  {{ t('admin.settings.basic.description') }}
                </p>
              </div>
              <UBadge color="neutral" variant="soft" class="border border-slate-200 dark:border-zinc-800 font-mono">
                site.name
              </UBadge>
            </div>
          </template>
  
          <div class="grid max-w-2xl gap-4">
            <UFormField :label="t('admin.settings.siteName')" name="site-name">
              <UInput
                v-model="siteName"
                icon="i-lucide-message-square-text"
                :placeholder="t('admin.settings.siteNamePlaceholder')"
                maxlength="80"
                required
                class="w-full"
              />
            </UFormField>
          </div>
        </UCard>
  
        <!-- 吸底保存组件 -->
        <SFAdminFormFooter
          :saving="saving"
          :show-unsaved-alert="hasUnsavedChanges"
          :submit-text="t('admin.settings.save')"
          @reset="resetForm"
        />
      </form>
    </div>
  </template>
  ```

- [ ] **Step 2: 运行验证测试以确保设置页面没有语法错误或引用失败**
  
  运行: `bun tests/validate-admin-framework.ts`
  Expected: PASS

- [ ] **Step 3: 提交**
  
  ```bash
  git add apps/web/app/pages/admin/settings/index.vue
  git commit -m "refactor: update site settings page to use unified admin form footer"
  ```

---

### Task 4: 更新后台开发决策规范 (Decision Record)

**Files:**
- Modify: `knowledge/decisions/2026-07-05-admin-multitabs-and-layout-rules.md`

- [ ] **Step 1: 在 `2026-07-05-admin-multitabs-and-layout-rules.md` 中追加第六条开发规范**
  
  打开 `/Users/inkedus/Code/SForum/knowledge/decisions/2026-07-05-admin-multitabs-and-layout-rules.md`，在第 58 行（`## Consequences` 上方）追加新规范：
  ```markdown
  ### 6. Standardized Form Actions & Sticky Footer (统一表单操作与吸底保存条)
  - **吸底布局规范**：所有后台配置页面、长表单编辑页面，若含有“保存”、“重置/取消”等操作，**严禁**直接平铺在卡片底部。
  - **统一使用 `SFAdminFormFooter`**：必须在 `<form>` 的底部引入 `<SFAdminFormFooter>` 组件。它会自动粘性吸附在内容视口底部，确保用户无需滚动页面即可执行保存。
  - **状态联动规范**：
    - 保存按钮必须绑定 `:loading="saving"`，防止重复提交。
    - 表单的 Input/Select 在提交中应当伴随禁用，或通过 `disabled` 属性统一传递给 Footer。
    - 按钮图标严格使用 `i-lucide-*`，遵循无 emoji 规范。
  ```

- [ ] **Step 2: 提交规范改动**
  
  ```bash
  git add knowledge/decisions/2026-07-05-admin-multitabs-and-layout-rules.md
  git commit -m "docs: document admin form actions & sticky footer rule in decision records"
  ```

---

### Task 5: 全面验证

- [ ] **Step 1: 运行所有验证脚本以确认无遗留影响**
  
  运行: `bun tests/validate-admin-framework.ts`
  Expected: PASS

- [ ] **Step 2: 完成**
  
  向用户报告修改结果。
