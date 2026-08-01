import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import {
  REVIEW_REQUIRED_ACTIONS,
  actionListForContext,
  parsePositiveInt,
  parseModerationSource,
  parseTargetType,
  parseWorkbenchTab,
  queueQueryForWorkbench,
  queueItemKey,
  reviewQuery,
  selectionFromQueueItem,
  selectionFromQuery,
  tabToSource
} from '../../app/utils/moderation/moderationWorkbench'
import type { ModerationDecision, ModerationPendingItem, ModerationReportItem, ModerationReviewContext } from '../../app/composables/moderation/useModerationApi'

const pendingItem: ModerationPendingItem = {
  targetType: 'topic',
  targetId: 62,
  title: 'Pending title',
  excerpt: 'Pending excerpt',
  authorId: 1,
  authorName: 'Author',
  category: 'General',
  triggers: ['new_user'],
  createdAt: '2026-07-23T00:00:00Z'
}

const reportItem: ModerationReportItem = {
  id: 7,
  reporterUserId: 2,
  reporterName: 'Reporter',
  targetType: 'comment',
  targetId: 184,
  reasonCode: 'spam',
  body: 'Spam report',
  status: 'open',
  reviewNote: '',
  createdAt: '2026-07-23T00:00:00Z',
  updatedAt: '2026-07-23T00:00:00Z',
  title: 'Reported comment',
  excerpt: 'Reported excerpt',
  targetAuthorId: 3,
  targetAuthorName: 'Target',
  category: 'General',
  targetStatus: 'active'
}

const historyItem: ModerationDecision = {
  id: 9,
  source: 'report',
  targetType: 'topic',
  targetId: 58,
  reportId: 7,
  action: 'hide_and_close',
  reviewerUserId: 1,
  reviewerName: 'Reviewer',
  reviewNote: 'Handled',
  triggers: [],
  createdAt: '2026-07-23T00:00:00Z'
}

const context: ModerationReviewContext = {
  source: 'pre_publish',
  targetType: 'topic',
  targetId: 62,
  topicId: 62,
  title: 'Pending title',
  html: '<p>Safe content</p>',
  authorId: 1,
  authorName: 'Author',
  category: 'General',
  status: 'pending',
  triggers: ['new_user'],
  createdAt: '2026-07-23T00:00:00Z'
}

describe('moderation workbench state', () => {
  test('defaults queue URL state without inventing abbreviations', () => {
    expect(parseWorkbenchTab(undefined)).toBe('pending')
    expect(parseWorkbenchTab('reports')).toBe('reports')
    expect(parseWorkbenchTab('unknown')).toBe('pending')
    expect(parseTargetType(undefined)).toBe('all')
    expect(parseTargetType('comment')).toBe('comment')
    expect(parseTargetType('bad')).toBe('all')
    expect(parsePositiveInt('0', 3)).toBe(3)
    expect(parsePositiveInt('4')).toBe(4)
    expect(parseModerationSource('report', 'pre_publish')).toBe('report')
    expect(parseModerationSource('bad', 'pre_publish')).toBe('pre_publish')
  })

  test('maps sources and stable identifiers without crossing queues', () => {
    expect(tabToSource('pending')).toBe('pre_publish')
    expect(tabToSource('reports')).toBe('report')
    expect(tabToSource('history')).toBe('pre_publish')
    expect(queueItemKey('pending', pendingItem)).toBe('pending:topic:62')
    expect(queueItemKey('reports', reportItem)).toBe('report:7:comment:184')
    expect(queueItemKey('history', historyItem)).toBe('history:9')
  })

  test('serializes review URL state without note or body content', () => {
    const selection = selectionFromQueueItem('reports', reportItem)
    expect(selection).toMatchObject({ tab: 'reports', source: 'report', targetType: 'comment', targetId: 184, reportId: 7 })
    expect(reviewQuery(selection)).toEqual({ source: 'report', reviewType: 'comment', reviewId: '184', reportId: '7', decisionId: undefined })
    expect(Object.keys(reviewQuery(selection))).not.toContain('reviewNote')
    expect(Object.keys(reviewQuery(selection))).not.toContain('html')
  })

  test('restores direct review selections from stable query fields', () => {
    expect(selectionFromQuery({ reviewType: 'topic', reviewId: '62' }, 'pending')).toMatchObject({ source: 'pre_publish', targetType: 'topic', targetId: 62 })
    expect(selectionFromQuery({ reviewType: 'comment', reviewId: '184', reportId: '7' }, 'reports')).toMatchObject({ source: 'report', targetType: 'comment', targetId: 184, reportId: 7 })
    expect(selectionFromQuery({ source: 'report', reviewType: 'topic', reviewId: '58', reportId: '7', decisionId: '9' }, 'history')).toMatchObject({ tab: 'history', source: 'report', decisionId: 9 })
    expect(selectionFromQuery({ source: 'report', reviewType: 'topic', reviewId: '58' }, 'pending')).toMatchObject({ tab: 'pending', source: 'pre_publish' })
    expect(selectionFromQuery({ reviewType: 'topic', reviewId: '0' }, 'pending')).toBeNull()
  })

  test('keeps queue filters while clearing review-only URL fields', () => {
    expect(queueQueryForWorkbench({
      tab: 'reports',
      source: 'report',
      targetType: 'comment',
      page: '3',
      reviewType: 'comment',
      reviewId: '184',
      reportId: '7',
      decisionId: '9'
    })).toEqual({
      tab: 'reports',
      source: undefined,
      targetType: 'comment',
      page: 3,
      reviewType: undefined,
      reviewId: undefined,
      reportId: undefined,
      decisionId: undefined
    })
    expect(queueQueryForWorkbench({ tab: 'pending', targetType: 'all', page: '1' })).toEqual({ tab: undefined, targetType: undefined, page: undefined, source: undefined, reviewType: undefined, reviewId: undefined, reportId: undefined, decisionId: undefined })
  })

  test('keeps destructive note rules and history readonly decisions explicit', () => {
    expect([...REVIEW_REQUIRED_ACTIONS]).toEqual(['reject', 'hide_and_close', 'delete_and_close'])
    expect(actionListForContext(context)).toEqual(['approve', 'reject'])
    expect(actionListForContext({ ...context, source: 'report', reportId: 7 })).toEqual(['keep_and_close', 'hide_and_close', 'delete_and_close'])
    expect(actionListForContext(context, true)).toEqual([])
  })
})

