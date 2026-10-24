import { describe, expect, test } from 'bun:test'
import { compileTemplate, parse } from '@vue/compiler-sfc'
import { installTestDom } from '../helpers/dom'

installTestDom()

const dialogPath = 'app/components/extensions/admin/SFAdminPageInstallDialog.vue'
const bridgePath = 'app/components/extensions/admin/SFTrustedAdminPageComponent.vue'
const noticePath = 'app/components/admin/SFAdminExtensionHostActionsNotice.vue'

async function source(relative: string) {
  return Bun.file(new URL(`../../${relative}`, import.meta.url)).text()
}

const dialog = await source(dialogPath)
const bridge = await source(bridgePath)
const notice = await source(noticePath)

function collectI18nKeys(value: string) {
  const keys = new Set<string>()
  for (const match of value.matchAll(/t\('([^']+)'/g)) keys.add(match[1] as string)
  for (const match of value.matchAll(/'((?:admin|validation)\.[a-zA-Z0-9_.]+)'/g)) keys.add(match[1] as string)
  return [...keys]
}

function lookup(locale: Record<string, unknown>, key: string) {
  return key.split('.').reduce<unknown>((current, segment) => {
    if (!current || typeof current !== 'object') return undefined
    return (current as Record<string, unknown>)[segment]
  }, locale)
}

describe('admin page install dialog', () => {
  test('compiles and keeps its template free of structural errors', () => {
    const { descriptor } = parse(dialog, { filename: 'SFAdminPageInstallDialog.vue' })
    const compiled = compileTemplate({
      source: descriptor.template?.content || '',
      filename: 'SFAdminPageInstallDialog.vue',
      id: 'admin-page-install-dialog'
    })
    expect(compiled.errors).toEqual([])
  })

  test('shows the URL host, the full URL, the digest and the declared identity', () => {
    expect(dialog).toContain("t('admin.extensionInstall.hostAction.subtitle', { host: urlHost })")
    expect(dialog).toContain('{{ request.url }}')
    expect(dialog).toContain('{{ request.sha256 }}')
    expect(dialog).toContain('{{ request.expectedExtensionId }}')
    expect(dialog).toContain('{{ request.expectedVersion ||')
    expect(dialog).toContain('{{ t(typeLabelKey) }}')
    expect(dialog).toContain('{{ permission }}')
  })

  test('warns about super_admin, the post-download verification and staging only', () => {
    expect(dialog).toContain("t('admin.extensionInstall.hostAction.superAdminTitle')")
    expect(dialog).toContain("t('admin.extensionInstall.hostAction.verifyBody')")
    expect(dialog).toContain("t('admin.extensionInstall.hostAction.stagedOnlyBody')")
    expect(dialog).toContain("t('admin.extensionInstall.hostAction.installingHint')")
  })

  test('renders cancel, retry, success and failure branches', () => {
    for (const key of ['cancel', 'retry', 'confirm', 'close', 'failedTitle', 'openPlugins', 'openThemes']) {
      expect(dialog).toContain(`admin.extensionInstall.hostAction.${key}`)
    }
    expect(dialog).toContain("@click=\"emit('confirm')\"")
    expect(dialog).toContain("@click=\"cancel\"")
    expect(dialog).toContain("@click=\"emit('retry')\"")
    expect(dialog).toContain("@click=\"emit('openManagement')\"")
  })

  test('returns focus to the trigger and leaves the initial focus to the modal scope', () => {
    expect(dialog).toContain('restoreFocusTo = document.activeElement instanceof HTMLElement')
    expect(dialog).toContain('previous?.focus()')
    expect(dialog).toContain('watch(open, syncFocus)')
    // 不和 FocusScope 抢初始焦点：确认动作不是默认焦点，必须显式点击/一次 Tab。
    expect(dialog).not.toContain('autofocus')
    expect(dialog).not.toContain('focusPrimaryAction')
    // 遮罩/ESC 关闭等于取消，不能留下悬挂的 promise。
    expect(dialog).toContain(':dismissible="false"')
  })

  test('every dialog and bridge copy key exists in both locales', async () => {
    const locales = ['zh-CN', 'en-US'] as const
    const keys = new Set([...collectI18nKeys(dialog), ...collectI18nKeys(bridge)])
    const installKeys = [...keys].filter(key => key.startsWith('admin.extensionInstall.'))
    expect(installKeys.length).toBeGreaterThan(0)
    for (const locale of locales) {
      const messages = JSON.parse(await Bun.file(
        new URL(`../../i18n/locales/${locale}.json`, import.meta.url)
      ).text()) as Record<string, unknown>
      for (const key of installKeys) {
        expect({ locale, key, value: lookup(messages, key) }).toMatchObject({ locale, key })
        const value = lookup(messages, key)
        expect(typeof value).toBe('string')
        expect(String(value).length > 0).toBeTrue()
      }
    }
  })
})

describe('bridge injection surface', () => {
  test('injects installExtension only for pages that declare the action', () => {
    expect(bridge).toContain('if (!adminPageCanInstallExtensions(props.page)) return base')
    expect(bridge).toContain('installExtension: (request: AdminHostInstallRequest) => hostInstall.start(request)')
    expect(bridge).toContain('satisfies AdminPageBridgeV1')
  })

  test('wires the shared transport and the shared error mapping, not a private copy', () => {
    expect(bridge).toContain('useAdminPageHostInstall({')
    expect(bridge).toContain('install: installFromUrl')
    expect(bridge).toContain('toResult: toAdminHostInstallResult')
    // 安装只能走共享 transport，不能在组件里自己拼安装请求。
    expect(bridge).not.toContain('install-from-url')
    expect(bridge).not.toContain('timeout:')
  })

  test('aborts an in-flight install when the page unmounts', () => {
    expect(bridge).toContain('hostInstall.abortOnUnmount()')
  })
})

describe('host action disclosure', () => {
  test('one shared notice serves the plugin detail, the theme detail and the enable dialog', async () => {
    const consumers = [
      'app/pages/admin/extensions/plugins.vue',
      'app/pages/admin/extensions/themes.vue',
      'app/components/admin/SFAdminExtensionEnableDialog.vue'
    ]
    for (const path of consumers) {
      const content = await source(path)
      expect(content).toContain('SFAdminExtensionHostActionsNotice')
      expect(content).toContain('<SFAdminExtensionHostActionsNotice :extension=')
    }
    // 共用同一个 i18n key，避免两处文案漂移。
    expect(notice).toContain("t('admin.extensions.hostActions.title')")
    expect(notice).toContain("t('admin.extensions.hostActions.body')")
    expect(notice).toContain('extensionCanRequestInstall')
  })

  test('the notice stays out of capability risk levels', async () => {
    expect(notice).not.toContain('capabilityRisk')
    expect(notice).not.toContain('capabilityGrants')
    const enableDialog = await source('app/components/admin/SFAdminExtensionEnableDialog.vue')
    // capability 列表仍是独立分支；披露是它前面的一条单独提示。
    expect(enableDialog).toContain("t(`admin.extensions.capabilityRisk.${grant.risk}`)")
    const noticeIndex = enableDialog.indexOf('<SFAdminExtensionHostActionsNotice')
    const capabilityIndex = enableDialog.indexOf('capabilityGrants?.length')
    expect(noticeIndex).toBeGreaterThan(0)
    expect(noticeIndex).toBeLessThan(capabilityIndex)
  })
})
