const phaseContracts = {
  P0: 'v3.governance.catalog@1',
  P1: 'sforum.trust@1 / sforum.recovery@1',
  P2: 'sforum.manifest@3',
  P3: 'sforum.host@2',
  P4: 'sforum.lifecycle@2',
  P5: 'sforum.database@1 / sforum.command@1',
  P6: 'sforum.route@1',
  P7: 'sforum.hook@1 / dedicated registry contracts',
  P8: 'sforum.theme.runtime@1 / sforum.page.viewmodel@1',
  P9: 'sforum.component@1 / sforum.asset@1 / sforum.frontend@1',
  P10: 'sforum.content@1 / sforum.media@1 / sforum.entity@1',
  P11: 'sforum.cache@1 / sforum.platform.services@1',
  P12: 'sforum.distribution@1 / Host and Frontend API LTS',
  P13: 'V3 reference-package and parity gates'
}

const phaseRollback = {
  P0: 'Revert catalog and governance documents; runtime is unchanged.',
  P1: 'Disable new grants while retaining host-owned safe mode and CLI recovery.',
  P2: 'Reject Manifest V3 while keeping V1/V2 packages usable.',
  P3: 'Select protocol v1 per plugin; retain the v2 contract additively.',
  P4: 'Keep v1 lifecycle for old packages; never downgrade after migrations start.',
  P5: 'Revoke new grants and retain plugin data; never drop it automatically.',
  P6: 'Atomically select the previous route snapshot and namespaced proxy.',
  P7: 'Atomically remove the registry revision and preserve Host-owned grants.',
  P8: 'Select the core provider and retain the legacy Page/L1 path behind a flag.',
  P9: 'Revoke the component grant and restore the previous/core provider snapshot.',
  P10: 'Render stable fallbacks and preserve source content and original media.',
  P11: 'Restore core providers and preserve plugin namespaces for retry or uninstall.',
  P12: 'Point desired revision to prior artifacts and retain published LTS shims.',
  P13: 'Delete compatibility paths only after parity; otherwise restore them.'
}

const phaseBySurface = new Map(Object.entries({
  // Template and theme rows (27).
  'Page ownership': 'P8',
  'Template executor': 'P8',
  'Load timing': 'P8',
  Compilation: 'P8',
  'Runtime cache': 'P8',
  'Provider lookup': 'P8',
  'Template safety': 'P8',
  Layout: 'P8',
  Partial: 'P8',
  'Control syntax': 'P8',
  'Page data': 'P8',
  'Default theme': 'P13',
  'Missing-template fallback': 'P8',
  'Host islands': 'P9',
  'Island parsing': 'P8',
  L0: 'P8',
  L1: 'P8',
  L2: 'P9',
  'Plugin templates': 'P9',
  'Theme override of plugin templates': 'P9',
  SEO: 'P11',
  'Lazy loading': 'P9',
  'Theme switch': 'P8',
  'Nuxt build': 'P8',
  'Request-time template I/O': 'P8',
  'Request-time theme database lookup': 'P8',
  'Browser without JavaScript': 'P13',

  // Plugin rows (72).
  Trust: 'P1',
  'Package install': 'P1',
  Enable: 'P1',
  'Backend runtime': 'P3',
  'Extension surface coverage': 'P0',
  'Route add': 'P6',
  'Route alias': 'P6',
  Redirect: 'P6',
  'Internal rewrite': 'P6',
  'Before processing': 'P6',
  'After processing': 'P6',
  'Request/response filtering': 'P6',
  'Handler wrapping': 'P6',
  'Handler replacement': 'P6',
  'Streaming routes': 'P6',
  'Route guard': 'P6',
  'Route conflict': 'P6',
  'UI extension': 'P9',
  'Admin surfaces': 'P7',
  'Admin lists and actions': 'P7',
  'Component actions': 'P9',
  'Component data': 'P9',
  'SSR fragments': 'P9',
  'Frontend code': 'P9',
  'Asset injection': 'P9',
  'Navigation and regions': 'P9',
  'Media pipeline': 'P10',
  'Own database': 'P5',
  'SQL migrations': 'P5',
  'Core data read': 'P5',
  'Core data mutation': 'P5',
  'Transactional core workflows': 'P5',
  'Own transactions': 'P5',
  'Custom entity': 'P10',
  'Custom taxonomy': 'P10',
  'Custom fields': 'P10',
  'Query pipeline': 'P7',
  'Identity and permissions': 'P7',
  'Authentication and profile': 'P7',
  'Actions and filters': 'P7',
  'Plugin-defined hooks': 'P7',
  'Plugin extends plugin': 'P7',
  'Custom blocks': 'P10',
  'Editor extension': 'P10',
  'Shortcode and embed': 'P10',
  'Content rendering': 'P10',
  'Cache API': 'P11',
  'Cache policy': 'P11',
  'Cache provider': 'P11',
  'Provider slots': 'P7',
  Jobs: 'P7',
  Schedules: 'P7',
  CLI: 'P7',
  OpenAPI: 'P6',
  Secrets: 'P11',
  Filesystem: 'P11',
  'HTTP client': 'P11',
  Localization: 'P11',
  'Settings lifecycle': 'P11',
  Dependencies: 'P2',
  'Compatibility policy': 'P12',
  Updates: 'P12',
  Distribution: 'P12',
  'Deployment extensions': 'P12',
  Lifecycle: 'P4',
  Uninstall: 'P4',
  'Data retention': 'P4',
  'External cleanup': 'P4',
  'Failure recovery': 'P1',
  'Multi-node': 'P12',
  Debugging: 'P12',
  'Developer testing': 'P12'
}))

function slug(value) {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
}

export function buildTraceability(rows) {
  const seen = new Set()
  return rows.map((row, index) => {
    const key = `${row.family}:${row.surface}`
    if (seen.has(key)) {
      throw new Error(`duplicate authoritative traceability row: ${key}`)
    }
    seen.add(key)
    const phase = phaseBySurface.get(row.surface)
    if (!phase) {
      throw new Error(`authoritative row has no phase mapping: ${row.surface}`)
    }
    return {
      row: index + 1,
      ...row,
      phase,
      contract: phaseContracts[phase],
      test: `v3.${phase.toLowerCase()}.${row.family}.${slug(row.surface)}`,
      rollback: phaseRollback[phase]
    }
  })
}
