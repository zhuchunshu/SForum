import { describe, expect, test } from 'bun:test'

import {
  commentListChanged,
  commentLiveAppendLimit,
  decideCommentLive,
  type CommentLiveContext
} from '../../app/utils/forum/commentLiveDecision'

const atBottom: CommentLiveContext = {
  isLastPage: true,
  nearBottom: true,
  composerOpen: false,
  selfJustPosted: false,
  autoAppendLimit: 20
}

const local = { total: 40, lastItemId: 903, revision: 7 }

describe('comment live decision matrix', () => {
  test('ignores revisions that did not advance', () => {
    expect(decideCommentLive({ revision: 7, commentCount: 41 }, local, atBottom)).toEqual({ kind: 'ignore' })
    expect(decideCommentLive({ revision: 6, commentCount: 41 }, local, atBottom)).toEqual({ kind: 'ignore' })
  })

  test('appends silently when the reader is at the bottom of the last page', () => {
    expect(decideCommentLive({ revision: 8, commentCount: 41 }, local, atBottom)).toEqual({ kind: 'append', newCount: 1 })
  })

  test('only offers a pill when the reader has scrolled up', () => {
    expect(decideCommentLive({ revision: 8, commentCount: 43 }, local, { ...atBottom, nearBottom: false }))
      .toEqual({ kind: 'pill-new', newCount: 3, jumpToLastPage: false })
  })

  test('never moves DOM while the composer is open', () => {
    expect(decideCommentLive({ revision: 8, commentCount: 41 }, local, { ...atBottom, composerOpen: true }))
      .toEqual({ kind: 'pill-new', newCount: 1, jumpToLastPage: false })
  })

  test('points to the last page when the reader is behind', () => {
    expect(decideCommentLive({ revision: 8, commentCount: 90 }, local, { ...atBottom, isLastPage: false }))
      .toEqual({ kind: 'pill-new', newCount: 50, jumpToLastPage: true })
  })

  test('caps one-shot catch-up at a single page', () => {
    expect(decideCommentLive({ revision: 8, commentCount: 61 }, local, { ...atBottom, autoAppendLimit: 20 }))
      .toEqual({ kind: 'pill-new', newCount: 21, jumpToLastPage: true })
    expect(decideCommentLive({ revision: 8, commentCount: 60 }, local, { ...atBottom, autoAppendLimit: 20 }))
      .toEqual({ kind: 'append', newCount: 20 })
  })

  test('suppresses the reader own fresh comment', () => {
    expect(decideCommentLive({ revision: 8, commentCount: 41 }, local, { ...atBottom, selfJustPosted: true }))
      .toEqual({ kind: 'ignore' })
  })

  test('treats an unchanged count as a silent reconcile', () => {
    expect(decideCommentLive({ revision: 9, commentCount: 40 }, local, atBottom)).toEqual({ kind: 'reconcile' })
  })

  test('clamps a shrinking count to zero new comments', () => {
    // 隐藏/删除让公开数量下降：不能算出「负数条新评论」。
    expect(decideCommentLive({ revision: 9, commentCount: 39 }, local, atBottom)).toEqual({ kind: 'reconcile' })
  })
})

describe('comment live helpers', () => {
  test('append limit follows the page size with a safe floor', () => {
    expect(commentLiveAppendLimit(50)).toBe(50)
    expect(commentLiveAppendLimit(0)).toBe(20)
    expect(commentLiveAppendLimit(-5)).toBe(20)
  })

  test('detects visible list changes only when something actually differs', () => {
    const before = [
      { id: 1, updatedAt: '2026-10-03T10:00:00Z', status: 'active', replyCount: 0 },
      { id: 2, updatedAt: '2026-10-03T10:05:00Z', status: 'active', replyCount: 1 }
    ]
    expect(commentListChanged(before, [...before])).toBe(false)
    expect(commentListChanged(before, [before[0]!, { ...before[1]!, updatedAt: '2026-10-03T11:00:00Z' }])).toBe(true)
    expect(commentListChanged(before, [before[0]!, { ...before[1]!, replyCount: 2 }])).toBe(true)
    expect(commentListChanged(before, [before[0]!, { ...before[1]!, status: 'deleted' }])).toBe(true)
    expect(commentListChanged(before, [before[0]!])).toBe(true)
    expect(commentListChanged(before, [before[0]!, before[1]!, { id: 3, updatedAt: '2026-10-03T10:06:00Z' }])).toBe(true)
  })
})
