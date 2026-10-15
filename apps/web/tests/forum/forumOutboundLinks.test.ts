import { describe, expect, it } from 'bun:test'
import {
  contentLinkInternalHosts,
  decodeOutboundTarget,
  decorateContentLinks,
  encodeOutboundTarget,
  isInterstitialHref,
  isOffsiteContentHref,
  isSafeOutboundTarget,
  isTrustedOutboundHost,
  outboundInterstitialHref,
  outboundTargetHost,
  parseTrustedDomains
} from '~/utils/forum/forumOutboundLinks'

function container(html: string) {
  const root = document.createElement('div')
  root.innerHTML = html
  return root
}

const basePolicy = {
  interstitial: true,
  trustedDomains: [] as string[],
  internalHosts: ['forum.example.com']
}

describe('forumOutboundLinks 编解码', () => {
  it('round-trips targets through the interstitial payload', () => {
    const target = 'https://example.com/spec/rfc-9457?section=4#中文'
    const href = outboundInterstitialHref(target)
    expect(href.startsWith('/outbound?u=')).toBe(true)
    expect(decodeOutboundTarget(href.split('u=')[1])).toBe(target)
    expect(isInterstitialHref(href)).toBe(true)
  })

  it('rejects unsafe or unparseable targets', () => {
    expect(decodeOutboundTarget('not-base64!!')).toBeNull()
    expect(decodeOutboundTarget(encodeOutboundTarget('javascript:alert(1)'))).toBeNull()
    expect(decodeOutboundTarget(encodeOutboundTarget('https://user:pass@example.com'))).toBeNull()
    expect(decodeOutboundTarget(encodeOutboundTarget(`https://example.com/${'a'.repeat(3000)}`))).toBeNull()
    expect(decodeOutboundTarget('')).toBeNull()
    expect(isSafeOutboundTarget('/relative')).toBe(false)
    expect(outboundTargetHost('https://Example.COM:8443/x')).toBe('example.com:8443')
  })
})

describe('forumOutboundLinks 判定', () => {
  it('collects internal hosts from site url, site domain and current host', () => {
    const hosts = contentLinkInternalHosts({
      siteUrl: 'http://127.0.0.1:3000',
      siteDomain: 'forum.example.com',
      currentHost: 'forum.example.com:3000'
    })
    expect(hosts).toContain('127.0.0.1:3000')
    expect(hosts).toContain('127.0.0.1')
    expect(hosts).toContain('forum.example.com')
    expect(hosts).toContain('forum.example.com:3000')
  })

  it('classifies offsite hrefs and trusted hosts', () => {
    const hosts = contentLinkInternalHosts({ siteUrl: 'https://forum.example.com' })
    expect(isOffsiteContentHref('https://example.com/a', hosts)).toBe(true)
    expect(isOffsiteContentHref('https://forum.example.com/a', hosts)).toBe(false)
    expect(isOffsiteContentHref('/t/hello', hosts)).toBe(false)
    expect(isOffsiteContentHref('mailto:hi@example.com', hosts)).toBe(false)

    const trusted = parseTrustedDomains('Example.COM\nhttps://github.com/octocat\n example.com ')
    expect(trusted).toEqual(['example.com', 'github.com'])
    expect(isTrustedOutboundHost('example.com', trusted)).toBe(true)
    expect(isTrustedOutboundHost('example.com:8443', trusted)).toBe(true)
    expect(isTrustedOutboundHost('evil.com', trusted)).toBe(false)
  })
})

describe('decorateContentLinks', () => {
  it('forces new tab and wraps offsite links', () => {
    const root = container('<p><a href="https://example.com/a">外</a><a href="/t/1">内</a></p>')
    expect(decorateContentLinks(root, basePolicy)).toBe(1)
    const anchors = Array.from(root.querySelectorAll('a'))
    expect(anchors[0]!.getAttribute('href')).toBe(outboundInterstitialHref('https://example.com/a'))
    expect(anchors[0]!.getAttribute('target')).toBe('_blank')
    expect(anchors[0]!.getAttribute('rel')).toBe('nofollow noreferrer noopener')
    // 站内链接只补 target/rel，不改写地址。
    expect(anchors[1]!.getAttribute('href')).toBe('/t/1')
    expect(anchors[1]!.getAttribute('target')).toBe('_blank')
  })

  it('is idempotent', () => {
    const root = container('<a href="https://example.com/a">外</a>')
    decorateContentLinks(root, basePolicy)
    const once = root.innerHTML
    expect(decorateContentLinks(root, basePolicy)).toBe(0)
    expect(root.innerHTML).toBe(once)
  })

  it('keeps mentions, image viewer triggers, anchors and mailto untouched', () => {
    const root = container([
      '<a class="sf-mention" href="/u/yg">@yg</a>',
      '<a data-sforum-image-viewer="1" href="/media/a.png"><img src="/media/a.png"></a>',
      '<a href="#section">锚点</a>',
      '<a href="mailto:hi@example.com">邮件</a>'
    ].join(''))
    expect(decorateContentLinks(root, basePolicy)).toBe(0)
    for (const anchor of Array.from(root.querySelectorAll('a'))) {
      expect(anchor.hasAttribute('target')).toBe(false)
    }
  })

  it('honours the disabled switch and the trusted domain list', () => {
    const offsite = container('<a href="https://github.com/octocat">gh</a>')
    decorateContentLinks(offsite, { ...basePolicy, interstitial: false })
    expect(offsite.querySelector('a')!.getAttribute('href')).toBe('https://github.com/octocat')
    expect(offsite.querySelector('a')!.getAttribute('target')).toBe('_blank')

    const trusted = container('<a href="https://github.com/octocat">gh</a>')
    decorateContentLinks(trusted, { ...basePolicy, trustedDomains: ['github.com'] })
    expect(trusted.querySelector('a')!.getAttribute('href')).toBe('https://github.com/octocat')
  })
})
