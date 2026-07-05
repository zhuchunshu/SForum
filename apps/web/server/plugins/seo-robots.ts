const aiCrawlerAgents = ['GPTBot', 'ChatGPT-User', 'ClaudeBot', 'PerplexityBot', 'CCBot']
const nonSEOCrawlerAgents = ['AhrefsBot', 'SemrushBot', 'MJ12bot', 'DotBot']

export default defineNitroPlugin((nitroApp) => {
  nitroApp.hooks.hook('robots:robots-txt', async (ctx: { robotsTxt: string }) => {
    const settings = await loadServerSEOSettings()
    if (!serverSEOIndexable(settings)) {
      ctx.robotsTxt = [
        'User-agent: *',
        'Disallow: /'
      ].join('\n')
      return
    }

    const lines: string[] = []
    if (settings.robotsExtraAllow.length || settings.robotsExtraDisallow.length) {
      lines.push('', 'User-agent: *')
      for (const path of settings.robotsExtraAllow) {
        lines.push(`Allow: ${path}`)
      }
      for (const path of settings.robotsExtraDisallow) {
        lines.push(`Disallow: ${path}`)
      }
    }

    appendBlockedAgents(lines, settings.blockAiBots ? aiCrawlerAgents : [])
    appendBlockedAgents(lines, settings.blockNonSeoBots ? nonSEOCrawlerAgents : [])

    if (lines.length > 0) {
      ctx.robotsTxt = `${ctx.robotsTxt.trimEnd()}\n${lines.join('\n')}\n`
    }
  })
})

function appendBlockedAgents(lines: string[], agents: string[]) {
  for (const agent of agents) {
    lines.push('', `User-agent: ${agent}`, 'Disallow: /')
  }
}
