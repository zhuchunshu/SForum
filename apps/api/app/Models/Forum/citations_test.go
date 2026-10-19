package forum

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// fakeCitationReader 只实现引用读取这一条路径。CitationService 的可见性判定走
// s.service.store（真实 store），所以这里的 fake 不参与可见性——这正是设计意图：
// 读取能力与可见性权威分离，注入一个 fake 不能绕过主题级判定。
type fakeCitationReader struct {
	items  []Citation
	err    error
	called int
	// gotID 记录被查询的产出评论 id，用来断言「可见性在前、读取在后」。
	gotID int64
}

func (f *fakeCitationReader) ListBySourceComment(_ context.Context, sourceCommentID int64) ([]Citation, error) {
	f.called++
	f.gotID = sourceCommentID
	return f.items, f.err
}

// 零引用 / 未装配都必须返回空切片而不是 nil：序列化成 `{"items":[]}` 而不是
// `{"items":null}`，前端据「有没有条目」决定渲染。
func TestCitationServiceUnwiredReturnsEmptyList(t *testing.T) {
	service := &CitationService{service: &Service{store: stubCitationStore{}}}
	list, err := service.CommentCitations(context.Background(), 501)
	if err != nil {
		t.Fatalf("CommentCitations: %v", err)
	}
	if list.Items == nil {
		t.Fatal("items must be an empty slice, not nil")
	}
	if len(list.Items) != 0 {
		t.Fatalf("items = %+v", list.Items)
	}
	encoded, err := json.Marshal(list)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"items":[]}` {
		t.Fatalf("wire shape = %s", encoded)
	}
}

// 非法 commentID 直接 404，不触达存储。
func TestCitationServiceRejectsInvalidCommentID(t *testing.T) {
	service := &CitationService{service: &Service{store: stubCitationStore{}}}
	for _, id := range []int64{0, -1} {
		if _, err := service.CommentCitations(context.Background(), id); !errors.Is(err, ErrCommentNotFound) {
			t.Fatalf("id=%d err = %v, want ErrCommentNotFound", id, err)
		}
	}
}

// 核心不变量：**可见性先于读取**。主题不可见时即使注入了 reader，也一次都不该
// 被调用——否则这个端点就成了用 commentID 探测隐藏主题的绕道。
func TestCitationServiceChecksVisibilityBeforeReading(t *testing.T) {
	reader := &fakeCitationReader{items: []Citation{{TopicID: 9, CommentID: 913, Excerpt: "内容"}}}
	service := &CitationService{
		service: &Service{store: stubCitationStore{invisibleTopic: true}},
		reader:  reader,
	}
	_, err := service.CommentCitations(context.Background(), 501)
	if !errors.Is(err, ErrCommentNotFound) && !errors.Is(err, ErrTopicNotFound) {
		t.Fatalf("err = %v, want a not-found error", err)
	}
	if reader.called != 0 {
		t.Fatalf("the reader must not run for an invisible topic: called=%d", reader.called)
	}
}

// 评论不存在与主题不可见给出同一个 not-found 结论：不区分，否则枚举就能探测存在性。
func TestCitationServiceDoesNotDistinguishMissingFromInvisible(t *testing.T) {
	missing := &CitationService{service: &Service{store: stubCitationStore{missingComment: true}}}
	invisible := &CitationService{service: &Service{store: stubCitationStore{invisibleTopic: true}}}

	_, errMissing := missing.CommentCitations(context.Background(), 501)
	_, errInvisible := invisible.CommentCitations(context.Background(), 501)
	if errMissing == nil || errInvisible == nil {
		t.Fatalf("both must fail: missing=%v invisible=%v", errMissing, errInvisible)
	}
	// 错误必须是同一族（同一个 sentinel 或其包装），不泄漏哪一侧失败。
	if !errors.Is(errMissing, ErrCommentNotFound) || !errors.Is(errInvisible, ErrCommentNotFound) {
		t.Fatalf("errors must be indistinguishable: missing=%v invisible=%v", errMissing, errInvisible)
	}
}

// 可见评论：返回 reader 给的依据，顺序保持。
func TestCitationServiceReturnsReaderItemsInOrder(t *testing.T) {
	reader := &fakeCitationReader{items: []Citation{
		{TopicID: 9, CommentID: 913, Excerpt: "第一处"},
		{TopicID: 9, Excerpt: "主题级引用"},
	}}
	service := &CitationService{service: &Service{store: stubCitationStore{}}, reader: reader}
	list, err := service.CommentCitations(context.Background(), 501)
	if err != nil {
		t.Fatalf("CommentCitations: %v", err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("items = %+v", list.Items)
	}
	if list.Items[0].CommentID != 913 || list.Items[1].CommentID != 0 {
		t.Fatalf("order = %+v", list.Items)
	}
	if reader.gotID != 501 {
		t.Fatalf("the reader must be queried with the produced comment id: %d", reader.gotID)
	}
}

// reader 故障要冒泡成错误（让控制器 500），不能被吞成空列表——空列表意味着
// 「这条评论没有依据」，那是一句具体的话，不该用来掩盖一次数据库故障。
func TestCitationServiceSurfacesReaderFailure(t *testing.T) {
	reader := &fakeCitationReader{err: errors.New("pool exhausted")}
	service := &CitationService{service: &Service{store: stubCitationStore{}}, reader: reader}
	if _, err := service.CommentCitations(context.Background(), 501); err == nil {
		t.Fatal("a reader failure must not be reported as an empty citation list")
	}
}

// JSON 字段名是给前端（T6）的稳定契约，必须与 aireply.Citation 逐字一致。
func TestForumCitationJSONContractMatchesAIReply(t *testing.T) {
	encoded, err := json.Marshal(Citation{TopicID: 42, CommentID: 913, Excerpt: "内容"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"topicId":42`, `"commentId":913`, `"excerpt":"内容"`} {
		if !strings.Contains(string(encoded), want) {
			t.Fatalf("citation JSON %s is missing %s", encoded, want)
		}
	}
	encoded, err = json.Marshal(Citation{TopicID: 42})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "commentId") {
		t.Fatalf("a topic-only citation must omit commentId: %s", encoded)
	}
}

