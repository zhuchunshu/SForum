// Nuxt UI 主题微调。Alert 默认 root 带 overflow-hidden，在后台纵向 flex
// 布局里高度被压缩时会裁切正文，看起来像下方 tab 把提示“压住”。
export default defineAppConfig({
  ui: {
    alert: {
      slots: {
        root: 'relative w-full rounded-lg p-4 flex items-start gap-2.5 shrink-0',
        wrapper: 'min-w-0 flex-1 flex flex-col',
        title: 'text-sm font-medium',
        description: 'text-sm opacity-90 leading-6',
        icon: 'shrink-0 size-5',
        avatar: 'shrink-0',
        avatarSize: '2xl',
        actions: 'flex flex-wrap gap-1.5 shrink-0',
        close: 'p-0'
      }
    }
  }
})
