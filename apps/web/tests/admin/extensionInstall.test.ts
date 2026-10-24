import { beforeAll, describe, expect, test } from 'bun:test'
import { fileURLToPath } from 'node:url'
import { compileVueSfc, flushPromises, mount, testVue } from '../helpers/vueSfc'
import * as installUtils from '../../app/utils/admin/extensions/extensionInstall'
import { adminExtensionInstallErrorMessage } from '../../app/utils/admin/extensions/hostActions'
import type { AdminExtensionInstallResult } from '../../app/utils/admin/adminExtensions'

const panelPath = fileURLToPath(new URL('../../app/components/admin/extensions/SFAdminExtensionInstallPanel.vue', import.meta.url))

let Panel: Awaited<ReturnType<typeof compileVueSfc>>
let toasts: unknown[] = []
let archiveCalls: Array<{ name: string, size: number }> = []
let urlCalls: Array<{ url: string, sha256: string }> = []
let archiveResult: AdminExtensionInstallResult
let archiveError: Error | null = null
let urlError: Error | null = null

beforeAll(async () => {
  Object.assign(globalThis, {
    ref: testVue.ref,
    computed: testVue.computed,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => params ? `${key}:${JSON.stringify(params)}` : key
    }),
    useToast: () => ({ add: (toast: unknown) => { toasts.push(toast) } }),
    useAdminRoutes: () => ({ path: (path: string) => `/control-panel${path}` }),
    useAdminExtensionInstall: () => ({
      installArchive: async (file: File) => {
        archiveCalls.push({ name: file.name, size: file.size })
        if (archiveError) throw archiveError
        return archiveResult
      },
      installFromUrl: async (url: string, sha256: string) => {
        urlCalls.push({ url, sha256 })
        if (urlError) throw urlError
        return archiveResult
      }
    }),
    apiErrorMessage: (error: unknown) => error instanceof Error ? error.message : ''
  })
  Panel = await compileVueSfc(panelPath, 'extension-install-panel', {
    EXTENSION_ARCHIVE_ACCEPT: installUtils.EXTENSION_ARCHIVE_ACCEPT,
    EXTENSION_ARCHIVE_MAX_BYTES: installUtils.EXTENSION_ARCHIVE_MAX_BYTES,
    formatFileSize: installUtils.formatFileSize,
    normalizeRemoteArchiveUrl: installUtils.normalizeRemoteArchiveUrl,
    normalizeSha256: installUtils.normalizeSha256,
    remoteArchiveHost: installUtils.remoteArchiveHost,
    validateArchiveFile: installUtils.validateArchiveFile,
    adminExtensionInstallErrorMessage
  })
})

function resetState() {
  toasts = []
  archiveCalls = []
  urlCalls = []
  archiveError = null
  urlError = null
  archiveResult = {
    upgraded: false,
    extension: {
      id: 'demo.plugin',
      name: 'Demo Plugin',
      version: '1.0.0',
      type: 'plugin',
      status: 'installed',
      manifest: {
        id: 'demo.plugin',
        name: 'Demo Plugin',
        version: '1.0.0',
        type: 'plugin'
      } as AdminExtensionInstallResult['extension']['manifest'],
      packageDigest: 'digest-1',
      packagePath: '/tmp/demo',
      installedAt: '2026-10-23T00:00:00Z',
      updatedAt: '2026-10-23T00:00:00Z'
    }
  }
}

function mountPanel() {
  return mount(Panel, { global: { stubs: uiStubs() } })
}

