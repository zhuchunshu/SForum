import { describe, expect, test } from 'bun:test'

import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

function source(relative: string): string {
  return readFileSync(fileURLToPath(new URL(relative, import.meta.url)), 'utf8')
}

// 注释里也会提到被禁止的写法（说明为什么不用它），断言必须只看真实代码。
function code(relative: string): string {
  return source(relative)
    .split('\n')
    .filter(line => !line.trimStart().startsWith('//') && !line.trimStart().startsWith('*'))
    .join('\n')
}

const observabilityTab = source('../../app/components/admin/ai/tabs/SFAdminAIObservabilityTab.vue')
const pageShell = source('../../app/pages/admin/ai.vue')

describe('admin AI observability budget display', () => {
  test('says the budget is unset instead of showing 0 / 0', () => {
    // 预算为 0 表示闸门停用；显示 0/0 会被读成「已用完」。
    expect(observabilityTab).toContain('budgetMicros > 0')
    expect(observabilityTab).toContain("t('admin.ai.budgetUnset')")
    expect(observabilityTab).toContain("t('admin.ai.budgetUnsetHint')")
  })

  test('hides the progress bar when there is no budget to fill', () => {
    expect(observabilityTab).toContain('<UProgress v-if="budgetMicros > 0"')
  })
})

describe('admin AI observability timestamps', () => {
  test('passes an explicit locale so the format does not depend on the runtime', () => {
    // locale code 是 zh-CN / en；language 才是 en-US。
    expect(observabilityTab).toContain("locale.value === 'en' ? 'en-US' : 'zh-CN'")
    expect(code('../../app/components/admin/ai/tabs/SFAdminAIObservabilityTab.vue')).not.toMatch(/toLocaleString\(\)/)
  })
})

describe('admin AI execution filtering', () => {
  test('offers a status filter bound to the rendered rows', () => {
    expect(observabilityTab).toContain('statusFilter')
    expect(observabilityTab).toContain('v-for="item in filteredExecutions"')
  })

  test('distinguishes "nothing recorded" from "nothing matches the filter"', () => {
    expect(observabilityTab).toContain("t('admin.ai.executionsEmptyFiltered')")
  })
})

describe('admin AI tab deep links', () => {
  test('syncs the tab into the query without a navigation', () => {
    // push 会触发完整导航并白屏；replace 只改地址栏，刷新后仍能落在原页签。
    expect(pageShell).toContain('router.replace({')
    expect(code('../../app/pages/admin/ai.vue')).not.toContain('router.push(')
  })

  test('still switches local state first so no reload is required', () => {
    expect(pageShell).toContain('activeTab.value = next')
  })
})
