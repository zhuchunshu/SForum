package aiderivedjobs

import (
	"testing"

	"github.com/riverqueue/river"
)

// 两个对外的 args 必须各自实现 river.JobArgs（编译期断言），且 kind 固定、互不相同。
var (
	_ river.JobArgs = (DigestArgs{})
	_ river.JobArgs = (TourArgs{})
)

// 编译期守卫：deriveRequest 绝不能实现 river.JobArgs。
//
// 这条约束不是风格偏好。River 在 `AddWorkerSafely[T]` 里取 `var jobArgs T` 的
// **零值**决定 kind（`river@v0.40.0/worker.go:133-136`），因此一个「Kind() 取决于
// 字段值」的 args 类型有两种失败形态：
//
//  1. 注册侧：零值 Purpose 为空 → 只注册出 ai.forum_digest，ai.forum_tour 入队后
//     被丢弃（`job kind is not registered in the client's Workers bundle`）。
//  2. payload 侧：Enqueuer 用新形状的 args 入队（无 purpose 字段），worker 却按
//     kind 绑定了旧类型 → 反序列化后 Purpose 为空 → 产物被判非法用途而 cancel
//     （`artifact is invalid: unsupported purpose ""`）。这一种在纯单测里完全看不见，
//     只有在真实 River 上跑端到端才会现形。
//
// 把 deriveRequest 降级为纯载体（无 Kind/EnqueueOptions）同时封死两条路。
// Go 没有「类型不实现某接口」的编译期断言，因此这里用运行期断言 + 下面一条失败即
// 说明有人给它加回了 Kind()。
func TestDeriveRequestCarriesNoKind(t *testing.T) {
	if _, ok := any(deriveRequest{}).(river.JobArgs); ok {
		t.Fatal("deriveRequest must stay a plain carrier; a conditional Kind() reintroduces the discarded/mismatch failure")
	}
	if (DigestArgs{}).Kind() == (TourArgs{}).Kind() {
		t.Fatal("the two purposes must not share a job kind")
	}
	if got := (DigestArgs{}).Kind(); got != KindDigest {
		t.Fatalf("digest kind = %q, want %q", got, KindDigest)
	}
	if got := (TourArgs{}).Kind(); got != KindTour {
		t.Fatalf("tour kind = %q, want %q", got, KindTour)
	}
}

// 两种 args 到内部载体的转换必须只补 purpose，不丢字段：丢了 RequestID 会让
// 「同一次触发只产出一份草稿」这条保证失效，丢了 TopicID 会让任务读错楼。
func TestDeriveRequestConversionPreservesEveryField(t *testing.T) {
	digest := (DigestArgs{TopicID: 42, RequestedBy: 7, RequestID: "r1"}).deriveRequest()
	if digest.Purpose != "forum.digest" || digest.TopicID != 42 || digest.RequestedBy != 7 || digest.RequestID != "r1" {
		t.Fatalf("digest carrier = %+v", digest)
	}
	tour := (TourArgs{TopicID: 43, RequestedBy: 8, RequestID: "r2"}).deriveRequest()
	if tour.Purpose != "forum.tour" || tour.TopicID != 43 || tour.RequestedBy != 8 || tour.RequestID != "r2" {
		t.Fatalf("tour carrier = %+v", tour)
	}
}
