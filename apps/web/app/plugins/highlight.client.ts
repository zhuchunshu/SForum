import { enhanceCodeBlocks } from '~/utils/forum/codeHighlight'

export default defineNuxtPlugin((nuxtApp) => {
  const toast = nuxtApp.vueApp.runWithContext(() => useToast())
  const i18n = nuxtApp.$i18n as { t?: (key: string) => unknown } | undefined
  const translate = (key: string, fallback: string) => {
    const value = i18n?.t?.(key)
    return typeof value === 'string' && value !== key ? value : fallback
  }
  const enhance = (el: Element) => enhanceCodeBlocks(el, {
    labels: {
      code: translate('codeBlock.code', '代码块'),
      copy: translate('codeBlock.copy', '复制'),
      copied: translate('codeBlock.copied', '已复制'),
      plainText: translate('codeBlock.plainText', '纯文本')
    },
    onCopySuccess: () => toast.add({
      color: 'success',
      icon: 'i-lucide-check',
      title: translate('codeBlock.copySuccess', '代码已复制'),
      duration: 10000
    }),
    onCopyError: () => toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: translate('codeBlock.copyFailed', '复制失败，请手动选择代码')
    })
  })

  nuxtApp.vueApp.directive('highlight', {
    mounted(el: Element) {
      enhance(el)
    },
    updated(el: Element) {
      enhance(el)
    },
    getSSRProps() {
      // 服务端渲染时不输出额外属性，高亮完全在客户端进行。
      return {}
    },
  })
})
