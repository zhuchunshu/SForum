import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const root = process.cwd()
const read = path => readFileSync(resolve(root, path), 'utf8')
const assert = (condition, message) => {
  if (!condition) throw new Error(message)
}

const doc = read('docs/extensions/host-api-v2.md')
const guide = read('docs/extensions/authoring-guide.md')
const commonProto = read('contracts/proto/sforum/protocol/v2/common.proto')
const controlProto = read('contracts/proto/sforum/protocol/v2/control.proto')
const pluginProto = read('contracts/proto/sforum/plugin/v2/runtime.proto')
const hostProtoPaths = [
  'contracts/proto/sforum/host/v2/data_work.proto',
  'contracts/proto/sforum/host/v2/platform.proto',
  'contracts/proto/sforum/host/v2/query_command.proto',
  'contracts/proto/sforum/host/v2/resources.proto'
]
const hostProto = hostProtoPaths.map(read).join('\n')
const server = read('apps/api/app/Support/Extensions/protocol_v2_server.go')
const hostRuntime = read('apps/api/app/Support/Extensions/protocol_v2_host.go')
const clientRuntime = read('apps/api/app/Support/Extensions/protocol_v2_client.go')
const testScript = read('scripts/test.sh')

for (const [source, packageName] of [
  [commonProto + controlProto, 'sforum.protocol.v2'],
  [pluginProto, 'sforum.plugin.v2'],
  [hostProto, 'sforum.host.v2']
]) {
  assert(source.includes(`package ${packageName};`), `missing Protobuf package ${packageName}`)
  assert(doc.includes(`\`${packageName}\``), `Host API v2 docs omit package ${packageName}`)
}

for (const field of [
  'host_broker_id', 'runtime_token', 'host_api_version', 'RequestContext',
  'ExtensionIdentity', 'AuthorityGrant', 'Actor', 'deadline'
]) {
  assert(commonProto.includes(field) || controlProto.includes(field), `wire contract no longer defines ${field}`)
  assert(doc.includes(field), `Host API v2 docs omit wire field ${field}`)
}

const metadata = hostRuntime.match(/ProtocolV2RuntimeTokenMetadataKey\s*=\s*"([^"]+)"/)?.[1]
assert(metadata === 'x-sforum-runtime-token-bin', `unexpected runtime token metadata key ${metadata}`)
assert(doc.includes(`\`${metadata}\``), 'Host API v2 docs omit the runtime token metadata key')

const maxMessageParts = server.match(/DefaultProtocolV2MaxMessageBytes\s*=\s*(\d+)\s*<<\s*(\d+)/)
assert(maxMessageParts, 'cannot resolve protocol v2 message limit')
const maxMessage = Number(maxMessageParts[1]) * (2 ** Number(maxMessageParts[2]))
const concurrency = Number(server.match(/DefaultProtocolV2ConcurrentCalls\s*=\s*(\d+)/)?.[1])
const timeout = Number(server.match(/DefaultProtocolV2RequestTimeout\s*=\s*(\d+)\s*\*\s*time\.Second/)?.[1])

assert(maxMessage === 4194304, `unexpected protocol v2 message limit ${maxMessage}`)
assert(concurrency === 16, `unexpected protocol v2 concurrency limit ${concurrency}`)
assert(timeout === 5, `unexpected protocol v2 request timeout ${timeout}`)
for (const value of ['4194304', String(concurrency), `${timeout} seconds`]) {
  assert(doc.includes(value), `Host API v2 docs omit current safety value ${value}`)
}

for (const invariant of [
  'base.AutoMTLS = true',
  'plugin.ProtocolGRPC',
  'hostAPIV2Version     = "sforum.host/v2"',
  'hostAPIV2Contract    = "sforum.host@2"',
  'hostAPIV2Legacy      = "sforum.host-api@2"',
  'HostBrokerId:',
  'RuntimeToken:'
]) {
  assert(clientRuntime.includes(invariant), `runtime no longer contains documented invariant: ${invariant}`)
}

for (const phrase of [
  'AutoMTLS',
  'no downgrade',
  'go-plugin v1.8.0',
  'PLUGIN_CLIENT_CERT',
  'SFORUM_PLUGIN=sforum-plugin-v1',
  'plugin.GRPCBroker/StartStream',
  'previously trusted v1',
  'thin Go launcher built against',
  '"hostApiVersion": "sforum.host@2"',
  '`sforum.host@2`',
  '`sforum.host-api@2`',
  '`sforum.host/v2`'
]) {
  assert(doc.includes(phrase), `Host API v2 docs omit required interoperability rule: ${phrase}`)
}

assert(guide.includes('[Host API v2 and non-Go runtimes](./host-api-v2.md)'), 'authoring guide must link the Host API v2 guide')
assert(testScript.includes('node tests/validate-host-api-v2-docs.mjs'), 'repository test gate must run Host API v2 documentation validation')

console.log('Host API v2 SDK and non-Go interoperability documentation validated.')
