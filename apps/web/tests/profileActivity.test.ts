import { describe, expect, test } from 'bun:test'

import {
  groupProfileActivitiesByDate,
  profileActivityLink
} from '../app/utils/profileActivity'
import type { ProfileActivity } from '../app/composables/useProfileApi'
import type { SiteDateTimeSettings } from '../app/utils/siteDateTime'

const settings: SiteDateTimeSettings = {
  timezone: 'Asia/Shanghai',
  dateFormat: 'Y-m-d',
  timeFormat: 'H:i',
  startOfWeek: 1
}

const topic = {
  id: 42,
  slug: 'hello-world',
  title: 'Hello world',
  status: 'active',
  categorySlug: 'general',
  categoryName: 'General',
  commentCount: 3,
  createdAt: '2026-07-22T09:00:00.000Z',
  updatedAt: '2026-07-22T09:00:00.000Z',
  lastActivityAt: '2026-07-22T10:00:00.000Z'
}

describe('profile activity presentation', () => {
  test('links topics and comments through legal topic URLs and comment anchors', () => {
    expect(profileActivityLink({
      kind: 'topic',
      topic,
      excerpt: 'Topic body',
      createdAt: '2026-07-22T09:00:00.000Z'
    }, 'id_slug')).toBe('/t/42/hello-world')

    expect(profileActivityLink({
      kind: 'comment',
      topic,
      commentId: 99,
      excerpt: 'Reply body',
      createdAt: '2026-07-22T10:00:00.000Z'
    }, 'id_slug')).toBe('/t/42/hello-world#comment-99')
  })

  test('groups activities by site timezone date with today and yesterday labels', () => {
    const activities: ProfileActivity[] = [
      {
        kind: 'topic',
        topic,
        excerpt: 'Today in Shanghai',
        createdAt: '2026-07-23T01:30:00.000Z'
      },
      {
        kind: 'comment',
        topic,
        commentId: 7,
        excerpt: 'Yesterday in Shanghai',
        createdAt: '2026-07-22T02:00:00.000Z'
      }
    ]

    const groups = groupProfileActivitiesByDate(activities, {
      settings,
      locale: 'zh-CN',
      topicUrlMode: 'id_slug',
      labels: { today: '今天', yesterday: '昨天' },
      now: new Date('2026-07-23T12:00:00.000Z')
    })

    expect(groups.map(group => group.label)).toEqual(['今天', '昨天'])
    expect(groups[0].key).toBe('2026-07-23')
    expect(groups[0].items[0].timeLabel).toMatch(/09:30/)
    expect(groups[1].items[0].to).toBe('/t/42/hello-world#comment-7')
  })

  test('drops invalid timestamps instead of creating hydration-unstable groups', () => {
    const groups = groupProfileActivitiesByDate([{
      kind: 'topic',
      topic,
      excerpt: '',
      createdAt: 'not-a-date'
    }], {
      settings,
      locale: 'en-US',
      topicUrlMode: 'id',
      labels: { today: 'Today', yesterday: 'Yesterday' },
      now: new Date('2026-07-23T12:00:00.000Z')
    })

    expect(groups).toEqual([])
  })
})
