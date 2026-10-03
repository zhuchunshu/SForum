package ai

// DiagnoseResult 是一次连通性探测的结论。失败时带上足够定位问题的信息，
// 而不是一个笼统的错误——诊断的价值全在这里。
type DiagnoseResult struct {
	OK           bool   `json:"ok"`
	ProviderID   string `json:"providerId,omitempty"`
	Model        string `json:"model,omitempty"`
	Protocol     string `json:"protocol,omitempty"`
	Reply        string `json:"reply,omitempty"`
	LatencyMS    int64  `json:"latencyMs,omitempty"`
	InputTokens  int    `json:"inputTokens,omitempty"`
	OutputTokens int    `json:"outputTokens,omitempty"`
	// Reason 是稳定原因码，供前端选择文案；Error 是给人看的说明。
	Reason string `json:"reason,omitempty"`
	Error  string `json:"error,omitempty"`
}

// 诊断结论的原因码。前三项是配置问题，不需要真的发请求就能判定。
const (
	DiagnoseReasonDisabled       = "gateway_disabled"
	DiagnoseReasonNoProvider     = "profile_unavailable"
	DiagnoseReasonNoCredential   = "credential_missing"
	DiagnoseReasonProviderFailed = "provider_failed"
)

// ReplyBotInfo 描述当前充当 AI 助手的账号。它让控制台能回答「到底用哪个账号
// 发言」，而不是让运营者去猜。
type ReplyBotInfo struct {
	UserID   int64  `json:"userId,omitempty"`
	Username string `json:"username,omitempty"`
	// Configured 为 false 表示站内还没有机器人账号，回复功能不会生效。
	Configured bool `json:"configured"`
}

// DiagnosePurpose 是探测调用的用途标识。它出现在执行记录里，让运维探测与
// 真实业务调用在用量表中可区分。
const DiagnosePurpose = "admin.connectivity_test"

// DefaultDiagnoseMessage 是未指定内容时的探测消息：越短越省钱，且要求一个
// 确定性的短回复，便于人工一眼判断是否正常。
const DefaultDiagnoseMessage = "请只回复两个字：收到"

// DiagnosePromptVersion 参与缓存键与执行记录，改动探测提示词时递增。
const DiagnosePromptVersion = "diagnose@1"
