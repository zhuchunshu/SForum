export type ForumImageViewerSlide = {
  src: string
  msrc: string
  width: number
  height: number
  alt: string
  element: HTMLElement
}

function imageDimension(image: HTMLImageElement, name: 'width' | 'height') {
  const attribute = Number(image.getAttribute(name))
  if (Number.isFinite(attribute) && attribute > 0) return attribute

  const natural = name === 'width' ? image.naturalWidth : image.naturalHeight
  if (natural > 0) return natural

  const rect = image.getBoundingClientRect()
  const rendered = name === 'width' ? rect.width : rect.height
  return Math.max(1, Math.round(rendered || 1))
}

export function forumImageViewerSlides(target: Element) {
  const trigger = target.closest<HTMLAnchorElement>('a[data-sforum-image-viewer="1"]')
  const currentImage = trigger?.querySelector<HTMLImageElement>('img')
    || target.closest<HTMLImageElement>('img')
  const gallery = currentImage?.closest<HTMLElement>('[data-sforum-image-gallery]')

  if (!currentImage || !gallery) return null

  const images = Array.from(gallery.querySelectorAll<HTMLImageElement>('img'))
    .filter(image => Boolean(image.currentSrc || image.getAttribute('src')))
  const index = images.indexOf(currentImage)
  if (index < 0) return null

  const slides: ForumImageViewerSlide[] = images.map(image => {
    const link = image.closest<HTMLAnchorElement>('a[data-sforum-image-viewer="1"]')
    const displayedSource = image.currentSrc || image.src
    return {
      src: link?.href || displayedSource,
      msrc: displayedSource,
      width: imageDimension(image, 'width'),
      height: imageDimension(image, 'height'),
      alt: image.alt || '',
      element: link || image
    }
  })

  return { index, slides }
}
