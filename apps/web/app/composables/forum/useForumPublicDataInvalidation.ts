const FORUM_HOME_STATE_KEYS = [
  'forum-home-loaded-topics',
  'forum-home-topic-total',
  'forum-home-loaded-feed-key',
  'forum-home-rendered-at'
]

export function invalidateForumPublicData() {
  clearNuxtData(key => key.startsWith('forum-home-topics:')
    || key === 'forum-home-category-groups'
    || key === 'forum-home-tags')
  for (const key of FORUM_HOME_STATE_KEYS) {
    clearNuxtState(key)
  }
}
