export default defineEventHandler((event) => {
  setHeader(event, 'Cache-Control', 'no-store')
  return {
    releaseId: process.env.SFORUM_WEB_RELEASE_ID || 'core',
    reloadMode: process.env.SFORUM_WEB_RELEASE_RELOAD_MODE === 'force' ? 'force' : 'prompt'
  }
})
