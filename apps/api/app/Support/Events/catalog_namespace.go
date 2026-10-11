package events

import "strings"

// OwnsEventName 判断事件名是否严格落在 extension 命名空间下：以
// `<extensionID>.` 开头且不是 extensionID 本身。发布侧与声明侧共用同一把尺，
// 防止插件能声明一个事件却发不出去（或反之，发到别人的命名空间）。
func OwnsEventName(extensionID, eventName string) bool {
	extensionID = strings.TrimSpace(extensionID)
	eventName = strings.TrimSpace(eventName)
	if extensionID == "" || eventName == "" || eventName == extensionID {
		return false
	}
	return strings.HasPrefix(eventName, extensionID+".") && len(eventName) > len(extensionID)+1
}

// ResolveDefinition 解析一个事件名的权威定义：先命中 Core 目录；未命中时按
// 插件自有命名空间合成 observe 定义（fail-open，使用默认异步超时）。命名空间
// 不属于调用方的插件事件一律拒绝；Core 事件与命名空间无关，始终可用。
func ResolveDefinition(eventName, extensionID string) (Definition, bool) {
	if definition, ok := FindDefinition(eventName); ok {
		return definition, true
	}
	if !OwnsEventName(extensionID, eventName) {
		return Definition{}, false
	}
	return normalizeDefinition(Definition{
		Name: eventName, Kind: KindObserve, FailurePolicy: FailurePolicyFailOpen,
	}), true
}
