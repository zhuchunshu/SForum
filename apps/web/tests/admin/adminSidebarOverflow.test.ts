import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')

describe('admin sidebar overflow contract', () => {
  test('sidebar layout clips horizontal overflow and truncates long labels', () => {
    const layout = source('../../app/layouts/admin.vue')
    const css = source('../../app/assets/css/main.css')

    // 根侧栏与菜单滚动容器禁止横向滚动
    expect(layout).toContain('sforum-admin-sidebar min-w-0 overflow-x-hidden')
    expect(layout).toContain('overflow-x-hidden overflow-y-auto')
    expect(layout).not.toContain('class="-mx-2"')

    // 菜单文案可收缩截断，避免 flex 子项把侧栏撑宽
    expect(layout).toContain("linkLabel: '!min-w-0 !truncate !leading-tight'")
    expect(layout).toContain("childLinkLabel: '!min-w-0 !truncate !leading-tight'")

    // Web 与 Core 共用一个版本，展示在左上角 SForum 品牌名右侧。
    expect(layout).toContain("formatOverviewVersion('', sforumBuild.version, sforumBuild.commit)")
    expect(layout).toContain('{{ sforumVersion }}')
    expect(layout).toContain('class="ml-auto shrink-0 font-mono text-[10px] font-semibold"')

    expect(css).toContain('.sforum-admin-sidebar {\n  overflow-x: hidden;')
    expect(css).toContain('text-overflow: ellipsis !important;')
  })
})
