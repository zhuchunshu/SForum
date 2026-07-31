import type PhotoSwipeLightbox from 'photoswipe/lightbox'
import 'photoswipe/style.css'
import { forumImageViewerSlides } from '~/utils/forum/forumImageViewer'

export function useForumImageViewer() {
  const { t } = useI18n()
  let activeLightbox: PhotoSwipeLightbox | null = null
  let disposed = false

  onBeforeUnmount(() => {
    disposed = true
    activeLightbox?.destroy()
    activeLightbox = null
  })

  async function openForumImageViewer(event: MouseEvent) {
    if (!(event.target instanceof Element)) return
    if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return

    const data = forumImageViewerSlides(event.target)
    if (!data) return

    event.preventDefault()
    activeLightbox?.destroy()

    try {
      const { default: Lightbox } = await import('photoswipe/lightbox')
      if (disposed) return

      const lightbox = new Lightbox({
        pswpModule: () => import('photoswipe'),
        bgOpacity: 0.92,
        loop: false,
        wheelToZoom: true,
        closeTitle: t('topicDetail.imageViewer.close'),
        zoomTitle: t('topicDetail.imageViewer.zoom'),
        arrowPrevTitle: t('topicDetail.imageViewer.previous'),
        arrowNextTitle: t('topicDetail.imageViewer.next'),
        errorMsg: t('topicDetail.imageViewer.loadFailed')
      })
      activeLightbox = lightbox
      lightbox.init()
      lightbox.loadAndOpen(data.index, data.slides, {
        x: event.clientX,
        y: event.clientY
      })
    } catch {
      const source = data.slides[data.index]?.src
      if (source) window.open(source, '_blank', 'noopener,noreferrer')
    }
  }

  return { openForumImageViewer }
}