function uiStubs() {
  return {
    UCard: { template: '<section><slot name="header" /><slot /></section>' },
    UButton: {
      props: ['disabled', 'loading', 'type', 'to'],
      emits: ['click'],
      template: '<button :disabled="disabled" @click="$emit(\'click\', $event)"><slot /></button>'
    },
    UInput: {
      props: ['modelValue', 'disabled'],
      emits: ['update:modelValue'],
      template: '<input :value="modelValue" :disabled="disabled" @input="$emit(\'update:modelValue\', $event.target.value)">'
    },
    UFormField: { props: ['label', 'error', 'description'], template: '<label><span v-if="error" data-form-error>{{ error }}</span><slot /></label>' },
    UAlert: { props: ['title', 'description'], template: '<div data-alert>{{ title }} {{ description }}</div>' },
    UBadge: { template: '<span><slot /></span>' },
    UIcon: { template: '<i />' }
  }
}

async function chooseFile(wrapper: ReturnType<typeof mount>, file: File) {
  const input = wrapper.get('[data-testid="extension-archive-input"]')
  Object.defineProperty(input.element, 'files', { value: [file], configurable: true })
  await input.trigger('change')
  await flushPromises()
}

describe('extension install panel', () => {
  test('rejects a non-zip file without calling the installer', async () => {
    resetState()
    const wrapper = mountPanel()
    try {
      await chooseFile(wrapper, new File([new Uint8Array([1])], 'demo.txt', { type: 'text/plain' }))
      expect(wrapper.text()).toContain('admin.extensionInstall.upload.errorType')
      expect(archiveCalls).toHaveLength(0)
    } finally {
      wrapper.unmount()
    }
  })

  test('uploads a valid zip and renders the install result', async () => {
    resetState()
    const wrapper = mountPanel()
    try {
      await chooseFile(wrapper, new File([new Uint8Array([0x50, 0x4b, 0x03, 0x04])], 'demo.zip', { type: 'application/zip' }))
      await wrapper.get('[data-testid="extension-archive-submit"]').trigger('click')
      await flushPromises()
      expect(archiveCalls).toEqual([{ name: 'demo.zip', size: 4 }])
      expect(wrapper.get('[data-testid="extension-install-result"]').text()).toContain('Demo Plugin')
      expect(wrapper.get('[data-testid="extension-install-result"]').text()).toContain('demo.plugin')
      expect(toasts).toHaveLength(1)
    } finally {
      wrapper.unmount()
    }
  })

  test('submits an https url with an optional sha256', async () => {
    resetState()
    const wrapper = mountPanel()
    try {
      await wrapper.get('[data-testid="extension-url-input"]').setValue('https://example.com/demo.zip')
      await wrapper.get('[data-testid="extension-url-checksum"]').setValue('a'.repeat(64))
      await wrapper.get('[data-testid="extension-url-form"]').trigger('submit')
      await flushPromises()
      expect(urlCalls).toEqual([{ url: 'https://example.com/demo.zip', sha256: 'a'.repeat(64) }])
      expect(wrapper.get('[data-testid="extension-install-result"]').text()).toContain('Demo Plugin')
    } finally {
      wrapper.unmount()
    }
  })

  test('rejects a non-https url before any request', async () => {
    resetState()
    const wrapper = mountPanel()
    try {
      await wrapper.get('[data-testid="extension-url-input"]').setValue('http://example.com/demo.zip')
      await wrapper.get('[data-testid="extension-url-form"]').trigger('submit')
      await flushPromises()
      expect(urlCalls).toHaveLength(0)
      expect(wrapper.text()).toContain('admin.extensionInstall.url.errorScheme')
    } finally {
      wrapper.unmount()
    }
  })

  test('rejects a malformed sha256 before any request', async () => {
    resetState()
    const wrapper = mountPanel()
    try {
      await wrapper.get('[data-testid="extension-url-input"]').setValue('https://example.com/demo.zip')
      await wrapper.get('[data-testid="extension-url-checksum"]').setValue('not-a-digest')
      await wrapper.get('[data-testid="extension-url-form"]').trigger('submit')
      await flushPromises()
      expect(urlCalls).toHaveLength(0)
      expect(wrapper.text()).toContain('admin.extensionInstall.url.errorChecksum')
    } finally {
      wrapper.unmount()
    }
  })

  test('keeps a blocking install error next to the form', async () => {
    resetState()
    archiveError = new Error('upload rejected')
    const wrapper = mountPanel()
    try {
      await chooseFile(wrapper, new File([new Uint8Array([0x50, 0x4b, 0x03, 0x04])], 'demo.zip', { type: 'application/zip' }))
      await wrapper.get('[data-testid="extension-archive-submit"]').trigger('click')
      await flushPromises()
      // 非信封错误按面板兜底文案呈现，不把传输层文本直接甩给管理员。
      expect(wrapper.text()).toContain('admin.extensionInstall.upload.failed')
      expect(toasts).toHaveLength(1)
    } finally {
      wrapper.unmount()
    }
  })

  test('maps a stable API reason to the shared install message', async () => {
    resetState()
    archiveError = Object.assign(new Error('mismatch'), {
      code: 422,
      message: 'extension.expected_mismatch',
      data: { reason: 'extension.expected_mismatch' }
    })
    const wrapper = mountPanel()
    try {
      await chooseFile(wrapper, new File([new Uint8Array([0x50, 0x4b, 0x03, 0x04])], 'demo.zip', { type: 'application/zip' }))
      await wrapper.get('[data-testid="extension-archive-submit"]').trigger('click')
      await flushPromises()
      expect(wrapper.text()).toContain('admin.extensionInstall.errors.expectedMismatch')
    } finally {
      wrapper.unmount()
    }
  })
})

