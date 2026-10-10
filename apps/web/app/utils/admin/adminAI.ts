// AI 面板的比较与归一化：它决定「提示词算不算自定义」与「面板算不算脏」。
//
// 为什么需要归一化：富文本编辑器会把 Markdown 重新排版（段落与列表之间补空行、
// 去掉行尾空白），所以「与内置默认逐字相同」在编辑器往返之后必然不成立。直接拿
// 原字符串比较，会让面板一打开就显示「已自定义 / 有未保存的改动」，而运营者什么
// 都没改。这里把空白差异视为相同，并把「空 / 等于默认」统一规范成同一个形状，
// 使服务端文档与表单副本可比。
//
// 服务端在提示词为空时会省略该字段（omitempty），表单侧则可能持有显式空串——
// 两者语义相同，必须在比较前对齐，否则一次「清空提示词」也会被当成未保存改动。

export type AIComparableSettings = {
  reply?: { systemPrompt?: string }
  gates?: { toolCallsPerReply?: number | null }
}

// DEFAULT_TOOL_CALLS_PER_REPLY 与后端 ai.DefaultToolCallsPerReply 一致，仅用于
// 界面展示：真正的缺省仍然由服务端的 nil 语义决定。
export const DEFAULT_TOOL_CALLS_PER_REPLY = 3

// toolCallsPerReplyUsesDefault 判断工具步数是否等价于「未设置」。
// 服务端用指针字段区分 nil（按默认 3 生效）与 0（停用工具调用）：一旦运营者碰过
// 这个输入框，nil 就会被固化成一个数字，之后升级默认也影响不到这个站点。
export function toolCallsPerReplyUsesDefault(value: number | null | undefined): boolean {
  return value === undefined || value === null || Number.isNaN(value)
}

// promptUsesDefault 判断提示词是否等价于「使用内置默认」：空字符串，或忽略空白
// 差异后与默认提示词相同。
export function promptUsesDefault(prompt: string | undefined, defaultPrompt: string): boolean {
  const normalize = (value: string) => value.replace(/\s+/g, ' ').trim()
  const current = normalize(prompt ?? '')
  if (current === '') {
    return true
  }
  return current === normalize(defaultPrompt ?? '')
}

export type AIModelOption = { label: string, value: string }

export type AIModelSelection = {
  options: AIModelOption[]
  /** 当前模型是否出现在供应商返回的列表里。 */
  currentListed: boolean
}

// modelOptionsForSelect 把供应商返回的模型列表变成下拉选项，并保证「当前值」始终
// 在选项里：供应商可能已经下线了站点正在使用的模型，把它过滤掉会让下拉框看起来
// 「什么都没选中」，而配置里明明写着一个模型。currentListed 让调用方能就这一点
// 单独提示，而不是把差异藏起来。
//
// 顺序保持供应商给出的顺序（多数供应商把最新或最常用的排在前面），并去掉重复与
// 空白条目——重复只会让下拉框变长，不会带来新信息。
export function modelOptionsForSelect(models: readonly string[], current: string): AIModelSelection {
  const seen = new Set<string>()
  const options: AIModelOption[] = []
  for (const entry of models ?? []) {
    const value = typeof entry === 'string' ? entry.trim() : ''
    if (!value || seen.has(value)) continue
    seen.add(value)
    options.push({ label: value, value })
  }
  const selected = (current ?? '').trim()
  const currentListed = selected !== '' && seen.has(selected)
  if (selected !== '' && !currentListed) {
    options.push({ label: selected, value: selected })
  }
  return { options, currentListed }
}

// normalizedAISettingsForComparison 返回可逐字比较的副本：把「使用默认」的提示词
// 统一写成空串，其余字段原样保留。
//
// toolCallsPerReply 必须显式对齐：JSON.stringify 会丢掉 undefined，nil（未设置）
// 与 3（显式等于默认）在字符串层面不可区分，而它们对服务端是两种语义。
export function normalizedAISettingsForComparison<T extends AIComparableSettings>(
  settings: T,
  defaultPrompt: string
): T {
  const copy = JSON.parse(JSON.stringify(settings)) as T
  const prompt = typeof copy.reply?.systemPrompt === 'string' ? copy.reply.systemPrompt : ''
  if (promptUsesDefault(prompt, defaultPrompt)) {
    copy.reply = { systemPrompt: '' }
  }
  if (copy.gates && 'toolCallsPerReply' in copy.gates) {
    if (toolCallsPerReplyUsesDefault(copy.gates.toolCallsPerReply)) {
      delete copy.gates.toolCallsPerReply
    } else {
      // 显式值统一成数字，避免 "3" 与 3 被当成改动。
      copy.gates.toolCallsPerReply = Number(copy.gates.toolCallsPerReply)
    }
  }
  return copy
}
