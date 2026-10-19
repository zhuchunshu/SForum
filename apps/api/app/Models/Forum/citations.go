package forum

import (
	"context"
	"errors"
)

// Citation 是 AI 产出评论的一条依据。它与 aireply.Citation 是同一份数据的不同
// 消费面：aireply 侧负责「校验并写入」，Forum 侧负责「公开读取」。
//
// 为什么在 Forum 里再定义一次而不是 import aireply：Forum 是更底层的领域包，
// 让论坛读路径依赖 AI 回复包会把依赖方向倒过来（AI 已经依赖 Forum）。两个结构
// 的 JSON 字段名必须保持一致——那是同一个对前端（T6）的契约：
// topicId / commentId / excerpt，全部 omitempty。
type Citation struct {
	TopicID   int64  `json:"topicId,omitempty"`
	CommentID int64  `json:"commentId,omitempty"`
	Excerpt   string `json:"excerpt,omitempty"`
}

// CitationList 是一条评论的全部依据。零引用时 Items 是空切片（不是 nil），
// 序列化成 `{"items":[]}`——前端据「有没有条目」决定渲不渲染折叠块。
type CitationList struct {
	Items []Citation `json:"items"`
}

// CitationReader 读取某条评论作为 AI 产出时留下的依据。
//
// 它由 aireply 的引用存储提供（PostgresCitationStore），经装配注入。未装配时
// 端点返回空列表而不是错误：引用是增强面，缺它不该让一个公开读端点 500。
type CitationReader interface {
	ListBySourceComment(ctx context.Context, sourceCommentID int64) ([]Citation, error)
}

// CitationService 是 AI 依据读取的聚焦协作者。
//
// 它刻意**不是** *Service 上的接收者方法集：Forum 的 Service 已经在
// receiverMethodCaps 上限（77），而 AGENTS.md 要求「一个 Service 跨多个独立能力
// 时必须变成聚焦协作者的兼容门面」。本文件是那个协作者，形状与
// TaxonomyService / EditableSourceService / ReferenceSelector 一致：
// 持有 *Service、只通过 s.service.store 取数、不往 Service 上加方法或字段。
type CitationService struct {
	service *Service
	// reader 可选：nil 时该方法仍会过可见性判定（不成为枚举豁免口），然后
	// 返回空列表。
	reader CitationReader
}

func NewCitationService(service *Service) *CitationService {
	return &CitationService{service: service}
}

// WithReader 注入引用读取能力。返回同一协作者以便链式装配。
func (s *CitationService) WithReader(reader CitationReader) *CitationService {
	if s == nil {
		return s
	}
	s.reader = reader
	return s
}

// CommentCitations 返回一条评论的依据列表。
//
// 可见性判定**完全复用评论本身那套**（与 ListCommentRepliesForViewer 同一个
// 顺序）：先经评论摘要追溯到所属主题，再走 GetTopic 的公开可见性规则。任何
// 一步不可见都返回 ErrCommentNotFound——**不区分「不存在」与「不可见」**，
// 因为这条区分会让 commentID 枚举成为探测隐藏主题的手段。
//
// 关于 viewer：本方法的可见性**只由主题级公开判定决定**，与 viewer 的登录态
// 无关——`store.GetTopic` 的谓词是 `status IN (active,locked) AND deleted_at IS
// NULL AND categories.visibility='public'`，它在匿名与登录两种情形下返回同一
// 结论。因此本方法不接收 viewer 参数（接收却不用是代码味道，读代码的人会以为
// 这里本该有身份判断）。身份只在控制器层用于 `requireGuestRead` 与
// `OptionalActor`（与 replies 同族），不参与引用可见性。
//
// 若将来出现「登录成员能看到 AI 在私密板块的依据」这种需求，那必须给
// 主题级判定本身加 actor 维度（与公开读同一条改造），而不是在这里补一个分支。
func (s *CitationService) CommentCitations(ctx context.Context, commentID int64) (CitationList, error) {
	empty := CitationList{Items: []Citation{}}
	if commentID <= 0 {
		return CitationList{}, ErrCommentNotFound
	}
	if s == nil || s.service == nil {
		return CitationList{}, ErrTopicNotFound
	}
	// 可见性先于读取：即使没有 reader，也必须过这一关，否则这个端点会成为
	// 「用 commentID 探测隐藏主题」的绕道。
	if _, err := s.visibleCommentTopic(ctx, commentID); err != nil {
		return CitationList{}, err
	}
	if s.reader == nil {
		return empty, nil
	}
	items, err := s.reader.ListBySourceComment(ctx, commentID)
	if err != nil {
		return CitationList{}, err
	}
	if items == nil {
		return empty, nil
	}
	return CitationList{Items: items}, nil
}

// visibleCommentTopic 是引用读取的唯一可见性入口：评论存在、其所属主题对访客
// 公开可见。
//
// 它与 ListCommentRepliesForViewer 的前两段同形——两套判定必然分叉成「回复列表
// 看不到、依据列表看得到」这种自相矛盾，所以刻意复用同一条 store 路径。
func (s *CitationService) visibleCommentTopic(ctx context.Context, commentID int64) (CommentSummary, error) {
	summary, err := s.service.store.GetCommentSummary(ctx, commentID)
	if err != nil {
		// 归一为评论不存在：GetCommentSummary 对不存在的行返回 ErrCommentNotFound，
		// 对真实故障返回别的错误。前者归一，后者照旧冒泡（那是 500，不是 404）。
		if errors.Is(err, ErrCommentNotFound) {
			return CommentSummary{}, ErrCommentNotFound
		}
		return CommentSummary{}, err
	}
	if _, err := s.service.store.GetTopic(ctx, summary.TopicID); err != nil {
		if errors.Is(err, ErrTopicNotFound) {
			// 主题不可见对调用方的结论必须是「这条评论不存在」：两种错误文本不同
			// 会让枚举者据此区分「评论存在但主题隐藏」，而 API 响应体里的 message
			// 就是这个错误字符串。这正是 replies 端点要防的那条泄漏。
			return CommentSummary{}, ErrCommentNotFound
		}
		return CommentSummary{}, err
	}
	return summary, nil
}