describe('moderation workbench implementation constraints', () => {
  const page = readFileSync(new URL('../../app/components/moderation/SFModerationReviewPage.vue', import.meta.url), 'utf8')
  const reader = readFileSync(new URL('../../app/components/moderation/ModerationReviewReader.vue', import.meta.url), 'utf8')
  const rail = readFileSync(new URL('../../app/components/moderation/ModerationDecisionRail.vue', import.meta.url), 'utf8')
  const nav = readFileSync(new URL('../../app/components/moderation/ModerationWorkbenchNav.vue', import.meta.url), 'utf8')
  const state = readFileSync(new URL('../../app/utils/moderation/moderationWorkbench.ts', import.meta.url), 'utf8')
  const css = readFileSync(new URL('../../app/assets/css/sforum-moderation.css', import.meta.url), 'utf8')
  const nuxtConfig = readFileSync(new URL('../../nuxt.config.ts', import.meta.url), 'utf8')

  test('uses the formal moderation API composable and query-backed review state', () => {
    expect(page).toContain('useModerationApi()')
    expect(page).toContain('error: countsError')
    expect(page).toContain('error: listError')
    expect(page).toContain('queueErrorMessage')
    expect(page).toContain('!queueErrorMessage && !items.length')
    expect(page).toContain('countsAvailable')
    expect(state).toContain('parseModerationSource')
    expect(state).toContain('source')
    expect(state).toContain('reviewType')
    expect(state).toContain('reviewId')
    expect(state).toContain('reportId')
    expect(page).toContain('router.push')
    expect(page).toContain('router.replace')
    expect(page).toContain('queueScroll')
  })

  test('keeps body rendering on the sanitizer path', () => {
    expect(reader).toContain('sanitizeHtml(context.html)')
    expect(page).not.toContain('v-html')
  })

  test('keeps history readonly and reuses home three-column chrome tokens', () => {
    expect(page).toContain('readonlyReview')
    expect(page).toContain("tab === 'history'")
    expect(rail).toContain('historyReadonly')
    // 直接复用首页三栏 chrome class，不自造 layout/sidebar 壳
    expect(page).toContain('sforum-home__layout sforum-home__layout--with-right')
    expect(page).toContain('sforum-home__sidebar')
    expect(page).toContain('sforum-home__main')
    expect(page).toContain('SFHomeNavigation')
    expect(page).toContain('ModerationWorkbenchNav')
    expect(page).toContain('ModerationQueueRail')
    expect(page).toContain('<SFResponsivePublicSidebar')
    expect(page).toContain('owner-id="forum.moderation.review"')
    expect(page).not.toContain('forum-mobile-menu-open')
    expect(page).not.toContain('@click="mobileInfoOpen = true"')
    expect(page).not.toContain('@click="openMobileMenu"')
    expect(page).toContain('sforum-mobile-drawer__backdrop')
    // 历史 tab 徽章必须用 historyTotal
    expect(page).toContain('historyTotal')
    expect(page).toContain('sourceCountFor')
    // 右栏决策/队列壳走 home right + right-rail 卡片
    expect(rail).toContain('sforum-home__right')
    expect(rail).toContain('sf-home-right-rail')
    expect(css).toContain('.sforum-moderation__overview-summary')
    expect(css).toMatch(/\.sforum-moderation__main\s*\{[^}]*background: var\(--sf-public-surface\);/s)
    // 右栏内容不再额外水平缩进，与 home flat rail 同轨
    expect(css).not.toContain('padding: 4px 14px 10px')
    expect(css).toContain('padding: 4px 0 10px')
  })

  test('host chrome sidebar tokens retain the shared host geometry', () => {
    // fail-closed host 路径仍依赖 sforum-home 共享三栏轨道。
    const homeCss = readFileSync(new URL('../../app/assets/css/sforum-home.css', import.meta.url), 'utf8')
    expect(homeCss).toContain('padding: 24px 24px 20px 28px;')
    expect(homeCss).toContain('padding: 34px 28px;')
    expect(homeCss).toContain('min-height: 40px;')
    expect(homeCss).toContain('.sf-home-navigation__link.is-active::before')
    expect(homeCss).toContain('padding-left: var(--sf-public-edge-inset, 24px);')
    expect(nav).toContain('sf-home-navigation__link')
    expect(nav).toContain('sf-home-navigation__label')
  })

  test('loads workbench styles with the initial Nuxt document instead of the async island', () => {
    expect(nuxtConfig).toContain("'~/assets/css/sforum-moderation.css'")
    expect(page).not.toContain('<style src="~/assets/css/sforum-moderation.css"')
  })
})