describe('extension install helpers', () => {
  test('validates archive files', () => {
    expect(installUtils.validateArchiveFile(null)).toEqual({ ok: false, reason: 'empty' })
    expect(installUtils.validateArchiveFile({ name: 'demo.zip', size: 0 })).toEqual({ ok: false, reason: 'empty' })
    expect(installUtils.validateArchiveFile({ name: 'demo.txt', size: 10 })).toEqual({ ok: false, reason: 'type' })
    expect(installUtils.validateArchiveFile({ name: 'demo.zip', size: installUtils.EXTENSION_ARCHIVE_MAX_BYTES + 1 })).toEqual({ ok: false, reason: 'size' })
    expect(installUtils.validateArchiveFile({ name: 'demo.zip', size: 10 })).toEqual({ ok: true })
    expect(installUtils.validateArchiveFile({ name: 'demo', type: 'application/zip', size: 10 })).toEqual({ ok: true })
  })

  test('normalizes remote archive urls', () => {
    expect(installUtils.normalizeRemoteArchiveUrl('')).toEqual({ ok: false, reason: 'empty' })
    expect(installUtils.normalizeRemoteArchiveUrl('http://example.com/demo.zip')).toEqual({ ok: false, reason: 'scheme' })
    expect(installUtils.normalizeRemoteArchiveUrl('https://user:pass@example.com/demo.zip')).toEqual({ ok: false, reason: 'credentials' })
    expect(installUtils.normalizeRemoteArchiveUrl('not a url')).toEqual({ ok: false, reason: 'invalid' })
    expect(installUtils.normalizeRemoteArchiveUrl(`https://example.com/${'a'.repeat(2048)}`)).toEqual({ ok: false, reason: 'length' })
    expect(installUtils.normalizeRemoteArchiveUrl('https://example.com/demo.zip')).toEqual({ ok: true, url: 'https://example.com/demo.zip' })
  })

  test('normalizes optional sha256 values', () => {
    expect(installUtils.normalizeSha256('')).toEqual({ ok: true, value: '' })
    expect(installUtils.normalizeSha256('not-a-digest')).toEqual({ ok: false })
    expect(installUtils.normalizeSha256('A'.repeat(64))).toEqual({ ok: true, value: 'a'.repeat(64) })
  })

  test('formats byte sizes for the drop hint', () => {
    expect(installUtils.formatFileSize(0)).toBe('0 B')
    expect(installUtils.formatFileSize(1024)).toBe('1 KB')
    expect(installUtils.formatFileSize(50 * 1024 * 1024)).toBe('50 MB')
  })
})
