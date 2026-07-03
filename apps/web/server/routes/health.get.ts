export default defineEventHandler(() => ({
  name: 'SForum Web',
  status: 'ok',
  locale: process.env.APP_LOCALE || 'zh-CN',
  supportedLocales: (process.env.SUPPORTED_LOCALES || 'zh-CN,en-US').split(','),
  time: new Date().toISOString()
}))
