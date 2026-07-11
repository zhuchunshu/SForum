import { normalizeForumSettings } from '~/utils/adminForum'
import {
  recommendedForumSettings,
  type ForumSettings
} from '~/utils/forumTaxonomy'

/**
 * 从 public web-options 解析论坛内容限制，供发帖/评论页实时校验。
 */
export function useForumContentLimits() {
  const { options } = useWebOptions()

  const limits = computed<ForumSettings>(() => {
    const o = options.value
    return normalizeForumSettings({
      defaultCategorySlug: o['forum.default_category_slug'],
      tagCreationMode: o['forum.tags.creation_mode'],
      tagPublicPages: o['forum.tags.public_pages'],
      tagMinPerTopic: o['forum.tags.min_per_topic'],
      tagMaxPerTopic: o['forum.tags.max_per_topic'],
      topicsPerPage: o['forum.pagination.topics_per_page'],
      commentsPerPage: o['forum.pagination.comments_per_page'],
      topicTitleMinRunes: o['forum.topics.title_min_runes'],
      topicTitleMaxRunes: o['forum.topics.title_max_runes'],
      topicContentMinRunes: o['forum.topics.content_min_runes'],
      topicContentMaxRunes: o['forum.topics.content_max_runes'],
      topicEditWindowMinutes: o['forum.topics.edit_window_minutes'],
      topicCooldownSeconds: o['forum.topics.cooldown_seconds'],
      dailyTopicLimit: o['forum.topics.daily_limit'],
      commentMinRunes: o['forum.comments.min_runes'],
      commentMaxRunes: o['forum.comments.max_runes'],
      commentMaxNestingDepth: o['forum.comments.max_nesting_depth'],
      commentEditWindowMinutes: o['forum.comments.edit_window_minutes'],
      commentCooldownSeconds: o['forum.comments.cooldown_seconds'],
      dailyCommentLimit: o['forum.comments.daily_limit'],
      excerptRuneLimit: o['forum.reading.excerpt_rune_limit']
    })
  })

  function runeLength(value: string) {
    return Array.from(value.trim()).length
  }

  function validateTopicTitle(title: string) {
    const count = runeLength(title)
    const { topicTitleMinRunes, topicTitleMaxRunes } = limits.value
    if (count < topicTitleMinRunes) {
      return 'titleTooShort'
    }
    if (count > topicTitleMaxRunes) {
      return 'titleTooLong'
    }
    return null
  }

  function validateTopicBody(body: string) {
    const count = runeLength(body)
    const { topicContentMinRunes, topicContentMaxRunes } = limits.value
    if (count < topicContentMinRunes) {
      return 'contentTooShort'
    }
    if (count > topicContentMaxRunes) {
      return 'contentTooLong'
    }
    return null
  }

  function validateCommentBody(body: string) {
    const count = runeLength(body)
    const { commentMinRunes, commentMaxRunes } = limits.value
    if (count < commentMinRunes) {
      return 'commentTooShort'
    }
    if (count > commentMaxRunes) {
      return 'commentTooLong'
    }
    return null
  }

  function validateTagCount(count: number) {
    const { tagMinPerTopic, tagMaxPerTopic } = limits.value
    if (count < tagMinPerTopic) {
      return 'tagMin'
    }
    if (count > tagMaxPerTopic) {
      return 'tagMax'
    }
    return null
  }

  return {
    limits,
    recommended: recommendedForumSettings,
    runeLength,
    validateTopicTitle,
    validateTopicBody,
    validateCommentBody,
    validateTagCount
  }
}
