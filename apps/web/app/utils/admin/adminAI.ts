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

// normalizedAISettingsForComparison 返回可逐字比较的副本：把「使用默认」的提示词
// 统一写成空串，其余字段原样保留。
export function normalizedAISettingsForComparison<T extends AIComparableSettings>(
  settings: T,
  defaultPrompt: string
): T {
  const copy = JSON.parse(JSON.stringify(settings)) as T
  const prompt = typeof copy.reply?.systemPrompt === 'string' ? copy.reply.systemPrompt : ''
  if (promptUsesDefault(prompt, defaultPrompt)) {
    copy.reply = { systemPrompt: '' }
  }
  return copy
}