// stubCitationStore 是最小的 store 替身：只让 GetCommentSummary / GetTopic 两条
// 路径可用。其余方法由内嵌的 nil 接口提供——被调用时会 panic，因此这同时是一条
// 断言：CitationService 不得触碰 store 的其他能力。
type stubCitationStore struct {
	Store
	missingComment bool
	invisibleTopic bool
}

func (s stubCitationStore) GetCommentSummary(_ context.Context, commentID int64) (CommentSummary, error) {
	if s.missingComment {
		return CommentSummary{}, ErrCommentNotFound
	}
	return CommentSummary{ID: commentID, TopicID: 9, Status: CommentStatusActive}, nil
}

func (s stubCitationStore) GetTopic(_ context.Context, topicID int64) (TopicDetail, error) {
	if s.invisibleTopic {
		return TopicDetail{}, ErrTopicNotFound
	}
	summary := TopicSummary{}
	summary.ID = topicID
	return TopicDetail{TopicSummary: summary}, nil
}

// fakeToolCallReader 只实现工具轨迹读取。与 fakeCitationReader 一样不参与可见性：
// 可见性权威始终在 s.service.store。
type fakeToolCallReader struct {
	items  []ToolCallEntry
	err    error
	called int
}

func (f *fakeToolCallReader) ListBySourceComment(_ context.Context, _ int64) ([]ToolCallEntry, error) {
	f.called++
	return f.items, f.err
}

// 公开投影的边界：四种「模型撞了配置」的结局必须消失，真实尝试必须留下。
func TestPublicToolCallsFiltersConfigurationMisses(t *testing.T) {
	items := []ToolCallEntry{
		{Tool: "web-search", Arguments: `{"query":"SForum"}`, LatencyMS: 120},
		{Tool: "forum-search", Failure: toolFailureNotAllowed},
		{Tool: "made-up", Failure: toolFailureUnknownTool},
		{Tool: "web-search", Failure: toolFailureStepExhausted},
		{Tool: "fetch-url", Failure: toolFailureBudgetExhausted},
		{Tool: "web-search", IsError: true, Failure: "internal_error"},
		{Tool: "  "},
	}
	visible := publicToolCalls(items)
	if len(visible) != 2 {
		t.Fatalf("visible = %+v", visible)
	}
	if visible[0].Tool != "web-search" || visible[0].IsError {
		t.Fatalf("第一次调用应保留：%+v", visible[0])
	}
	if visible[1].Tool != "web-search" || !visible[1].IsError {
		t.Fatalf("真实失败的尝试必须保留：%+v", visible[1])
	}
}

// 工具调用是 omitempty 分区：零调用时字段不出现，与「这个能力没装配」同形。
func TestCitationServiceOmitsToolCallsWhenThereAreNone(t *testing.T) {
	service := &CitationService{
		service:   &Service{store: stubCitationStore{}},
		reader:    &fakeCitationReader{},
		toolCalls: &fakeToolCallReader{},
	}
	list, err := service.CommentCitations(context.Background(), 501)
	if err != nil {
		t.Fatalf("CommentCitations: %v", err)
	}
	encoded, err := json.Marshal(list)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"items":[]}` {
		t.Fatalf("零工具调用时不得出现 toolCalls 字段：%s", encoded)
	}
}

// 工具轨迹读失败只丢那个分区：依据一直是这个端点的主体，不该被一起拖垮。
func TestCitationServiceKeepsCitationsWhenToolCallsFail(t *testing.T) {
	service := &CitationService{
		service:   &Service{store: stubCitationStore{}},
		reader:    &fakeCitationReader{items: []Citation{{TopicID: 9, CommentID: 913, Excerpt: "内容"}}},
		toolCalls: &fakeToolCallReader{err: errors.New("db down")},
	}
	list, err := service.CommentCitations(context.Background(), 501)
	if err != nil {
		t.Fatalf("CommentCitations: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("items = %+v", list.Items)
	}
	if list.ToolCalls != nil {
		t.Fatalf("toolCalls = %+v", list.ToolCalls)
	}
}

// 可见性仍然先于两个读取：不可见主题一次都不该触达工具轨迹存储。
func TestCitationServiceChecksVisibilityBeforeReadingToolCalls(t *testing.T) {
	tools := &fakeToolCallReader{items: []ToolCallEntry{{Tool: "web-search"}}}
	service := &CitationService{
		service:   &Service{store: stubCitationStore{invisibleTopic: true}},
		toolCalls: tools,
	}
	if _, err := service.CommentCitations(context.Background(), 501); err == nil {
		t.Fatal("invisible topic must not return citations")
	}
	if tools.called != 0 {
		t.Fatalf("工具轨迹在可见性判定之前被读取了 %d 次", tools.called)
	}
}
