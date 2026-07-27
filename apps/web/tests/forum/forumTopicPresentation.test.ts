import { describe, expect, test } from 'bun:test'

import {
  buildCommentActionMenuItems,
  buildTopicActionMenuItems,
  type TopicActionLabels,
  type TopicActionMenuInput
} from '../../app/utils/forum/forumTopicPresentation'

const labels: TopicActionLabels = {
  edit: 'Edit',
  delete: 'Delete',
  lock: 'Lock',
  unlock: 'Unlock',
  pin: 'Pin',
  unpin: 'Unpin',
  hide: 'Hide',
  restore: 'Restore',
  report: 'Report'
}

function input(overrides: Partial<TopicActionMenuInput> = {}): TopicActionMenuInput {
  return {
    canEdit: false,
    canDelete: false,
    canLock: false,
    canPin: false,
    canModerate: false,
    canReport: false,
    locked: false,
    pinned: false,
    hidden: false,
    labels,
    extensions: [],
    ...overrides
  }
}

describe('topic action presentation', () => {
  test('does not invent core actions when every permission is denied', () => {
    expect(buildTopicActionMenuItems(input())).toEqual([])
  })

  test('selects the lifecycle command matching the current topic state', () => {
    const items = buildTopicActionMenuItems(input({
      canEdit: true,
      canLock: true,
      canPin: true,
      canModerate: true,
      canReport: true,
      locked: true,
      pinned: true,
      extensions: [{
        extensionId: 'demo.plugin',
        id: 'bookmark',
        label: 'Bookmark',
        icon: 'i-lucide-bookmark'
      }]
    }))

    expect(items.map(item => item.id)).toEqual([
      'edit',
      'unlock',
      'unpin',
      'hide',
      'report',
      'extension:demo.plugin:bookmark'
    ])
    expect(items.at(-1)?.extension).toEqual({
      extensionId: 'demo.plugin',
      actionId: 'bookmark'
    })
  })

  test('marks destructive commands and preserves extension ordering', () => {
    const items = buildTopicActionMenuItems(input({
      canDelete: true,
      canModerate: true,
      hidden: true,
      extensions: [
        { extensionId: 'demo', id: 'first', label: 'First', confirm: true },
        { extensionId: 'demo', id: 'second', label: 'Second' }
      ]
    }))

    expect(items.map(item => item.id)).toEqual([
      'restore',
      'delete',
      'extension:demo:first',
      'extension:demo:second'
    ])
    expect(items.find(item => item.id === 'delete')).toMatchObject({
      tone: 'danger',
      requiresConfirm: true
    })
    expect(items.find(item => item.id === 'extension:demo:first')?.requiresConfirm).toBe(true)
  })
})

describe('comment action presentation', () => {
  test('keeps authorized core actions ahead of extension actions', () => {
    expect(buildCommentActionMenuItems({
      canReply: true,
      canEdit: false,
      canDelete: true,
      canReport: false,
      labels: { reply: 'Reply', link: 'Link', edit: 'Edit', delete: 'Delete', report: 'Report' },
      extensions: [{ label: 'Resolve', value: 'extension:demo:resolve', icon: 'i-lucide-check' }]
    })).toEqual([
      { label: 'Reply', value: 'reply', icon: 'i-lucide-reply' },
      { label: 'Link', value: 'link', icon: 'i-lucide-link' },
      { label: 'Delete', value: 'delete', icon: 'i-lucide-trash-2' },
      { label: 'Resolve', value: 'extension:demo:resolve', icon: 'i-lucide-check' }
    ])
  })
})
