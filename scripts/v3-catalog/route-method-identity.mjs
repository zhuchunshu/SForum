const identities = new Map([
  ['*', 'all'],
  ['DELETE', 'delete'],
  ['GET', 'get'],
  ['PATCH', 'patch'],
  ['POST', 'post'],
  ['PUT', 'put']
])

export function routeMethodIdentity(method) {
  const identity = identities.get(method)
  if (!identity) throw new Error(`unsupported route registration method: ${method}`)
  return identity
}
