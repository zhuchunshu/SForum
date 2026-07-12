<script setup lang="ts">
/**
 * L2 预构建 widget 入口 — 当前安全关闭。
 *
 * 审阅结论：任意远程/相对 entry 的 dynamic import 会执行未受保护的 JavaScript，
 * integrity 校验与 package-digest 信任授权未完成。在完整 L2 安全模型落地前，
 * 宿主拒绝加载任何可执行 widget，仅渲染明确的禁用占位。
 *
 * 不得恢复为「半成品可执行」路径。完整实现需要：
 * - 同源、由宿主根据 已安装扩展 + 版本 + package digest + widget manifest 生成的资源 URL
 * - super_admin 信任授权绑定精确 package digest
 * - 服务端摘要校验与 CSP
 * - 扩展升级/撤销/禁用时立即失效
 */
const props = defineProps<{
  extensionId?: string
  entry?: string
  integrity?: string
  name?: string
  widgetId?: string
}>()

// 始终禁用：不读取 entry，不 dynamic import，不加载远程脚本。
const disabled = true
</script>

<template>
  <div
    class="sf-extension-widget sf-extension-widget--disabled"
    data-l2-disabled="1"
    :data-extension-id="extensionId || ''"
    :data-widget-id="widgetId || name || ''"
    role="status"
  >
    <slot name="disabled">
      <SFAlert
        variant="warning"
        title="L2 extension widgets are disabled"
      >
        Prebuilt interactive widgets (L2) are not available until host integrity,
        package-digest trust, and CSP enforcement are complete. Template authors
        must not rely on executable remote modules.
      </SFAlert>
    </slot>
    <!-- 保留 props 引用，避免 unused 告警并便于将来安全实现读取 -->
    <span
      v-if="false"
      class="hidden"
    >{{ entry }}{{ integrity }}{{ disabled }}</span>
  </div>
</template>
