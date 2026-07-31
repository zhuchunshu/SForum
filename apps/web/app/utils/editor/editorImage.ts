export const sfEditorImageDisplaySizes = ['compact', 'standard', 'wide'] as const

export type SFEditorImageDisplaySize = typeof sfEditorImageDisplaySizes[number]

export function normalizeEditorImageDisplaySize(value: unknown): SFEditorImageDisplaySize {
  return sfEditorImageDisplaySizes.includes(value as SFEditorImageDisplaySize)
    ? value as SFEditorImageDisplaySize
    : 'standard'
}

export function normalizeEditorImageDimension(value: unknown) {
  const dimension = typeof value === 'string' ? Number(value) : Number(value)
  return Number.isSafeInteger(dimension) && dimension > 0 && dimension <= 100000
    ? dimension
    : null
}

export function editorImageRenderAttributes(attrs: Record<string, unknown>) {
  const width = normalizeEditorImageDimension(attrs.width)
  const height = normalizeEditorImageDimension(attrs.height)
  const rendered: Record<string, string | number> = {
    'data-sforum-image-size': normalizeEditorImageDisplaySize(attrs.displaySize)
  }

  if (width && height) {
    rendered.width = width
    rendered.height = height
    if (height >= width * 2.5) {
      rendered['data-sforum-image-long'] = '1'
    }
  }
  return rendered
}
