export const defaultSiteLogoUrl = '/brand/sforum-logo.svg'
export const defaultSiteFaviconUrl = defaultSiteLogoUrl

// 后台空值保持“未配置”语义，Core 品牌回退只在展示边界解析。
export function resolveSiteBrandAssetUrl(value: string | null | undefined, fallbackUrl: string) {
  return value?.trim() || fallbackUrl
}
